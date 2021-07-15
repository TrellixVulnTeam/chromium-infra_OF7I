// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gitiles

import (
	"context"
	"infra/chromium/bootstrapper/gitiles"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	. "go.chromium.org/luci/common/testing/assertions"
)

func strPtr(s string) *string {
	return &s
}

func TestFactory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Factory", t, func() {

		Convey("returns an RPC client by default", func() {
			factory := Factory(nil)

			client, err := factory(ctx, "fake-host")

			So(err, ShouldBeNil)
			So(client, ShouldNotBeNil)
		})

		Convey("fails for a nil host", func() {
			factory := Factory(map[string]*Host{
				"fake-host": nil,
			})

			client, err := factory(ctx, "fake-host")

			So(err, ShouldNotBeNil)
			So(client, ShouldBeNil)
		})

		Convey("returns RPC client for provided host", func() {
			host := &Host{}
			factory := Factory(map[string]*Host{
				"fake-host": host,
			})

			client, err := factory(ctx, "fake-host")

			So(err, ShouldBeNil)
			So(client, ShouldResemble, &Client{
				hostname: "fake-host",
				gitiles:  host,
			})
		})
	})
}

func logRequest(project, ref string) *gitilespb.LogRequest {
	return &gitilespb.LogRequest{
		Project:    project,
		Committish: ref,
		PageSize:   1,
	}
}

func downloadFileRequest(project, revision, path string) *gitilespb.DownloadFileRequest {
	return &gitilespb.DownloadFileRequest{
		Project:    project,
		Committish: revision,
		Path:       path,
	}
}

func TestGitilesClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("gitilesClient", t, func() {

		Convey("Log", func() {

			Convey("returns a revision by default", func() {
				client, _ := Factory(nil)(ctx, "fake-host")

				response, err := client.Log(ctx, logRequest("fake/project", "refs/heads/fake-branch"))

				So(err, ShouldBeNil)
				So(response, ShouldNotBeNil)
				So(response.Log, ShouldHaveLength, 1)
				So(response.Log[0].Id, ShouldNotBeEmpty)
			})

			Convey("fails for a nil project", func() {
				client, _ := Factory(map[string]*Host{
					"fake-host": {
						Projects: map[string]*Project{
							"fake/project": nil,
						},
					},
				})(ctx, "fake-host")

				response, err := client.Log(ctx, logRequest("fake/project", "refs/heads/fake-branch"))

				So(err, ShouldErrLike, `unknown project "fake/project" on host "fake-host"`)
				So(response, ShouldBeNil)
			})

			Convey("fails for an empty ref revision", func() {
				client, _ := Factory(map[string]*Host{
					"fake-host": {
						Projects: map[string]*Project{
							"fake/project": {
								Refs: map[string]string{
									"refs/heads/fake-branch": "",
								},
							},
						},
					},
				})(ctx, "fake-host")

				response, err := client.Log(ctx, logRequest("fake/project", "refs/heads/fake-branch"))

				So(err, ShouldErrLike, `unknown ref "refs/heads/fake-branch" for project "fake/project" on host "fake-host"`)
				So(response, ShouldBeNil)
			})

			Convey("returns log for provided revision", func() {
				client, _ := Factory(map[string]*Host{
					"fake-host": {
						Projects: map[string]*Project{
							"fake/project": {
								Refs: map[string]string{
									"refs/heads/fake-branch": "fake-revision",
								},
							},
						},
					},
				})(ctx, "fake-host")

				response, err := client.Log(ctx, logRequest("fake/project", "refs/heads/fake-branch"))

				So(err, ShouldBeNil)
				So(response, ShouldNotBeNil)
				So(response.Log, ShouldHaveLength, 1)
				So(response.Log[0].Id, ShouldEqual, "fake-revision")
			})

		})

		Convey("DownloadFile", func() {

			Convey("returns contents by default", func() {
				client, _ := Factory(nil)(ctx, "fake-host")

				response, err := client.DownloadFile(ctx, downloadFileRequest("fake/project", "fake-revision", "fake/file"))
				So(err, ShouldBeNil)
				So(response, ShouldNotBeEmpty)
			})

			Convey("fails for a nil project", func() {
				client, _ := Factory(map[string]*Host{
					"fake-host": {
						Projects: map[string]*Project{
							"fake/project": nil,
						},
					},
				})(ctx, "fake-host")

				response, err := client.DownloadFile(ctx, downloadFileRequest("fake/project", "fake-revision", "fake/file"))

				So(err, ShouldErrLike, `unknown project "fake/project" on host "fake-host"`)
				So(response, ShouldBeNil)
			})

			Convey("fails for nil contents", func() {
				client, _ := Factory(map[string]*Host{
					"fake-host": {
						Projects: map[string]*Project{
							"fake/project": {
								Files: map[FileRevId]*string{
									{"fake-revision", "fake/file"}: nil,
								},
							},
						},
					},
				})(ctx, "fake-host")

				response, err := client.DownloadFile(ctx, downloadFileRequest("fake/project", "fake-revision", "fake/file"))
				So(err, ShouldErrLike, `unknown file "fake/file" at revision "fake-revision" of project "fake/project" on host "fake-host"`)
				So(response, ShouldBeNil)
			})

			Convey("returns contents for provided file rev ID", func() {
				client, _ := Factory(map[string]*Host{
					"fake-host": {
						Projects: map[string]*Project{
							"fake/project": {
								Files: map[FileRevId]*string{
									{"fake-revision", "fake/file"}: strPtr("fake-contents"),
								},
							},
						},
					},
				})(ctx, "fake-host")

				response, err := client.DownloadFile(ctx, downloadFileRequest("fake/project", "fake-revision", "fake/file"))

				So(err, ShouldBeNil)
				So(response, ShouldNotBeNil)
				So(response.Contents, ShouldEqual, "fake-contents")
			})

		})

	})
}

func TestIntegration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("gitiles using fake factory", t, func() {

		Convey("succeeds when calling FetchLatestRevision", func() {
			ctx := gitiles.UseGitilesClientFactory(ctx, Factory(nil))
			client := gitiles.NewClient(ctx)

			revision, err := client.FetchLatestRevision(ctx, "fake-host", "fake/project", "refs/heads/fake-branch")

			So(err, ShouldBeNil)
			So(revision, ShouldNotBeEmpty)
		})

		Convey("succeeds when calling DownloadFile", func() {
			ctx := gitiles.UseGitilesClientFactory(ctx, Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Files: map[FileRevId]*string{
								{"fake-revision", "fake/file"}: strPtr("fake-contents"),
							},
						},
					},
				},
			}))
			client := gitiles.NewClient(ctx)

			contents, err := client.DownloadFile(ctx, "fake-host", "fake/project", "fake-revision", "fake/file")

			So(err, ShouldBeNil)
			So(contents, ShouldEqual, "fake-contents")
		})

	})
}
