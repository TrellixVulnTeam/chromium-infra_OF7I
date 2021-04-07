// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package operations

import (
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/cli"

	"infra/cmd/shivas/ufs/subcmds/asset"
	"infra/cmd/shivas/ufs/subcmds/cachingservice"
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
	"infra/cmd/shivas/ufs/subcmds/schedulingunit"
	"infra/cmd/shivas/ufs/subcmds/switches"
	"infra/cmd/shivas/ufs/subcmds/vlan"
	"infra/cmd/shivas/ufs/subcmds/vm"
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
	machine-prototype/rack-prototype/chromeplatform/vlan`,
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
func (c deleteApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		asset.DeleteAssetCmd,
		dut.DeleteDUTCmd,
		schedulingunit.DeleteSchedulingUnitCmd,
		machine.DeleteMachineCmd,
		host.DeleteHostCmd,
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
	}
}

// GetName is cli.Application interface implementation
func (c deleteApp) GetName() string {
	return "delete"
}
