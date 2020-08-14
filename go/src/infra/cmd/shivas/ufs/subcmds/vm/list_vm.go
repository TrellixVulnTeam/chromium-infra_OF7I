// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package vm

import (
	"context"
	"fmt"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

// ListVMCmd list all VMs on a host.
var ListVMCmd = &subcommands.Command{
	UsageLine: "vm -h {Hostname}",
	ShortDesc: "List all VMs on a host",
	LongDesc:  cmdhelp.ListVMLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &listVM{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.outputFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.pageSize, "n", 0, cmdhelp.ListPageSizeDesc)
		c.Flags.BoolVar(&c.keysOnly, "keys", false, cmdhelp.KeysOnlyText)
		c.Flags.StringVar(&c.filter, "filter", "", cmdhelp.VMFilterHelp)
		return c
	},
}

type listVM struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	outputFlags site.OutputFlags

	pageSize int
	filter   string
	keysOnly bool
}

func (c *listVM) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *listVM) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
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

	if !c.outputFlags.JSON() {
		return utils.PrintListTableFormat(ctx, ic, printVMs, false, int32(c.pageSize), c.filter, c.keysOnly, utils.VMTitle, c.outputFlags.Tsv())
	}
	return utils.PrintListJSONFormat(ctx, ic, printVMs, true, int32(c.pageSize), c.filter, c.keysOnly)
}

func printVMs(ctx context.Context, ic ufsAPI.FleetClient, json bool, pageSize int32, pageToken, filter string, keysOnly, tsv bool) (string, error) {
	req := &ufsAPI.ListVMsRequest{
		PageSize:  pageSize,
		PageToken: pageToken,
		Filter:    filter,
		KeysOnly:  keysOnly,
	}
	res, err := ic.ListVMs(ctx, req)
	if err != nil {
		return "", err
	}
	if json {
		utils.PrintVMsJSON(res.GetVms())
	} else if tsv {
		utils.PrintTSVVMs(res.GetVms(), keysOnly)
	} else {
		utils.PrintVMs(res.GetVms(), keysOnly)
	}
	return res.GetNextPageToken(), nil
}

func (c *listVM) validateArgs() error {
	if c.filter != "" {
		filter := fmt.Sprintf(strings.Replace(c.filter, " ", "", -1))
		if !ufsAPI.FilterRegex.MatchString(filter) {
			return cmdlib.NewUsageError(c.Flags, ufsAPI.InvalidFilterFormat)
		}
		var err error
		c.filter, err = utils.ReplaceZoneNames(filter)
		if err != nil {
			return err
		}
	}
	return nil
}
