// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/registration"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func mockDrac(id string) *ufspb.Drac {
	return &ufspb.Drac{
		Name: id,
	}
}

func TestCreateDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	machine1 := &ufspb.Machine{
		Name: "machine-1",
		Device: &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
		},
	}
	registration.CreateMachine(ctx, machine1)
	Convey("CreateDrac", t, func() {
		Convey("Create new drac with non existing machine", func() {
			drac1 := &ufspb.Drac{
				Name: "drac-1",
			}
			resp, err := CreateDrac(ctx, drac1, "machine-5")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-5 in the system.")
		})

		Convey("Create new drac with existing machine with drac", func() {
			machine := &ufspb.Machine{
				Name: "machine-10",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Drac: "drac-5",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			_, err = registration.CreateDrac(ctx, &ufspb.Drac{
				Name: "drac-5",
			})
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name: "drac-20",
			}
			resp, err := CreateDrac(ctx, drac, "machine-10")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac)

			mresp, err := registration.GetMachine(ctx, "machine-10")
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserMachine().GetDrac(), ShouldResemble, "drac-20")
		})

		Convey("Create new drac with existing machine without drac", func() {
			machine := &ufspb.Machine{
				Name: "machine-15",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name: "drac-25",
			}
			resp, err := CreateDrac(ctx, drac, "machine-15")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac)

			mresp, err := registration.GetMachine(ctx, "machine-15")
			So(err, ShouldBeNil)
			So(mresp.GetChromeBrowserMachine().GetDrac(), ShouldResemble, "drac-25")
		})

		Convey("Create new drac with non existing switch", func() {
			drac1 := &ufspb.Drac{
				Name: "drac-1",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := CreateDrac(ctx, drac1, "machine-1")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Switch with SwitchID switch-1")
		})

		Convey("Create new drac with existing switch", func() {
			switch2 := &ufspb.Switch{
				Name: "switch-2",
			}
			_, err := registration.CreateSwitch(ctx, switch2)
			So(err, ShouldBeNil)

			drac2 := &ufspb.Drac{
				Name: "drac-2",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-2",
				},
			}
			resp, err := CreateDrac(ctx, drac2, "machine-1")
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, drac2)
		})
	})
}

func TestUpdateDrac(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("UpdateDrac", t, func() {
		Convey("Update drac with non-existing drac", func() {
			machine1 := &ufspb.Machine{
				Name: "machine-1",
			}
			registration.CreateMachine(ctx, machine1)
			drac := &ufspb.Drac{
				Name: "drac-1",
			}
			resp, err := UpdateDrac(ctx, drac, "machine-1")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Drac with DracID drac-1 in the system.")
		})

		Convey("Update drac with non existing switch", func() {
			drac := &ufspb.Drac{
				Name: "drac-2",
			}
			_, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			drac2 := &ufspb.Drac{
				Name: "drac-2",
				SwitchInterface: &ufspb.SwitchInterface{
					Switch: "switch-1",
				},
			}
			resp, err := UpdateDrac(ctx, drac2, "machine-1")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Switch with SwitchID switch-1")
		})

		Convey("Update drac with new machine", func() {
			machine3 := &ufspb.Machine{
				Name: "machine-3",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Drac: "drac-3",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine3)
			So(err, ShouldBeNil)

			machine4 := &ufspb.Machine{
				Name: "machine-4",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Drac: "drac-4",
					},
				},
			}
			_, err = registration.CreateMachine(ctx, machine4)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name: "drac-3",
			}
			_, err = registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			_, err = registration.CreateDrac(ctx, &ufspb.Drac{
				Name: "drac-4",
			})
			So(err, ShouldBeNil)

			drac = &ufspb.Drac{
				Name: "drac-3",
			}
			resp, err := UpdateDrac(ctx, drac, "machine-4")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, drac)

			mresp, err := registration.GetMachine(ctx, "machine-3")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(mresp.GetChromeBrowserMachine().GetDrac(), ShouldResemble, "")

			mresp, err = registration.GetMachine(ctx, "machine-4")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(mresp.GetChromeBrowserMachine().GetDrac(), ShouldResemble, "drac-3")

			_, err = registration.GetDrac(ctx, "drac-4")
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "Entity not found.")
		})

		Convey("Update drac with same machine", func() {
			machine := &ufspb.Machine{
				Name: "machine-5",
				Device: &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
						Drac: "drac-5",
					},
				},
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			drac := &ufspb.Drac{
				Name: "drac-5",
			}
			_, err = registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			drac = &ufspb.Drac{
				Name:       "drac-5",
				MacAddress: "ab:cd:ef",
			}
			resp, err := UpdateDrac(ctx, drac, "machine-5")
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			So(resp, ShouldResembleProto, drac)
		})

		Convey("Update drac with non existing machine", func() {
			drac := &ufspb.Drac{
				Name: "drac-6",
			}
			_, err := registration.CreateDrac(ctx, drac)
			So(err, ShouldBeNil)

			drac = &ufspb.Drac{
				Name: "drac-6",
			}
			resp, err := UpdateDrac(ctx, drac, "machine-6")
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "There is no Machine with MachineID machine-6 in the system.")
		})

	})
}
