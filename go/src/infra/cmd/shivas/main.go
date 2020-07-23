// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/data/rand/mathrand"

	"infra/cmd/shivas/audit"
	"infra/cmd/shivas/meta"
	"infra/cmd/shivas/query"
	"infra/cmd/shivas/site"
	sw_cmds "infra/cmd/shivas/swarming/cmds"
	"infra/cmd/shivas/ufs/cmds/configuration"
	"infra/cmd/shivas/ufs/cmds/labsetup/host"
	"infra/cmd/shivas/ufs/cmds/labsetup/vm"
	q "infra/cmd/shivas/ufs/cmds/query"
	"infra/cmd/shivas/ufs/cmds/registration/drac"
	"infra/cmd/shivas/ufs/cmds/registration/kvm"
	"infra/cmd/shivas/ufs/cmds/registration/machine"
	"infra/cmd/shivas/ufs/cmds/registration/nic"
	"infra/cmd/shivas/ufs/cmds/registration/rack"
	"infra/cmd/shivas/ufs/cmds/registration/switches"
)

func getApplication() *cli.Application {
	return &cli.Application{
		Name:  "shivas",
		Title: "Unified Fleet System Management",
		Context: func(ctx context.Context) context.Context {
			return ctx
		},
		Commands: []*subcommands.Command{
			subcommands.CmdHelp,
			subcommands.Section("Meta"),
			meta.Version,
			meta.Update,
			subcommands.Section("Authentication"),
			authcli.SubcommandInfo(site.DefaultAuthOptions, "whoami", false),
			authcli.SubcommandLogin(site.DefaultAuthOptions, "login", false),
			authcli.SubcommandLogout(site.DefaultAuthOptions, "logout", false),
			subcommands.Section("ChromeOS Lab (only) audit"),
			audit.AuditCmd,
			query.GetAssetsCmd,
			subcommands.Section("Registration Management"),
			machine.AddMachineCmd,
			machine.UpdateMachineCmd,
			machine.DeleteMachineCmd,
			nic.AddNicCmd,
			nic.UpdateNicCmd,
			nic.DeleteNicCmd,
			drac.AddDracCmd,
			drac.UpdateDracCmd,
			drac.DeleteDracCmd,
			rack.AddRackCmd,
			rack.UpdateRackCmd,
			rack.DeleteRackCmd,
			switches.AddSwitchCmd,
			switches.UpdateSwitchCmd,
			switches.DeleteSwitchCmd,
			kvm.AddKVMCmd,
			kvm.UpdateKVMCmd,
			kvm.DeleteKVMCmd,
			subcommands.Section("Lab Setup Management"),
			host.AddHostCmd,
			host.UpdateHostCmd,
			host.DeleteHostCmd,
			vm.AddVMCmd,
			vm.UpdateVMCmd,
			vm.DeleteVMCmd,
			subcommands.Section("Configuration Management"),
			configuration.AddMachineLSEPrototypeCmd,
			configuration.UpdateMachineLSEPrototypeCmd,
			configuration.DeleteMachineLSEPrototypeCmd,
			configuration.AddRackLSEPrototypeCmd,
			configuration.UpdateRackLSEPrototypeCmd,
			configuration.DeleteRackLSEPrototypeCmd,
			configuration.AddChromePlatformCmd,
			configuration.UpdateChromePlatformCmd,
			configuration.DeleteChromePlatformCmd,
			subcommands.Section("Query Unified Fleet"),
			q.GetCmd,
			q.ListCmd,
			subcommands.Section("State"),
			sw_cmds.ReserveDutsCmd,
		},
	}
}

func main() {
	mathrand.SeedRandomly()
	log.SetOutput(ioutil.Discard)
	os.Exit(subcommands.Run(getApplication(), nil))
}
