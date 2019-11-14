// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package cloudkms implements decryption of Cloud KMS encrypted ciphertext.
package cloudkms

import (
	"context"
	"encoding/base64"
	"time"

	cloudkms "google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/option"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/retry"
	"go.chromium.org/luci/common/retry/transient"
)

// Client provides a high level Cloud KMS interface.
type Client interface {
	Decrypt(ctx context.Context, ciphertext string) ([]byte, error)
}

type client struct {
	service *cloudkms.Service
	keyPath string
}

// NewClient creates a new Cloud KMS client.
func NewClient(ctx context.Context, o auth.Options, keyPath string) (Client, error) {
	o.Scopes = append(o.Scopes, cloudkms.CloudkmsScope)
	c, err := auth.NewAuthenticator(ctx, auth.SilentLogin, o).Client()
	if err != nil {
		return nil, errors.Annotate(err, "create new Cloud KMS client").Err()
	}
	s, err := cloudkms.NewService(ctx, option.WithHTTPClient(c))
	if err != nil {
		return nil, errors.Annotate(err, "create new Cloud KMS client").Err()
	}
	return &client{
		service: s,
		keyPath: keyPath,
	}, nil
}

// Decrypt decrypts a base64 encoded ciphertext that was previously encrypted
// using a key in Cloud KMS.
func (c *client) Decrypt(ctx context.Context, ciphertext string) ([]byte, error) {
	req := cloudkms.DecryptRequest{
		Ciphertext: ciphertext,
	}
	var resp *cloudkms.DecryptResponse
	err := retry.Retry(ctx, transient.Only(retry.Default), func() error {
		var err error
		resp, err = c.service.Projects.Locations.KeyRings.CryptoKeys.
			Decrypt(c.keyPath, &req).Context(ctx).Do()
		return err
	}, func(
		err error,
		d time.Duration,
	) {
		logging.Warningf(ctx, "Transient error while making request, retrying in %s...", d)
	})
	if err != nil {
		return nil, errors.Annotate(err, "decrypt ciphertext using Cloud KMS").Err()
	}
	decoded, err := base64.StdEncoding.DecodeString(resp.Plaintext)
	if err != nil {
		return nil, errors.Annotate(err, "decrypt ciphertext using Cloud KMS").Err()
	}
	return decoded, nil
}
