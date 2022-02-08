// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cas

import (
	"context"
	"testing"

	bscas "infra/chromium/bootstrapper/cas"
	"infra/chromium/util"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/filemetadata"
	. "github.com/smartystreets/goconvey/convey"
	apipb "go.chromium.org/luci/swarming/proto/api"
)

func TestFactory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Factory", t, func() {

		Convey("succeeds by default", func() {
			factory := Factory(nil)

			client, err := factory(ctx, "fake-instance")

			So(err, ShouldBeNil)
			So(client, ShouldNotBeNil)
		})

		Convey("fails for a nil instance", func() {
			factory := Factory(map[string]*Instance{
				"fake-instance": nil,
			})

			client, err := factory(ctx, "fake-instance")

			So(err, ShouldNotBeNil)
			So(client, ShouldBeNil)
		})

		Convey("succeeds for provided instance", func() {
			factory := Factory(map[string]*Instance{
				"fake-instance": {},
			})

			client, err := factory(ctx, "fake-instance")

			So(err, ShouldBeNil)
			So(client, ShouldNotBeNil)
		})

	})

}

func TestDownload(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("casClient.Download", t, func() {

		execRoot := t.TempDir()

		d := digest.Digest{
			Hash: "fake-hash",
			Size: 42,
		}

		cache := filemetadata.NewNoopCache()

		Convey("succeeds by default", func() {
			client, err := Factory(nil)(ctx, "fake-instance")
			util.PanicOnError(err)

			_, _, err = client.DownloadDirectory(ctx, d, execRoot, cache)

			So(err, ShouldBeNil)
		})

		Convey("fails for a false blob", func() {
			client, err := Factory(map[string]*Instance{
				"fake-instance": {
					blobs: map[string]bool{
						"fake-hash": false,
					},
				},
			})(ctx, "fake-instance")
			util.PanicOnError(err)

			_, _, err = client.DownloadDirectory(ctx, d, execRoot, cache)

			So(err, ShouldNotBeNil)
		})

		Convey("succeeds for a true blob", func() {
			client, err := Factory(map[string]*Instance{
				"fake-instance": {
					blobs: map[string]bool{
						"fake-hash": true,
					},
				},
			})(ctx, "fake-instance")
			util.PanicOnError(err)

			_, _, err = client.DownloadDirectory(ctx, d, execRoot, cache)

			So(err, ShouldBeNil)
		})

	})
}

func TestIntegration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("package using fake CAS client", t, func() {

		execRoot := t.TempDir()

		ctx := bscas.UseCasClientFactory(ctx, Factory(nil))

		Convey("succeeds when calling Download", func() {
			client := bscas.NewClient(ctx, execRoot)

			packagePath, err := client.Download(ctx, "fake-instance", &apipb.Digest{
				Hash:      "fake-hash",
				SizeBytes: 42,
			})

			So(err, ShouldBeNil)
			So(packagePath, ShouldNotBeEmpty)
		})

	})
}
