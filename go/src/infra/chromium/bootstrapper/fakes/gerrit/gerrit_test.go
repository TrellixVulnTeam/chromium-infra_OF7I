// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gerrit

import (
	"context"
	"infra/chromium/bootstrapper/gerrit"
	"sort"
	"strconv"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	. "go.chromium.org/luci/common/testing/assertions"
)

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
				gerrit:   host,
			})
		})
	})
}

func getChangeRequest(project string, number int64) *gerritpb.GetChangeRequest {
	return &gerritpb.GetChangeRequest{
		Project: project,
		Number:  number,
		Options: []gerritpb.QueryOption{gerritpb.QueryOption_ALL_REVISIONS},
	}
}

func TestGetChange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Client.GetChange", t, func() {

		Convey("returns a change info by default", func() {
			client, _ := Factory(nil)(ctx, "fake-host")

			changeInfo, err := client.GetChange(ctx, getChangeRequest("fake/project", 123))

			So(err, ShouldBeNil)
			So(changeInfo, ShouldNotBeNil)
			So(changeInfo.Project, ShouldEqual, "fake/project")
			So(changeInfo.Number, ShouldEqual, 123)
			So(changeInfo.Revisions, ShouldHaveLength, 1)
			for rev, revInfo := range changeInfo.Revisions {
				So(rev, ShouldNotBeEmpty)
				So(revInfo.Number, ShouldEqual, 1)
			}
		})

		Convey("fails for a nil project", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": nil,
					},
				},
			})(ctx, "fake-host")

			changeInfo, err := client.GetChange(ctx, getChangeRequest("fake/project", 123))

			So(err, ShouldErrLike, `unknown project "fake/project" on host "fake-host"`)
			So(changeInfo, ShouldBeNil)
		})

		Convey("fails for a nil change", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Changes: map[int64]*Change{
								123: nil,
							},
						},
					},
				},
			})(ctx, "fake-host")

			changeInfo, err := client.GetChange(ctx, getChangeRequest("fake/project", 123))

			So(err, ShouldErrLike, `change 123 does not exist for project "fake/project" on host "fake-host"`)
			So(changeInfo, ShouldBeNil)
		})

		Convey("returns info for provided change without revisions", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Changes: map[int64]*Change{
								123: {
									Ref: "fake-ref",
								},
							},
						},
					},
				},
			})(ctx, "fake-host")

			changeInfo, err := client.GetChange(ctx, getChangeRequest("fake/project", 123))

			So(err, ShouldBeNil)
			So(changeInfo, ShouldNotBeNil)
			So(changeInfo.Project, ShouldEqual, "fake/project")
			So(changeInfo.Number, ShouldEqual, 123)
			So(changeInfo.Ref, ShouldEqual, "fake-ref")
			for rev, revInfo := range changeInfo.Revisions {
				So(rev, ShouldNotBeEmpty)
				So(revInfo.Number, ShouldEqual, 1)
			}
		})

		Convey("returns info for provided change with revisions", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Changes: map[int64]*Change{
								123: {
									Ref: "fake-ref",
									Patchsets: map[int32]*Patchset{
										4: {Revision: "fake-revision-4"},
										7: {Revision: ""},
									},
								},
							},
						},
					},
				},
			})(ctx, "fake-host")

			changeInfo, err := client.GetChange(ctx, getChangeRequest("fake/project", 123))

			So(err, ShouldBeNil)
			So(changeInfo, ShouldNotBeNil)
			So(changeInfo.Project, ShouldEqual, "fake/project")
			So(changeInfo.Number, ShouldEqual, 123)
			So(changeInfo.Ref, ShouldEqual, "fake-ref")
			So(changeInfo.Revisions, ShouldContainKey, "fake-revision-4")
			So(changeInfo.Revisions["fake-revision-4"].Number, ShouldEqual, 4)
			var patchsets []int
			for rev, revInfo := range changeInfo.Revisions {
				So(rev, ShouldNotBeEmpty)
				patchsets = append(patchsets, int(revInfo.Number))
			}
			sort.Ints(patchsets)
			So(patchsets, ShouldResemble, []int{1, 2, 3, 4, 5, 6, 7})
		})

	})

}

func listFilesRequest(project string, change int64, patchset int32) *gerritpb.ListFilesRequest {
	return &gerritpb.ListFilesRequest{
		Project:    project,
		Number:     change,
		RevisionId: strconv.FormatInt(int64(patchset), 10),
	}
}

func TestListFiles(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Client.ListFiles", t, func() {

		Convey("returns no files by default", func() {
			client, _ := Factory(nil)(ctx, "fake-host")

			response, err := client.ListFiles(ctx, listFilesRequest("fake/project", 123, 1))

			So(err, ShouldBeNil)
			So(response, ShouldNotBeNil)
			So(response.Files, ShouldBeEmpty)
		})

		Convey("fails for a nil project", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": nil,
					},
				},
			})(ctx, "fake-host")

			response, err := client.ListFiles(ctx, listFilesRequest("fake/project", 123, 1))

			So(err, ShouldErrLike, `unknown project "fake/project" on host "fake-host"`)
			So(response, ShouldBeNil)
		})

		Convey("fails for a nil change", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Changes: map[int64]*Change{
								123: nil,
							},
						},
					},
				},
			})(ctx, "fake-host")

			response, err := client.ListFiles(ctx, listFilesRequest("fake/project", 123, 1))

			So(err, ShouldErrLike, `change 123 does not exist for project "fake/project" on host "fake-host"`)
			So(response, ShouldBeNil)
		})

		Convey("fails for a missing patchset", func() {
			client, _ := Factory(nil)(ctx, "fake-host")

			response, err := client.ListFiles(ctx, listFilesRequest("fake/project", 123, 2))

			So(err, ShouldErrLike, `patchset 2 does not exist for change 123 of project "fake/project" on host "fake-host"`)
			So(response, ShouldBeNil)
		})

		Convey("returns info for provided affected files", func() {
			client, _ := Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Changes: map[int64]*Change{
								123: {
									Patchsets: map[int32]*Patchset{
										1: {
											AffectedFiles: map[string]*AffectedFile{
												"fake-file-1": nil,
												"fake-file-2": {
													Status: gerrit.ADDED,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			})(ctx, "fake-host")

			response, err := client.ListFiles(ctx, listFilesRequest("fake/project", 123, 1))

			So(err, ShouldBeNil)
			So(response, ShouldResembleProtoJSON, `{
				"files": {
					"fake-file-1": {
						"status": 0
					},
					"fake-file-2": {
						"status": 1
					}
				}
			}`)
		})

	})

}

func TestIntegration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("gerrit using fake factory", t, func() {

		Convey("succeeds when calling GetTargetRef", func() {
			ctx := gerrit.UseGerritClientFactory(ctx, Factory(nil))
			client := gerrit.NewClient(ctx)

			ref, err := client.GetTargetRef(ctx, "fake-host", "fake/project", 123)

			So(err, ShouldBeNil)
			So(ref, ShouldNotBeEmpty)
		})

		Convey("succeeds when calling GetRevision", func() {
			ctx := gerrit.UseGerritClientFactory(ctx, Factory(nil))
			client := gerrit.NewClient(ctx)

			revision, err := client.GetRevision(ctx, "fake-host", "fake/project", 123, 1)

			So(err, ShouldBeNil)
			So(revision, ShouldNotBeEmpty)
		})

		Convey("succeeds when calling ListFiles", func() {
			ctx := gerrit.UseGerritClientFactory(ctx, Factory(map[string]*Host{
				"fake-host": {
					Projects: map[string]*Project{
						"fake/project": {
							Changes: map[int64]*Change{
								123: {
									Patchsets: map[int32]*Patchset{
										1: {
											AffectedFiles: map[string]*AffectedFile{
												"fake/file": nil,
											},
										},
									},
								},
							},
						},
					},
				},
			}))
			client := gerrit.NewClient(ctx)

			fileInfo, err := client.GetAffectedFileInfo(ctx, "fake-host", "fake/project", 123, 1, "fake/file")

			So(err, ShouldBeNil)
			So(fileInfo, ShouldNotBeNil)
		})

	})
}
