// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cas

import (
	"context"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/filemetadata"
	"go.chromium.org/luci/client/casclient"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/hardcoded/chromeinfra"
	apipb "go.chromium.org/luci/swarming/proto/api"
)

// Client provides the CAS-related operations required for bootstrapping.
type Client struct {
	execRoot string
	clients  map[string]CasClient
	factory  CasClientFactory
}

// CasClient provides a subset of the client.Client interface.
type CasClient interface {
	DownloadDirectory(ctx context.Context, d digest.Digest, execRoot string, cache filemetadata.Cache) (map[string]*client.TreeOutput, *client.MovedBytesMetadata, error)
}

// Enforce that the CasClient interface is a subset of the client.Client
// interface.
var _ CasClient = (*client.Client)(nil)

// CasClientFactory creates the client for accessing CAS that will deploy
// packages to the directory identified by cipdRoot.
type CasClientFactory func(ctx context.Context, instance string) (CasClient, error)

var ctxKey = "infra/chromium/bootstrapper/recipe.CasClientFactory"

// UseCasClientFactory returns a context that causes new Client instances to
// use the given factory when getting the CAS client.
func UseCasClientFactory(ctx context.Context, factory CasClientFactory) context.Context {
	return context.WithValue(ctx, &ctxKey, factory)
}

func NewClient(ctx context.Context, execRoot string) *Client {
	factory, _ := ctx.Value(&ctxKey).(CasClientFactory)
	if factory == nil {
		factory = func(ctx context.Context, instance string) (CasClient, error) {
			return casclient.NewLegacy(ctx, casclient.AddrProd, instance, chromeinfra.DefaultAuthOptions(), true)
		}
	}
	return &Client{execRoot, map[string]CasClient{}, factory}
}

func (c *Client) clientForInstance(ctx context.Context, instance string) (CasClient, error) {
	if client, ok := c.clients[instance]; ok {
		return client, nil
	}
	client, err := c.factory(ctx, instance)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errors.Reason("returned client for %s is nil", instance).Err()
	}
	c.clients[instance] = client
	return client, nil
}

// Download downloads a CAS blob with the given digest from the given CAS
// instance.
func (c *Client) Download(ctx context.Context, instance string, d *apipb.Digest) (string, error) {
	client, err := c.clientForInstance(ctx, instance)
	if err != nil {
		return "", err
	}
	casDigest := digest.Digest{
		Hash: d.Hash,
		Size: d.SizeBytes,
	}
	_, _, err = client.DownloadDirectory(ctx, casDigest, c.execRoot, filemetadata.NewNoopCache())
	if err != nil {
		return "", err
	}
	return c.execRoot, nil
}
