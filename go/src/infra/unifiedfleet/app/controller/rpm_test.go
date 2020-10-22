// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/models"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
)

func mockRPM(id string) *ufspb.RPM {
	return &ufspb.RPM{
		Name: id,
	}
}

func TestDeleteRPM(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	RPM1 := mockRPM("RPM-1")
	RPM3 := mockRPM("RPM-3")
	RPM4 := mockRPM("RPM-4")
	Convey("DeleteRPM", t, func() {
		Convey("Delete RPM by existing ID with machine reference", func() {
			resp, cerr := CreateRPM(ctx, RPM1)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)

			chromeBrowserMachine1 := &ufspb.Machine{
				Name: "machine-1",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						RpmInterface: &ufspb.RPMInterface{
							Rpm: "RPM-1",
						},
					},
				},
			}
			mresp, merr := registration.CreateMachine(ctx, chromeBrowserMachine1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeBrowserMachine1)

			err := DeleteRPM(ctx, "RPM-1")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetRPM(ctx, "RPM-1")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM1)
		})

		Convey("Delete RPM by existing ID with racklse reference", func() {
			resp, cerr := CreateRPM(ctx, RPM3)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM3)

			chromeOSRackLSE1 := &ufspb.RackLSE{
				Name: "racklse-1",
				Lse: &ufspb.RackLSE_ChromeosRackLse{
					ChromeosRackLse: &ufspb.ChromeOSRackLSE{
						Rpms: []string{"RPM-3", "RPM-5"},
					},
				},
			}
			mresp, merr := inventory.CreateRackLSE(ctx, chromeOSRackLSE1)
			So(merr, ShouldBeNil)
			So(mresp, ShouldResembleProto, chromeOSRackLSE1)

			err := DeleteRPM(ctx, "RPM-3")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, CannotDelete)

			resp, cerr = GetRPM(ctx, "RPM-3")
			So(resp, ShouldNotBeNil)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM3)
		})
		Convey("Delete RPM successfully by existing ID without references", func() {
			resp, cerr := CreateRPM(ctx, RPM4)
			So(cerr, ShouldBeNil)
			So(resp, ShouldResembleProto, RPM4)

			err := DeleteRPM(ctx, "RPM-4")
			So(err, ShouldBeNil)

			resp, cerr = GetRPM(ctx, "RPM-4")
			So(resp, ShouldBeNil)
			So(cerr, ShouldNotBeNil)
			So(cerr.Error(), ShouldContainSubstring, NotFound)
		})
	})
}

func TestListRPMs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	rpms := make([]*ufspb.RPM, 0, 2)
	for i := 0; i < 4; i++ {
		rpm := mockRPM(fmt.Sprintf("rpm-%d", i))
		resp, _ := registration.CreateRPM(ctx, rpm)
		rpms = append(rpms, resp)
	}
	Convey("ListRPMs", t, func() {
		Convey("List RPMs - filter invalid - error", func() {
			_, _, err := ListRPMs(ctx, 5, "", "invalid=mx-1", false)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Invalid field name invalid")
		})

		Convey("ListRPMs - Full listing - happy path", func() {
			resp, _, _ := ListRPMs(ctx, 5, "", "", false)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, rpms)
		})
	})
}

func TestBatchGetRPMs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetRPMs", t, func() {
		Convey("Batch get rpms - happy path", func() {
			entities := make([]*ufspb.RPM, 4)
			for i := 0; i < 4; i++ {
				entities[i] = &ufspb.RPM{
					Name: fmt.Sprintf("rpm-batchGet-%d", i),
				}
			}
			_, err := registration.BatchUpdateRPMs(ctx, entities)
			So(err, ShouldBeNil)
			resp, err := registration.BatchGetRPMs(ctx, []string{"rpm-batchGet-0", "rpm-batchGet-1", "rpm-batchGet-2", "rpm-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, entities)
		})
		Convey("Batch get rpms  - missing id", func() {
			resp, err := registration.BatchGetRPMs(ctx, []string{"rpm-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "rpm-batchGet-non-existing")
		})
		Convey("Batch get rpms  - empty input", func() {
			resp, err := registration.BatchGetRPMs(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = registration.BatchGetRPMs(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}
