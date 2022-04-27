// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cipd

import (
	"context"
	bscipd "infra/chromium/bootstrapper/cipd"
	"infra/chromium/util"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/cipd/client/cipd"
	"go.chromium.org/luci/cipd/common"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/common/testing/testfs"
)

func collect(cipdRoot, subdir string) map[string]string {
	layout, err := testfs.Collect(filepath.Join(cipdRoot, subdir))
	util.PanicOnError(err)
	return layout
}

func TestCipdClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("cipdClient", t, func() {

		Convey("ResolveVersion", func() {

			Convey("returns an instance ID by default", func() {
				client, _ := Factory(nil)(ctx, "fake-root")

				pin, err := client.ResolveVersion(ctx, "fake-package", "fake-version")

				So(err, ShouldBeNil)
				So(pin.PackageName, ShouldEqual, "fake-package")
				So(pin.InstanceID, ShouldNotBeEmpty)
			})

			Convey("fails for a nil package", func() {
				client, _ := Factory(map[string]*Package{
					"fake-package": nil,
				})(ctx, "fake-root")

				pin, err := client.ResolveVersion(ctx, "fake-package", "fake-version")

				So(err, ShouldErrLike, "unknown package")
				So(pin.PackageName, ShouldBeEmpty)
				So(pin.InstanceID, ShouldBeEmpty)
			})

			Convey("fails for an empty instance ID", func() {
				client, _ := Factory(map[string]*Package{
					"fake-package": {
						Refs: map[string]string{
							"fake-version": "",
						},
					},
				})(ctx, "fake-root")

				pin, err := client.ResolveVersion(ctx, "fake-package", "fake-version")

				So(err, ShouldErrLike, "unknown version")
				So(pin.PackageName, ShouldBeEmpty)
				So(pin.InstanceID, ShouldBeEmpty)
			})

			Convey("returns pin for provided instance ID", func() {
				client, _ := Factory(map[string]*Package{
					"fake-package": {
						Refs: map[string]string{
							"fake-version": "fake-instance-id",
						},
					},
				})(ctx, "fake-root")

				pin, err := client.ResolveVersion(ctx, "fake-package", "fake-version")

				So(err, ShouldBeNil)
				So(pin.PackageName, ShouldEqual, "fake-package")
				So(pin.InstanceID, ShouldEqual, "fake-instance-id")
			})

		})

		Convey("EnsurePackages", func() {

			cipdRoot := t.TempDir()

			pkgs := common.PinSliceBySubdir{
				"fake-subdir": common.PinSlice{common.Pin{PackageName: "fake-package", InstanceID: "fake-instance-id"}},
			}

			Convey("succeeds by default", func() {
				client, _ := Factory(nil)(ctx, cipdRoot)

				_, err := client.EnsurePackages(ctx, pkgs, &cipd.EnsureOptions{Paranoia: cipd.CheckIntegrity})

				So(err, ShouldBeNil)
			})

			Convey("fails for a nil package", func() {
				client, _ := Factory(map[string]*Package{
					"fake-package": nil,
				})(ctx, cipdRoot)

				_, err := client.EnsurePackages(ctx, pkgs, &cipd.EnsureOptions{Paranoia: cipd.CheckIntegrity})

				So(err, ShouldErrLike, "unknown package")
			})

			Convey("fails for a nil bundle", func() {
				client, _ := Factory(map[string]*Package{
					"fake-package": {
						Instances: map[string]*PackageInstance{
							"fake-instance-id": nil,
						},
					},
				})(ctx, cipdRoot)

				_, err := client.EnsurePackages(ctx, pkgs, &cipd.EnsureOptions{Paranoia: cipd.CheckIntegrity})

				So(err, ShouldErrLike, "unknown instance ID")
			})

			Convey("deploys specified files", func() {
				client, _ := Factory(map[string]*Package{
					"fake-package": {
						Instances: map[string]*PackageInstance{
							"fake-instance-id": {
								Contents: map[string]string{
									"infra/config/recipes.cfg": "fake-recipes.cfg",
									"recipes/foo.py":           "fake-recipe-foo",
								},
							},
						},
					},
				})(ctx, cipdRoot)

				_, err := client.EnsurePackages(ctx, pkgs, &cipd.EnsureOptions{Paranoia: cipd.CheckIntegrity})

				So(err, ShouldBeNil)
				layout := collect(cipdRoot, "fake-subdir")
				So(layout, ShouldResemble, map[string]string{
					"infra/config/recipes.cfg": "fake-recipes.cfg",
					"recipes/foo.py":           "fake-recipe-foo",
				})
			})

		})

	})
}

func TestIntegration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("CIPD using fake factory", t, func() {

		cipdRoot := t.TempDir()

		ctx := bscipd.UseCipdClientFactory(ctx, Factory(nil))

		Convey("succeeds when calling DownloadPackage", func() {
			client, err := bscipd.NewClient(ctx, cipdRoot)
			util.PanicOnError(err)

			packagePath, err := client.DownloadPackage(ctx, "fake-package", "fake-version", "fake-subdir")

			So(err, ShouldBeNil)
			So(packagePath, ShouldNotBeEmpty)
		})

	})
}
