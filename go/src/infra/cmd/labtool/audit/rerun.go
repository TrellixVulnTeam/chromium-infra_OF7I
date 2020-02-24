// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package audit

import (
	"fmt"
	"sort"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	fleetAPI "infra/appengine/cros/lab_inventory/api/v1"
	"infra/cmd/labtool/site"
	"infra/cmd/labtool/utils"
	"infra/cmdsupport/cmdlib"
)

// RerunCmd rerun a previous audit based on its log.
var RerunCmd = &subcommands.Command{
	UsageLine: "rerun",
	ShortDesc: "Run previous audit from logs",
	LongDesc: `Runs the operations in the log of previous audits.
	
Please note that each rerun will generate its own logs.`,
	CommandRun: func() subcommands.CommandRun {
		c := &rerun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.logDir, "log-dir", getLogDir(), "Log directory")
		c.Flags.IntVar(&c.idx, "index", 0, "Index of the log to run, run logls for index checking")
		// To be implemented
		c.Flags.StringVar(&c.tStamp, "timestamp", "", "Timestamp of log to run")
		return c
	},
}

type rerun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
	logDir    string
	idx       int
	tStamp    string
}

func (c *rerun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *rerun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	fmt.Printf("listing logs in %s\n", c.logDir)
	logs, err := utils.ListLogs(c.logDir)
	if err != nil {
		return err
	}
	if c.idx >= len(logs) || c.idx < 0 {
		return cmdlib.NewUsageError(c.Flags, fmt.Sprintf("index is beyond the scope of [%d, %d)", 0, len(logs)))
	}

	sort.Sort(logs)
	runStats := logs[c.idx]
	ast, err := utils.GetAssetsInOrder(runStats.LogPath)
	if err != nil {
		return err
	}
	if len(ast) == 0 {
		fmt.Println("No asset to rerun, return")
		return nil
	}

	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	fmt.Printf("Using inventory service %s\n", e)
	ic := fleetAPI.NewInventoryPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.InventoryService,
		Options: site.DefaultPRPCOptions,
	})

	gsc, err := getGSClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}

	u, err := utils.NewUpdater(ctx, ic, gsc, c.logDir)
	if err != nil {
		return err
	}
	u.AddAsset(ast)
	u.Close()
	return nil
}
