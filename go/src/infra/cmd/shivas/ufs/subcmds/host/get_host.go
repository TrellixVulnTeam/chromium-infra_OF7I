// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package host

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
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

// GetHostCmd get host by given name.
var GetHostCmd = &subcommands.Command{
	UsageLine: "host {Host name}",
	ShortDesc: "Get host details by name",
	LongDesc: `Get host details by name.

Example:

shivas get host {name1} {name2}

shivas get host -rack rack1 -rack2 -state serving -state needs_repair

Gets the host and prints the output in JSON format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getHost{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

		c.Flags.Var(flag.StringSlice(&c.zones), "zone", "Name(s) of a zone to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.racks), "rack", "Name(s) of a rack to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.machines), "machine", "Name(s) of a machine to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.prototypes), "prototype", "Name(s) of a host prototype to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.vlans), "vlan", "Name(s) of a vlan to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.manufacturers), "man", "Name(s) of a manufacturer to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.oses), "os", "Name(s) of an os to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.nics), "nic", "Name(s) of a nic to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.vdcs), "vdc", "Name(s) of a vdc to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.switches), "switch", "Name(s) of a switch to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of a tag to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.states), "state", "Name(s) of a state to filter by. Can be specified multiple times."+cmdhelp.StateFilterHelpText)
		return c
	},
}

type getHost struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	outputFlags site.OutputFlags

	// Filters
	zones         []string
	racks         []string
	machines      []string
	prototypes    []string
	vlans         []string
	manufacturers []string
	oses          []string
	nics          []string
	vdcs          []string
	switches      []string
	tags          []string
	states        []string

	pageSize int
	keysOnly bool
}

func (c *getHost) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getHost) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	var res []proto.Message
	if len(args) > 0 {
		res, err = c.batchGet(ctx, ic, args)
	} else {
		res, err = c.batchList(ctx, ic, c.formatFilters())
	}
	if err != nil {
		return err
	}
	emit := !utils.NoEmitMode(c.outputFlags.NoEmit())
	full := utils.FullMode(c.outputFlags.Full())
	return utils.PrintEntities(ctx, ic, res, utils.PrintMachineLSEsJSON, printHostFull, printHostNormal,
		c.outputFlags.JSON(), emit, full, c.outputFlags.Tsv(), c.keysOnly)
}

func (c *getHost) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("zone", c.zones)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("rack", c.racks)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("machine", c.machines)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("machineprototype", c.prototypes)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("man", c.manufacturers)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("os", c.oses)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("nic", c.nics)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("vdc", c.vdcs)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("tag", c.tags)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("state", c.states)...)
	return filters
}

func (c *getHost) batchList(ctx context.Context, ic ufsAPI.FleetClient, filters []string) ([]proto.Message, error) {
	errs := make(map[string]error)
	res := make([]proto.Message, 0)
	for _, filter := range filters {
		protos, err := utils.DoList(ctx, ic, listHosts, int32(c.pageSize), filter, c.keysOnly)
		if err != nil {
			errs[filter] = err
		}
		res = append(res, protos...)
		if c.pageSize > 0 && len(res) >= c.pageSize {
			res = res[0:c.pageSize]
			break
		}
	}
	if len(errs) > 0 {
		fmt.Println("Fail to do some queries:")
		resErr := make([]error, 0, len(errs))
		for f, err := range errs {
			fmt.Printf("Filter %s: %s\n", f, err.Error())
			resErr = append(resErr, err)
		}
		return nil, errors.MultiError(resErr)
	}
	return res, nil
}

func (c *getHost) batchGet(ctx context.Context, ic ufsAPI.FleetClient, names []string) ([]proto.Message, error) {
	res, err := ic.BatchGetMachineLSEs(ctx, &ufsAPI.BatchGetMachineLSEsRequest{
		Names: names,
	})
	if err != nil {
		return nil, err
	}
	protos := make([]proto.Message, len(res.GetMachineLses()))
	for i, r := range res.GetMachineLses() {
		protos[i] = r
	}
	return protos, nil
}

func listHosts(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly bool) ([]proto.Message, string, error) {
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
	for i, kvm := range res.GetMachineLSEs() {
		protos[i] = kvm
	}
	return protos, res.GetNextPageToken(), nil
}

func printHostFull(ctx context.Context, ic ufsAPI.FleetClient, msgs []proto.Message, tsv bool) error {
	entities := make([]*ufspb.MachineLSE, len(msgs))
	names := make([]string, len(msgs))
	for i, r := range msgs {
		entities[i] = r.(*ufspb.MachineLSE)
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
			utils.PrintTSVHostFull(e, dhcpMap[e.GetName()])
		}
		return nil
	}
	utils.PrintTitle(utils.MachineLSETFullitle)
	utils.PrintMachineLSEFull(entities, dhcpMap)
	return nil
}

func printHostNormal(msgs []proto.Message, tsv, keysOnly bool) error {
	if tsv {
		utils.PrintTSVMachineLSEs(msgs, false)
		return nil
	}
	utils.PrintTableTitle(utils.MachineLSETitle, tsv, keysOnly)
	utils.PrintMachineLSEs(msgs, keysOnly)
	return nil
}
