// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/cmd/skylab/internal/site"
	"infra/cmd/skylab/internal/userinput"
)

// BatchUpdateDuts subcommand: batch update duts with some common labels.
var BatchUpdateDuts = &subcommands.Command{
	UsageLine: "batch-update-duts -pool POOL -f FILE [FLAGS...] HOSTNAMES...",
	ShortDesc: "update a fixed set of common labels for a batch of existing DUTs",
	LongDesc: `Update some common labels of existing DUTs' in inventory.

	Currently common labels only include pool.
	`,
	CommandRun: func() subcommands.CommandRun {
		c := &batchUpdateDutsRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.pool, "pool", "DUT_POOL_QUOTA", "The pool to update for the given hostnames")
		c.Flags.StringVar(&c.inputFile, "input_file", "", "A file which contains a list of hostnames at each line")
		return c
	},
}

type batchUpdateDutsRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  envFlags
	pool      string
	inputFile string
}

// Run implements the subcommands.CommandRun interface.
func (c *batchUpdateDutsRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		PrintError(a.GetErr(), err)
		return 1
	}
	return 0
}

func (c *batchUpdateDutsRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if c.pool == "" {
		return NewUsageError(c.Flags, "must specify non-empty pool to update")
	}
	hostnames, err := appendHostnames(c.Flags.Args(), c.inputFile)
	if err != nil {
		return errors.Annotate(err, "fail to collect hostnames").Err()
	}
	if len(hostnames) == 0 {
		return NewUsageError(c.Flags, "must specify at least one hostname to update")
	}
	prompt := userinput.CLIPrompt(a.GetOut(), os.Stdin, false)
	if !prompt(fmt.Sprintf("Ready to update hosts: %v", hostnames)) {
		return nil
	}

	ctx := cli.GetContext(a, c, env)
	hc, err := newHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "fail to new http client").Err()
	}
	e := c.envFlags.Env()
	ic := fleet.NewInventoryPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.AdminService,
		Options: site.DefaultPRPCOptions,
	})
	ds, err := ic.BatchUpdateDuts(ctx, &fleet.BatchUpdateDutsRequest{
		Hostnames: hostnames,
		Pool:      c.pool,
	})
	if err != nil {
		return errors.Annotate(err, "fail to update Duts").Err()
	}
	if err := printUpdates(a.GetOut(), ds); err != nil {
		return errors.Annotate(err, "fail to print updates").Err()
	}
	return nil
}

func appendHostnames(existingHosts []string, fp string) ([]string, error) {
	if fp == "" {
		return existingHosts, nil
	}
	raw, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	parsed := strings.Split(string(raw), "\n")
	return append(existingHosts, parsed...), nil
}

func printUpdates(w io.Writer, ds *fleet.BatchUpdateDutsResponse) (err error) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	if ds.GetUrl() == "" {
		fmt.Fprintf(tw, "The pool label is already set in inventory, no commit change needed.\n")
	} else {
		fmt.Fprintf(tw, "Inventory change URL:\t%s\n", ds.GetUrl())
	}
	return tw.Flush()
}
