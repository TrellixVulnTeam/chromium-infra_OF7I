// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package kvm

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
	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetKVMCmd get kvm by given name.
var GetKVMCmd = &subcommands.Command{
	UsageLine: "kvm ...",
	ShortDesc: "Get kvm details by filters",
	LongDesc: `Get kvm details by filters.

Example:

shivas get kvm {name1} {name2}

shivas get kvm -platform platform1 -zone mtv97 -rack rack1 -rack rack2 -rack rack3

Gets the kvm and prints the output in the user-specified format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getKVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

		c.Flags.Var(flag.StringSlice(&c.zones), "zone", "Name(s) of a zone to filter by. Can be specified multiple times."+cmdhelp.ZoneFilterHelpText)
		c.Flags.Var(flag.StringSlice(&c.racks), "rack", "Name(s) of a rack to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.platforms), "platform", "Name(s) of a platform to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of a tag to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.macs), "mac", "Name(s) of a mac to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.states), "state", "Name(s) of a state to filter by. Can be specified multiple times."+cmdhelp.StateFilterHelpText)
		return c
	},
}

type getKVM struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags

	// Filters
	zones     []string
	racks     []string
	platforms []string
	tags      []string
	macs      []string
	states    []string

	pageSize int
	keysOnly bool
}

func (c *getKVM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getKVM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	var res []proto.Message
	if len(args) > 0 {
		res, err = c.batchGet(ctx, ic, args)
	} else {
		res, err = utils.BatchList(ctx, ic, listKVMs, c.formatFilters(), c.pageSize, c.keysOnly)
	}
	if err != nil {
		return err
	}
	emit := !utils.NoEmitMode(c.outputFlags.NoEmit())
	full := utils.FullMode(c.outputFlags.Full())
	return utils.PrintEntities(ctx, ic, res, utils.PrintKVMsJSON, printKVMFull, printKVMNormal,
		c.outputFlags.JSON(), emit, full, c.outputFlags.Tsv(), c.keysOnly)
}

func (c *getKVM) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("zone", c.zones)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("rack", c.racks)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("platform", c.platforms)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("tag", c.tags)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("mac", c.macs)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("state", c.states)...)
	return filters
}

func (c *getKVM) batchGet(ctx context.Context, ic ufsAPI.FleetClient, names []string) ([]proto.Message, error) {
	res, err := ic.BatchGetKVMs(ctx, &ufsAPI.BatchGetKVMsRequest{
		Names: names,
	})
	if err != nil {
		return nil, err
	}
	protos := make([]proto.Message, len(res.GetKVMs()))
	for i, r := range res.GetKVMs() {
		protos[i] = r
	}
	return protos, nil
}

func printKVMFull(ctx context.Context, ic ufsAPI.FleetClient, msgs []proto.Message, tsv bool) error {
	kvms := make([]*ufspb.KVM, len(msgs))
	names := make([]string, len(msgs))
	for i, r := range msgs {
		kvms[i] = r.(*ufspb.KVM)
		kvms[i].Name = ufsUtil.RemovePrefix(kvms[i].Name)
		names[i] = kvms[i].GetName()
	}
	res, _ := ic.BatchGetDHCPConfigs(ctx, &ufsAPI.BatchGetDHCPConfigsRequest{
		Names: names,
	})
	dhcpMap := make(map[string]*ufspb.DHCPConfig, 0)
	for _, d := range res.GetDhcpConfigs() {
		dhcpMap[d.GetHostname()] = d
	}
	if tsv {
		for _, kvm := range kvms {
			utils.PrintTSVKVMFull(kvm, dhcpMap[kvm.GetName()])
		}
		return nil
	}
	utils.PrintTitle(utils.KvmFullTitle)
	utils.PrintKVMFull(kvms, dhcpMap)
	return nil
}

func printKVMNormal(kvms []proto.Message, tsv, keysOnly bool) error {
	if tsv {
		utils.PrintTSVKVMs(kvms, false)
		return nil
	}
	utils.PrintTableTitle(utils.KvmTitle, tsv, keysOnly)
	utils.PrintKVMs(kvms, keysOnly)
	return nil
}

func listKVMs(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly bool) ([]proto.Message, string, error) {
	req := &ufsAPI.ListKVMsRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
	}
	res, err := ic.ListKVMs(ctx, req)
	if err != nil {
		return nil, "", err
	}
	protos := make([]proto.Message, len(res.GetKVMs()))
	for i, kvm := range res.GetKVMs() {
		protos[i] = kvm
	}
	return protos, res.GetNextPageToken(), nil
}
