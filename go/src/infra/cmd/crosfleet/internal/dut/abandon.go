// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"fmt"
	"infra/cmd/crosfleet/internal/buildbucket"
	"infra/cmd/crosfleet/internal/common"
	"infra/cmd/crosfleet/internal/site"
	"infra/cmdsupport/cmdlib"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
)

const abandonCmd = "abandon"

var abandon = &subcommands.Command{
	UsageLine: fmt.Sprintf("%s [HOST...]", abandonCmd),
	ShortDesc: "abandon DUTs which were previously leased via 'dut lease'",
	LongDesc: `Abandon DUTs which were previously leased via 'dut lease'.

If no hostnames are entered, all pending or active leases by the current user
will be abandoned.

This command's behavior is subject to change without notice.
Do not build automation around this subcommand.`,
	CommandRun: func() subcommands.CommandRun {
		c := &abandonRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.reason, "reason", "", "Optional reason for abandoning.")
		return c
	},
}

type abandonRun struct {
	subcommands.CommandRunBase
	reason    string
	authFlags authcli.Flags
	envFlags  common.EnvFlags
}

func (c *abandonRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *abandonRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)
	userEmail, err := common.GetUserEmail(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	swarmingService, err := newSwarmingService(ctx, c.envFlags.Env().SwarmingService, &c.authFlags)
	if err != nil {
		return err
	}
	leasesBBClient, err := buildbucket.NewClient(ctx, c.envFlags.Env().DUTLeaserBuilder, c.envFlags.Env().BuildbucketService, c.authFlags)
	if err != nil {
		return err
	}
	earliestCreateTime := common.OffsetTimestamp(-1 * maxLeaseLengthMinutes)
	var botIDs []string
	for _, hostname := range args {
		correctedHostname := correctedHostname(hostname)
		id, err := hostnameToBotID(ctx, swarmingService, correctedHostname)
		if err != nil {
			return err
		}
		botIDs = append(botIDs, id)
	}

	err = leasesBBClient.CancelBuildsByUser(ctx, a.GetOut(), earliestCreateTime, userEmail, botIDs, c.reason)
	if err != nil {
		return err
	}
	return nil
}
