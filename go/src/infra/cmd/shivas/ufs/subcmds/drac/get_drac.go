// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package drac

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
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetDracCmd get drac by given name.
var GetDracCmd = &subcommands.Command{
	UsageLine: "drac ...",
	ShortDesc: "Get drac details by filters",
	LongDesc: `Get drac details by filters.

Example:

shivas get drac {name1} {name2}

shivas get drac -zone atl97 -rack rack1 -rack rack2

Gets the drac and prints the output in the user-specified format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getDrac{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

		c.Flags.Var(flag.StringSlice(&c.zones), "zone", "Name(s) of a zone to filter by. Can be specified multiple times."+cmdhelp.ZoneFilterHelpText)
		c.Flags.Var(flag.StringSlice(&c.racks), "rack", "Name(s) of a rack to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.machines), "machine", "Name(s) of a machine to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.switches), "switch", "Name(s) of a switch to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.switchPorts), "switchport", "Name(s) of a switch port to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.macs), "mac", "Name(s) of a mac to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of a tag to filter by. Can be specified multiple times.")
		return c
	},
}

type getDrac struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags

	// Filters
	zones       []string
	racks       []string
	machines    []string
	switches    []string
	switchPorts []string
	macs        []string
	tags        []string

	pageSize int
	keysOnly bool
}

func (c *getDrac) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getDrac) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
		res, err = utils.BatchList(ctx, ic, listDracs, c.formatFilters(), c.pageSize, c.keysOnly, full)
	}
	if err != nil {
		return err
	}
	return utils.PrintEntities(ctx, ic, res, utils.PrintDracsJSON, printDracFull, printDracNormal,
		c.outputFlags.JSON(), emit, full, c.outputFlags.Tsv(), c.keysOnly)
}

func (c *getDrac) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("zone", c.zones)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("rack", c.racks)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("machine", c.machines)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("switch", c.switches)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("switchport", c.switchPorts)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("mac", c.macs)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("tag", c.tags)...)
	return filters
}

func (c *getDrac) getSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) (proto.Message, error) {
	return ic.GetDrac(ctx, &ufsAPI.GetDracRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.DracCollection, name),
	})
}

func listDracs(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly, full bool) ([]proto.Message, string, error) {
	req := &ufsAPI.ListDracsRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
	}
	res, err := ic.ListDracs(ctx, req)
	if err != nil {
		return nil, "", err
	}
	protos := make([]proto.Message, len(res.GetDracs()))
	for i, m := range res.GetDracs() {
		protos[i] = m
	}
	return protos, res.GetNextPageToken(), nil
}

func printDracFull(ctx context.Context, ic ufsAPI.FleetClient, msgs []proto.Message, tsv bool) error {
	entities := make([]*ufspb.Drac, len(msgs))
	names := make([]string, len(msgs))
	for i, r := range msgs {
		entities[i] = r.(*ufspb.Drac)
		entities[i].Name = ufsUtil.RemovePrefix(entities[i].Name)
		names[i] = entities[i].GetName()
	}
	res, _ := ic.BatchGetDHCPConfigs(ctx, &ufsAPI.BatchGetDHCPConfigsRequest{
		Names: names,
	})
	dhcpMap := make(map[string]*ufspb.DHCPConfig, 0)
	for _, d := range res.GetDhcpConfigs() {
		dhcpMap[d.GetHostname()] = d
	}
	if tsv {
		for _, e := range entities {
			utils.PrintTSVDracFull(e, dhcpMap[e.GetName()])
		}
		return nil
	}
	utils.PrintTitle(utils.DracFullTitle)
	utils.PrintDracFull(entities, dhcpMap)
	return nil
}

func printDracNormal(msgs []proto.Message, tsv, keysOnly bool) error {
	if tsv {
		utils.PrintTSVDracs(msgs, keysOnly)
		return nil
	}
	utils.PrintTableTitle(utils.DracTitle, tsv, keysOnly)
	utils.PrintDracs(msgs, keysOnly)
	return nil
}
