// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package schedulingunit

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
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetSchedulingUnitCmd get chrome platform by given name.
var GetSchedulingUnitCmd = &subcommands.Command{
	UsageLine: "schedulingunit ...",
	ShortDesc: "Get SchedulingUnit details by filters",
	LongDesc: `Get SchedulingUnit details by filters.

Example:

shivas get schedulingunit {name1} {name2}

shivas get schedulingunit -dut dut1

shivas get schedulingunit

Gets the SchedulingUnit and prints the output in the user-specified format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getSchedulingUnit{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

		c.Flags.Var(flag.StringSlice(&c.types), "type", "Name(s) of a type to filter by. Can be specified multiple times."+cmdhelp.SchedulingUnitTypesHelpText)
		c.Flags.Var(flag.StringSlice(&c.duts), "duts", "Name(s) of a dut to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.pools), "pools", "Name(s) of a pool to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of a tag to filter by. Can be specified multiple times.")
		return c
	},
}

type getSchedulingUnit struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags

	// Filters
	types []string
	duts  []string
	pools []string
	tags  []string

	pageSize int
	keysOnly bool
}

func (c *getSchedulingUnit) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getSchedulingUnit) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx, ufsUtil.OSNamespace)
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
	emit := !utils.NoEmitMode(c.outputFlags.NoEmit())
	full := utils.FullMode(c.outputFlags.Full())
	var res []proto.Message
	if len(args) > 0 {
		res = utils.ConcurrentGet(ctx, ic, args, c.getSingle)
	} else {
		res, err = utils.BatchList(ctx, ic, listSchedulingUnits, c.formatFilters(), c.pageSize, c.keysOnly, full)
	}
	if err != nil {
		return err
	}
	return utils.PrintEntities(ctx, ic, res, utils.PrintSchedulingUnitsJSON, printSchedulingUnitFull, printSchedulingUnitNormal,
		c.outputFlags.JSON(), emit, full, c.outputFlags.Tsv(), c.keysOnly)
}

func (c *getSchedulingUnit) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.PoolsFilterName, c.pools)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.MachineLSEsFilterName, c.duts)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.TagFilterName, c.tags)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.TypeFilterName, c.types)...)
	return filters
}

func (c *getSchedulingUnit) getSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) (proto.Message, error) {
	return ic.GetSchedulingUnit(ctx, &ufsAPI.GetSchedulingUnitRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.SchedulingUnitCollection, name),
	})
}

func listSchedulingUnits(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly, full bool) ([]proto.Message, string, error) {
	req := &ufsAPI.ListSchedulingUnitsRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
	}
	res, err := ic.ListSchedulingUnits(ctx, req)
	if err != nil {
		return nil, "", err
	}
	protos := make([]proto.Message, len(res.GetSchedulingUnits()))
	for i, m := range res.GetSchedulingUnits() {
		protos[i] = m
	}
	return protos, res.GetNextPageToken(), nil
}

func printSchedulingUnitFull(ctx context.Context, ic ufsAPI.FleetClient, msgs []proto.Message, tsv bool) error {
	return printSchedulingUnitNormal(msgs, tsv, false)
}

func printSchedulingUnitNormal(entities []proto.Message, tsv, keysOnly bool) error {
	if len(entities) == 0 {
		return nil
	}
	if tsv {
		utils.PrintTSVSchedulingUnits(entities, keysOnly)
		return nil
	}
	utils.PrintTableTitle(utils.SchedulingUnitTitle, tsv, keysOnly)
	utils.PrintSchedulingUnits(entities, keysOnly)
	return nil
}
