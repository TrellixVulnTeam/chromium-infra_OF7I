// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cipd

import (
	"context"
	"path/filepath"

	"go.chromium.org/luci/cipd/client/cipd"
	"go.chromium.org/luci/cipd/common"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

// Client provides the recipe-related operations required for bootstrapping.
type Client struct {
	cipdRoot string
	client   CipdClient
}

// CipdClient provides a subset of the cipd.Client interface
type CipdClient interface {
	ResolveVersion(ctx context.Context, packageName, version string) (common.Pin, error)
	EnsurePackages(ctx context.Context, packages common.PinSliceBySubdir, paranoia cipd.ParanoidMode, maxThreads int, dryRun bool) (cipd.ActionMap, error)
}

// Enforce that the CIPD client interface is a subset of the cipd.Client
// interface.
var _ CipdClient = (cipd.Client)(nil)

// CipdClientFactory creates the client for accessing CIPD that will
// deploy packages to the directory identified by cipdRoot.
type CipdClientFactory func(ctx context.Context, cipdRoot string) (CipdClient, error)

var ctxKey = "infra/chromium/bootstrapper/recipe.CipdClientFactory"

// UseCipdClientFactory returns a context that causes new Client instances to
// use the given factory when getting the CIPD client.
func UseCipdClientFactory(ctx context.Context, factory CipdClientFactory) context.Context {
	return context.WithValue(ctx, &ctxKey, factory)
}

// NewClient returns a new recipe client.
//
// If ctx is a context returned from UseCipdClientFactory, then the returned
// client will use the factory that was passed to UseCipdClientFactory to get a
// CIPD client. Otherwise, a client created using cipd.NewClient with default
// options will be used.
func NewClient(ctx context.Context, cipdRoot string) (*Client, error) {
	factory, _ := ctx.Value(&ctxKey).(CipdClientFactory)
	if factory == nil {
		factory = func(ctx context.Context, cipdRoot string) (CipdClient, error) {
			return cipd.NewClient(cipd.ClientOptions{
				ServiceURL: chromeinfra.CIPDServiceURL,
				Root:       cipdRoot,
			})
		}
	}
	cipdClient, err := factory(ctx, cipdRoot)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get recipe client for CIPD root: <%s>", cipdRoot).Err()
	}
	if cipdClient == nil {
		panic("nil CIPD client returned from factory")
	}
	return &Client{cipdRoot, cipdClient}, nil
}

// DownloadPackage downloads and installs the CIPD package with the given name
// at the given version and returns the path to the deployed package.
func (c *Client) DownloadPackage(ctx context.Context, name, version string) (string, error) {
	pin, err := c.client.ResolveVersion(ctx, name, version)
	if err != nil {
		return "", err
	}
	packages := common.PinSliceBySubdir{name: common.PinSlice{pin}}
	if _, err := c.client.EnsurePackages(ctx, packages, cipd.CheckIntegrity, 0, false); err != nil {
		return "", err
	}
	return filepath.Join(c.cipdRoot, name), nil
}
