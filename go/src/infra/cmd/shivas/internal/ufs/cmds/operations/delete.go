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

type delete struct {
	subcommands.CommandRunBase
}

// DeleteCmd contains delete command specification
var DeleteCmd = &subcommands.Command{
	UsageLine: "delete <sub-command>",
	ShortDesc: "Delete a resource/entity",
	LongDesc: `Delete a
	machine/rack/kvm/rpm/switch/drac/nic
	host/vm
	asset/dut/cachingservice/schedulingunit
	machine-prototype/rack-prototype/chromeplatform/vlan
	attached-device-machine (aliased as adm/attached-device-machine)
	attached-device-host (aliased as adh/attached-device-host)
	peripheral-wifi
	bluetooth-peers`,
	CommandRun: func() subcommands.CommandRun {
		c := &delete{}
		return c
	},
}

type deleteApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *delete) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&deleteApp{*d}, args)
}

// GetCommands lists all the subcommands under delete
//
// Aliases:
//   attacheddevicemachine.DeleteAttachedDeviceMachineCmd = attacheddevicemachine.DeleteADMCmd
//   attacheddevicehost.DeleteAttachedDeviceHostCmd = attacheddevicehost.DeleteADHCmd
func (c deleteApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		asset.DeleteAssetCmd,
		dut.DeleteDUTCmd,
		schedulingunit.DeleteSchedulingUnitCmd,
		machine.DeleteMachineCmd,
		attacheddevicemachine.DeleteAttachedDeviceMachineCmd,
		attacheddevicemachine.DeleteADMCmd,
		host.DeleteHostCmd,
		attacheddevicehost.DeleteAttachedDeviceHostCmd,
		attacheddevicehost.DeleteADHCmd,
		kvm.DeleteKVMCmd,
		rpm.DeleteRPMCmd,
		switches.DeleteSwitchCmd,
		drac.DeleteDracCmd,
		nic.DeleteNicCmd,
		vm.DeleteVMCmd,
		rack.DeleteRackCmd,
		machineprototype.DeleteMachineLSEPrototypeCmd,
		rackprototype.DeleteRackLSEPrototypeCmd,
		chromeplatform.DeleteChromePlatformCmd,
		cachingservice.DeleteCachingServiceCmd,
		vlan.DeleteVlanCmd,
		peripherals.DeleteBluetoothPeersCmd,
		peripherals.DeletePeripheralWifiCmd,
	}
}

// GetName is cli.Application interface implementation
func (c deleteApp) GetName() string {
	return "delete"
}
