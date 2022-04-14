// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
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
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"
)

func TestInitData(t *testing.T) {
	Convey("Given an init data server", t, func() {
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

		server := NewInitDataGeneratorServer()
		cfg, err := config.CreatePlaceholderConfig()
		So(err, ShouldBeNil)

		config.SetTestConfig(ctx, cfg)

		Convey("Unauthorised requests are rejected", func() {
			ctx = auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:someone@example.com",
				// Not a member of weetbix-access.
				IdentityGroups: []string{"other-group"},
			})

			// Make some request (the request should not matter, as
			// a common decorator is used for all requests.)
			request := &pb.GenerateInitDataRequest{
				ReferrerUrl: "/p/chromium",
			}

			rule, err := server.GenerateInitData(ctx, request)
			st, _ := grpcStatus.FromError(err)
			So(st.Code(), ShouldEqual, codes.PermissionDenied)
			So(st.Message(), ShouldEqual, "not a member of weetbix-access")
			So(rule, ShouldBeNil)
		})
		Convey("When getting data", func() {
			request := &pb.GenerateInitDataRequest{
				ReferrerUrl: "/p/chromium",
			}

			result, err := server.GenerateInitData(ctx, request)

			So(err, ShouldBeNil)

			expected := &pb.GenerateInitDataResponse{
				InitData: &pb.InitData{
					Hostnames: &pb.Hostnames{
						MonorailHostname: "monorail-test.appspot.com",
					},
					User: &pb.User{
						Email: "someone@example.com",
					},
					AuthUrls: &pb.AuthUrls{
						LogoutUrl: "http://fake.example.com/logout?dest=%2Fp%2Fchromium",
					},
				},
			}

			So(result, ShouldResembleProto, expected)
		})
	})
}
