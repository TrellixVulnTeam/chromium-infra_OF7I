// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"fmt"
	"io"
	"os"
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
	LongDesc: fmt.Sprintf(`Update some common labels of existing DUTs' in inventory.

Currently common labels only include pool and rpm info. RPM info are only supported by passing an input file via -f.

The format of the input file is guided by:
* each row is separated by "\n"
* each column is separated by ","
* the first column is the hostname
* the following columns must has format of "label-name=label-value".

The supported label names are [%s]

Some example usages of this tool:

* skylab batch-update-duts -pool DUT_POOL_QUOTA host1 host2 host3

* skylab batch-update-duts -f input_file

the content of the input file could be:
hostname1,pool=fake_pool1
hostname2,powerunit_hostname=fake_host,powerunit_outlet=fake_outlet
...
	`, userinput.SupportedLabels),
	CommandRun: func() subcommands.CommandRun {
		c := &batchUpdateDutsRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.pool, "pool", "DUT_POOL_QUOTA", "The pool to update for the given hostnames")
		c.Flags.StringVar(&c.inputFile, "input_file", "", "A file which contains a list of hostnames, and required info to update at each line. It's mutually exclusive with all other parameters")

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
	hostnames := c.Flags.Args()
	if len(hostnames) > 0 && c.inputFile != "" {
		return NewUsageError(c.Flags, "inputFile or HOSTNAMES are mutually exclusive, can only specify one of them")
	}
	if len(hostnames) == 0 && c.inputFile == "" {
		return NewUsageError(c.Flags, "must specify at least one hostname or pass in a file via -f")
	}
	if len(hostnames) > 0 && c.pool == "" {
		return NewUsageError(c.Flags, "must specify non-empty pool to update")
	}

	var req *fleet.BatchUpdateDutsRequest
	var err error
	if len(hostnames) > 0 {
		req = getRequestForHostnames(hostnames, c.pool)
	} else if c.inputFile != "" {
		req, err = userinput.GetRequestFromFiles(c.inputFile)
		if err != nil {
			return err
		}
	}

	prompt := userinput.CLIPrompt(a.GetOut(), os.Stdin, false)
	if !prompt(fmt.Sprintf("Ready to update %d hosts", len(req.GetDutProperties()))) {
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
	ds, err := ic.BatchUpdateDuts(ctx, req)
	if err != nil {
		return errors.Annotate(err, "fail to update Duts").Err()
	}
	if err := printUpdates(a.GetOut(), ds); err != nil {
		return errors.Annotate(err, "fail to print updates").Err()
	}
	return nil
}

func getRequestForHostnames(hostnames []string, pool string) *fleet.BatchUpdateDutsRequest {
	var duts []*fleet.DutProperty
	for _, h := range hostnames {
		duts = append(duts, &fleet.DutProperty{
			Hostname: h,
			Pool:     pool,
		})
	}
	return &fleet.BatchUpdateDutsRequest{
		DutProperties: duts,
	}
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
