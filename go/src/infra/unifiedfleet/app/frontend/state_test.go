// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/genproto/googleapis/rpc/code"

	ufspb "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
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
			So(parseAssets(states, "ResourceName"), ShouldResemble, []string{"machines/machine1", "machines/machine2", "machines/machine3", "vms/vm578-m4"})
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
			_, err := tf.Fleet.UpdateState(ctx, req)
			So(err, ShouldBeNil)
		})
	})
}

func TestGetState(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Update state", t, func() {
		Convey("happy path", func() {
			req := &api.GetStateRequest{
				ResourceName: "hosts/chromeos1-row2-rack3-host4",
			}
			_, err := tf.Fleet.GetState(ctx, req)
			So(err, ShouldBeNil)
		})
	})
}
