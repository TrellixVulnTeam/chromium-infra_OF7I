// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package attacheddevicemachine

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

// GetAttachedDeviceMachineCmd gets the attached device machine for a given name.
var GetAttachedDeviceMachineCmd = &subcommands.Command{
	UsageLine:  "attached-device-machine ...",
	ShortDesc:  "Get attached device machine details by filters",
	LongDesc:   cmdhelp.GetADMText,
	CommandRun: getADMCommandRun,
}

// GetADMCmd is an alias to GetAttachedDeviceMachineCmd
var GetADMCmd = &subcommands.Command{
	UsageLine:  "adm ...",
	ShortDesc:  "Get attached device machine details by filters",
	LongDesc:   cmdhelp.GetADMText,
	CommandRun: getADMCommandRun,
}

func getADMCommandRun() subcommands.CommandRun {
	c := &getAttachedDeviceMachine{}
	c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
	c.envFlags.Register(&c.Flags)
	c.outputFlags.Register(&c.Flags)

	c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
	c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

	c.Flags.Var(flag.StringSlice(&c.zones), "zone", "Name(s) of a zone to filter by. Can be specified multiple times."+cmdhelp.ZoneFilterHelpText)
	c.Flags.Var(flag.StringSlice(&c.racks), "rack", "Name(s) of a rack to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of a tag to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.states), "state", "Name(s) of a state to filter by. Can be specified multiple times."+cmdhelp.StateFilterHelpText)
	c.Flags.Var(flag.StringSlice(&c.devicetypes), "devicetype", "Name(s) of a devicetype to filter by. Can be specified multiple times."+cmdhelp.AttachedDeviceTypeHelpText)
	c.Flags.Var(flag.StringSlice(&c.models), "model", "Name(s) of a model to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.buildTargets), "build-target", "Name(s) of a build target to filter by. Can be specified multiple times.")
	return c
}

type getAttachedDeviceMachine struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
	outputFlags site.OutputFlags

	// Filters
	zones        []string
	racks        []string
	tags         []string
	states       []string
	devicetypes  []string
	models       []string
	buildTargets []string

	pageSize int
	keysOnly bool
}

func (c *getAttachedDeviceMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getAttachedDeviceMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
		c.formatAttachedDeviceTypes()
		res, err = utils.BatchList(ctx, ic, listAttachedDeviceMachines, c.formatFilters(), c.pageSize, c.keysOnly, full)
	}
	if err != nil {
		return err
	}
	return utils.PrintEntities(ctx, ic, res, utils.PrintMachinesJSON, printAttachedDeviceMachineFull, printAttachedDeviceMachineNormal,
		c.outputFlags.JSON(), emit, full, c.outputFlags.Tsv(), c.keysOnly)
}

func (c *getAttachedDeviceMachine) getSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) (proto.Message, error) {
	return ic.GetMachine(ctx, &ufsAPI.GetMachineRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, name),
	})
}

func (c *getAttachedDeviceMachine) formatAttachedDeviceTypes() {
	for i, v := range c.devicetypes {
		c.devicetypes[i] = ufsUtil.StrToUFSAttachedDeviceType[v]
	}
}

func (c *getAttachedDeviceMachine) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.ZoneFilterName, c.zones)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.RackFilterName, c.racks)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.TagFilterName, c.tags)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.StateFilterName, c.states)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.DeviceTypeFilterName, c.devicetypes)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.ModelFilterName, c.models)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.BuildTargetFilterName, c.buildTargets)...)
	return filters
}

func printAttachedDeviceMachineFull(ctx context.Context, ic ufsAPI.FleetClient, msgs []proto.Message, tsv bool) error {
	return printAttachedDeviceMachineNormal(msgs, tsv, false)
}

func printAttachedDeviceMachineNormal(entities []proto.Message, tsv, keysOnly bool) error {
	if len(entities) == 0 {
		return nil
	}
	if tsv {
		utils.PrintTSVMachines(entities, keysOnly)
		return nil
	}
	utils.PrintTableTitle(utils.AttachedDeviceMachineTitle, tsv, keysOnly)
	utils.PrintMachines(entities, keysOnly)
	return nil
}

func listAttachedDeviceMachines(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly, full bool) ([]proto.Message, string, error) {
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
	protos := make([]proto.Message, 0)
	for _, m := range res.GetMachines() {
		if m.GetAttachedDevice() != nil {
			protos = append(protos, m)
		}
	}
	return protos, res.GetNextPageToken(), nil
}

func (c *getAttachedDeviceMachine) validateArgs() error {
	for _, s := range c.states {
		if !ufsUtil.IsUFSState(s) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid state, please check help info for '-state'.", s)
		}
	}
	for _, d := range c.devicetypes {
		if !ufsUtil.IsAttachedDeviceType(d) {
			return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n%s is not a valid attached device type, please check help info for '-devicetype'.", d)
		}
	}
	return nil
}
