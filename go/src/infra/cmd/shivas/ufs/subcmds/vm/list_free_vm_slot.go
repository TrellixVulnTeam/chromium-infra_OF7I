// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package vm

import (
	"fmt"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

// ListVMSlotCmd returns vm slots by some filters.
var ListVMSlotCmd = &subcommands.Command{
	UsageLine: "vm-slots -filter",
	ShortDesc: "Get free VM slots",
	LongDesc: `Get free VM slots by filters.

Examples:
shivas list vm-slots -n 5 -filter man=apple
Fetches 5 vm slots by manufacturer of chrome platform.

shivas list vm-slots -n 5 -filter 'man=apple & lab=mtv97'
Fetches 5 vm slots by location & manufacturer.`,
	CommandRun: func() subcommands.CommandRun {
		c := &listVMSlot{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)

		c.Flags.IntVar(&c.number, "n", 0, "the number of free vm slots to fetch.")
		c.Flags.StringVar(&c.filter, "filter", "", cmdhelp.VMSlotFilterHelp)
		return c
	},
}

type listVMSlot struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	filter string
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

	// Get the host machineLSE
	resp, err := ic.ListMachineLSEs(ctx, &ufsAPI.ListMachineLSEsRequest{
		PageSize: int32(c.number),
		Filter:   c.filter + "& free=true",
	})
	if err != nil {
		return errors.Annotate(err, "No free vm slots found").Err()
	}

	utils.PrintFreeVMs(ctx, ic, resp.GetMachineLSEs())
	return nil
}

func (c *listVMSlot) validateArgs() error {
	if c.number == 0 {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\n'-n' is required")
	}
	if c.filter == "" {
		return cmdlib.NewUsageError(c.Flags, "Wrong usage!!\nAt least one filter ('-filter') is required to get free vm slots")
	}
	if c.filter != "" {
		filter := fmt.Sprintf(strings.Replace(c.filter, " ", "", -1))
		if !ufsAPI.FilterRegex.MatchString(filter) {
			return cmdlib.NewUsageError(c.Flags, ufsAPI.InvalidFilterFormat)
		}
		var err error
		c.filter, err = utils.ReplaceLabNames(filter)
		if err != nil {
			return err
		}
	}
	return nil
}
