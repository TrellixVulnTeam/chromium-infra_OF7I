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

	"infra/cmd/shivas/meta"
	queen_cmds "infra/cmd/shivas/queen/cmds"
	"infra/cmd/shivas/runlocal/cmds/run"
	"infra/cmd/shivas/site"
	sw_cmds "infra/cmd/shivas/swarming/cmds"
	bot_cmds "infra/cmd/shivas/ufs/cmds/bot"
	"infra/cmd/shivas/ufs/cmds/operations"
	"infra/cmd/shivas/ufs/cmds/state"
)

func getApplication() *cli.Application {
	return &cli.Application{
		Name: "shivas",
		Title: `Unified Fleet System Management

Tool uses a default RPC retry strategy with five attempts and exponential backoff.
Full documentation http://go/shivas-cli`,
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
			subcommands.Section("Resource Management"),
			operations.AddCmd,
			operations.UpdateCmd,
			operations.DeleteCmd,
			operations.GetCmd,
			operations.RenameCmd,
			subcommands.Section("Repair"),
			sw_cmds.RepairDutsCmd,
			sw_cmds.AuditDutsCmd,
			subcommands.Section("State"),
			sw_cmds.ReserveDutsCmd,
			state.DutStateCmd,
			subcommands.Section("Drone Queen Inspection"),
			queen_cmds.InspectDuts,
			queen_cmds.InspectDrones,
			queen_cmds.PushDuts,
			subcommands.Section("Internal use"),
			bot_cmds.PrintBotInfo,
			run.RunCmd,
		},
	}
}

func main() {
	mathrand.SeedRandomly()
	log.SetOutput(ioutil.Discard)
	os.Exit(subcommands.Run(getApplication(), nil))
}
