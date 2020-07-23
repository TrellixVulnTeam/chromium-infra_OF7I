// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package chromeplatform

import (
	"context"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

// ListChromePlatformCmd list all Chrome Platforms.
var ListChromePlatformCmd = &subcommands.Command{
	UsageLine: "chrome-platform",
	ShortDesc: "List all chrome platforms",
	LongDesc:  cmdhelp.ListChromePlatformLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &listChromePlatform{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.json, "json", false, `print output in JSON format`)
		return c
	},
}

type listChromePlatform struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
	pageSize  int
	json      bool
}

func (c *listChromePlatform) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *listChromePlatform) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	if c.json {
		return utils.PrintListJSONFormat(ctx, ic, printChromePlatforms, c.json, int32(c.pageSize), "")
	}
	return utils.PrintListTableFormat(ctx, ic, printChromePlatforms, c.json, int32(c.pageSize), "", utils.ChromePlatformTitle)
}

func printChromePlatforms(ctx context.Context, ic ufsAPI.FleetClient, json bool, pageSize int32, pageToken, filter string) (string, error) {
	req := &ufsAPI.ListChromePlatformsRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
	}
	res, err := ic.ListChromePlatforms(ctx, req)
	if err != nil {
		return "", err
	}
	if json {
		utils.PrintChromePlatformsJSON(res.ChromePlatforms)
	} else {
		utils.PrintChromePlatforms(res.ChromePlatforms)
	}
	return res.GetNextPageToken(), nil
}
