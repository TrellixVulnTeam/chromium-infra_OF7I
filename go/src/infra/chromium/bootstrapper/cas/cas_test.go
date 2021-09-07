// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cas

import (
	"context"
	"errors"
	"testing"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/filemetadata"
	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	apipb "go.chromium.org/luci/swarming/proto/api"
)

type fakeCasClient struct {
	downloadDirectory func(ctx context.Context, d digest.Digest, execRoot string, cache filemetadata.Cache) (map[string]*client.TreeOutput, *client.MovedBytesMetadata, error)
}

func (f *fakeCasClient) DownloadDirectory(ctx context.Context, d digest.Digest, execRoot string, cache filemetadata.Cache) (map[string]*client.TreeOutput, *client.MovedBytesMetadata, error) {
	downloadDirectory := f.downloadDirectory
	if downloadDirectory != nil {
		return downloadDirectory(ctx, d, execRoot, cache)
	}
	return nil, nil, nil
}

func TestClientForHost(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Client.clientForInstance", t, func() {

		Convey("fails if factory fails", func() {
			ctx := UseCasClientFactory(ctx, func(ctx context.Context, instance string) (CasClient, error) {
				return nil, errors.New("test client factory failure")
			})

			client := NewClient(ctx, "fake-exec-root")
			casClient, err := client.clientForInstance(ctx, "fake-instance")

			So(err, ShouldErrLike, "test client factory failure")
			So(casClient, ShouldBeNil)
		})

		Convey("returns CAS client from factory", func() {
			fakeClient := &fakeCasClient{}
			ctx := UseCasClientFactory(ctx, func(ctx context.Context, host string) (CasClient, error) {
				return fakeClient, nil
			})

			client := NewClient(ctx, "fake-exec-root")
			casClient, err := client.clientForInstance(ctx, "fake-instance")

			So(err, ShouldBeNil)
			So(casClient, ShouldEqual, fakeClient)
		})

		Convey("re-uses CAS client for instance", func() {
			ctx := UseCasClientFactory(ctx, func(ctx context.Context, host string) (CasClient, error) {
				return &fakeCasClient{}, nil
			})

			client := NewClient(ctx, "fake-exec-root")
			casClientFoo1, _ := client.clientForInstance(ctx, "fake-instance-foo")
			casClientFoo2, _ := client.clientForInstance(ctx, "fake-instance-foo")
			casClientBar, _ := client.clientForInstance(ctx, "fake-instance-bar")

			So(casClientFoo1, ShouldNotBeNil)
			So(casClientFoo2, ShouldPointTo, casClientFoo1)
			So(casClientBar, ShouldNotPointTo, casClientFoo1)
		})

	})
}

func TestDownload(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Client.Download", t, func() {

		execRoot := t.TempDir()

		Convey("fails if getting client for instance fails", func() {
			ctx := UseCasClientFactory(ctx, func(ctx context.Context, instance string) (CasClient, error) {
				return nil, errors.New("test client factory failure")
			})

			client := NewClient(ctx, execRoot)
			packagePath, err := client.Download(ctx, "fake-instance", &apipb.Digest{
				Hash:      "fake-hash",
				SizeBytes: 42,
			})

			So(err, ShouldErrLike, "test client factory failure")
			So(packagePath, ShouldBeEmpty)
		})

		Convey("fails if downloading directory fails", func() {
			ctx := UseCasClientFactory(ctx, func(ctx context.Context, instance string) (CasClient, error) {
				return &fakeCasClient{
					downloadDirectory: func(ctx context.Context, d digest.Digest, execRoot string, cache filemetadata.Cache) (map[string]*client.TreeOutput, *client.MovedBytesMetadata, error) {
						return nil, nil, errors.New("test DownloadDirectory failure")
					},
				}, nil
			})

			client := NewClient(ctx, execRoot)
			packagePath, err := client.Download(ctx, "fake-instance", &apipb.Digest{
				Hash:      "fake-hash",
				SizeBytes: 42,
			})

			So(err, ShouldErrLike, "test DownloadDirectory failure")
			So(packagePath, ShouldBeEmpty)
		})

		Convey("returns path to deployed package", func() {
			ctx := UseCasClientFactory(ctx, func(ctx context.Context, instance string) (CasClient, error) {
				return &fakeCasClient{}, nil
			})

			client := NewClient(ctx, execRoot)
			packagePath, err := client.Download(ctx, "fake-instance", &apipb.Digest{
				Hash:      "fake-hash",
				SizeBytes: 42,
			})

			So(err, ShouldBeNil)
			So(packagePath, ShouldEqual, execRoot)
		})

	})
}
