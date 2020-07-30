// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rack

import (
	"context"
	"fmt"
	"strings"

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

// ListRackCmd list all Racks.
var ListRackCmd = &subcommands.Command{
	UsageLine: "rack",
	ShortDesc: "List all racks",
	LongDesc:  cmdhelp.ListRackLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &listRack{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.json, "json", false, `print output in JSON format`)
		c.Flags.StringVar(&c.filter, "filter", "", cmdhelp.RackFilterHelp)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)
		c.commonFlags.Register(&c.Flags)
		return c
	},
}

type listRack struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	pageSize    int
	json        bool
	filter      string
	keysOnly    bool
}

func (c *listRack) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *listRack) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UnifiedFleet service %s\n", e.UnifiedFleetService)
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	if c.json {
		return utils.PrintListJSONFormatDup(ctx, ic, printRacks, c.json, int32(c.pageSize), c.filter, c.keysOnly)
	}
	return utils.PrintListTableFormatDup(ctx, ic, printRacks, c.json, int32(c.pageSize), c.filter, c.keysOnly, utils.RackTitle)
}

func printRacks(ctx context.Context, ic ufsAPI.FleetClient, json bool, pageSize int32, pageToken, filter string, keysOnly bool) (string, error) {
	req := &ufsAPI.ListRacksRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
	}
	res, err := ic.ListRacks(ctx, req)
	if err != nil {
		return "", err
	}
	if json {
		utils.PrintRacksJSON(res.Racks)
	} else {
		utils.PrintRacks(res.Racks, keysOnly)
	}
	return res.GetNextPageToken(), nil
}

func (c *listRack) validateArgs() error {
	if c.filter != "" {
		filter := fmt.Sprintf(strings.Replace(c.filter, " ", "", -1))
		if !ufsAPI.FilterRegex.MatchString(filter) {
			return cmdlib.NewUsageError(c.Flags, ufsAPI.InvalidFilterFormat)
		}
		var err error
		c.filter, err = utils.ReplaceLabNames(filter)
		if err != nil {
			return err
		}
	}
	return nil
}
