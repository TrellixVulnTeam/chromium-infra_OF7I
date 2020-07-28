// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	. "infra/unifiedfleet/app/model/datastore"
)

func TestCreateSwitch(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	switch1 := mockSwitch("Switch-1")
	switch2 := mockSwitch("")
	Convey("CreateSwitch", t, func() {
		Convey("Create new switch", func() {
			resp, err := CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)
		})
		Convey("Create existing switch", func() {
			resp, err := CreateSwitch(ctx, switch1)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})
		Convey("Create switch - invalid ID", func() {
			resp, err := CreateSwitch(ctx, switch2)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestUpdateSwitch(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	switch1 := mockSwitch("Switch-1")
	switch2 := mockSwitch("Switch-1")
	switch3 := mockSwitch("Switch-3")
	switch4 := mockSwitch("")
	Convey("UpdateSwitch", t, func() {
		Convey("Update existing switch", func() {
			resp, err := CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)

			resp, err = UpdateSwitch(ctx, switch2)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch2)
		})
		Convey("Update non-existing switch", func() {
			resp, err := UpdateSwitch(ctx, switch3)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Update switch - invalid ID", func() {
			resp, err := UpdateSwitch(ctx, switch4)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestGetSwitch(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	switch1 := mockSwitch("Switch-1")
	Convey("GetSwitch", t, func() {
		Convey("Get switch by existing ID", func() {
			resp, err := CreateSwitch(ctx, switch1)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)
			resp, err = GetSwitch(ctx, "Switch-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switch1)
		})
		Convey("Get switch by non-existing ID", func() {
			resp, err := GetSwitch(ctx, "switch-2")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get switch - invalid ID", func() {
			resp, err := GetSwitch(ctx, "")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestListSwitches(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	switches := make([]*ufspb.Switch, 0, 4)
	for i := 0; i < 4; i++ {
		switch1 := mockSwitch(fmt.Sprintf("switch-%d", i))
		resp, _ := CreateSwitch(ctx, switch1)
		switches = append(switches, resp)
	}
	Convey("ListSwitches", t, func() {
		Convey("List switches - page_token invalid", func() {
			resp, nextPageToken, err := ListSwitches(ctx, 5, "abc", nil, false)
			So(resp, ShouldBeNil)
			So(nextPageToken, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InvalidPageToken)
		})

		Convey("List switches - Full listing with no pagination", func() {
			resp, nextPageToken, err := ListSwitches(ctx, 4, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switches)
		})

		Convey("List switches - listing with pagination", func() {
			resp, nextPageToken, err := ListSwitches(ctx, 3, "", nil, false)
			So(resp, ShouldNotBeNil)
			So(nextPageToken, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switches[:3])

			resp, _, err = ListSwitches(ctx, 2, nextPageToken, nil, false)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, switches[3:])
		})
	})
}

func TestDeleteSwitch(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	switch2 := mockSwitch("switch-2")
	Convey("DeleteSwitch", t, func() {
		Convey("Delete switch successfully by existing ID", func() {
			resp, cerr := CreateSwitch(ctx, switch2)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, switch2)

			err := DeleteSwitch(ctx, "switch-2")
			So(err, ShouldBeNil)

			resp, cerr = GetSwitch(ctx, "switch-2")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete switch by non-existing ID", func() {
			err := DeleteSwitch(ctx, "switch-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete switch - invalid ID", func() {
			err := DeleteSwitch(ctx, "")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, InternalError)
		})
	})
}

func TestImportSwitches(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	Convey("import switches", t, func() {
		switches := []*ufspb.Switch{
			mockSwitch("switch1"),
			mockSwitch("switch2"),
		}
		Convey("happy path", func() {
			resp, err := ImportSwitches(ctx, switches)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, len(switches))
			getRes, _, err := ListSwitches(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(getRes, ShouldResembleProto, switches)
		})
		Convey("happy path also for importing existing switches", func() {
			switch1 := []*ufspb.Switch{
				mockSwitch("switch1"),
			}
			resp, err := ImportSwitches(ctx, switch1)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, len(switch1))
			s, err := GetSwitch(ctx, "switch1")
			So(err, ShouldBeNil)
			So(s, ShouldResembleProto, switch1[0])
		})
	})
}

func mockSwitch(id string) *ufspb.Switch {
	return &ufspb.Switch{
		Name: id,
	}
}
