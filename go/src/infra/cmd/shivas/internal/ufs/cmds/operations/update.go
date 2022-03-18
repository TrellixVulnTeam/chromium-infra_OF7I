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
	"infra/cmd/shivas/internal/ufs/subcmds/lsedeployment"
	"infra/cmd/shivas/internal/ufs/subcmds/machine"
	"infra/cmd/shivas/internal/ufs/subcmds/machineprototype"
	"infra/cmd/shivas/internal/ufs/subcmds/nic"
	"infra/cmd/shivas/internal/ufs/subcmds/rack"
	"infra/cmd/shivas/internal/ufs/subcmds/rackprototype"
	"infra/cmd/shivas/internal/ufs/subcmds/rpm"
	"infra/cmd/shivas/internal/ufs/subcmds/schedulingunit"
	"infra/cmd/shivas/internal/ufs/subcmds/switches"
	"infra/cmd/shivas/internal/ufs/subcmds/vlan"
	"infra/cmd/shivas/internal/ufs/subcmds/vm"
)

type update struct {
	subcommands.CommandRunBase
}

// UpdateCmd contains update command specification
var UpdateCmd = &subcommands.Command{
	UsageLine: "update <sub-command>",
	ShortDesc: "Update details of a resource/entity",
	LongDesc: `Update details for
	machine/rack/kvm/rpm/switch/drac/nic
	host/vm
	asset/dut/labstation/cachingservice/schedulingunit
	machine-prototype/rack-prototype/chromeplatform/vlan/host-deployment
	attached-device-machine (aliased as adm/attached-device-machine)
	attached-device-host (aliased as adh/attached-device-host)`,
	CommandRun: func() subcommands.CommandRun {
		c := &update{}
		return c
	},
}

type updateApp struct {
	cli.Application
}

// Run implementing subcommands.CommandRun interface
func (c *update) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	d := a.(*cli.Application)
	return subcommands.Run(&updateApp{*d}, args)
}

// GetCommands lists all the subcommands under update
// Aliases:
//   attacheddevicemachine.UpdateAttachedDeviceMachineCmd = attacheddevicemachine.UpdateADMCmd
//   attacheddevicehost.UpdateAttachedDeviceHostCmd = attacheddevicehost.UpdateADHCmd
func (c updateApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		asset.UpdateAssetCmd,
		dut.UpdateDUTCmd,
		labstation.UpdateLabstationCmd,
		cachingservice.UpdateCachingServiceCmd,
		schedulingunit.UpdateSchedulingUnitCmd,
		machine.UpdateMachineCmd,
		attacheddevicemachine.UpdateAttachedDeviceMachineCmd,
		attacheddevicemachine.UpdateADMCmd,
		host.UpdateHostCmd,
		attacheddevicehost.UpdateAttachedDeviceHostCmd,
		attacheddevicehost.UpdateADHCmd,
		kvm.UpdateKVMCmd,
		rpm.UpdateRPMCmd,
		switches.UpdateSwitchCmd,
		drac.UpdateDracCmd,
		nic.UpdateNicCmd,
		vm.UpdateVMCmd,
		rack.UpdateRackCmd,
		machineprototype.UpdateMachineLSEPrototypeCmd,
		rackprototype.UpdateRackLSEPrototypeCmd,
		chromeplatform.UpdateChromePlatformCmd,
		vlan.UpdateVlanCmd,
		lsedeployment.UpdateMachineLSEDeploymentCmd,
	}
}

// GetName is cli.Application interface implementation
func (c updateApp) GetName() string {
	return "update"
}
