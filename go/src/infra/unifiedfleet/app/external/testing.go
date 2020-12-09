// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package external

import (
	"context"

	luciconfig "go.chromium.org/luci/config"
	"go.chromium.org/luci/machine-db/api/common/v1"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"

	"infra/libs/git"
	"infra/libs/sheet"
	"infra/unifiedfleet/app/frontend/fake"
)

// WithTestingContext allows for mocked external interface.
func WithTestingContext(ctx context.Context) context.Context {
	_, err := GetServerInterface(ctx)
	if err != nil {
		es := &InterfaceFactory{
			cfgInterfaceFactory:           fakeCfgInterfaceFactory,
			machineDBInterfaceFactory:     fakeMachineDBInterface,
			crosInventoryInterfaceFactory: fakeCrosInventoryInterface,
			sheetInterfaceFactory:         fakeSheetInterfaceFactory,
			gitInterfaceFactory:           fakeGitInterfaceFactory,
		}
		return context.WithValue(ctx, InterfaceFactoryKey, es)
	}
	return ctx
}

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
var testDracs = []*crimson.DRAC{
	{
		Name:       "drac-hostname",
		Machine:    "machine1",
		MacAddress: "machine1-mac1",
		Switch:     "eq017.atl97",
		Switchport: 1,
		Ipv4:       "ip1.1",
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
		Dracs:         testDracs,
	}, nil
}

func fakeCrosInventoryInterface(ctx context.Context, host string) (CrosInventoryClient, error) {
	return &fake.InventoryClient{}, nil
}

func fakeCfgInterfaceFactory(ctx context.Context) luciconfig.Interface {
	return &fake.LuciConfigClient{}
}

func fakeSheetInterfaceFactory(ctx context.Context) (sheet.ClientInterface, error) {
	return &fake.SheetClient{}, nil
}

func fakeGitInterfaceFactory(ctx context.Context, gitilesHost, project, branch string) (git.ClientInterface, error) {
	return &fake.GitClient{}, nil
}
