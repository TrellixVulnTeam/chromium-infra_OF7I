// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/flag"
	"go.chromium.org/luci/grpc/prpc"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/libs/skylab/autotest/hostinfo"
	inventoryclient "infra/libs/skylab/inventory/inventoryclient"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/ufs/subcmds/host"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetDutCmd get host by given name.
var GetDutCmd = &subcommands.Command{
	UsageLine: "dut ...",
	ShortDesc: "Get DUT/labstation details by filters",
	LongDesc: `Get DUT/labstation details by filters.

Example:

shivas get dut {name1} {name2}

shivas get dut -rack rack1 -rack2 -state serving -state needs_repair

Gets the ChromeOS DUT and prints the output in user-specified format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getDut{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

		c.Flags.Var(flag.StringSlice(&c.zones), "zone", "Name(s) of a zone to filter by. Can be specified multiple times."+cmdhelp.ZoneFilterHelpText)
		c.Flags.Var(flag.StringSlice(&c.racks), "rack", "Name(s) of a rack to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.machines), "machine", "Name(s) of a machine/asset to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.prototypes), "prototype", "Name(s) of a host prototype to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of a tag to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.states), "state", "Name(s) of a state to filter by. Can be specified multiple times."+cmdhelp.StateFilterHelpText)
		c.Flags.Var(flag.StringSlice(&c.servos), "servo", "Name(s) of a servo:port to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.servotypes), "servotype", "Name(s) of a servo type to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.switches), "switch", "Name(s) of a switch to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.rpms), "rpm", "Name(s) of a rpm to filter by. Can be specified multiple times.")
		c.Flags.BoolVar(&c.wantHostInfoStore, "host-info-store", false, "write host info store to stdout")

		return c
	},
}

type getDut struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	outputFlags site.OutputFlags
	commonFlags site.CommonFlags

	// Filters
	zones      []string
	racks      []string
	machines   []string
	prototypes []string
	tags       []string
	states     []string
	servos     []string
	servotypes []string
	switches   []string
	rpms       []string

	pageSize          int
	keysOnly          bool
	wantHostInfoStore bool
}

func (c *getDut) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getDut) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	ctx = utils.SetupContext(ctx, ufsUtil.OSNamespace)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	if c.commonFlags.Verbose() {
		fmt.Printf("Using UFS service %s\n", e.UnifiedFleetService)
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	if c.wantHostInfoStore {
		return c.getHostInfoStore(ctx, hc, e.InventoryService, e.AdminService, args)
	}

	emit := !utils.NoEmitMode(c.outputFlags.NoEmit())
	full := utils.FullMode(c.outputFlags.Full())
	var res []proto.Message
	if len(args) > 0 {
		res = utils.ConcurrentGet(ctx, ic, args, c.getSingle)
	} else {
		res, err = utils.BatchList(ctx, ic, host.ListHosts, c.formatFilters(), c.pageSize, c.keysOnly, full)
	}
	if err != nil {
		return err
	}
	return utils.PrintEntities(ctx, ic, res, utils.PrintMachineLSEsJSON, printDutFull, printDutNormal,
		c.outputFlags.JSON(), emit, full, c.outputFlags.Tsv(), c.keysOnly)
}

func (c *getDut) getHostInfoStore(ctx context.Context, hc *http.Client, inventoryService string, adminService string, hostnames []string) error {
	invWithSVClient := fleet.NewInventoryPRPCClient(
		&prpc.Client{
			C:       hc,
			Host:    adminService,
			Options: site.DefaultPRPCOptions,
		},
	)

	invC := inventoryclient.NewInventoryClient(
		hc,
		inventoryService,
		nil,
	)

	g := hostinfo.NewGetter(invC, invWithSVClient)
	for _, hostname := range hostnames {
		contents, err := g.GetContentsForHostname(ctx, hostname)
		if err != nil {
			// TODO: log an error message
			continue
		}
		fmt.Printf("%s\n", contents)
	}
	return nil
}

func (c *getDut) getSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) (proto.Message, error) {
	return ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, name),
	})
}

func (c *getDut) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.ZoneFilterName, c.zones)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.RackFilterName, c.racks)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.MachineFilterName, c.machines)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.MachinePrototypeFilterName, c.prototypes)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.ServoFilterName, c.servos)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.ServoTypeFilterName, c.servotypes)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.SwitchFilterName, c.switches)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.RPMFilterName, c.rpms)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.TagFilterName, c.tags)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.StateFilterName, c.states)...)
	return filters
}

func printDutFull(ctx context.Context, ic ufsAPI.FleetClient, msgs []proto.Message, tsv bool) error {
	machineMap := make(map[string]*ufspb.Machine, 0)
	lses := make([]*ufspb.MachineLSE, len(msgs))
	for i, r := range msgs {
		lses[i] = r.(*ufspb.MachineLSE)
		lses[i].Name = ufsUtil.RemovePrefix(lses[i].Name)
		if len(lses[i].GetMachines()) == 0 {
			fmt.Println("Invalid host ", lses[i].Name)
			continue
		}
		res, _ := ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
			Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, lses[i].GetMachines()[0]),
		})
		machineMap[lses[i].Name] = res
	}
	utils.PrintDutsFull(lses, machineMap)
	return nil
}

func printDutNormal(msgs []proto.Message, tsv, keysOnly bool) error {
	utils.PrintDutsShort(msgs, keysOnly)
	return nil
}
