// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cipd

import (
	"context"
	"path/filepath"

	"go.chromium.org/luci/auth"
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
	EnsurePackages(ctx context.Context, packages common.PinSliceBySubdir, opts *cipd.EnsureOptions) (cipd.ActionMap, error)
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
			authClient, err := auth.NewAuthenticator(ctx, auth.SilentLogin, auth.Options{}).Client()
			if err != nil {
				return nil, errors.Annotate(err, "could not initialize auth client").Err()
			}
			return cipd.NewClient(cipd.ClientOptions{
				ServiceURL:          chromeinfra.CIPDServiceURL,
				Root:                cipdRoot,
				AuthenticatedClient: authClient,
			})
		}
	}
	cipdClient, err := factory(ctx, cipdRoot)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get recipe client for CIPD root: <%s>", cipdRoot).Err()
	}
	if cipdClient == nil {
		return nil, errors.New("nil CIPD client returned from factory")
	}
	return &Client{cipdRoot, cipdClient}, nil
}

// ResolveVersion resolves the requested version of a package to an instance ID,
// returning the pin for the instance.
func (c *Client) ResolveVersion(ctx context.Context, name, version string) (common.Pin, error) {
	return c.client.ResolveVersion(ctx, name, version)
}

// DownloadPackage downloads and installs the CIPD package with the given name
// and instance ID and returns the path to the deployed package.
func (c *Client) DownloadPackage(ctx context.Context, name, instanceId, subdir string) (string, error) {
	pin := common.Pin{
		PackageName: name,
		InstanceID:  instanceId,
	}
	packages := common.PinSliceBySubdir{subdir: common.PinSlice{pin}}
	if _, err := c.client.EnsurePackages(ctx, packages, &cipd.EnsureOptions{
		Paranoia: cipd.CheckIntegrity,
	}); err != nil {
		return "", err
	}
	return filepath.Join(c.cipdRoot, subdir), nil
}
