// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

func TestGetState(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	bm1, _ := state.UpdateStateRecord(ctx, &ufspb.StateRecord{
		ResourceName: "machine/browser-machine-1",
	})
	state.UpdateStateRecord(ctx, &ufspb.StateRecord{
		ResourceName: "machine/os-machine-1",
		State:        ufspb.State_STATE_REGISTERED,
	})
	os2Registered, _ := state.UpdateStateRecord(ctx, &ufspb.StateRecord{
		ResourceName: "machine/os-machine-2",
		State:        ufspb.State_STATE_REGISTERED,
	})
	osCtx, _ := util.SetupDatastoreNamespace(ctx, util.OSNamespace)
	os1Serving, _ := state.UpdateStateRecord(osCtx, &ufspb.StateRecord{
		ResourceName: "machine/os-machine-1",
		State:        ufspb.State_STATE_SERVING,
	})
	os3Serving, _ := state.UpdateStateRecord(osCtx, &ufspb.StateRecord{
		ResourceName: "machine/os-machine-3",
		State:        ufspb.State_STATE_SERVING,
	})
	Convey("GetState", t, func() {
		Convey("GetState for a browser machine with default namespace context", func() {
			res, err := GetState(ctx, "machine/browser-machine-1")
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, bm1)
		})

		Convey("GetState for a os machine with default namespace context", func() {
			res, err := GetState(ctx, "machine/os-machine-1")
			So(err, ShouldBeNil)
			// TODO(eshwarn): change this check when fall back read is removed
			So(res, ShouldResembleProto, os1Serving)
			res, err = GetState(ctx, "machine/os-machine-2")
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, os2Registered)
			res, err = GetState(ctx, "machine/os-machine-3")
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, os3Serving)
		})

		Convey("GetState for a os machine with os namespace context", func() {
			res, err := GetState(osCtx, "machine/os-machine-1")
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, os1Serving)
			res, err = GetState(osCtx, "machine/os-machine-2")
			So(err, ShouldBeNil)
			// TODO(eshwarn): change this check when fall back read is removed
			So(res, ShouldResembleProto, os2Registered)
			res, err = GetState(osCtx, "machine/os-machine-3")
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, os3Serving)
		})
	})
}
