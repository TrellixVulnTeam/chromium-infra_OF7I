// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package buildbucket

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/protobuf/proto"

	bbpb "go.chromium.org/luci/buildbucket/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestGetBuild(t *testing.T) {
	t.Parallel()

	Convey("Get build", t, func() {

		ctl := gomock.NewController(t)
		defer ctl.Finish()
		mc := NewMockedClient(context.Background(), ctl)

		bID := int64(87654321)
		inv := "invocations/build-87654321"

		req := &bbpb.GetBuildRequest{
			Id: bID,
			Mask: &bbpb.BuildMask{
				Fields: &field_mask.FieldMask{
					Paths: []string{"builder", "infra.resultdb", "status"},
				},
			},
		}

		res := &bbpb.Build{
			Builder: &bbpb.BuilderID{
				Project: "chromium",
				Bucket:  "ci",
				Builder: "builder",
			},
			Infra: &bbpb.BuildInfra{
				Resultdb: &bbpb.BuildInfra_ResultDB{
					Hostname:   "results.api.cr.dev",
					Invocation: inv,
				},
			},
			Status: bbpb.Status_FAILURE,
		}
		reqCopy := proto.Clone(req).(*bbpb.GetBuildRequest)
		mc.GetBuild(reqCopy, res)

		bc, err := NewClient(mc.Ctx, "bbhost")
		So(err, ShouldBeNil)
		b, err := bc.GetBuild(mc.Ctx, req)
		So(err, ShouldBeNil)
		So(b, ShouldResembleProto, res)
	})
}
