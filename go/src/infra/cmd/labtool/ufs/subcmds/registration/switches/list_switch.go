// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package switches

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"
	"infra/cmd/labtool/site"
	"infra/cmd/labtool/utils"
	"infra/cmdsupport/cmdlib"
	UfleetAPI "infra/unifiedfleet/api/v1/rpc"
)

// ListSwitchCmd list all switches.
var ListSwitchCmd = &subcommands.Command{
	UsageLine: "ls",
	ShortDesc: "list all Switches",
	LongDesc: `list all Switches

	./labtool switch ls
	Fetches 100 items and prints the output in table format

	./labtool switch ls -n 50
	Fetches 50 items and prints the output in table format

	./labtool switch ls -json
	Fetches 100 items and prints the output in JSON format

	./labtool switch ls -n 50 -json
	Fetches 50 items and prints the output in JSON format
	`,
	CommandRun: func() subcommands.CommandRun {
		c := &listSwitch{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.pageSize, "n", 100,
			`number of items to get. The service may return fewer than this value.If unspecified, at most 100 items will be returned.
The maximum value is 1000; values above 1000 will be coerced to 1000.`)
		c.Flags.BoolVar(&c.json, "json", false, `print output in JSON format`)
		return c
	},
}

type listSwitch struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
	pageSize  int
	json      bool
}

func (c *listSwitch) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *listSwitch) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	ic := UfleetAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	var pageToken string
	for {
		req := &UfleetAPI.ListSwitchesRequest{
			PageSize:  int32(c.pageSize),
			PageToken: pageToken,
		}
		res, err := ic.ListSwitches(ctx, req)
		if err != nil {
			return err
		}
		if c.json {
			utils.PrintSwitchesJSON(res.Switches)
		} else {
			utils.PrintSwitches(res.Switches)
		}
		pageToken = res.GetNextPageToken()
		nextPage, err := utils.GetNextPage(pageToken)
		if err != nil {
			return err
		}
		if !nextPage {
			return nil
		}
	}
}

func (c *listSwitch) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return nil
	}
	return nil
}
