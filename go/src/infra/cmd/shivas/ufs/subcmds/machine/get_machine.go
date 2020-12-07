// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package machine

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

// GetMachineCmd get machine by given name.
var GetMachineCmd = &subcommands.Command{
	UsageLine: "machine ...",
	ShortDesc: "Get machine details by filters",
	LongDesc: `Get machine details by filters.

Example:

shivas get machine {name1} {name2}

shivas get machine -platform platform1 -state serving -state needs_repair -zone atl97

Gets the machine and prints the output in the user-specified format.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getMachine{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

		c.Flags.Var(flag.StringSlice(&c.zones), "zone", "Name(s) of a zone to filter by. Can be specified multiple times."+cmdhelp.ZoneFilterHelpText)
		c.Flags.Var(flag.StringSlice(&c.racks), "rack", "Name(s) of a rack to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.kvms), "kvm", "Name(s) of a kvm to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.kvmPorts), "kvm-port", "Name(s) of a kvm port to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.platforms), "platform", "Name(s) of a platform to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of a tag to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.states), "state", "Name(s) of a state to filter by. Can be specified multiple times."+cmdhelp.StateFilterHelpText)

		// Chrome os machine specific filters
		c.Flags.Var(flag.StringSlice(&c.devicetypes), "devicetype", "Name(s) of a devicetype to filter by. Can be specified multiple times."+cmdhelp.DeviceTypeFilterHelpText)
		c.Flags.Var(flag.StringSlice(&c.models), "model", "Name(s) of a model to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.buildTargets), "target", "Name(s) of a build target to filter by. Can be specified multiple times.")
		c.Flags.Var(flag.StringSlice(&c.phases), "phase", "Name(s) of a phase to filter by. Can be specified multiple times.")
		return c
	},
}

type getMachine struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags

	// Filters
	zones        []string
	racks        []string
	kvms         []string
	kvmPorts     []string
	platforms    []string
	tags         []string
	states       []string
	devicetypes  []string
	models       []string
	buildTargets []string
	phases       []string

	pageSize int
	keysOnly bool
}

func (c *getMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
		res, err = utils.BatchList(ctx, ic, listMachines, c.formatFilters(), c.pageSize, c.keysOnly, full)
	}
	if err != nil {
		return err
	}
	return utils.PrintEntities(ctx, ic, res, utils.PrintMachinesJSON, printMachineFull, printMachineNormal,
		c.outputFlags.JSON(), emit, full, c.outputFlags.Tsv(), c.keysOnly)
}

func (c *getMachine) getSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) (proto.Message, error) {
	return ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, name),
	})
}

func (c *getMachine) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("zone", c.zones)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("rack", c.racks)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("kvm", c.kvms)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("kvmport", c.kvmPorts)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("platform", c.platforms)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("tag", c.tags)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("state", c.states)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("devicetype", c.devicetypes)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("model", c.models)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("target", c.buildTargets)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters("phase", c.phases)...)
	return filters
}

func printMachineFull(ctx context.Context, ic ufsAPI.FleetClient, msgs []proto.Message, tsv bool) error {
	return printMachineNormal(msgs, tsv, false)
}

func printMachineNormal(entities []proto.Message, tsv, keysOnly bool) error {
	if len(entities) == 0 {
		return nil
	}
	if tsv {
		utils.PrintTSVMachines(entities, keysOnly)
		return nil
	}
	machine := entities[0].(*ufspb.Machine)
	if machine.GetChromeBrowserMachine() != nil {
		utils.PrintTableTitle(utils.BrowserMachineTitle, tsv, keysOnly)
	} else if machine.GetChromeosMachine() != nil {
		utils.PrintTableTitle(utils.OSMachineTitle, tsv, keysOnly)
	}
	utils.PrintMachines(entities, keysOnly)
	return nil
}

func listMachines(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly, full bool) ([]proto.Message, string, error) {
	req := &ufsAPI.ListMachinesRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
		Full:      full,
	}
	res, err := ic.ListMachines(ctx, req)
	if err != nil {
		return nil, "", err
	}
	protos := make([]proto.Message, len(res.GetMachines()))
	for i, m := range res.GetMachines() {
		protos[i] = m
	}
	return protos, res.GetNextPageToken(), nil
}
