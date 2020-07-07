// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/cmd/utils"
	inv "infra/cmd/skylab/internal/inventory"
	"infra/cmd/skylab/internal/site"
	"infra/cmd/skylab/internal/userinput"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/inventory"
	"infra/libs/skylab/swarming"
)

// UpdateDut subcommand: update and redeploy an existing DUT.
var UpdateDut = &subcommands.Command{
	UsageLine: "update-dut [FLAGS...] HOSTNAME",
	ShortDesc: "update an existing DUT",
	LongDesc: `Update existing DUT's inventory information.

A repair task to validate DUT deployment is triggered after DUT update. See
flags to run costlier DUT preparation steps.

By default, this subcommand opens up your favourite text editor to enter the
new specs for the DUT requested. Use -new-specs-file to run non-interactively.`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateDutRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "new-specs-file", "",
			`Path to a file containing updated DUT inventory specification.
This file must contain one inventory.DeviceUnderTest JSON-encoded protobuf
message.

The JSON-encoding for protobuf messages is described at
https://developers.google.com/protocol-buffers/docs/proto3#json

The protobuf definition of inventory.DeviceUnderTest is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/libs/skylab/inventory/device.proto`)

		c.Flags.BoolVar(&c.installOS, "install-os", false, "Force DUT OS re-install.")
		c.Flags.BoolVar(&c.installFirmware, "install-firmware", false, "Force DUT firmware re-install.")
		c.Flags.BoolVar(&c.skipImageDownload, "skip-image-download", false, `Some DUT preparation steps require downloading OS image onto an external drive
connected to the DUT. This flag disables the download, instead using whatever
image is already downloaded onto the external drive.`)
		return c
	},
}

type updateDutRun struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     skycmdlib.EnvFlags
	newSpecsFile string
	tail         bool

	installOS         bool
	installFirmware   bool
	skipImageDownload bool
}

// Run implements the subcommands.CommandRun interface.
func (c *updateDutRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateDutRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) != 1 {
		return cmdlib.NewUsageError(c.Flags, "want exactly one DUT to update, got %d", len(args))
	}
	hostname := args[0]

	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "update dut").Err()
	}

	icV2 := inv.NewInventoryClient(hc, c.envFlags.Env())
	oldSpecs, err := getOldDeviceSpecs(ctx, icV2, hostname)
	if err != nil {
		return errors.Annotate(err, "update dut").Err()
	}
	newSpecs, err := c.getNewSpecs(a, oldSpecs)
	if err != nil {
		return err
	}

	prompt := userinput.CLIPrompt(a.GetOut(), os.Stdin, false)
	if !prompt(fmt.Sprintf("Ready to update host: %s", hostname)) {
		return nil
	}

	creator, err := utils.NewTaskCreator(ctx, &c.authFlags, c.envFlags)
	if err != nil {
		return err
	}
	taskID, err := c.triggerRedeploy(ctx, icV2, oldSpecs, newSpecs, creator)
	if err != nil {
		if utils.IsSwarmingTaskErr(err) {
			fmt.Fprintf(a.GetOut(), "DUT change has been updated to inventory, but fails to trigger deploy task. Please rerun `skylab update-dut`.\n")
		}
		return err
	}
	fmt.Fprintf(a.GetOut(), "Deploy task URL:\t%s\n", swarming.TaskURL(creator.Environment.SwarmingService, taskID))
	return nil
}

const deployStatusCheckDelay = 30 * time.Second

// tailDeployment tails an ongoing deployment, reporting status updates to the
// user.
func tailDeployment(ctx context.Context, w io.Writer, ic fleet.InventoryClient, deploymentID string, ds *fleet.GetDeploymentStatusResponse) error {
	for !isStatusFinal(ds.GetStatus()) {
		fmt.Fprintln(w, "")
		fmt.Fprintf(w, "Checking again in %s...\n", deployStatusCheckDelay)
		time.Sleep(deployStatusCheckDelay)

		var err error
		ds, err = ic.GetDeploymentStatus(ctx, &fleet.GetDeploymentStatusRequest{
			DeploymentId: deploymentID,
		})
		if err != nil {
			return errors.Annotate(err, "report deployment status").Err()
		}
		fmt.Fprintf(w, "Current status: %s", ds.GetStatus().String())
	}

	if ds.GetStatus() != fleet.GetDeploymentStatusResponse_DUT_DEPLOYMENT_STATUS_SUCCEEDED {
		return errors.Reason("Deployment failed. Final status: %s", ds.GetStatus().String()).Err()
	}
	fmt.Fprintln(w, "Deployment successful!")
	return nil
}

const updateDUTHelpText = "Remove the 'servo_port' attribute to auto-generate a valid servo_port."

// getNewSpecs parses the DeviceUnderTest from specsFile, or from the user.
//
// If c.newSpecsFile is provided, it is parsed.
// If c.newSpecsFile is "", getNewSpecs obtains the specs interactively from the user.
func (c *updateDutRun) getNewSpecs(a subcommands.Application, oldSpecs *inventory.DeviceUnderTest) (*inventory.DeviceUnderTest, error) {
	if c.newSpecsFile != "" {
		return parseSpecsFile(c.newSpecsFile)
	}
	return userinput.GetDeviceSpecs(oldSpecs, updateDUTHelpText, userinput.CLIPrompt(a.GetOut(), os.Stdin, true), nil)
}

// parseSpecsFile parses device specs from the user provided file.
func parseSpecsFile(specsFile string) (*inventory.DeviceUnderTest, error) {
	rawText, err := ioutil.ReadFile(specsFile)
	if err != nil {
		return nil, errors.Annotate(err, "parse specs file").Err()
	}
	text := userinput.DropCommentLines(string(rawText))
	var specs inventory.DeviceUnderTest
	err = jsonpb.Unmarshal(strings.NewReader(text), &specs)
	return &specs, err
}

// triggerRedeploy kicks off a RedeployDut attempt via crosskylabadmin.
//
// This function returns the deployment task ID for the attempt.
func (c *updateDutRun) triggerRedeploy(ctx context.Context, ic inv.Client, old, updated *inventory.DeviceUnderTest, creator *utils.TaskCreator) (string, error) {
	newSpecs := updated.GetCommon()
	if !proto.Equal(old.GetCommon(), newSpecs) {
		if err := ic.UpdateDUT(ctx, newSpecs); err != nil {
			return "", errors.Annotate(err, "update DUT to inventory").Err()
		}
	}
	return creator.DeployTask(ctx, old.GetCommon().GetId(), c.deployActions())
}

func (c *updateDutRun) stageImageToUsb() bool {
	return (c.installFirmware || c.installOS) && !c.skipImageDownload
}

// getOldDeviceSpecs gets the current device specs for hostname from Inventory v2.
func getOldDeviceSpecs(ctx context.Context, ic inv.Client, hostname string) (*inventory.DeviceUnderTest, error) {
	oldDut, err := ic.GetDutInfo(ctx, hostname, true)
	if err != nil {
		return nil, errors.Annotate(err, "get old specs").Err()
	}
	return oldDut, nil
}

func printDeploymentStatus(w io.Writer, deploymentID string, ds *fleet.GetDeploymentStatusResponse) (err error) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, "Deployment ID:\t%s\n", deploymentID)
	fmt.Fprintf(tw, "Status:\t%s\n", ds.GetStatus())
	fmt.Fprintf(tw, "Deploy task URL:\t%s\n", ds.GetTaskUrl())
	fmt.Fprintf(tw, "Message:\t%s\n", ds.GetMessage())
	return tw.Flush()
}

func isStatusFinal(s fleet.GetDeploymentStatusResponse_Status) bool {
	return s != fleet.GetDeploymentStatusResponse_DUT_DEPLOYMENT_STATUS_IN_PROGRESS
}

func (c *updateDutRun) deployActions() string {
	s := make([]string, 0, 5)
	if c.stageImageToUsb() {
		s = append(s, "stage-usb")
	}
	if c.installOS {
		s = append(s, "install-test-image")
		s = append(s, "update-label")
	}
	if c.installFirmware {
		s = append(s, "install-firmware")
		s = append(s, "verify-recovery-mode")
	}
	s = append(s, "run-pre-deploy-verification")
	return strings.Join(s, ",")
}
