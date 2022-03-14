// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package attacheddevicehost

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
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// GetAttachedDeviceHostCmd gets the attached device host for a given name.
var GetAttachedDeviceHostCmd = &subcommands.Command{
	UsageLine:  "attached-device-host ...",
	ShortDesc:  "Get attached device host details by filters",
	LongDesc:   cmdhelp.GetADHText,
	CommandRun: getADHCommandRun,
}

// GetADHCmd is an alias to GetAttachedDeviceHostCmd
var GetADHCmd = &subcommands.Command{
	UsageLine:  "adh ...",
	ShortDesc:  "Get attached device host details by filters",
	LongDesc:   cmdhelp.GetADHText,
	CommandRun: getADHCommandRun,
}

func getADHCommandRun() subcommands.CommandRun {
	c := &getAttachedDeviceHost{}
	c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
	c.envFlags.Register(&c.Flags)
	c.outputFlags.Register(&c.Flags)

	c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
	c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)

	c.Flags.Var(flag.StringSlice(&c.zones), "zone", "Name(s) of a zone to filter by. Can be specified multiple times."+cmdhelp.ZoneFilterHelpText)
	c.Flags.Var(flag.StringSlice(&c.racks), "rack", "Name(s) of a rack to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.machines), "machine", "Name(s) of a machine to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.manufacturers), "man", "Name(s) of a manufacturer to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.oses), "os", "Name(s) of an os to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.tags), "tag", "Name(s) of a tag to filter by. Can be specified multiple times.")
	c.Flags.Var(flag.StringSlice(&c.states), "state", "Name(s) of a state to filter by. Can be specified multiple times."+cmdhelp.StateFilterHelpText)
	return c
}

type getAttachedDeviceHost struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	outputFlags site.OutputFlags

	// Filters
	zones         []string
	racks         []string
	machines      []string
	manufacturers []string
	oses          []string
	tags          []string
	states        []string

	pageSize int
	keysOnly bool
}

func (c *getAttachedDeviceHost) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getAttachedDeviceHost) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	emit := !utils.NoEmitMode(c.outputFlags.NoEmit())
	full := utils.FullMode(c.outputFlags.Full())
	var res []proto.Message
	if len(args) > 0 {
		res = utils.ConcurrentGet(ctx, ic, args, c.getSingle)
	} else {
		res, err = utils.BatchList(ctx, ic, listAttachedDeviceHosts, c.formatFilters(), c.pageSize, c.keysOnly, full)
	}
	if err != nil {
		return err
	}
	return utils.PrintEntities(ctx, ic, res, utils.PrintMachineLSEsJSON, printAttachedDeviceHostFull, printAttachedDeviceHostNormal,
		c.outputFlags.JSON(), emit, full, c.outputFlags.Tsv(), c.keysOnly)
}

func (c *getAttachedDeviceHost) formatFilters() []string {
	filters := make([]string, 0)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.ZoneFilterName, c.zones)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.RackFilterName, c.racks)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.MachineFilterName, c.machines)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.ManufacturerFilterName, c.manufacturers)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.OSFilterName, c.oses)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.TagFilterName, c.tags)...)
	filters = utils.JoinFilters(filters, utils.PrefixFilters(ufsUtil.StateFilterName, c.states)...)
	return filters
}

func (c *getAttachedDeviceHost) getSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) (proto.Message, error) {
	return ic.GetMachineLSE(ctx, &ufsAPI.GetMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, name),
	})
}

func listAttachedDeviceHosts(ctx context.Context, ic ufsAPI.FleetClient, pageSize int32, pageToken, filter string, keysOnly, full bool) ([]proto.Message, string, error) {
	req := &ufsAPI.ListMachineLSEsRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
		Full:      full,
	}
	res, err := ic.ListMachineLSEs(ctx, req)
	if err != nil {
		return nil, "", err
	}
	protos := make([]proto.Message, 0)
	for _, lse := range res.GetMachineLSEs() {
		if lse.GetAttachedDeviceLse() != nil {
			protos = append(protos, lse)
		}
	}
	return protos, res.GetNextPageToken(), nil
}

func printAttachedDeviceHostFull(ctx context.Context, ic ufsAPI.FleetClient, entities []proto.Message, tsv bool) error {
	return printAttachedDeviceHostNormal(entities, tsv, false)
}

func printAttachedDeviceHostNormal(entities []proto.Message, tsv, keysOnly bool) error {
	if len(entities) == 0 {
		return nil
	}
	if tsv {
		utils.PrintTSVAttachedDeviceLSEs(entities, keysOnly)
		return nil
	}
	utils.PrintTableTitle(utils.AttachedDeviceLSETitle, tsv, keysOnly)
	utils.PrintAttachedDeviceLSEs(entities, keysOnly)
	return nil
}
