// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc/status"

	proto "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"

	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
)

var localPlatforms = []*crimsonconfig.Platform{
	{Name: "platform1"},
	{Name: "platform2"},
	{Name: "platform3"},
}

func mockParsePlatformsFunc(path string) (*crimsonconfig.Platforms, error) {
	return &crimsonconfig.Platforms{
		Platform: localPlatforms,
	}, nil
}

func TestImportChromePlatforms(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import chrome platforms", t, func() {
		Convey("happy path", func() {
			req := &api.ImportChromePlatformsRequest{
				Source: &api.ImportChromePlatformsRequest_ConfigSource{
					ConfigSource: &api.ConfigSource{
						ConfigServiceName: "",
						FileName:          "test.config",
					},
				},
			}
			parsePlatformsFunc = mockParsePlatformsFunc
			res, err := tf.Fleet.ImportChromePlatforms(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)
			getRes, err := configuration.GetAllChromePlatforms(ctx)
			So(err, ShouldBeNil)
			So(getRes, ShouldHaveLength, len(localPlatforms))
			wants := getLocalPlatformNames()
			gets := getReturnedPlatformNames(*getRes)
			So(gets, ShouldResemble, wants)
		})
		Convey("import platforms with invalid argument", func() {
			req := &api.ImportChromePlatformsRequest{
				Source: &api.ImportChromePlatformsRequest_ConfigSource{},
			}
			_, err := tf.Fleet.ImportChromePlatforms(ctx, req)
			So(err, ShouldNotBeNil)
			s, ok := status.FromError(err)
			So(ok, ShouldBeTrue)
			So(s.Code(), ShouldEqual, code.Code_INVALID_ARGUMENT)
		})
	})
}

func getLocalPlatformNames() []string {
	wants := make([]string, len(localPlatforms))
	for i, p := range localPlatforms {
		wants[i] = p.GetName()
	}
	return wants
}

func getReturnedPlatformNames(res datastore.OpResults) []string {
	gets := make([]string, len(res))
	for i, r := range res {
		gets[i] = r.Data.(*proto.ChromePlatform).GetName()
	}
	return gets
}
