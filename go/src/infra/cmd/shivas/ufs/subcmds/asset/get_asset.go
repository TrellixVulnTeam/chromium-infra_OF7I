// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package asset

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

// GetAssetCmd get asset by given name.
var GetAssetCmd = &subcommands.Command{
	UsageLine: "asset ...",
	ShortDesc: "Get asset details by filters",
	LongDesc: `Get asset details by filters.

Example:

shivas get asset {name1} {name2}

shivas get asset -zone atl97

Gets the asset and prints the output in the user-specified format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getAsset{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

		c.Flags.Var(flag.StringSlice(&c.zones), "zone", "Name(s) of a zone to filter by. Can be specified multiple times."+cmdhelp.ZoneFilterHelpText)
		c.Flags.Var(flag.StringSlice(&c.racks), "rack", "Name(s) of a rack to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.assettypes), "assettype", "Name(s) of a assettype to filter by. Can be specified multiple times."+cmdhelp.AssetTypesHelpText)
		c.Flags.Var(flag.StringSlice(&c.models), "model", "Name(s) of a model to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.buildTargets), "target", "Name(s) of a build target to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.phases), "phase", "Name(s) of a phase to filter by. Can be specified multiple times.")
		return c
	},
}

type getAsset struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags

	// Filters
	zones        []string
	racks        []string
	assettypes   []string
	models       []string
	buildTargets []string
	phases       []string

	pageSize int
	keysOnly bool
}

func (c *getAsset) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getAsset) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ns, err := c.envFlags.Namespace()
	if err != nil {
		return err
	}
	ctx = utils.SetupContext(ctx, ns)
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
		res, err = utils.BatchList(ctx, ic, listAssets, c.formatFilters(), c.pageSize, c.keysOnly, full)
	}
	if err != nil {
		return err
	}
	return utils.PrintEntities(ctx, ic, res, utils.PrintAssetsJSON, printAssetFull, printAssetNormal,
		c.outputFlags.JSON(), emit, full, c.outputFlags.Tsv(), c.keysOnly)
}

func (c *getAsset) getSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) (proto.Message, error) {
	return ic.GetAsset(ctx, &ufsAPI.GetAssetRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.AssetCollection, name),
	})
}

func (c *getAsset) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.ZoneFilterName, c.zones)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.RackFilterName, c.racks)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.AssetTypeFilterName, c.assettypes)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.ModelFilterName, c.models)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.BuildTargetFilterName, c.buildTargets)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.PhaseFilterName, c.phases)...)
	return filters
}

func printAssetFull(ctx context.Context, ic ufsAPI.FleetClient, msgs []proto.Message, tsv bool) error {
	return printAssetNormal(msgs, tsv, false)
}

func printAssetNormal(entities []proto.Message, tsv, keysOnly bool) error {
	if len(entities) == 0 {
		return nil
	}
	if tsv {
		utils.PrintTSVAssets(entities, keysOnly)
		return nil
	}
	utils.PrintTableTitle(utils.AssetTitle, tsv, keysOnly)
	utils.PrintAssets(entities, keysOnly)
	return nil
}

func listAssets(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly, full bool) ([]proto.Message, string, error) {
	req := &ufsAPI.ListAssetsRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
	}
	res, err := ic.ListAssets(ctx, req)
	if err != nil {
		return nil, "", err
	}
	protos := make([]proto.Message, len(res.GetAssets()))
	for i, m := range res.GetAssets() {
		protos[i] = m
	}
	return protos, res.GetNextPageToken(), nil
}
