// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recipe

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/cipd/client/cipd"
	"go.chromium.org/luci/cipd/common"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/common/testing/testfs"
)

type fakeCipdClient struct {
	resolveVersion func(context.Context, string, string) (common.Pin, error)
	ensurePackages func(context.Context, common.PinSliceBySubdir, cipd.ParanoidMode, int, bool) (cipd.ActionMap, error)
}

func (f *fakeCipdClient) ResolveVersion(ctx context.Context, packageName, version string) (common.Pin, error) {
	resolveVersion := f.resolveVersion
	if resolveVersion != nil {
		return resolveVersion(ctx, packageName, version)
	}
	return common.Pin{
		PackageName: packageName,
		InstanceID:  "fake-instance-id",
	}, nil
}

func (f *fakeCipdClient) EnsurePackages(ctx context.Context, packages common.PinSliceBySubdir, paranoia cipd.ParanoidMode, maxThreads int, dryRun bool) (cipd.ActionMap, error) {
	ensurePackages := f.ensurePackages
	if ensurePackages != nil {
		return ensurePackages(ctx, packages, paranoia, maxThreads, dryRun)
	}
	return nil, nil
}

func factoryForRecipesCfg(contents string) CipdClientFactory {
	return func(ctx context.Context, cipdRoot string) (CipdClient, error) {
		return &fakeCipdClient{
			ensurePackages: func(ctx context.Context, packages common.PinSliceBySubdir, paranoia cipd.ParanoidMode, maxThreads int, dryRun bool) (cipd.ActionMap, error) {
				layout := map[string]string{}
				for subdir := range packages {
					layout[strings.Join([]string{subdir, "infra", "config", "recipes.cfg"}, "/")] = contents
				}
				if err := testfs.Build(cipdRoot, layout); err != nil {
					panic(err)
				}
				return nil, nil
			},
		}, nil
	}
}

func TestClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Client", t, func() {

		Convey("NewClient", func() {

			Convey("fails if client factory fails", func() {
				factory := func(ctx context.Context, cipdRoot string) (CipdClient, error) {
					return &fakeCipdClient{}, nil
				}
				ctx := UseCipdClientFactory(ctx, factory)

				client, err := NewClient(ctx, "fake-root")

				So(err, ShouldBeNil)
				So(client, ShouldNotBeNil)
			})

			Convey("succeeds if factory succeeds", func() {
				factory := func(ctx context.Context, cipdRoot string) (CipdClient, error) {
					return nil, errors.New("test factory failure")
				}
				ctx := UseCipdClientFactory(ctx, factory)

				client, err := NewClient(ctx, "fake-root")

				So(err, ShouldErrLike, "test factory failure")
				So(client, ShouldBeNil)
			})

		})

		Convey("SetupRecipe", func() {

			cipdRoot := t.TempDir()

			Convey("fails if resolving version fails", func() {
				factory := func(ctx context.Context, cipdRoot string) (CipdClient, error) {
					return &fakeCipdClient{resolveVersion: func(ctx context.Context, packageName, version string) (common.Pin, error) {
						return common.Pin{}, errors.New("test ResolveVersion failure")
					}}, nil
				}
				ctx := UseCipdClientFactory(ctx, factory)
				client, _ := NewClient(ctx, cipdRoot)

				recipesPyPath, err := client.SetupRecipe(ctx, "fake-package", "fake-version")

				So(err, ShouldErrLike, "test ResolveVersion failure")
				So(recipesPyPath, ShouldBeEmpty)
			})

			Convey("fails if fetching and deploying instance fails", func() {
				factory := func(ctx context.Context, cipdRoot string) (CipdClient, error) {
					return &fakeCipdClient{ensurePackages: func(ctx context.Context, packages common.PinSliceBySubdir, paranoia cipd.ParanoidMode, maxThreads int, dryRun bool) (cipd.ActionMap, error) {
						return nil, errors.New("test EnsurePackages failure")
					}}, nil
				}
				ctx := UseCipdClientFactory(ctx, factory)
				client, _ := NewClient(ctx, cipdRoot)

				recipesPyPath, err := client.SetupRecipe(ctx, "fake-package", "fake-version")

				So(err, ShouldErrLike, "test EnsurePackages failure")
				So(recipesPyPath, ShouldBeEmpty)
			})

			Convey("fails if reading recipes.cfg fails", func() {
				factory := func(ctx context.Context, cipdRoot string) (CipdClient, error) {
					return &fakeCipdClient{}, nil
				}
				ctx := UseCipdClientFactory(ctx, factory)
				client, _ := NewClient(ctx, cipdRoot)

				recipesPyPath, err := client.SetupRecipe(ctx, "fake-package", "fake-version")

				So(err, ShouldErrLike, "could not read recipes.cfg")
				So(recipesPyPath, ShouldBeEmpty)
			})

			Convey("fails if unmarshalling recipes.cfg fails", func() {
				ctx := UseCipdClientFactory(ctx, factoryForRecipesCfg("invalid-recipes.cfg"))
				client, _ := NewClient(ctx, cipdRoot)

				recipesPyPath, err := client.SetupRecipe(ctx, "fake-package", "fake-version")

				So(err, ShouldErrLike, "invalid value")
				So(recipesPyPath, ShouldBeEmpty)
			})

			Convey("returns path to recipes.py", func() {

				Convey("with default recipe path", func() {
					ctx := UseCipdClientFactory(ctx, factoryForRecipesCfg("{}"))
					client, _ := NewClient(ctx, cipdRoot)

					recipesPyPath, err := client.SetupRecipe(ctx, "fake-package", "fake-version")

					So(err, ShouldBeNil)
					So(recipesPyPath, ShouldEqual, filepath.Join(cipdRoot, "fake-package", "recipes.py"))
				})

				Convey("with explicit recipes path", func() {
					ctx := UseCipdClientFactory(ctx, factoryForRecipesCfg(`{
						"recipes_path": "recipes"
					}`))
					client, _ := NewClient(ctx, cipdRoot)

					recipesPyPath, err := client.SetupRecipe(ctx, "fake-package", "fake-version")

					So(err, ShouldBeNil)
					So(recipesPyPath, ShouldEqual, filepath.Join(cipdRoot, "fake-package", "recipes", "recipes.py"))
				})

			})

		})

	})
}
