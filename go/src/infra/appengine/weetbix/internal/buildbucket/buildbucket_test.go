// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package buildbucket

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc"

	bbpb "go.chromium.org/luci/buildbucket/proto"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetBuild(t *testing.T) {
	t.Parallel()

	Convey("Get build", t, func() {

		ctl := gomock.NewController(t)
		defer ctl.Finish()
		mc := NewMockedClient(context.Background(), ctl)

		bId := int64(87654321)
		inv := "invocations/build-87654321"

		req := &bbpb.GetBuildRequest{
			Id: bId,
			Fields: &field_mask.FieldMask{
				Paths: []string{"infra.resultdb"},
			},
		}

		resF := func(ctx context.Context, in *bbpb.GetBuildRequest, opt grpc.CallOption) (*bbpb.Build, error) {
			return &bbpb.Build{
				Infra: &bbpb.BuildInfra{
					Resultdb: &bbpb.BuildInfra_ResultDB{
						Hostname:   "results.api.cr.dev",
						Invocation: inv,
					},
				},
			}, nil
		}
		mc.GetBuild(req, resF)

		bc, err := NewClient(mc.Ctx, "bbhost")
		So(err, ShouldBeNil)
		b, err := bc.GetResultDBInfo(mc.Ctx, bId)
		So(err, ShouldBeNil)
		So(b.Infra.Resultdb.Invocation, ShouldEqual, inv)
	})
}
