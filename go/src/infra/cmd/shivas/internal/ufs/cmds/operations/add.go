// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package operations

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cmd/shivas/internal/ufs/subcmds/asset"
	"infra/cmd/shivas/internal/ufs/subcmds/attacheddevicehost"
	"infra/cmd/shivas/internal/ufs/subcmds/attacheddevicemachine"
	"infra/cmd/shivas/internal/ufs/subcmds/cachingservice"
	"infra/cmd/shivas/internal/ufs/subcmds/chromeplatform"
	"infra/cmd/shivas/internal/ufs/subcmds/drac"
	"infra/cmd/shivas/internal/ufs/subcmds/dut"
	"infra/cmd/shivas/internal/ufs/subcmds/host"
	"infra/cmd/shivas/internal/ufs/subcmds/kvm"
	"infra/cmd/shivas/internal/ufs/subcmds/labstation"
	"infra/cmd/shivas/internal/ufs/subcmds/machine"
	"infra/cmd/shivas/internal/ufs/subcmds/machineprototype"
	"infra/cmd/shivas/internal/ufs/subcmds/nic"
	"infra/cmd/shivas/internal/ufs/subcmds/peripherals"
	"infra/cmd/shivas/internal/ufs/subcmds/rack"
	"infra/cmd/shivas/internal/ufs/subcmds/rackprototype"
	"infra/cmd/shivas/internal/ufs/subcmds/rpm"
	"infra/cmd/shivas/internal/ufs/subcmds/schedulingunit"
	"infra/cmd/shivas/internal/ufs/subcmds/switches"
	"infra/cmd/shivas/internal/ufs/subcmds/vlan"
	"infra/cmd/shivas/internal/ufs/subcmds/vm"
)

type add struct {
	subcommands.CommandRunBase
}

// AddCmd contains add command specification
var AddCmd = &subcommands.Command{
	UsageLine: "add <sub-command>",
	ShortDesc: "Add details of a resource/entity",
	LongDesc: `Add details for
	machine/rack/kvm/rpm/switch/drac/nic
	host/vm
	asset/dut/labstation/cachingservice/schedulingunit
	machine-prototype/rack-prototype/chromeplatform/vlan
	attached-device-machine (aliased as adm/attached-device-machine)
	attached-device-host (aliased as adh/attached-device-host)
	peripheral-wifi
	bluetooth-peers`,
	CommandRun: func() subcommands.CommandRun {
		c := &add{}
		return c
	},
}

type addApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *add) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&addApp{*d}, args)
}

// GetCommands lists all the subcommands under add
// Aliases:
//   attacheddevicemachine.AddAttachedDeviceMachineCmd = attacheddevicemachine.AddADMCmd
//   attacheddevicehost.AddAttachedDeviceHostCmd = attacheddevicehost.AddADHCmd
func (c addApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		asset.AddAssetCmd,
		dut.AddDUTCmd,
		labstation.AddLabstationCmd,
		cachingservice.AddCachingServiceCmd,
		schedulingunit.AddSchedulingUnitCmd,
		machine.AddMachineCmd,
		attacheddevicemachine.AddAttachedDeviceMachineCmd,
		attacheddevicemachine.AddADMCmd,
		host.AddHostCmd,
		attacheddevicehost.AddAttachedDeviceHostCmd,
		attacheddevicehost.AddADHCmd,
		kvm.AddKVMCmd,
		rpm.AddRPMCmd,
		switches.AddSwitchCmd,
		drac.AddDracCmd,
		nic.AddNicCmd,
		vm.AddVMCmd,
		rack.AddRackCmd,
		machineprototype.AddMachineLSEPrototypeCmd,
		rackprototype.AddRackLSEPrototypeCmd,
		chromeplatform.AddChromePlatformCmd,
		vlan.AddVlanCmd,
		peripherals.AddBluetoothPeersCmd,
		peripherals.AddPeripheralWifiCmd,
	}
}

// GetName is cli.Application interface implementation
func (c addApp) GetName() string {
	return "add"
}
