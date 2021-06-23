// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recipe

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"go.chromium.org/luci/cipd/client/cipd"
	"go.chromium.org/luci/cipd/common"
	"go.chromium.org/luci/common/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
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
			return cipd.NewClient(cipd.ClientOptions{Root: cipdRoot})
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

// SetupRecipe downloads and installs the recipe package with the given name at
// the given version and returns the path to the recipes.py script.
func (c *Client) SetupRecipe(ctx context.Context, name, version string) (string, error) {
	pin, err := c.client.ResolveVersion(ctx, name, version)
	if err != nil {
		return "", err
	}
	packages := common.PinSliceBySubdir{name: common.PinSlice{pin}}
	if _, err := c.client.EnsurePackages(ctx, packages, cipd.CheckIntegrity, 0, false); err != nil {
		return "", err
	}
	recipeRoot := filepath.Join(c.cipdRoot, name)
	recipesCfgContents, err := ioutil.ReadFile(filepath.Join(recipeRoot, "infra", "config", "recipes.cfg"))
	if err != nil {
		return "", errors.Annotate(err, "could not read recipes.cfg").Err()
	}
	recipesCfg := &structpb.Struct{}
	if err := protojson.Unmarshal(recipesCfgContents, recipesCfg); err != nil {
		return "", err
	}
	var recipesPath string
	if recipesPathField, ok := recipesCfg.Fields["recipes_path"]; ok {
		if value, ok := recipesPathField.Kind.(*structpb.Value_StringValue); ok {
			recipesPath = value.StringValue
		} else {
			return "", errors.Reason(`unexpected type for "recipes_path" in infra/config/recipes.cfg: %T`, value).Err()
		}
	}
	return filepath.Join(recipeRoot, filepath.FromSlash(recipesPath), "recipes.py"), nil
}
