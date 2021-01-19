// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"
	"golang.org/x/time/rate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.chromium.org/luci/common/data/caching/lru"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"

	evalpb "infra/rts/presubmit/eval/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestGerritClient(t *testing.T) {
	t.Parallel()
	Convey(`GerritClient`, t, func() {
		ctx := context.Background()

		client := &gerritClient{
			limiter: rate.NewLimiter(100, 1),
			fileListCache: cache{
				dir:       t.TempDir(),
				memory:    lru.New(256),
				valueType: reflect.TypeOf(changedFiles{}),
			},
		}
		ps := &evalpb.GerritPatchset{
			Change: &evalpb.GerritChange{
				Host:    "example.googlesource.com",
				Project: "repo",
				Number:  123,
			},
			Patchset: 1,
		}

		Convey(`Works`, func() {
			var actualHost string
			var actualReq *gerritpb.ListFilesRequest
			client.listFilesRPC = func(ctx context.Context, host string, req *gerritpb.ListFilesRequest) (*gerritpb.ListFilesResponse, error) {
				actualHost = host
				actualReq = req
				return &gerritpb.ListFilesResponse{
					Files: map[string]*gerritpb.FileInfo{
						"a.go":        {},
						"b.go":        {},
						"/COMMIT_MSG": {},
					},
				}, nil
			}

			files, err := client.ChangedFiles(ctx, ps)
			So(err, ShouldBeNil)
			So(files, ShouldResemble, []string{"a.go", "b.go"})
			So(actualHost, ShouldEqual, "example.googlesource.com")
			So(actualReq, ShouldResembleProto, &gerritpb.ListFilesRequest{
				Project:    "repo",
				Number:     123,
				RevisionId: "1",
			})
		})

		Convey(`CL not found`, func() {
			client.listFilesRPC = func(ctx context.Context, host string, req *gerritpb.ListFilesRequest) (*gerritpb.ListFilesResponse, error) {
				return nil, status.Errorf(codes.NotFound, "not found")
			}

			files, err := client.ChangedFiles(ctx, ps)
			So(err, ShouldBeNil)
			So(files, ShouldBeEmpty)
		})

		Convey(`Quota errors`, func() {
			returnQuotaError := true
			client.listFilesRPC = func(ctx context.Context, host string, req *gerritpb.ListFilesRequest) (*gerritpb.ListFilesResponse, error) {
				if returnQuotaError {
					returnQuotaError = false
					return nil, status.Errorf(codes.ResourceExhausted, "quota exhausted")
				}

				return &gerritpb.ListFilesResponse{
					Files: map[string]*gerritpb.FileInfo{
						"a.go": {},
						"b.go": {},
					},
				}, nil
			}

			files, err := client.ChangedFiles(ctx, ps)
			So(err, ShouldBeNil)
			So(files, ShouldResemble, []string{"a.go", "b.go"})
		})
	})
}
