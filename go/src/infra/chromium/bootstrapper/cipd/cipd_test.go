// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cipd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/cipd/client/cipd"
	"go.chromium.org/luci/cipd/common"
	. "go.chromium.org/luci/common/testing/assertions"
)

type fakeCipdClient struct {
	resolveVersion func(context.Context, string, string) (common.Pin, error)
	ensurePackages func(context.Context, common.PinSliceBySubdir, cipd.ParanoidMode, bool) (cipd.ActionMap, error)
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

func (f *fakeCipdClient) EnsurePackages(ctx context.Context, packages common.PinSliceBySubdir, opts *cipd.EnsureOptions) (cipd.ActionMap, error) {
	ensurePackages := f.ensurePackages
	if ensurePackages != nil {
		return ensurePackages(ctx, packages, opts.Paranoia, opts.DryRun)
	}
	return nil, nil
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("NewClient", t, func() {

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
}

func TestResolveVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Client.DownloadPackage", t, func() {

		Convey("fails if resolving version fails", func() {
			factory := func(ctx context.Context, cipdRoot string) (CipdClient, error) {
				return &fakeCipdClient{resolveVersion: func(ctx context.Context, pkg, version string) (common.Pin, error) {
					return common.Pin{}, errors.New("test ResolveVersion failure")
				}}, nil
			}
			ctx := UseCipdClientFactory(ctx, factory)
			client, _ := NewClient(ctx, "unneeded-cipd-root")

			pin, err := client.ResolveVersion(ctx, "fake-package", "fake-version")

			So(err, ShouldErrLike, "test ResolveVersion failure")
			So(pin, ShouldResemble, common.Pin{})
		})

		Convey("returns resolved pin", func() {
			factory := func(ctx context.Context, cipdRoot string) (CipdClient, error) {
				return &fakeCipdClient{resolveVersion: func(ctx context.Context, pkg, version string) (common.Pin, error) {
					return common.Pin{
						PackageName: pkg,
						InstanceID:  fmt.Sprintf("%s-instance-id", pkg),
					}, nil
				}}, nil
			}
			ctx := UseCipdClientFactory(ctx, factory)
			client, _ := NewClient(ctx, "unneeded-cipd-root")

			pin, err := client.ResolveVersion(ctx, "fake-package", "fake-version")

			So(err, ShouldBeNil)
			So(pin, ShouldResemble, common.Pin{
				PackageName: "fake-package",
				InstanceID:  "fake-package-instance-id",
			})
		})

	})

}

func TestDownloadPackage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Client.DownloadPackage", t, func() {

		cipdRoot := t.TempDir()

		Convey("fails if ensuring package fails", func() {
			factory := func(ctx context.Context, cipdRoot string) (CipdClient, error) {
				return &fakeCipdClient{ensurePackages: func(ctx context.Context, packages common.PinSliceBySubdir, paranoia cipd.ParanoidMode, dryRun bool) (cipd.ActionMap, error) {
					return nil, errors.New("test EnsurePackages failure")
				}}, nil
			}
			ctx := UseCipdClientFactory(ctx, factory)
			client, _ := NewClient(ctx, cipdRoot)

			packagePath, err := client.DownloadPackage(ctx, "fake-package", "fake-instance-id", "fake-subdir")

			So(err, ShouldErrLike, "test EnsurePackages failure")
			So(packagePath, ShouldBeEmpty)
		})

		Convey("returns path to deployed package", func() {
			factory := func(ctx context.Context, cipdRoot string) (CipdClient, error) {
				return &fakeCipdClient{ensurePackages: func(ctx context.Context, packages common.PinSliceBySubdir, paranoia cipd.ParanoidMode, dryRun bool) (cipd.ActionMap, error) {
					return nil, nil
				}}, nil
			}
			ctx := UseCipdClientFactory(ctx, factory)
			client, _ := NewClient(ctx, cipdRoot)

			packagePath, err := client.DownloadPackage(ctx, "fake-package", "fake-instance-id", "fake-subdir")

			So(err, ShouldBeNil)
			So(packagePath, ShouldEqual, filepath.Join(cipdRoot, "fake-subdir"))

		})

	})

}
