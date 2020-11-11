// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package internalcmds

import (
	"bufio"
	"fmt"
	"text/tabwriter"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
)

// DutStableVersion subcommand: Stable versions for DUT.
var DutStableVersion = &subcommands.Command{
	UsageLine: "stable-version HOSTNAME",
	ShortDesc: "Stable versions for DUT",
	LongDesc: `Stable versions for DUT.

For internal use only.`,
	CommandRun: func() subcommands.CommandRun {
		c := &dutStableVersionRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		return c
	},
}

type dutStableVersionRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  skycmdlib.EnvFlags
}

func (c *dutStableVersionRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *dutStableVersionRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) != 1 {
		return cmdlib.NewUsageError(c.Flags, "exactly one HOSTNAME must be provided")
	}
	hostname := args[0]
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	siteEnv := c.envFlags.Env()
	ic := fleet.NewInventoryPRPCClient(&prpc.Client{
		C:       hc,
		Host:    siteEnv.AdminService,
		Options: site.DefaultPRPCOptions,
	})
	req := fleet.GetStableVersionRequest{Hostname: hostname}
	res, err := ic.GetStableVersion(ctx, &req)
	if err != nil {
		return err
	}

	bw := bufio.NewWriter(a.GetOut())
	tw := tabwriter.NewWriter(bw, 0, 2, 2, ' ', 0)
	defer tw.Flush()
	defer bw.Flush()
	fmt.Fprintf(tw, "Hostname:\t%s\n", hostname)
	fmt.Fprintf(tw, "Cros:\t%s\n", res.GetCrosVersion())
	fmt.Fprintf(tw, "Firmware:\t%s\n", res.GetFirmwareVersion())
	fmt.Fprintf(tw, "Servo-cros:\t%s\n", res.GetServoCrosVersion())
	fmt.Fprintf(tw, "Faft:\t%s\n", res.GetFaftVersion())
	fmt.Fprintf(tw, "\n")
	return nil
}
