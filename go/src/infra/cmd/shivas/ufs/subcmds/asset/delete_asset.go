// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package asset

import (
	"context"
	"fmt"
	"os"

	"github.com/golang/protobuf/proto"
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

// DeleteAssetCmd delete a asset on a machine.
var DeleteAssetCmd = &subcommands.Command{
	UsageLine: "asset {assetname}...",
	ShortDesc: "Delete an asset(Chromebook, Servo, Labstation)",
	LongDesc: `Delete an asset.

Example:
shivas delete asset {assetname}

shivas delete asset {assetname1} {assetname2}

Deletes the Asset(s).`,
	CommandRun: func() subcommands.CommandRun {
		c := &deleteAsset{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		return c
	},
}

type deleteAsset struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags
}

func (c *deleteAsset) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *deleteAsset) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})
	prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
	if prompt != nil && !prompt(fmt.Sprintf("Are you sure you want to delete the asset(s): %s", args)) {
		return nil
	}
	assets := utils.ConcurrentGet(ctx, ic, args, c.getSingle)
	fmt.Fprintln(a.GetOut(), "\nAsset(s) before deletion:")
	utils.PrintAssetsJSON(assets, true)
	pass, fail := utils.ConcurrentDelete(ctx, ic, args, c.deleteSingle)
	fmt.Fprintln(a.GetOut(), "\nSuccessfully deleted Asset(s):")
	fmt.Fprintln(a.GetOut(), pass)
	fmt.Fprintln(a.GetOut(), "\nFailed to delete Asset(s):")
	fmt.Fprintln(a.GetOut(), fail)
	return nil
}

func (c *deleteAsset) getSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) (proto.Message, error) {
	return ic.GetAsset(ctx, &ufsAPI.GetAssetRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.AssetCollection, name),
	})
}

func (c *deleteAsset) deleteSingle(ctx context.Context, ic ufsAPI.FleetClient, name string) error {
	_, err := ic.DeleteAsset(ctx, &ufsAPI.DeleteAssetRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.AssetCollection, name),
	})
	return err
}

func (c *deleteAsset) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the name(s) of the asset to delete.")
	}
	return nil
}
