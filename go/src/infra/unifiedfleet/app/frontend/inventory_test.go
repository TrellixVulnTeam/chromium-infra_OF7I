// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"fmt"
	"google.golang.org/grpc/codes"
	"strconv"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	code "google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	. "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

func mockMachineLSE(id string) *ufspb.MachineLSE {
	return &ufspb.MachineLSE{
		Name:     util.AddPrefix(util.MachineLSECollection, id),
		Hostname: id,
	}
}

func mockRackLSE(id string) *ufspb.RackLSE {
	return &ufspb.RackLSE{
		Name: util.AddPrefix(util.RackLSECollection, id),
	}
}

func TestCreateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("CreateMachineLSEs", t, func() {
		Convey("Create new machineLSE with machineLSE_id", func() {
			machine := &ufspb.Machine{
				Name: "machine-1",
			}
			_, err := registration.CreateMachine(ctx, machine)
			So(err, ShouldBeNil)

			machineLSE1 := mockMachineLSE("machineLSE-1")
			req := &ufsAPI.CreateMachineLSERequest{
				MachineLSE:   machineLSE1,
				MachineLSEId: "machinelse-1",
				Machines:     []string{"machine-1"},
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)
		})

		Convey("Create new machineLSE - Invalid input nil", func() {
			req := &ufsAPI.CreateMachineLSERequest{
				MachineLSE: nil,
				Machines:   []string{"machine-1"},
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new machineLSE - Invalid input empty ID", func() {
			req := &ufsAPI.CreateMachineLSERequest{
				MachineLSE:   mockMachineLSE("machineLSE-3"),
				MachineLSEId: "",
				Machines:     []string{"machine-1"},
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new machineLSE - Invalid input invalid characters", func() {
			req := &ufsAPI.CreateMachineLSERequest{
				MachineLSE:   mockMachineLSE("machineLSE-4"),
				MachineLSEId: "a.b)7&",
				Machines:     []string{"machine-1"},
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})

		Convey("Create new machineLSE - Invalid input nil machines", func() {
			req := &ufsAPI.CreateMachineLSERequest{
				MachineLSE:   mockMachineLSE("machineLSE-4"),
				MachineLSEId: "machineLSE-4",
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyMachineName)
		})

		Convey("Create new machineLSE - Invalid input empty machines", func() {
			req := &ufsAPI.CreateMachineLSERequest{
				MachineLSE:   mockMachineLSE("machineLSE-4"),
				MachineLSEId: "machineLSE-4",
				Machines:     []string{""},
			}
			resp, err := tf.Fleet.CreateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyMachineName)
		})
	})
}

func TestUpdateMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("UpdateMachineLSEs", t, func() {
		Convey("Update existing machineLSEs", func() {
			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "machineLSE-1",
			})
			So(err, ShouldBeNil)

			machineLSE := mockMachineLSE("machineLSE-1")
			req := &ufsAPI.UpdateMachineLSERequest{
				MachineLSE: machineLSE,
			}
			resp, err := tf.Fleet.UpdateMachineLSE(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE)
		})

		Convey("Update existing machineLSEs with states", func() {
			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "machineLSE-state",
			})
			So(err, ShouldBeNil)

			machineLSE := mockMachineLSE("machineLSE-state")
			req := &ufsAPI.UpdateMachineLSERequest{
				MachineLSE: machineLSE,
				States: map[string]ufspb.State{
					"machineLSE-state": ufspb.State_STATE_DEPLOYED_TESTING,
				},
			}
			resp, err := tf.Fleet.UpdateMachineLSE(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE)
			s, err := state.GetStateRecord(ctx, "hosts/machineLSE-state")
			So(err, ShouldBeNil)
			So(s.GetState(), ShouldEqual, ufspb.State_STATE_DEPLOYED_TESTING)
		})

		Convey("Update machineLSE - Invalid input nil", func() {
			req := &ufsAPI.UpdateMachineLSERequest{
				MachineLSE: nil,
			}
			resp, err := tf.Fleet.UpdateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update machineLSE - Invalid input empty name", func() {
			machineLSE4 := mockMachineLSE("")
			machineLSE4.Name = ""
			req := &ufsAPI.UpdateMachineLSERequest{
				MachineLSE: machineLSE4,
			}
			resp, err := tf.Fleet.UpdateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update machineLSE - Invalid input invalid characters", func() {
			machineLSE5 := mockMachineLSE("a.b)7&")
			req := &ufsAPI.UpdateMachineLSERequest{
				MachineLSE: machineLSE5,
			}
			resp, err := tf.Fleet.UpdateMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestGetMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("GetMachineLSE", t, func() {
		Convey("Get machineLSE by existing ID", func() {
			machineLSE1, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name: "machineLSE-1",
			})
			So(err, ShouldBeNil)
			machineLSE1.Name = util.AddPrefix(util.MachineLSECollection, machineLSE1.Name)

			req := &ufsAPI.GetMachineLSERequest{
				Name: util.AddPrefix(util.MachineLSECollection, "machineLSE-1"),
			}
			resp, err := tf.Fleet.GetMachineLSE(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, machineLSE1)
		})
		Convey("Get machineLSE by non-existing ID", func() {
			req := &ufsAPI.GetMachineLSERequest{
				Name: util.AddPrefix(util.MachineLSECollection, "machineLSE-2"),
			}
			resp, err := tf.Fleet.GetMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get machineLSE - Invalid input empty name", func() {
			req := &ufsAPI.GetMachineLSERequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Get machineLSE - Invalid input invalid characters", func() {
			req := &ufsAPI.GetMachineLSERequest{
				Name: util.AddPrefix(util.MachineLSECollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListMachineLSEs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	machineLSEs := make([]*ufspb.MachineLSE, 0, 4)
	for i := 0; i < 4; i++ {
		resp, _ := inventory.CreateMachineLSE(tf.C, &ufspb.MachineLSE{
			Name:     fmt.Sprintf("machineLSEFilter-%d", i),
			Machines: []string{"mac-1"},
		})
		resp.Name = util.AddPrefix(util.MachineLSECollection, resp.Name)
		machineLSEs = append(machineLSEs, resp)
	}
	Convey("ListMachineLSEs", t, func() {
		Convey("ListMachineLSEs - page_size negative - error", func() {
			req := &ufsAPI.ListMachineLSEsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListMachineLSEs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListMachineLSEs - Full listing - happy path", func() {
			req := &ufsAPI.ListMachineLSEsRequest{}
			resp, err := tf.Fleet.ListMachineLSEs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.MachineLSEs, ShouldResembleProto, machineLSEs)
		})

		Convey("ListMachineLSEs - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListMachineLSEsRequest{
				Filter: "machine=mac-1|rpm=rpm-2",
			}
			_, err := tf.Fleet.ListMachineLSEs(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
		})

		Convey("ListMachineLSEs - filter format valid AND", func() {
			req := &ufsAPI.ListMachineLSEsRequest{
				Filter: "machine=mac-1 & machineprototype=mlsep-1",
			}
			resp, err := tf.Fleet.ListMachineLSEs(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.MachineLSEs, ShouldBeNil)
		})

		Convey("ListMachineLSEs - filter format valid", func() {
			req := &ufsAPI.ListMachineLSEsRequest{
				Filter: "machine=mac-1",
			}
			resp, err := tf.Fleet.ListMachineLSEs(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.MachineLSEs, ShouldResembleProto, machineLSEs)
		})
		Convey("ListMachineLSEs get free vm slots", func() {
			_, err := inventory.ImportMachineLSEs(tf.C, []*ufspb.MachineLSE{
				{
					Name:         "lse-vm-1",
					Hostname:     "lse-vm-1",
					Manufacturer: "apple",
					Lab:          "mtv97",
					Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
						ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
							VmCapacity: 2,
						},
					},
				},
				{
					Name:         "lse-vm-2",
					Hostname:     "lse-vm-2",
					Manufacturer: "apple",
					Lab:          "mtv97",
					Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
						ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
							VmCapacity: 3,
						},
					},
				},
				{
					Name:         "lse-vm-3",
					Hostname:     "lse-vm-3",
					Manufacturer: "apple",
					Lab:          "mtv1234",
					Lse: &ufspb.MachineLSE_ChromeBrowserMachineLse{
						ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
							VmCapacity: 2,
						},
					},
				},
			})
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{
				{
					Hostname: "lse-vm-1",
					Vlan:     "vlan-vm",
				},
				{
					Hostname: "lse-vm-2",
					Vlan:     "vlan-vm2",
				},
			})
			So(err, ShouldBeNil)

			resp, err := tf.Fleet.ListMachineLSEs(tf.C, &ufsAPI.ListMachineLSEsRequest{
				PageSize: 3,
				Filter:   "man=apple & lab=mtv97 & free=true",
			})
			So(err, ShouldBeNil)
			So(resp.GetMachineLSEs(), ShouldHaveLength, 2)
			for _, r := range resp.GetMachineLSEs() {
				switch r.GetName() {
				case "lse-vm-1":
					So(r.GetLab(), ShouldEqual, "mtv97")
				case "lse-vm-2":
					So(r.GetLab(), ShouldEqual, "mtv97")
				}
			}

			resp, err = tf.Fleet.ListMachineLSEs(tf.C, &ufsAPI.ListMachineLSEsRequest{
				PageSize: 2,
				Filter:   "man=apple & lab=mtv97 & free=true",
			})
			So(err, ShouldBeNil)
			// 1 host is enough for 2 slots
			So(resp.GetMachineLSEs(), ShouldHaveLength, 1)
		})
	})
}

func TestDeleteMachineLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("DeleteMachineLSE", t, func() {
		Convey("Delete machineLSE by existing ID", func() {
			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machineLSE-1",
				Hostname: "machineLSE-1",
			})
			So(err, ShouldBeNil)

			req := &ufsAPI.DeleteMachineLSERequest{
				Name: util.AddPrefix(util.MachineLSECollection, "machineLSE-1"),
			}
			_, err = tf.Fleet.DeleteMachineLSE(tf.C, req)
			So(err, ShouldBeNil)

			res, err := inventory.GetMachineLSE(tf.C, "machineLSE-1")
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete machineLSE by existing ID with assigned ip", func() {
			_, err := inventory.CreateMachineLSE(ctx, &ufspb.MachineLSE{
				Name:     "machineLSE-with-ip",
				Hostname: "machineLSE-with-ip",
				Nic:      "eth0",
			})
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateDHCPs(ctx, []*ufspb.DHCPConfig{
				{
					Hostname: "machineLSE-with-ip",
					Ip:       "1.2.3.4",
				},
			})
			So(err, ShouldBeNil)
			_, err = configuration.BatchUpdateIPs(ctx, []*ufspb.IP{
				{
					Id:       "vlan:1234",
					Ipv4:     1234,
					Ipv4Str:  "1.2.3.4",
					Vlan:     "vlan",
					Occupied: true,
				},
			})
			So(err, ShouldBeNil)

			req := &ufsAPI.DeleteMachineLSERequest{
				Name: util.AddPrefix(util.MachineLSECollection, "machineLSE-with-ip"),
			}
			_, err = tf.Fleet.DeleteMachineLSE(tf.C, req)
			So(err, ShouldBeNil)

			res, err := inventory.GetMachineLSE(tf.C, "machineLSE-1")
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
			dhcp, err := configuration.GetDHCPConfig(ctx, "machineLSE-with-ip")
			So(err, ShouldNotBeNil)
			So(dhcp, ShouldBeNil)
			s, _ := status.FromError(err)
			So(s.Code(), ShouldEqual, codes.NotFound)
			ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4_str": "1.2.3.4"})
			So(err, ShouldBeNil)
			So(ips, ShouldHaveLength, 1)
			So(ips[0].GetOccupied(), ShouldBeFalse)
		})
		Convey("Delete machineLSE by non-existing ID", func() {
			req := &ufsAPI.DeleteMachineLSERequest{
				Name: util.AddPrefix(util.MachineLSECollection, "machineLSE-2"),
			}
			_, err := tf.Fleet.DeleteMachineLSE(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete machineLSE - Invalid input empty name", func() {
			req := &ufsAPI.DeleteMachineLSERequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Delete machineLSE - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteMachineLSERequest{
				Name: util.AddPrefix(util.MachineLSECollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteMachineLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestCreateRackLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rackLSE1 := mockRackLSE("rackLSE-1")
	rackLSE2 := mockRackLSE("rackLSE-2")
	Convey("CreateRackLSEs", t, func() {
		Convey("Create new rackLSE with rackLSE_id", func() {
			req := &ufsAPI.CreateRackLSERequest{
				RackLSE:   rackLSE1,
				RackLSEId: "rackLSE-1",
			}
			resp, err := tf.Fleet.CreateRackLSE(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE1)
		})

		Convey("Create existing rackLSEs", func() {
			req := &ufsAPI.CreateRackLSERequest{
				RackLSE:   rackLSE1,
				RackLSEId: "rackLSE-1",
			}
			resp, err := tf.Fleet.CreateRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, AlreadyExists)
		})

		Convey("Create new rackLSE - Invalid input nil", func() {
			req := &ufsAPI.CreateRackLSERequest{
				RackLSE: nil,
			}
			resp, err := tf.Fleet.CreateRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Create new rackLSE - Invalid input empty ID", func() {
			req := &ufsAPI.CreateRackLSERequest{
				RackLSE:   rackLSE2,
				RackLSEId: "",
			}
			resp, err := tf.Fleet.CreateRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyID)
		})

		Convey("Create new rackLSE - Invalid input invalid characters", func() {
			req := &ufsAPI.CreateRackLSERequest{
				RackLSE:   rackLSE2,
				RackLSEId: "a.b)7&",
			}
			resp, err := tf.Fleet.CreateRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestUpdateRackLSE(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rackLSE1 := mockRackLSE("rackLSE-1")
	rackLSE2 := mockRackLSE("rackLSE-1")
	rackLSE3 := mockRackLSE("rackLSE-3")
	rackLSE4 := mockRackLSE("")
	rackLSE5 := mockRackLSE("a.b)7&")
	Convey("UpdateRackLSEs", t, func() {
		Convey("Update existing rackLSEs", func() {
			req := &ufsAPI.CreateRackLSERequest{
				RackLSE:   rackLSE1,
				RackLSEId: "rackLSE-1",
			}
			resp, err := tf.Fleet.CreateRackLSE(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE1)
			ureq := &ufsAPI.UpdateRackLSERequest{
				RackLSE: rackLSE2,
			}
			resp, err = tf.Fleet.UpdateRackLSE(tf.C, ureq)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE2)
		})

		Convey("Update non-existing rackLSEs", func() {
			ureq := &ufsAPI.UpdateRackLSERequest{
				RackLSE: rackLSE3,
			}
			resp, err := tf.Fleet.UpdateRackLSE(tf.C, ureq)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})

		Convey("Update rackLSE - Invalid input nil", func() {
			req := &ufsAPI.UpdateRackLSERequest{
				RackLSE: nil,
			}
			resp, err := tf.Fleet.UpdateRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.NilEntity)
		})

		Convey("Update rackLSE - Invalid input empty name", func() {
			rackLSE4.Name = ""
			req := &ufsAPI.UpdateRackLSERequest{
				RackLSE: rackLSE4,
			}
			resp, err := tf.Fleet.UpdateRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})

		Convey("Update rackLSE - Invalid input invalid characters", func() {
			req := &ufsAPI.UpdateRackLSERequest{
				RackLSE: rackLSE5,
			}
			resp, err := tf.Fleet.UpdateRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestGetRackLSE(t *testing.T) {
	t.Parallel()
	Convey("GetRackLSE", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		rackLSE1 := mockRackLSE("rackLSE-1")
		req := &ufsAPI.CreateRackLSERequest{
			RackLSE:   rackLSE1,
			RackLSEId: "rackLSE-1",
		}
		resp, err := tf.Fleet.CreateRackLSE(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, rackLSE1)
		Convey("Get rackLSE by existing ID", func() {
			req := &ufsAPI.GetRackLSERequest{
				Name: util.AddPrefix(util.RackLSECollection, "rackLSE-1"),
			}
			resp, err := tf.Fleet.GetRackLSE(tf.C, req)
			So(err, ShouldBeNil)
			So(resp, ShouldResembleProto, rackLSE1)
		})
		Convey("Get rackLSE by non-existing ID", func() {
			req := &ufsAPI.GetRackLSERequest{
				Name: util.AddPrefix(util.RackLSECollection, "rackLSE-2"),
			}
			resp, err := tf.Fleet.GetRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Get rackLSE - Invalid input empty name", func() {
			req := &ufsAPI.GetRackLSERequest{
				Name: "",
			}
			resp, err := tf.Fleet.GetRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Get rackLSE - Invalid input invalid characters", func() {
			req := &ufsAPI.GetRackLSERequest{
				Name: util.AddPrefix(util.RackLSECollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.GetRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestListRackLSEs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	rackLSEs := make([]*ufspb.RackLSE, 0, 4)
	for i := 0; i < 4; i++ {
		resp, _ := inventory.CreateRackLSE(tf.C, &ufspb.RackLSE{
			Name:  fmt.Sprintf("rackLSE-%d", i),
			Racks: []string{"rack-1"},
		})
		resp.Name = util.AddPrefix(util.RackLSECollection, resp.Name)
		rackLSEs = append(rackLSEs, resp)
	}
	Convey("ListRackLSEs", t, func() {
		Convey("ListRackLSEs - page_size negative - error", func() {
			req := &ufsAPI.ListRackLSEsRequest{
				PageSize: -5,
			}
			resp, err := tf.Fleet.ListRackLSEs(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidPageSize)
		})

		Convey("ListRackLSEs - Full listing - happy path", func() {
			req := &ufsAPI.ListRackLSEsRequest{}
			resp, err := tf.Fleet.ListRackLSEs(tf.C, req)
			So(resp, ShouldNotBeNil)
			So(err, ShouldBeNil)
			So(resp.RackLSEs, ShouldResembleProto, rackLSEs)
		})

		Convey("ListRackLSEs - filter format invalid format OR - error", func() {
			req := &ufsAPI.ListRackLSEsRequest{
				Filter: "rack=mac-1|rpm=rpm-2",
			}
			_, err := tf.Fleet.ListRackLSEs(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidFilterFormat)
		})

		Convey("ListRackLSEs - filter format valid AND", func() {
			req := &ufsAPI.ListRackLSEsRequest{
				Filter: "rack=rack-1 & rackprototype=mlsep-1",
			}
			resp, err := tf.Fleet.ListRackLSEs(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.RackLSEs, ShouldBeNil)
		})

		Convey("ListRackLSEs - filter format valid", func() {
			req := &ufsAPI.ListRackLSEsRequest{
				Filter: "rack=rack-1",
			}
			resp, err := tf.Fleet.ListRackLSEs(tf.C, req)
			So(err, ShouldBeNil)
			So(resp.RackLSEs, ShouldResembleProto, rackLSEs)
		})
	})
}

func TestDeleteRackLSE(t *testing.T) {
	t.Parallel()
	Convey("DeleteRackLSE", t, func() {
		ctx := testingContext()
		tf, validate := newTestFixtureWithContext(ctx, t)
		defer validate()
		rackLSE1 := mockRackLSE("")
		req := &ufsAPI.CreateRackLSERequest{
			RackLSE:   rackLSE1,
			RackLSEId: "rackLSE-1",
		}
		resp, err := tf.Fleet.CreateRackLSE(tf.C, req)
		So(err, ShouldBeNil)
		So(resp, ShouldResembleProto, rackLSE1)
		Convey("Delete rackLSE by existing ID", func() {
			req := &ufsAPI.DeleteRackLSERequest{
				Name: util.AddPrefix(util.RackLSECollection, "rackLSE-1"),
			}
			_, err := tf.Fleet.DeleteRackLSE(tf.C, req)
			So(err, ShouldBeNil)
			greq := &ufsAPI.GetRackLSERequest{
				Name: util.AddPrefix(util.RackLSECollection, "rackLSE-1"),
			}
			res, err := tf.Fleet.GetRackLSE(tf.C, greq)
			So(res, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rackLSE by non-existing ID", func() {
			req := &ufsAPI.DeleteRackLSERequest{
				Name: util.AddPrefix(util.RackLSECollection, "rackLSE-2"),
			}
			_, err := tf.Fleet.DeleteRackLSE(tf.C, req)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, NotFound)
		})
		Convey("Delete rackLSE - Invalid input empty name", func() {
			req := &ufsAPI.DeleteRackLSERequest{
				Name: "",
			}
			resp, err := tf.Fleet.DeleteRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.EmptyName)
		})
		Convey("Delete rackLSE - Invalid input invalid characters", func() {
			req := &ufsAPI.DeleteRackLSERequest{
				Name: util.AddPrefix(util.RackLSECollection, "a.b)7&"),
			}
			resp, err := tf.Fleet.DeleteRackLSE(tf.C, req)
			So(resp, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, ufsAPI.InvalidCharacters)
		})
	})
}

func TestImportMachineLSEs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import machine lses", t, func() {
		Convey("happy path", func() {
			req := &ufsAPI.ImportMachineLSEsRequest{
				Source: &ufsAPI.ImportMachineLSEsRequest_MachineDbSource{
					MachineDbSource: &ufsAPI.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			tf.Fleet.importPageSize = 25
			res, err := tf.Fleet.ImportMachineLSEs(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)

			// Verify machine lse prototypes
			lps, _, err := configuration.ListMachineLSEPrototypes(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(ufsAPI.ParseResources(lps, "Name"), ShouldResemble, []string{"browser-lab:no-vm", "browser-lab:vm"})

			// Verify machine lses
			machineLSEs, _, err := inventory.ListMachineLSEs(ctx, 100, 100, "", nil, false, nil)
			So(err, ShouldBeNil)
			So(ufsAPI.ParseResources(machineLSEs, "Name"), ShouldResemble, []string{"esx-8", "web"})
			for _, r := range machineLSEs {
				switch r.GetName() {
				case "esx-8":
					So(r.GetChromeBrowserMachineLse().GetVmCapacity(), ShouldEqual, 10)
				case "web":
					So(r.GetChromeBrowserMachineLse().GetVmCapacity(), ShouldEqual, 100)
				}
			}
			lse, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", "machine1", false)
			So(err, ShouldBeNil)
			So(lse, ShouldHaveLength, 1)
			So(lse[0].GetMachineLsePrototype(), ShouldEqual, "browser-lab:vm")
			So(lse[0].GetHostname(), ShouldEqual, "esx-8")
			So(lse[0].GetChromeBrowserMachineLse().GetVms(), ShouldHaveLength, 1)
			So(lse[0].GetChromeBrowserMachineLse().GetVms()[0].GetHostname(), ShouldEqual, "vm578-m4")

			// Verify DHCPs
			dhcp, err := configuration.GetDHCPConfig(ctx, lse[0].GetHostname())
			So(err, ShouldBeNil)
			So(dhcp.GetIp(), ShouldEqual, "192.168.40.60")
			So(dhcp.GetMacAddress(), ShouldEqual, "00:3e:e1:c8:57:f9")

			// Verify IPs
			ipv4, err := util.IPv4StrToInt(dhcp.GetIp())
			So(err, ShouldBeNil)
			ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"ipv4": strconv.FormatUint(uint64(ipv4), 10)})
			So(err, ShouldBeNil)
			So(ips, ShouldHaveLength, 1)
			So(ips[0].GetOccupied(), ShouldBeTrue)
			So(ips[0].GetVlan(), ShouldEqual, "browser-lab:40")
			So(ips[0].GetId(), ShouldEqual, "browser-lab:40/3232245820")
			So(ips[0].GetIpv4Str(), ShouldEqual, "192.168.40.60")
		})
	})
}

func TestImportOSMachineLSEs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	tf, validate := newTestFixtureWithContext(ctx, t)
	defer validate()
	Convey("Import ChromeOS machine lses", t, func() {
		Convey("happy path", func() {
			req := &ufsAPI.ImportOSMachineLSEsRequest{
				Source: &ufsAPI.ImportOSMachineLSEsRequest_MachineDbSource{
					MachineDbSource: &ufsAPI.MachineDBSource{
						Host: "fake_host",
					},
				},
			}
			tf.Fleet.importPageSize = 25
			res, err := tf.Fleet.ImportOSMachineLSEs(ctx, req)
			So(err, ShouldBeNil)
			So(res.Code, ShouldEqual, code.Code_OK)

			// Verify machine lse prototypes
			lps, _, err := configuration.ListMachineLSEPrototypes(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(ufsAPI.ParseResources(lps, "Name"), ShouldResemble, []string{"acs-lab:camera", "acs-lab:wificell", "atl-lab:labstation", "atl-lab:standard"})

			// Verify machine lses
			machineLSEs, _, err := inventory.ListMachineLSEs(ctx, 100, 100, "", nil, false, nil)
			So(err, ShouldBeNil)
			So(ufsAPI.ParseResources(machineLSEs, "Name"), ShouldResemble, []string{"chromeos2-test_host", "chromeos3-test_host", "chromeos5-test_host", "test_servo"})
			// Spot check some fields
			for _, r := range machineLSEs {
				switch r.GetName() {
				case "test_host", "chromeos1-test_host", "chromeos3-test_host":
					So(r.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPools(), ShouldResemble, []string{"DUT_POOL_QUOTA", "hotrod"})
					So(r.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetSmartUsbhub(), ShouldBeTrue)
				case "test_servo":
					So(r.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetPools(), ShouldResemble, []string{"labstation_main"})
					So(r.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetRpm().GetPowerunitName(), ShouldEqual, "test_power_unit_name")
				}
			}
			lse, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", "mock_dut_id", false)
			So(err, ShouldBeNil)
			So(lse, ShouldHaveLength, 1)
			So(lse[0].GetMachineLsePrototype(), ShouldEqual, "atl-lab:standard")
			So(lse[0].GetHostname(), ShouldEqual, "chromeos2-test_host")
			lse, err = inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", "mock_camera_dut_id", false)
			So(err, ShouldBeNil)
			So(lse, ShouldHaveLength, 1)
			So(lse[0].GetMachineLsePrototype(), ShouldEqual, "acs-lab:camera")
			So(lse[0].GetHostname(), ShouldEqual, "chromeos3-test_host")
			lse, err = inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", "mock_wifi_dut_id", false)
			So(err, ShouldBeNil)
			So(lse, ShouldHaveLength, 1)
			So(lse[0].GetMachineLsePrototype(), ShouldEqual, "acs-lab:wificell")
			So(lse[0].GetHostname(), ShouldEqual, "chromeos5-test_host")
			lse, err = inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", "mock_labstation_id", false)
			So(err, ShouldBeNil)
			So(lse, ShouldHaveLength, 1)
			So(lse[0].GetMachineLsePrototype(), ShouldEqual, "atl-lab:labstation")
			So(lse[0].GetHostname(), ShouldEqual, "test_servo")
		})
	})
}
