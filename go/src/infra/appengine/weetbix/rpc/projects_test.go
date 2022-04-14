// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"sort"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/secrets"
	"go.chromium.org/luci/server/secrets/testsecrets"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"

	"infra/appengine/weetbix/internal/config"
	configpb "infra/appengine/weetbix/internal/config/proto"
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"
)

func TestProjects(t *testing.T) {
	Convey("Given a projects server", t, func() {

		ctx := testutil.SpannerTestContext(t)

		// For user identification.
		ctx = authtest.MockAuthConfig(ctx)
		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity:       "user:someone@example.com",
			IdentityGroups: []string{"weetbix-access"},
		})
		ctx = secrets.Use(ctx, &testsecrets.Store{})

		// Provides datastore implementation needed for project config.
		ctx = memory.Use(ctx)
		server := NewProjectsServer()

		Convey("Unauthorised requests are rejected", func() {
			ctx = auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:someone@example.com",
				// Not a member of weetbix-access.
				IdentityGroups: []string{"other-group"},
			})

			// Make some request (the request should not matter, as
			// a common decorator is used for all requests.)
			request := &pb.ListProjectsRequest{}

			rule, err := server.List(ctx, request)
			st, _ := grpcStatus.FromError(err)
			So(st.Code(), ShouldEqual, codes.PermissionDenied)
			So(st.Message(), ShouldEqual, "not a member of weetbix-access")
			So(rule, ShouldBeNil)
		})
		Convey("When a list of projects is provided", func() {
			// Setup
			projectChromium := config.CreatePlaceholderProjectConfig()
			projectChrome := config.CreatePlaceholderProjectConfigWithKey("chrome")
			configs := make(map[string]*configpb.ProjectConfig)
			configs["chromium"] = projectChromium
			configs["chrome"] = projectChrome
			config.SetTestProjectConfig(ctx, configs)

			// Run
			request := &pb.ListProjectsRequest{}
			projectsResponse, err := server.List(ctx, request)

			// Verify
			So(err, ShouldBeNil)
			expected := &pb.ListProjectsResponse{Projects: []*pb.Project{
				{
					Name:        "projects/chrome",
					DisplayName: "Chrome",
					Project:     "chrome",
				},
				{
					Name:        "projects/chromium",
					DisplayName: "Chromium",
					Project:     "chromium",
				},
			}}
			sort.Slice(expected.Projects, func(i, j int) bool {
				return expected.Projects[i].Name < expected.Projects[j].Name
			})
			sort.Slice(projectsResponse.Projects, func(i, j int) bool {
				return projectsResponse.Projects[i].Name < projectsResponse.Projects[j].Name
			})
			So(projectsResponse, ShouldResembleProto, expected)
		})

		Convey("When a displayName is empty", func() {
			// Setup
			emptyKeyProject := config.CreatePlaceholderProjectConfigWithKey("")
			configs := make(map[string]*configpb.ProjectConfig)
			configs["chrome"] = emptyKeyProject
			config.SetTestProjectConfig(ctx, configs)

			// Run
			request := &pb.ListProjectsRequest{}
			projectsResponse, err := server.List(ctx, request)

			// Verify
			expected := &pb.ListProjectsResponse{Projects: []*pb.Project{
				{
					Name:        "projects/chrome",
					DisplayName: "Chrome",
					Project:     "chrome",
				},
			}}
			So(err, ShouldBeNil)
			So(projectsResponse, ShouldResembleProto, expected)
		})

	})

}
