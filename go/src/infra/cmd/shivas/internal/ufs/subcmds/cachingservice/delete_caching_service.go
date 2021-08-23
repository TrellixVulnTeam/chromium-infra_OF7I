// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cachingservice

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

// DeleteCachingServiceCmd delete CachingService by given name.
var DeleteCachingServiceCmd = &subcommands.Command{
	UsageLine: "cachingservice",
	ShortDesc: "Delete CachingService",
	LongDesc: `Delete CachingService.

Example:
shivas delete cachingservice {CachingService Name}
Deletes the given CachingService.`,
	CommandRun: func() subcommands.CommandRun {
		c := &deleteCachingService{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		return c
	},
}

type deleteCachingService struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
}

func (c *deleteCachingService) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *deleteCachingService) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	if err := utils.PrintExistingCachingService(ctx, ic, args[0]); err != nil {
		return err
	}
	prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
	if prompt != nil && !prompt(fmt.Sprintf("Are you sure you want to delete the CachingService: %s", args[0])) {
		return nil
	}
	_, err = ic.DeleteCachingService(ctx, &ufsAPI.DeleteCachingServiceRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.CachingServiceCollection, args[0]),
	})
	if err == nil {
		fmt.Fprintln(a.GetOut(), args[0], "is deleted successfully.")
		return nil
	}
	return err
}

func (c *deleteCachingService) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the CachingService name to be deleted.")
	}
	return nil
}
