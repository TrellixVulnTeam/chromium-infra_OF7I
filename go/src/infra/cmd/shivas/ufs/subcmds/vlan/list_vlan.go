// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package vlan

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

// ListVlanCmd list all vlans.
var ListVlanCmd = &subcommands.Command{
	UsageLine: "vlan",
	ShortDesc: "List all vlans",
	LongDesc:  cmdhelp.ListVlanLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &listVlan{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.StringVar(&c.filter, "filter", "", cmdhelp.VlanFilterHelp)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)
		return c
	},
}

type listVlan struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags
	pageSize    int
	filter      string
	keysOnly    bool
}

func (c *listVlan) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *listVlan) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	if c.outputFlags.JSON() {
		return utils.PrintListJSONFormat(ctx, ic, printVlans, true, int32(c.pageSize), c.filter, c.keysOnly)
	}
	return utils.PrintListTableFormat(ctx, ic, printVlans, false, int32(c.pageSize), c.filter, c.keysOnly, utils.VlanTitle, c.outputFlags.Tsv())
}

func printVlans(ctx context.Context, ic ufsAPI.FleetClient, json bool, pageSize int32, pageToken, filter string, keysOnly, tsv bool) (string, error) {
	req := &ufsAPI.ListVlansRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
	}
	res, err := ic.ListVlans(ctx, req)
	if err != nil {
		return "", err
	}
	if json {
		utils.PrintVlansJSON(res.Vlans)
	} else if tsv {
		utils.PrintTSVVlans(res.Vlans, keysOnly)
	} else {
		utils.PrintVlans(res.Vlans, keysOnly)
	}
	return res.GetNextPageToken(), nil
}

func (c *listVlan) validateArgs() error {
	if c.filter != "" {
		filter := fmt.Sprintf(strings.Replace(c.filter, " ", "", -1))
		if !ufsAPI.FilterRegex.MatchString(filter) {
			return cmdlib.NewUsageError(c.Flags, ufsAPI.InvalidFilterFormat)
		}
	}
	return nil
}
