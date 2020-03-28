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
	"runtime/debug"
	"strings"
	"text/tabwriter"
	"time"

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
	protos "infra/libs/fleet/protos"
	"infra/libs/skylab/inventory"
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

	// Use the default inventory configured in site.go.
	ic := NewInventoryClient(hc, e, true)

	modified, err := ic.deleteDUTs(ctx, c.Flags.Args(), &c.authFlags, c.removalReason, a.GetOut())
	if err != nil {
		return err
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

func (client *inventoryClientV1) deleteDUTs(ctx context.Context, hostnames []string, authFlags *authcli.Flags, rr skycmdlib.RemovalReason, stdout io.Writer) (modified bool, err error) {
	// RemovalReason is not used in V1 deletion.
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

func (client *inventoryClientV2) deleteDUTs(ctx context.Context, hostnames []string, authFlags *authcli.Flags, rr skycmdlib.RemovalReason, stdout io.Writer) (modified bool, err error) {
	var devIds []*invV2Api.DeviceID
	for _, h := range hostnames {
		devIds = append(devIds, &invV2Api.DeviceID{Id: &invV2Api.DeviceID_Hostname{Hostname: h}})
	}
	// RemovalReason is to be added into DeleteCrosDevicesRequest.
	rsp, err := client.ic.DeleteCrosDevices(ctx, &invV2Api.DeleteCrosDevicesRequest{
		Ids: devIds,
		Reason: &invV2Api.DeleteCrosDevicesRequest_Reason{
			Bug:     rr.Bug,
			Comment: rr.Comment,
		},
	})
	if err != nil {
		return false, errors.Annotate(err, "remove devices for %s ...", hostnames[0]).Err()
	}
	if len(rsp.FailedDevices) > 0 {
		var reasons []string
		for _, d := range rsp.FailedDevices {
			reasons = append(reasons, fmt.Sprintf("%s:%s", d.Hostname, d.ErrorMsg))
		}
		return false, errors.Reason("failed to remove device: %s", strings.Join(reasons, ", ")).Err()
	}
	b := bufio.NewWriter(stdout)
	fmt.Fprintln(b, "Deleted DUT hostnames")
	for _, d := range rsp.RemovedDevices {
		fmt.Fprintln(b, d.Hostname)
	}
	// TODO(eshwarn) : move this into DeleteCrosDevices in inventoryV2 layer
	updateAssets(ctx, client, rsp.RemovedDevices, b)
	b.Flush()
	return len(rsp.RemovedDevices) > 0, nil
}

func updateAssets(ctx context.Context, client *inventoryClientV2, deletedDevices []*invV2Api.DeviceOpResult, b io.Writer) {
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
		}
	}()
	if len(deletedDevices) < 0 {
		return
	}
	var existingAssetsIDs = make([]string, 0, len(deletedDevices))
	var existingAssets = make([]*protos.ChopsAsset, 0, len(deletedDevices))
	for _, deletedDevice := range deletedDevices {
		existingAssetsIDs = append(existingAssetsIDs, deletedDevice.GetId())
		existingAssets = append(existingAssets,
			&protos.ChopsAsset{
				Id:       deletedDevice.GetId(),
				Location: &protos.Location{},
			})
	}
	assetResponse, _ := client.ic.GetAssets(ctx, &invV2Api.AssetIDList{Id: existingAssetsIDs})
	if assetResponse != nil {
		for _, assetResult := range assetResponse.Passed {
			fmt.Fprintf(b, "AssetId: %s , Old Location: %s\n", assetResult.GetAsset().GetId(), assetResult.GetAsset().GetLocation().String())
		}
		for _, assetResult := range assetResponse.Failed {
			fmt.Fprintf(b, "failed to get asset from registration for %s : %s\n", assetResult.Asset.GetId(), assetResult.GetErrorMsg())
		}
	}
	// Update existing assets in registration system
	assetResponse, _ = client.ic.UpdateAssets(ctx, &invV2Api.AssetList{Asset: existingAssets})
	if assetResponse != nil {
		for _, assetResult := range assetResponse.Passed {
			fmt.Fprintf(b, "AssetId: %s , New Location: %s\n", assetResult.GetAsset().GetId(), assetResult.GetAsset().GetLocation().String())
		}
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
