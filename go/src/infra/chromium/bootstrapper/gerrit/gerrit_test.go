// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gerrit

import (
	"context"
	"errors"
	"infra/chromium/bootstrapper/gob"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/common/proto"
	gerritpb "go.chromium.org/luci/common/proto/gerrit"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGerritClientForHost(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Client.gerritClientForHost", t, func() {

		Convey("fails if factory fails", func() {
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return nil, errors.New("fake client factory failure")
			})

			client := NewClient(ctx)
			gerritClient, err := client.gerritClientForHost(ctx, "fake-host")

			So(err, ShouldNotBeNil)
			So(gerritClient, ShouldBeNil)
		})

		Convey("returns gerrit client from factory", func() {
			mockGerritClient := gerritpb.NewMockGerritClient(gomock.NewController(t))
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return mockGerritClient, nil
			})

			client := NewClient(ctx)
			gerritClient, err := client.gerritClientForHost(ctx, "fake-host")

			So(err, ShouldBeNil)
			So(gerritClient, ShouldEqual, mockGerritClient)
		})

		Convey("re-uses gerrit client for host", func() {
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return gerritpb.NewMockGerritClient(gomock.NewController(t)), nil
			})

			client := NewClient(ctx)
			gerritClientFoo1, _ := client.gerritClientForHost(ctx, "fake-host-foo")
			gerritClientFoo2, _ := client.gerritClientForHost(ctx, "fake-host-foo")
			gerritClientBar, _ := client.gerritClientForHost(ctx, "fake-host-bar")

			So(gerritClientFoo1, ShouldNotBeNil)
			So(gerritClientFoo2, ShouldPointTo, gerritClientFoo1)
			So(gerritClientBar, ShouldNotPointTo, gerritClientFoo1)
		})

	})
}

func TestGetChangeInfo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = gob.CtxForTest(ctx)

	Convey("Client.getChangeInfo", t, func() {

		Convey("fails if getting client for host fails", func() {
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return nil, errors.New("fake client factory failure")
			})

			client := NewClient(ctx)
			changeInfo, err := client.getChangeInfo(ctx, "fake-host", "fake/project", 123)

			So(err, ShouldErrLike, "fake client factory failure")
			So(changeInfo, ShouldBeNil)
		})

		Convey("fails if API call fails", func() {
			mockGerritClient := gerritpb.NewMockGerritClient(gomock.NewController(t))
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return mockGerritClient, nil
			})
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Project: "fake/project",
					Number:  123,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_ALL_REVISIONS},
				})).
				Return(nil, errors.New("fake GetChange failure"))

			client := NewClient(ctx)
			changeInfo, err := client.getChangeInfo(ctx, "fake-host", "fake/project", 123)

			So(err, ShouldErrLike, "fake GetChange failure")
			So(changeInfo, ShouldBeNil)
		})

		Convey("returns change info for change", func() {
			mockGerritClient := gerritpb.NewMockGerritClient(gomock.NewController(t))
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return mockGerritClient, nil
			})
			mockChangeInfo := &gerritpb.ChangeInfo{
				Project: "fake/project",
				Number:  123,
				Ref:     "fake-ref",
				Revisions: map[string]*gerritpb.RevisionInfo{
					"fake-revision": {
						Number: 1,
					},
				},
			}
			matcher := proto.MatcherEqual(&gerritpb.GetChangeRequest{
				Project: "fake/project",
				Number:  123,
				Options: []gerritpb.QueryOption{gerritpb.QueryOption_ALL_REVISIONS},
			})
			// Check that potentially transient errors are retried
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), matcher).
				Return(nil, status.Error(codes.NotFound, "fake transient GetChange failure"))
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), matcher).
				Return(nil, status.Error(codes.Unavailable, "fake transient GetChange failure"))
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), matcher).
				Return(mockChangeInfo, nil)

			client := NewClient(ctx)
			changeInfo, err := client.getChangeInfo(ctx, "fake-host", "fake/project", 123)

			So(err, ShouldBeNil)
			So(changeInfo, ShouldEqual, mockChangeInfo)
		})

		Convey("re-uses change info", func() {
			mockGerritClient := gerritpb.NewMockGerritClient(gomock.NewController(t))
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return mockGerritClient, nil
			})
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), gomock.Any()).
				AnyTimes().
				DoAndReturn(func(ctx context.Context, req *gerritpb.GetChangeRequest, opts ...grpc.CallOption) (*gerritpb.ChangeInfo, error) {
					return &gerritpb.ChangeInfo{
						Project: req.Project,
						Number:  req.Number,
						Ref:     "fake-ref",
						Revisions: map[string]*gerritpb.RevisionInfo{
							"fake-revision": {
								Number: 1,
							},
						},
					}, nil
				})

			client := NewClient(ctx)
			changeInfo, _ := client.getChangeInfo(ctx, "fake-host", "fake/project", 123)
			changeInfo2, _ := client.getChangeInfo(ctx, "fake-host", "fake/project", 123)
			changeInfoOther, _ := client.getChangeInfo(ctx, "fake-host", "fake/project", 456)

			So(changeInfo, ShouldNotBeNil)
			So(changeInfo, ShouldPointTo, changeInfo2)
			So(changeInfoOther, ShouldNotPointTo, changeInfo)
		})

	})
}

func TestGetTargetRef(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = gob.CtxForTest(ctx)

	Convey("Client.GetTargetRef", t, func() {

		Convey("fails if getting change info fails", func() {
			mockGerritClient := gerritpb.NewMockGerritClient(gomock.NewController(t))
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return mockGerritClient, nil
			})
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Project: "fake/project",
					Number:  123,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_ALL_REVISIONS},
				})).
				Return(nil, errors.New("fake GetChange failure"))

			client := NewClient(ctx)
			ref, err := client.GetTargetRef(ctx, "fake-host", "fake/project", 123)

			So(err, ShouldErrLike, "fake GetChange failure")
			So(ref, ShouldBeEmpty)
		})

		Convey("returns the target ref", func() {
			mockGerritClient := gerritpb.NewMockGerritClient(gomock.NewController(t))
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return mockGerritClient, nil
			})
			mockChangeInfo := &gerritpb.ChangeInfo{
				Project: "fake/project",
				Number:  123,
				Ref:     "fake-ref",
				Revisions: map[string]*gerritpb.RevisionInfo{
					"fake-revision": {
						Number: 1,
					},
				},
			}
			matcher := proto.MatcherEqual(&gerritpb.GetChangeRequest{
				Project: "fake/project",
				Number:  123,
				Options: []gerritpb.QueryOption{gerritpb.QueryOption_ALL_REVISIONS},
			})
			// Check that potentially transient errors are retried
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), matcher).
				Return(nil, status.Error(codes.NotFound, "fake transient GetChange failure"))
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), matcher).
				Return(nil, status.Error(codes.Unavailable, "fake transient GetChange failure"))
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), matcher).
				Return(mockChangeInfo, nil)

			client := NewClient(ctx)
			ref, err := client.GetTargetRef(ctx, "fake-host", "fake/project", 123)

			So(err, ShouldBeNil)
			So(ref, ShouldEqual, "fake-ref")
		})

	})
}

func TestGetRevision(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = gob.CtxForTest(ctx)

	Convey("Client.GetRevision", t, func() {

		Convey("fails if getting change info fails", func() {
			mockGerritClient := gerritpb.NewMockGerritClient(gomock.NewController(t))
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return mockGerritClient, nil
			})
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Project: "fake/project",
					Number:  123,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_ALL_REVISIONS},
				})).
				Return(nil, errors.New("fake GetChange failure"))

			client := NewClient(ctx)
			revision, err := client.GetRevision(ctx, "fake-host", "fake/project", 123, 1)

			So(err, ShouldErrLike, "fake GetChange failure")
			So(revision, ShouldBeEmpty)
		})

		Convey("returns revision", func() {
			mockGerritClient := gerritpb.NewMockGerritClient(gomock.NewController(t))
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return mockGerritClient, nil
			})
			mockChangeInfo := &gerritpb.ChangeInfo{
				Project: "fake/project",
				Number:  123,
				Ref:     "fake-ref",
				Revisions: map[string]*gerritpb.RevisionInfo{
					"fake-revision": {
						Number: 1,
					},
				},
			}
			matcher := proto.MatcherEqual(&gerritpb.GetChangeRequest{
				Project: "fake/project",
				Number:  123,
				Options: []gerritpb.QueryOption{gerritpb.QueryOption_ALL_REVISIONS},
			})
			// Check that potentially transient errors are retried
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), matcher).
				Return(nil, status.Error(codes.NotFound, "fake transient GetChange failure"))
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), matcher).
				Return(nil, status.Error(codes.Unavailable, "fake transient GetChange failure"))
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), matcher).
				Return(mockChangeInfo, nil)

			client := NewClient(ctx)
			revision, err := client.GetRevision(ctx, "fake-host", "fake/project", 123, 1)

			So(err, ShouldBeNil)
			So(revision, ShouldEqual, "fake-revision")
		})

		Convey("fails for non-existent patchset", func() {
			mockGerritClient := gerritpb.NewMockGerritClient(gomock.NewController(t))
			ctx := UseGerritClientFactory(ctx, func(ctx context.Context, host string) (GerritClient, error) {
				return mockGerritClient, nil
			})
			mockChangeInfo := &gerritpb.ChangeInfo{
				Project: "fake/project",
				Number:  123,
				Ref:     "fake-ref",
				Revisions: map[string]*gerritpb.RevisionInfo{
					"fake-revision": {
						Number: 1,
					},
				},
			}
			mockGerritClient.EXPECT().
				GetChange(gomock.Any(), proto.MatcherEqual(&gerritpb.GetChangeRequest{
					Project: "fake/project",
					Number:  123,
					Options: []gerritpb.QueryOption{gerritpb.QueryOption_ALL_REVISIONS},
				})).
				Return(mockChangeInfo, nil)

			client := NewClient(ctx)
			revision, err := client.GetRevision(ctx, "fake-host", "fake/project", 123, 2)

			So(err, ShouldErrLike, "fake-host/c/fake/project/+/123 does not have patchset 2")
			So(revision, ShouldBeEmpty)
		})

	})
}
