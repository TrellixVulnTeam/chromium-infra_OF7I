// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	swarming "infra/libs/swarming"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

var defaultRedeployTaskActions = []string{"run-pre-deploy-verification"}

// UpdateDUTCmd update dut by given hostname.
var UpdateDUTCmd = &subcommands.Command{
	UsageLine: "dut [options]",
	ShortDesc: "Update a DUT.",
	LongDesc:  cmdhelp.UpdateDUTLongDesc,
	CommandRun: func() subcommands.CommandRun {
		c := &updateDUT{
			deployTags:    shivasTags,
			deployActions: defaultRedeployTaskActions,
		}
		// Initialize servo setup types
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.newSpecsFile, "f", "", cmdhelp.DUTUpdateFileText)

		c.Flags.Int64Var(&c.deployTaskTimeout, "deploy-timeout", swarming.DeployTaskExecutionTimeout, "execution timeout for deploy task in seconds.")
		c.Flags.BoolVar(&c.deployOnly, "deploy-only", false, "skip updating UFS. Starts a redeploy task.")
		c.Flags.Var(utils.CSVString(&c.deployTags), "deploy-tags", "comma seperated tags for deployment task.")
		c.Flags.BoolVar(&c.deployDownloadImage, "deploy-download-image", false, "download image and stage usb.")
		c.Flags.BoolVar(&c.deployInstallFirmware, "deploy-install-fw", false, "install firmware.")
		c.Flags.BoolVar(&c.deployInstallOS, "deploy-install-os", false, "install os image.")
		c.Flags.BoolVar(&c.deployUpdateLabels, "deploy-update-labels", false, "update labels during deployment.")
		return c
	},
}

type updateDUT struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	newSpecsFile string

	deployOnly            bool
	deployTaskTimeout     int64
	deployActions         []string
	deployTags            []string
	deployDownloadImage   bool
	deployInstallOS       bool
	deployInstallFirmware bool
	deployUpdateLabels    bool
}

func (c *updateDUT) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateDUT) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
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
		if !c.deployOnly {
			fmt.Printf("Using UFS service %s \n", e.UnifiedFleetService)
		}
		fmt.Printf("Using swarming service %s \n", e.SwarmingService)
	}

	req, err := c.parseArgs()
	if err != nil {
		return err
	}

	// Update the UFS database if enabled.
	if !c.deployOnly {
		ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
			C:       hc,
			Host:    e.UnifiedFleetService,
			Options: site.DefaultPRPCOptions,
		})

		if err := c.updateDUTToUFS(ctx, ic, req); err != nil {
			return err
		}
	}

	// Swarm a deploy task.
	c.updateDeployActions()
	tc, err := swarming.NewTaskCreator(ctx, &c.authFlags, e.SwarmingService)
	if err != nil {
		return err
	}
	tc.LogdogService = e.LogdogService
	tc.SwarmingServiceAccount = e.SwarmingServiceAccount
	// Start a swarming deploy task for the DUT.
	if err := c.deployDUTToSwarming(ctx, tc, req.GetMachineLSE()); err != nil {
		return err
	}

	return nil
}

func (c updateDUT) validateArgs() error {
	if c.newSpecsFile == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Need a json file containing dut description. Assign using -f")
	}
	return nil
}

// validateRequest checks if the req is valid based on the cmdline input.
func (c *updateDUT) validateRequest(ctx context.Context, req *ufsAPI.UpdateMachineLSERequest) error {
	lse := req.MachineLSE
	if lse == nil {
		return fmt.Errorf("Internal Error. Invalid UpdateMachineLSERequest")
	}
	if lse.Name == "" {
		return fmt.Errorf("Invalid update. Missing DUT name")
	}
	if len(lse.Machines) == 0 || lse.Machines[0] == "" {
		return fmt.Errorf("Invalid update. Cannot delete asset")
	}
	return nil
}

// parseArgs reads input from the cmd line parameters and generates update dut request.
func (c *updateDUT) parseArgs() (*ufsAPI.UpdateMachineLSERequest, error) {
	if utils.IsCSVFile(c.newSpecsFile) {
		return nil, fmt.Errorf("Not implemented yet")
	}
	machineLse := &ufspb.MachineLSE{}
	if err := utils.ParseJSONFile(c.newSpecsFile, machineLse); err != nil {
		return nil, err
	}
	// json input updates without a mask.
	return &ufsAPI.UpdateMachineLSERequest{
		MachineLSE: machineLse,
	}, nil
}

// updateDeployActions updates the deploySkipActions based on boolean skip options.
func (c *updateDUT) updateDeployActions() {
	if c.deployDownloadImage {
		c.deployActions = append(c.deployActions, "stage-usb")
	}
	if c.deployInstallOS {
		c.deployActions = append(c.deployActions, "install-test-image")
	}
	if c.deployInstallFirmware {
		c.deployActions = append(c.deployActions, "install-firmware", "verify-recovery-mode")
	}
	if c.deployInstallFirmware || c.deployInstallOS || c.deployUpdateLabels {
		c.deployActions = append(c.deployActions, "update-label")
	}
}

// updateDUTToUFS verifies the request and calls UpdateMachineLSE API with the given request.
func (c *updateDUT) updateDUTToUFS(ctx context.Context, ic ufsAPI.FleetClient, req *ufsAPI.UpdateMachineLSERequest) error {
	// Validate the update request.
	if err := c.validateRequest(ctx, req); err != nil {
		return err
	}
	// Print existing LSE before update.
	if err := utils.PrintExistingHost(ctx, ic, req.MachineLSE.GetName()); err != nil {
		return err
	}
	req.MachineLSE.Name = ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, req.MachineLSE.Name)

	res, err := ic.UpdateMachineLSE(ctx, req)
	if err != nil {
		return err
	}
	res.Name = ufsUtil.RemovePrefix(res.Name)
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	fmt.Printf("Successfully updated DUT to UFS: %s \n", res.GetName())
	return nil
}

// deployDUTToSwarming starts a re-deploy task for the given DUT.
func (c *updateDUT) deployDUTToSwarming(ctx context.Context, tc *swarming.TaskCreator, lse *ufspb.MachineLSE) error {
	var hostname, machine string
	// Using hostname because name has resource prefix
	hostname = lse.GetHostname()
	machines := lse.GetMachines()
	if len(machines) > 0 {
		machine = machines[0]
	}
	task, err := tc.DeployDut(ctx, hostname, machine, defaultSwarmingPool, c.deployTaskTimeout, c.deployActions, c.deployTags, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Triggered Deploy task for DUT %s. Follow the deploy job at %s\n", hostname, task.TaskURL)

	return nil
}
