// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/autotest/hostinfo"
)

// GetStableVersionCmd get stable version for given host and board/model.
var GetStableVersionCmd = &subcommands.Command{
	UsageLine: "stable-version ...",
	ShortDesc: "Get stable version details for DUT/labstation/model",
	LongDesc:  cmdhelp.GetStableVersionText,
	CommandRun: func() subcommands.CommandRun {
		c := &getStableVersion{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.board, "board", "", "Name of the board using for getting stable version. If board is not provided then model will be used as board.")
		c.Flags.StringVar(&c.model, "model", "", "Name of the model using for getting stable version")

		return c
	},
}

type getStableVersion struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	board string
	model string
}

func (c *getStableVersion) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *getStableVersion) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	err := c.validateArgs(args)
	if err != nil {
		return err
	}

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

	if c.commonFlags.Verbose() {
		fmt.Printf("Using CrosSkylabAdmin service: %s\n", e.AdminService)
	}

	invWithSVClient := fleet.NewInventoryPRPCClient(
		&prpc.Client{
			C:       hc,
			Host:    e.AdminService,
			Options: site.DefaultPRPCOptions,
		},
	)
	g := hostinfo.NewGetter(nil, invWithSVClient)

	for _, hostname := range args {
		sv, err := g.GetStableVersionForHostname(ctx, hostname)
		if err != nil && c.commonFlags.Verbose() {
			return err
		}
		if len(sv) > 0 {
			fmt.Printf("Stable version for host:%s\n", hostname)
			printStableVersion(a, sv)
		}
	}

	if c.model != "" {
		board := c.board
		if board == "" {
			board = c.model
		}
		sv, err := g.GetStableVersionForModel(ctx, board, c.model)
		if err != nil && c.commonFlags.Verbose() {
			return err
		}
		if len(sv) > 0 {
			fmt.Printf("Stable version for %s:%s\n", c.board, c.model)
			printStableVersion(a, sv)
		}
	}
	return nil
}

func (c *getStableVersion) validateArgs(args []string) error {
	if len(args) == 0 && c.model == "" {
		return cmdlib.NewUsageError(c.Flags, "Please provide hostname or board/model.")
	}
	return nil
}

func printStableVersion(a subcommands.Application, sv map[string]string) {
	keys := make([]string, 0, len(sv))
	for k := range sv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	tw := tabwriter.NewWriter(a.GetOut(), 0, 2, 2, ' ', 0)
	defer tw.Flush()
	for _, k := range keys {
		fmt.Fprintf(tw, "%s:\t%s\n", k, sv[k])
	}
	fmt.Fprintf(tw, "\n")
}
