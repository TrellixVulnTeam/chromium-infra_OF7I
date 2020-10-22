// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package vm

import (
	"context"

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

// ListVMSlotCmd returns vm slots by some filters.
var ListVMSlotCmd = &subcommands.Command{
	UsageLine: "vm-slots ...",
	ShortDesc: "Get free VM slots",
	LongDesc: `Get free VM slots by filters.

Examples:

shivas get vm-slots -n 5 -zone atl97 -man apple -os ESXi

Fetches 5 vm slots by manufacturer of chrome platform.
`,
	CommandRun: func() subcommands.CommandRun {
		c := &listVMSlot{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.number, "n", 0, "the number of free vm slots to fetch.")

		c.Flags.Var(flag.StringSlice(&c.zones), "zone", "Name(s) of a zone to filter by. Can be specified multiple times."+cmdhelp.ZoneFilterHelpText)
		c.Flags.Var(flag.StringSlice(&c.racks), "rack", "Name(s) of a rack to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.machines), "machine", "Name(s) of a machine to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.manufacturers), "man", "Name(s) of a manufacturer to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.oses), "os", "Name(s) of a tag to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.vdcs), "vdc", "Name(s) of a virtual datacenter to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.states), "state", "Name(s) of a state to filter by. Can be specified multiple times."+cmdhelp.StateFilterHelpText)
		return c
	},
}

type listVMSlot struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	// Filters
	zones         []string
	racks         []string
	machines      []string
	oses          []string
	manufacturers []string
	vdcs          []string
	states        []string

	number int
}

func (c *listVMSlot) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *listVMSlot) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
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
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	filters := c.formatFilters()
	for i := range filters {
		filters[i] += " & free=true"
	}
	entities, err := c.listFreeVMSlots(ctx, ic, filters)
	if err != nil {
		return err
	}
	return printVMSlotsFull(ctx, ic, entities)
}

func (c *listVMSlot) listFreeVMSlots(ctx context.Context, ic ufsAPI.FleetClient, filters []string) ([]*ufspb.MachineLSE, error) {
	var entities []*ufspb.MachineLSE
	var total int32
	for _, filter := range filters {
		protos, err := utils.DoList(ctx, ic, listFreeSlots, int32(c.number), filter, false)
		if err != nil {
			return nil, err
		}
		for _, p := range protos {
			host := p.(*ufspb.MachineLSE)
			total += host.GetChromeBrowserMachineLse().GetVmCapacity()
			entities = append(entities, host)
			if total >= int32(c.number) {
				return entities, nil
			}
		}
	}
	return entities, nil
}

func printVMSlotsFull(ctx context.Context, ic ufsAPI.FleetClient, entities []*ufspb.MachineLSE) error {
	names := make([]string, len(entities))
	for i := range entities {
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
	utils.PrintFreeVMs(entities, dhcpMap)
	return nil
}

func (c *listVMSlot) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("zone", c.zones)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("rack", c.racks)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("man", c.manufacturers)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("machine", c.machines)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("os", c.oses)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("vdc", c.vdcs)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("state", c.states)...)
	return filters
}

func listFreeSlots(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly bool) ([]proto.Message, string, error) {
	req := &ufsAPI.ListMachineLSEsRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
	}
	res, err := ic.ListMachineLSEs(ctx, req)
	if err != nil {
		return nil, "", err
	}
	protos := make([]proto.Message, len(res.GetMachineLSEs()))
	for i, m := range res.GetMachineLSEs() {
		protos[i] = m
	}
	return protos, res.GetNextPageToken(), nil
}

func (c *listVMSlot) validateArgs() error {
	if c.number == 0 {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\n'-n' is required")
	}
	if len(c.states) == 0 {
		c.states = []string{ufspb.State_STATE_SERVING.String()}
	}
	return nil
}
