// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"

	proto "infra/unifiedfleet/api/v1/proto"
)

func TestImportSwitches(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	Convey("import nics", t, func() {
		switches := []*proto.Switch{
			mockSwitch("nic1", 10),
			mockSwitch("nic2", 20),
		}
		Convey("happy path", func() {
			resp, err := ImportSwitches(ctx, switches)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, len(switches))
			getRes, _, err := ListSwitches(ctx, 100, "")
			So(err, ShouldBeNil)
			So(getRes, ShouldResembleProto, switches)
		})
		Convey("happy path also for importing existing switches", func() {
			switch1 := []*proto.Switch{
				mockSwitch("nic1", 100),
			}
			resp, err := ImportSwitches(ctx, switch1)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, len(switch1))
			s, err := GetSwitch(ctx, "nic1")
			So(err, ShouldBeNil)
			So(s.GetCapacityPort(), ShouldEqual, 100)
		})
	})
}

func mockSwitch(id string, capacity int) *proto.Switch {
	return &proto.Switch{
		Name:         id,
		CapacityPort: int32(capacity),
	}
}
