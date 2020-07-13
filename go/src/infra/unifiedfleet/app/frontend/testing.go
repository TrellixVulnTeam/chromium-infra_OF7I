// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package frontend

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/gologger"
	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/machine-db/api/common/v1"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"

	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/frontend/fake"
)

type testFixture struct {
	T *testing.T
	C context.Context

	Fleet *FleetServerImpl
}

func newTestFixtureWithContext(ctx context.Context, t *testing.T) (testFixture, func()) {
	tf := testFixture{T: t, C: ctx}
	mc := gomock.NewController(t)

	tf.Fleet = &FleetServerImpl{
		cfgInterfaceFactory:           fakeCfgInterfaceFactory,
		machineDBInterfaceFactory:     fakeMachineDBInterface,
		crosInventoryInterfaceFactory: fakeCrosInventoryInterface,
		importPageSize:                testImportPageSize,
	}

	validate := func() {
		mc.Finish()
	}
	return tf, validate
}

func testingContext() context.Context {
	c := gaetesting.TestingContextWithAppID("dev~infra-unified-fleet-system")
	c = gologger.StdConfig.Use(c)
	c = logging.SetLevel(c, logging.Debug)
	c = config.Use(c, &config.Config{
		CrosNetworkConfig: &config.OSNetworkConfig{
			GitilesHost: "test_gitiles",
			Project:     "test_project",
			Branch:      "test_branch",
			CrosNetworkTopology: []*config.OSNetworkConfig_OSNetworkTopology{
				{
					Name:       "test_name",
					RemotePath: "test_git_path",
					SheetId:    "test_sheet",
				},
			},
		},
	})
	datastore.GetTestable(c).Consistent(true)
	return c
}

var testImportPageSize = 2
var testMachines = []*crimson.Machine{
	{
		Name:     "machine1",
		Platform: "fake platform",
		State:    common.State_SERVING,
	},
	{
		Name:     "machine2",
		Platform: "fake platform",
		State:    common.State_SERVING,
	},
	{
		Name:     "machine3",
		Platform: "fake platform2",
		State:    common.State_SERVING,
	},
}

var testNics = []*crimson.NIC{
	{
		Name:       "drac",
		Machine:    "machine1",
		MacAddress: "machine1-mac1",
		Switch:     "eq017.atl97",
		Switchport: 1,
		Ipv4:       "ip1.1",
	},
	{
		Name:       "eth0",
		Machine:    "machine1",
		MacAddress: "machine1-mac2",
		Switch:     "eq017.atl97",
		Switchport: 2,
		Ipv4:       "ip1.2",
	},
	{
		Name:       "eth1",
		Machine:    "machine1",
		MacAddress: "machine1-mac3",
		Switch:     "eq017.atl97",
		Switchport: 3,
		Ipv4:       "ip1.3",
	},
	{
		Name:       "eth0",
		Machine:    "machine2",
		MacAddress: "machine2-mac1",
		Switch:     "eq017.atl97",
		Switchport: 4,
		Ipv4:       "ip2",
	},
	{
		Name:       "eth0",
		Machine:    "machine3",
		MacAddress: "machine3-mac1",
		Switch:     "eq041.atl97",
		Switchport: 1,
		Ipv4:       "ip3",
	},
}
var testPhysicalHosts = []*crimson.PhysicalHost{
	{
		Name:       "esx-8",
		Vlan:       40,
		Machine:    "machine1",
		Os:         "testing-os",
		Ipv4:       "192.168.40.60",
		MacAddress: "00:3e:e1:c8:57:f9",
		Nic:        "eth0",
		VmSlots:    10,
	},
	{
		Name:       "web",
		Vlan:       20,
		Machine:    "machine2",
		Os:         "testing-os",
		Ipv4:       "192.168.20.18",
		MacAddress: "84:2b:2b:01:b9:c6",
		Nic:        "eth0",
		VmSlots:    100,
	},
}
var testVMs = []*crimson.VM{
	{
		Name:     "vm578-m4",
		Vlan:     144,
		Ipv4:     "192.168.144.63",
		Host:     "esx-8",
		HostVlan: 40,
		Os:       "testing-vm",
		State:    common.State_REPAIR,
	},
}

func fakeMachineDBInterface(ctx context.Context, host string) (crimson.CrimsonClient, error) {
	return &fake.CrimsonClient{
		Machines:      testMachines,
		Nics:          testNics,
		PhysicalHosts: testPhysicalHosts,
		Vms:           testVMs,
	}, nil
}

func fakeCrosInventoryInterface(ctx context.Context, host string) (CrosInventoryClient, error) {
	return &fake.InventoryClient{}, nil
}

func fakeCfgInterfaceFactory(ctx context.Context) luciconfig.Interface {
	return &fake.LuciConfigClient{}
}
