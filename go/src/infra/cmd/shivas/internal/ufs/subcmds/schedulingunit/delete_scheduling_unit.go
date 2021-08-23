// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package schedulingunit

import (
	"fmt"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// DeleteSchedulingUnitCmd delete SchedulingUnit by given name.
var DeleteSchedulingUnitCmd = &subcommands.Command{
	UsageLine: "schedulingunit",
	ShortDesc: "Delete SchedulingUnit",
	LongDesc: `Delete SchedulingUnit.

Example:
shivas delete schedulingunit {SchedulingUnit Name}
Deletes the given SchedulingUnit. Multi-deletion in not supported.`,
	CommandRun: func() subcommands.CommandRun {
		c := &deleteSchedulingUnit{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		return c
	},
}

type deleteSchedulingUnit struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
}

func (c *deleteSchedulingUnit) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *deleteSchedulingUnit) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
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
	if err := utils.PrintExistingSchedulingUnit(ctx, ic, args[0]); err != nil {
		return err
	}
	prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
	if prompt != nil && !prompt(fmt.Sprintf("Are you sure you want to delete the SchedulingUnit: %s", args[0])) {
		return nil
	}
	_, err = ic.DeleteSchedulingUnit(ctx, &ufsAPI.DeleteSchedulingUnitRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.SchedulingUnitCollection, args[0]),
	})
	if err == nil {
		fmt.Fprintln(a.GetOut(), args[0], "is deleted successfully.")
		return nil
	}
	return err
}

func (c *deleteSchedulingUnit) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the SchedulingUnit name to be deleted.")
	}
	return nil
}
