// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

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

// DeleteChromePlatformCmd delete ChromePlatform by given name.
var DeleteChromePlatformCmd = &subcommands.Command{
	UsageLine: "delete-platform",
	ShortDesc: "Delete platform configuration for browser machine",
	LongDesc: `Delete platform configuration for browser machine.

Example:
shivas delete-platform {Platform Name}
Deletes the given platform configuration.`,
	CommandRun: func() subcommands.CommandRun {
		c := &deleteChromePlatform{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		return c
	},
}

type deleteChromePlatform struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
}

func (c *deleteChromePlatform) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *deleteChromePlatform) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
	if !prompt(fmt.Sprintf("Are you sure you want to delete the platform: %s", args[0])) {
		return nil
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
	_, err = ic.DeleteChromePlatform(ctx, &ufsAPI.DeleteChromePlatformRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.ChromePlatformCollection, args[0]),
	})
	if err == nil {
		fmt.Fprintln(a.GetOut(), args[0], "is deleted successfully.")
		return nil
	}
	return err
}

func (c *deleteChromePlatform) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the platform name to be deleted.")
	}
	return nil
}
