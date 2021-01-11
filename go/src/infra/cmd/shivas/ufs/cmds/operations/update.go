// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package operations

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cmd/shivas/ufs/subcmds/asset"
	"infra/cmd/shivas/ufs/subcmds/chromeplatform"
	"infra/cmd/shivas/ufs/subcmds/drac"
	"infra/cmd/shivas/ufs/subcmds/dut"
	"infra/cmd/shivas/ufs/subcmds/host"
	"infra/cmd/shivas/ufs/subcmds/kvm"
	"infra/cmd/shivas/ufs/subcmds/machine"
	"infra/cmd/shivas/ufs/subcmds/machineprototype"
	"infra/cmd/shivas/ufs/subcmds/nic"
	"infra/cmd/shivas/ufs/subcmds/rack"
	"infra/cmd/shivas/ufs/subcmds/rackprototype"
	"infra/cmd/shivas/ufs/subcmds/rpm"
	"infra/cmd/shivas/ufs/subcmds/switches"
	"infra/cmd/shivas/ufs/subcmds/vlan"
	"infra/cmd/shivas/ufs/subcmds/vm"
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
	machine-prototype/rack-prototype/chromeplatform/vlan`,
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
func (c updateApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		asset.UpdateAssetCmd,
		dut.UpdateDUTCmd,
		machine.UpdateMachineCmd,
		host.UpdateHostCmd,
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
	}
}

// GetName is cli.Application interface implementation
func (c updateApp) GetName() string {
	return "update"
}
