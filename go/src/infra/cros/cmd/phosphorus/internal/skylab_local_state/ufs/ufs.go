// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package ufs provides option t build connection to UFS service.
package ufs

import (
	"context"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/grpc/metadata"

	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsutil "infra/unifiedfleet/app/util"
)

// prpcOptions is used for UFS PRPC clients.
var prpcOptions = prpcOptionWithUserAgent("skylab_local_state/6.0.0")

// NewClient initialize and return new client to work with UFS service.
func NewClient(ctx context.Context, ufsService string, authFlags *authcli.Flags) (ufsAPI.FleetClient, error) {
	if ufsService == "" {
		return nil, errors.Reason("UFS service path not provided.").Err()
	}
	authOpts, err := authFlags.Options()
	if err != nil {
		return nil, errors.Annotate(err, "create UFS client").Err()
	}
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, authOpts)
	httpClient, err := a.Client()
	if err != nil {
		return nil, errors.Annotate(err, "create UFS client").Err()
	}
	ufsClient := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       httpClient,
		Host:    ufsService,
		Options: prpcOptions,
	})
	return ufsClient, nil
}

// prpcOptionWithUserAgent create prpc option with custom UserAgent.
//
// DefaultOptions provides Retry ability in case we have issue with service.
func prpcOptionWithUserAgent(userAgent string) *prpc.Options {
	options := *prpc.DefaultOptions()
	options.UserAgent = userAgent
	return &options
}

// SetupContext set up the outgoing context for API calls.
func SetupContext(ctx context.Context, namespace string) context.Context {
	md := metadata.Pairs(ufsutil.Namespace, namespace)
	return metadata.NewOutgoingContext(ctx, md)
}
