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

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/gerrit"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	iv "infra/cmd/skylab/internal/inventory"
	"infra/cmd/skylab/internal/site"
	"infra/cmd/skylab/internal/userinput"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/inventory"
)

// RemoveDuts subcommand: RemoveDuts a DUT from a drone.
var RemoveDuts = &subcommands.Command{
	UsageLine: "remove-duts -bug BUG [-delete] [FLAGS] DUT...",
	ShortDesc: "remove DUTs from a drone",
	LongDesc: `Remove DUTs from a drone

-bug is required unless -delete is passed.

If -drone is given, check that the DUTs are currently assigned to that
drone.  Otherwise, the DUTs are removed from whichever drone they are
currently assigned to.

Removing DUTs from a drone stops the DUTs from being able to run
tasks.  The DUT can be assigned with assign-duts to run tasks again.

Setting -delete deletes the DUTs from the inventory entirely.  After
deleting a DUT, it would have to be deployed from scratch to run tasks
again.`,
	CommandRun: func() subcommands.CommandRun {
		c := &removeDutsRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.server, "drone", "", "Drone to remove DUTs from.")
		c.Flags.BoolVar(&c.delete, "delete", false, "Delete DUT from inventory.")
		c.Flags.BoolVar(&c.v2, "v2", false, "[INTERNAL ONLY] Use ChromeOS Lab inventory v2 service.")
		c.removalReason.Register(&c.Flags)
		return c
	},
}

type removeDutsRun struct {
	subcommands.CommandRunBase
	authFlags     authcli.Flags
	envFlags      skycmdlib.EnvFlags
	server        string
	delete        bool
	v2            bool
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

	// Inventory v1 is the default client.
	// Only the result from main inventory client will be printed out.
	icMain := NewInventoryClient(hc, e, false)
	icBackup := NewInventoryClient(hc, e, true)
	if c.v2 {
		icMain, icBackup = icBackup, icMain
	}
	icBackup.removeDUTs(ctx, c.server, c.Flags.Args(), c.removalReason, a.GetOut())
	modified, err := icMain.removeDUTs(ctx, c.server, c.Flags.Args(), c.removalReason, a.GetOut())
	if err != nil {
		return err
	}
	if c.delete {
		icBackup.deleteDUTs(ctx, c.Flags.Args(), &c.authFlags, a.GetOut())
		mod, err := icMain.deleteDUTs(ctx, c.Flags.Args(), &c.authFlags, a.GetOut())
		if err != nil {
			return err
		}
		modified = modified || mod
	}
	if !modified {
		fmt.Fprintln(a.GetOut(), "No DUTs modified")
		return nil
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
	if c.removalReason.Bug != "" && !validBug(c.removalReason.Bug) {
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

// validBug returns true if the given bug string is acceptably formatted.
func validBug(bug string) bool {
	if strings.HasPrefix(bug, "b/") {
		return true
	}
	if strings.HasPrefix(bug, "crbug.com/") {
		return true
	}
	return false
}

func (client *inventoryClientV1) removeDUTs(ctx context.Context, drone string, hostnames []string, rr skycmdlib.RemovalReason, stdout io.Writer) (modified bool, err error) {
	req, err := removeRequest(drone, hostnames, rr)
	if err != nil {
		return false, err
	}
	resp, err := client.ic.RemoveDutsFromDrones(ctx, &req)
	if err != nil {
		return false, err
	}
	if resp.Url == "" {
		return false, nil
	}
	_ = printRemovals(stdout, resp)
	return true, nil
}

func (client *inventoryClientV2) removeDUTs(ctx context.Context, drone string, hostnames []string, rr skycmdlib.RemovalReason, stdout io.Writer) (bool, error) {
	var devIds []*invV2Api.DeviceID
	for _, h := range hostnames {
		devIds = append(devIds, &invV2Api.DeviceID{Id: &invV2Api.DeviceID_Hostname{Hostname: h}})
	}
	rsp, err := client.ic.DeleteCrosDevices(ctx, &invV2Api.DeleteCrosDevicesRequest{
		Ids: devIds,
	})
	if err != nil {
		return false, errors.Annotate(err, "[v2] remove devices for %s ...", hostnames[0]).Err()
	}
	if len(rsp.FailedDevices) > 0 {
		var reasons []string
		for _, d := range rsp.FailedDevices {
			reasons = append(reasons, fmt.Sprintf("%s:%s", d.Hostname, d.ErrorMsg))
		}
		return false, errors.Reason("[v2] failed to remove device: %s", strings.Join(reasons, ", ")).Err()
	}
	b := bufio.NewWriter(stdout)
	fmt.Fprintln(b, "== Inventory v2: output begin ==")
	fmt.Fprintln(b, "Deleted DUT hostnames")
	for _, d := range rsp.RemovedDevices {
		fmt.Fprintln(b, d.Hostname)
	}
	fmt.Fprintln(b, "== Inventory v2: output end ==")
	b.Flush()
	return len(rsp.RemovedDevices) > 0, nil
}

// removeRequest builds a RPC remove request.
func removeRequest(server string, hostnames []string, rr skycmdlib.RemovalReason) (fleet.RemoveDutsFromDronesRequest, error) {
	req := fleet.RemoveDutsFromDronesRequest{
		Removals: make([]*fleet.RemoveDutsFromDronesRequest_Item, len(hostnames)),
	}
	reason := inventory.RemovalReason{
		Bug:        &rr.Bug,
		Comment:    &rr.Comment,
		ExpireTime: protoTimestamp(rr.Expire),
	}
	enc, err := proto.Marshal(&reason)
	if err != nil {
		return req, errors.Annotate(err, "make remove request").Err()
	}
	for i, hn := range hostnames {
		req.Removals[i] = &fleet.RemoveDutsFromDronesRequest_Item{
			DutHostname:   hn,
			DroneHostname: server,
			RemovalReason: enc,
		}
	}
	return req, nil
}

func protoTimestamp(t time.Time) *inventory.Timestamp {
	s := t.Unix()
	ns := int32(t.Nanosecond())
	return &inventory.Timestamp{
		Seconds: &s,
		Nanos:   &ns,
	}
}

func (client *inventoryClientV1) deleteDUTs(ctx context.Context, hostnames []string, authFlags *authcli.Flags, stdout io.Writer) (modified bool, err error) {
	hc, err := cmdlib.NewHTTPClient(ctx, authFlags)
	if err != nil {
		return false, err
	}
	ic, err := iv.CreateC(hc)
	if err != nil {
		return false, err
	}

	var changeInfo *gerrit.ChangeInfo
	defer func() {
		if changeInfo != nil {
			err := ic.AbandonChange(ctx, changeInfo)
			if err != nil {
				b := bufio.NewWriter(stdout)
				fmt.Fprintf(b, "Failed to abandon change %v on error", changeInfo)
			}
		}
	}()
	changeInfo, err = ic.CreateChange(ctx, fmt.Sprintf("delete %d duts", len(hostnames)))
	if err != nil {
		return false, err
	}
	for _, host := range hostnames {
		if err := ic.MakeDeleteHostChange(ctx, changeInfo, host); err != nil {
			return false, err
		}
	}
	if err := ic.SubmitChange(ctx, changeInfo); err != nil {
		return false, err
	}
	cn := int(changeInfo.Number)
	// Successful: do not abandon change beyond this point.
	changeInfo = nil

	_ = printDeletions(stdout, cn, hostnames)
	return true, nil
}

func (client *inventoryClientV2) deleteDUTs(ctx context.Context, hostnames []string, authFlags *authcli.Flags, stdout io.Writer) (modified bool, err error) {
	return
}

// printRemovals prints a table of DUT removals from drones.
func printRemovals(w io.Writer, resp *fleet.RemoveDutsFromDronesResponse) error {
	fmt.Fprintf(w, "DUT removal from drone: %s\n", resp.Url)

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
