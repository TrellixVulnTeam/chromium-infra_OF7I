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
	"infra/cmd/shivas/ufs/subcmds/labstation"
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
	asset/dut/labstation/cachingservice
	machine-prototype/rack-prototype/chromeplatform/vlan`,
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
func (c addApp) GetCommands() []*subcommands.Command {
	return []*subcommands.Command{
		subcommands.CmdHelp,
		asset.AddAssetCmd,
		dut.AddDUTCmd,
		labstation.AddLabstationCmd,
		cachingservice.AddCachingServiceCmd,
		machine.AddMachineCmd,
		host.AddHostCmd,
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
	}
}

// GetName is cli.Application interface implementation
func (c addApp) GetName() string {
	return "add"
}
