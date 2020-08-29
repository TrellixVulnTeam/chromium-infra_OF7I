// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package nic

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

// GetNicCmd get nic by given name.
var GetNicCmd = &subcommands.Command{
	UsageLine: "nic ...",
	ShortDesc: "Get nic details by filters",
	LongDesc: `Get nic details by filters.

Example:

shivas get nic name1 name2

shivas get nic -mac mac1 -mac mac2

shivas get nic -n 10

Gets the nic and prints the output in the user-specified format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getNic{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

		c.Flags.Var(flag.StringSlice(&c.zones), "zone", "Name(s) of a zone to filter by. Can be specified multiple times."+cmdhelp.ZoneFilterHelpText)
		c.Flags.Var(flag.StringSlice(&c.racks), "rack", "Name(s) of a rack to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.switches), "switch", "Name(s) of a switch to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.switchPorts), "switchport", "Name(s) of a switch port to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.macs), "mac", "Name(s) of a mac to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.machines), "machine", "Name(s) of a machine to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of a tag to filter by. Can be specified multiple times.")
		return c
	},
}

type getNic struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags

	// Filters
	zones       []string
	racks       []string
	switches    []string
	switchPorts []string
	macs        []string
	machines    []string
	tags        []string

	pageSize int
	keysOnly bool
}

func (c *getNic) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getNic) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx)
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

	var res []proto.Message
	if len(args) > 0 {
		res, err = c.batchGet(ctx, ic, args)
	} else {
		res, err = utils.BatchList(ctx, ic, listNics, c.formatFilters(), c.pageSize, c.keysOnly)
	}
	if err != nil {
		return err
	}
	emit := !utils.NoEmitMode(c.outputFlags.NoEmit())
	full := utils.FullMode(c.outputFlags.Full())
	return utils.PrintEntities(ctx, ic, res, utils.PrintNicsJSON, printNicFull, printNicNormal,
		c.outputFlags.JSON(), emit, full, c.outputFlags.Tsv(), c.keysOnly)
}

func (c *getNic) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("zone", c.zones)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("rack", c.racks)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("switch", c.switches)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("switchport", c.switchPorts)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("mac", c.macs)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("machine", c.machines)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("tag", c.tags)...)
	return filters
}

func (c *getNic) batchGet(ctx context.Context, ic ufsAPI.FleetClient, names []string) ([]proto.Message, error) {
	res, err := ic.BatchGetNics(ctx, &ufsAPI.BatchGetNicsRequest{
		Names: names,
	})
	if err != nil {
		return nil, err
	}
	protos := make([]proto.Message, len(res.GetNics()))
	for i, r := range res.GetNics() {
		protos[i] = r
	}
	return protos, nil
}

func listNics(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly bool) ([]proto.Message, string, error) {
	req := &ufsAPI.ListNicsRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
	}
	res, err := ic.ListNics(ctx, req)
	if err != nil {
		return nil, "", err
	}
	protos := make([]proto.Message, len(res.GetNics()))
	for i, m := range res.GetNics() {
		protos[i] = m
	}
	return protos, res.GetNextPageToken(), nil
}

func printNicFull(ctx context.Context, ic ufsAPI.FleetClient, msgs []proto.Message, tsv bool) error {
	return printNicNormal(msgs, tsv, false)
}

func printNicNormal(entities []proto.Message, tsv, keysOnly bool) error {
	if len(entities) == 0 {
		return nil
	}
	if tsv {
		utils.PrintTSVNics(entities, false)
		return nil
	}
	utils.PrintTableTitle(utils.NicTitle, tsv, keysOnly)
	utils.PrintNics(entities, keysOnly)
	return nil
}
