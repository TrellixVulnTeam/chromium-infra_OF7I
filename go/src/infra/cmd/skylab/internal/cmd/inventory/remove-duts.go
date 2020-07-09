// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	iv "infra/cmd/skylab/internal/inventory"
	"infra/cmd/skylab/internal/site"
	"infra/cmd/skylab/internal/userinput"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/inventory"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// RemoveDuts subcommand: RemoveDuts a DUT from inventory system.
var RemoveDuts = &subcommands.Command{
	UsageLine: "remove-duts -bug BUG [-delete] [FLAGS] DUT...",
	ShortDesc: "remove DUTs from the inventory system",
	LongDesc: `Remove DUTs from the inventory system

Removing DUTs from the inventory system stops the DUTs from being able to run
tasks. Please note that we don't support "removing only" feature any more. When
you run this command, the duts to be removed will be completedly deleted from inventory.

After deleting a DUT, it needs to be deployed from scratch via "add-duts" to run tasks
again.`,
	CommandRun: func() subcommands.CommandRun {
		c := &removeDutsRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.BoolVar(&c.delete, "delete", false, "Delete DUT from inventory (to-be-deprecated).")
		c.removalReason.Register(&c.Flags)
		return c
	},
}

type removeDutsRun struct {
	subcommands.CommandRunBase
	authFlags     authcli.Flags
	envFlags      skycmdlib.EnvFlags
	delete        bool
	removalReason skycmdlib.RemovalReason
}

func (c *removeDutsRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *removeDutsRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	hostnames := c.Flags.Args()
	prompt := userinput.CLIPrompt(a.GetOut(), os.Stdin, false)
	if !prompt(fmt.Sprintf("Ready to remove hosts: %v", hostnames)) {
		return nil
	}

	// Duplicate traffic to UFS (in experiment)
	ufsClient := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UFSService,
		Options: site.DefaultPRPCOptions,
	})
	fmt.Fprintf(a.GetOut(), "####### TESTING with ufs service: %s #######\n", e.UFSService)
	if err := c.deleteFromUFS(ctx, ufsClient, hostnames); err != nil {
		fmt.Fprintf(a.GetOut(), "%s\n", err.Error())
		fmt.Fprintf(a.GetOut(), "####### The above error is NOT FATAL #######\n")
	} else {
		fmt.Fprintf(a.GetOut(), "Successfully undeploy the following machines from UFS:\n")
		for _, h := range hostnames {
			fmt.Fprintf(a.GetOut(), "\t%s\n", h)
		}
		fmt.Fprintf(a.GetOut(), "####### Finish TESTING #######\n")
	}
	fmt.Fprintf(a.GetOut(), "\n")

	ic := iv.NewInventoryClient(hc, e)
	modified, err := ic.DeleteDUTs(ctx, c.Flags.Args(), &c.authFlags, c.removalReason, a.GetOut())
	if err != nil {
		return err
	}
	if !modified {
		fmt.Fprintln(a.GetOut(), "No DUTs modified")
		return nil
	}
	return nil
}

// deleteFromUFS kicks off the inventory updates to UFS
func (c *removeDutsRun) deleteFromUFS(ctx context.Context, ufsClient ufsAPI.FleetClient, hostnames []string) error {
	for _, hostname := range hostnames {
		_, err := ufsClient.DeleteMachineLSE(ctx, &ufsAPI.DeleteMachineLSERequest{
			Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, hostname),
		})
		if err != nil {
			return errors.Annotate(err, "fail to delete host %s from UFS inventory system", hostname).Err()
		}
	}
	return nil
}

func (c *removeDutsRun) validateArgs() error {
	var errs []string
	if c.Flags.NArg() == 0 {
		errs = append(errs, "must specify at least 1 DUT")
	}
	if c.removalReason.Bug == "" && !c.delete {
		errs = append(errs, "-bug is required when not deleting")
	}
	if c.removalReason.Bug != "" && !userinput.ValidBug(c.removalReason.Bug) {
		errs = append(errs, "-bug must match crbug.com/NNNN or b/NNNN")
	}
	// Limit to roughly one line, like a commit message first line.
	if len(c.removalReason.Comment) > 80 {
		errs = append(errs, "-reason is too long (use the bug for details)")
	}
	if len(errs) > 0 {
		return cmdlib.NewUsageError(c.Flags, strings.Join(errs, ", "))
	}
	return nil
}

func protoTimestamp(t time.Time) *inventory.Timestamp {
	s := t.Unix()
	ns := int32(t.Nanosecond())
	return &inventory.Timestamp{
		Seconds: &s,
		Nanos:   &ns,
	}
}

// printRemovals prints a table of DUT removals from drones.
func printRemovals(w io.Writer, resp *fleet.RemoveDutsFromDronesResponse) error {
	fmt.Fprintf(w, "DUT removal: %s\n", resp.Url)

	t := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(t, "DUT ID\tRemoved from drone")
	for _, r := range resp.Removed {
		fmt.Fprintf(t, "%s\t%s\n", r.GetDutId(), r.GetDroneHostname())
	}
	return t.Flush()
}

// printDeletions prints a list of deleted DUTs.
func printDeletions(w io.Writer, cn int, hostnames []string) error {
	b := bufio.NewWriter(w)
	url, err := iv.ChangeURL(cn)
	if err != nil {
		return err
	}
	fmt.Fprintf(b, "DUT deletion: %s\n", url)
	fmt.Fprintln(b, "Deleted DUT hostnames")
	for _, h := range hostnames {
		fmt.Fprintln(b, h)
	}
	return b.Flush()
}
