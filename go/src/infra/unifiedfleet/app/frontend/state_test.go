// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/genproto/googleapis/rpc/code"

	ufspb "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/state"
)

func TestImportStates(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import machine lses", t, func() {
		Convey("happy path", func() {
			req := &api.ImportStatesRequest{
				Source: &api.ImportStatesRequest_MachineDbSource{
					MachineDbSource: &api.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			res, err := tf.Fleet.ImportStates(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)

			states, _, err := state.ListStateRecords(ctx, 100, "")
			So(err, ShouldBeNil)
			So(api.ParseResources(states, "ResourceName"), ShouldResemble, []string{"machines/machine1", "machines/machine2", "machines/machine3", "vms/vm578-m4"})
			s, err := state.GetStateRecord(ctx, "machines/machine1")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_SERVING)
			s, err = state.GetStateRecord(ctx, "vms/vm578-m4")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_NEEDS_REPAIR)
		})
	})
}

func TestUpdateState(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Update state", t, func() {
		Convey("happy path", func() {
			req := &api.UpdateStateRequest{
				State: &ufspb.StateRecord{
					ResourceName: "hosts/chromeos1-row2-rack3-host4",
					State:        ufspb.State_STATE_RESERVED,
				},
			}
			res, err := tf.Fleet.UpdateState(ctx, req)
			So(err, ShouldBeNil)
			s, err := state.GetStateRecord(ctx, "hosts/chromeos1-row2-rack3-host4")
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, s)
		})
		Convey("invalid resource prefix", func() {
			req := &api.UpdateStateRequest{
				State: &ufspb.StateRecord{
					ResourceName: "resources/chromeos1-row2-rack3-host4",
					State:        ufspb.State_STATE_RESERVED,
				},
			}
			_, err := tf.Fleet.UpdateState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
		Convey("empty resource name", func() {
			req := &api.UpdateStateRequest{
				State: &ufspb.StateRecord{
					ResourceName: "",
					State:        ufspb.State_STATE_RESERVED,
				},
			}
			_, err := tf.Fleet.UpdateState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
		Convey("invalid characters in resource name", func() {
			req := &api.UpdateStateRequest{
				State: &ufspb.StateRecord{
					ResourceName: "hosts/host1@_@",
					State:        ufspb.State_STATE_RESERVED,
				},
			}
			_, err := tf.Fleet.UpdateState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
	})
}

func TestGetState(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Get state", t, func() {
		Convey("happy path", func() {
			s := &ufspb.StateRecord{
				ResourceName: "hosts/chromeos1-row2-rack3-host4",
				State:        ufspb.State_STATE_RESERVED,
			}
			_, err := state.UpdateStateRecord(ctx, s)
			So(err, ShouldBeNil)
			req := &api.GetStateRequest{
				ResourceName: "hosts/chromeos1-row2-rack3-host4",
			}
			res, err := tf.Fleet.GetState(ctx, req)
			So(err, ShouldBeNil)
			So(res, ShouldResembleProto, s)
		})
		Convey("valid resource name, but not found", func() {
			res, err := tf.Fleet.GetState(ctx, &api.GetStateRequest{
				ResourceName: "hosts/chromeos-fakehost",
			})
			So(err, ShouldNotBeNil)
			So(res, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, datastore.NotFound)
		})
		Convey("invalid resource prefix", func() {
			req := &api.GetStateRequest{
				ResourceName: "resources/chromeos1-row2-rack3-host4",
			}
			_, err := tf.Fleet.GetState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
		Convey("empty resource name", func() {
			req := &api.GetStateRequest{
				ResourceName: "",
			}
			_, err := tf.Fleet.GetState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
		Convey("invalid characters in resource name", func() {
			req := &api.GetStateRequest{
				ResourceName: "hosts/host1@_@",
			}
			_, err := tf.Fleet.GetState(ctx, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, api.ResourceFormat)
		})
	})
}
