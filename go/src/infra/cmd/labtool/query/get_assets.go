// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package query

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"

	"go.chromium.org/luci/grpc/prpc"
	"infra/cmd/labtool/site"
	"infra/cmd/labtool/utils"
	"infra/cmdsupport/cmdlib"

	fleetAPI "infra/appengine/cros/lab_inventory/api/v1"
)

// GetAssetsCmd query assets by given asset tags.
var GetAssetsCmd = &subcommands.Command{
	UsageLine: "get-assets",
	ShortDesc: "get assets by asset tag",
	LongDesc: `get assets by asset tags.

Please note that the asset tags can be manually entered or scanned.`,
	CommandRun: func() subcommands.CommandRun {
		c := &getAssets{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.BoolVar(&c.interactive, "interactive", false, "if enabling the interactive mode to scan asset tags for querying")
		return c
	},
}

type getAssets struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	interactive bool
}

func (c *getAssets) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getAssets) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}

	tags := c.Flags.Args()
	if c.interactive {
		tags = append(tags, utils.GetInteractiveInput()...)
	}

	e := c.envFlags.Env()
	fmt.Printf("Using inventory service %s\n", e)

	ic := fleetAPI.NewInventoryPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.InventoryService,
		Options: site.DefaultPRPCOptions,
	})

	res, err := ic.GetAssets(ctx, &fleetAPI.AssetIDList{
		Id: tags,
	})
	printOutput(res)
	return nil
}

func (c *getAssets) validateArgs() error {
	if !c.interactive && c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Not in interactive mode and no asset tags are specified")
	}
	return nil
}

func printOutput(res *fleetAPI.AssetResponse) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	defer tw.Flush()
	passed := res.GetPassed()
	if len(passed) > 0 {
		fmt.Fprintln(tw, "\nSuccessful queries:")
		fmt.Fprintln(tw, "Asset Tag\t\tLocation")
		for _, r := range passed {
			fmt.Fprintf(tw, "%s\t\t%s\n", r.GetAsset().GetId(), r.GetAsset().GetLocation().String())
		}
	} else {
		fmt.Fprintln(tw, "\nNo successful queries")
	}

	failed := res.GetFailed()
	if len(failed) > 0 {
		fmt.Fprintln(tw, "\nFailed queries:")
		fmt.Fprintln(tw, "Asset Tag\t\tError")
		for _, r := range failed {
			fmt.Fprintf(tw, "%s\t\t%s\n", r.GetAsset().GetId(), r.GetErrorMsg())
		}
	} else {
		fmt.Fprintln(tw, "\nNo failed queries")
	}
}
