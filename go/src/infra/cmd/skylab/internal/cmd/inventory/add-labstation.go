// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"
	"fmt"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmd/skylab/internal/userinput"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/inventory"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

const addlabstationWarning = `

*********************************************************************************************
*                                                                                           *
*    WARNING: 'skylab add-labstation' is deprecated and will be removed                     *
*             in April 2021. Consider using 'shivas add labstation' instead.                *
*                                                                                           *
*********************************************************************************************

`

// AddLabstation subcommand: add a new labstation to inventory and prepare it for tasks.
var AddLabstation = &subcommands.Command{
	UsageLine: "add-labstation [FLAGS...]",
	ShortDesc: "add a new labstation, batch deploy option is also supported",
	LongDesc: `Add and a new labstation to the inventory and prepare it for tasks.
Add labstation as batch(via csv file) option is also supported.

A repair task to validate labstation deployment is always triggered after labstation
addition.

By default, this subcommand opens up your favourite text editor to enter the
specs for the new labstation. Use -new-specs-file to run non-interactively.` + addlabstationWarning,
	CommandRun: func() subcommands.CommandRun {
		c := &addLabstationRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.StringVar(&c.newSpecsFile, "specs-file", "",
			`Path to a file containing labstation inventory specification.
This file must contain one inventory.DeviceUnderTest JSON-encoded protobuf
message.

The JSON-encoding for protobuf messages is described at
https://developers.google.com/protocol-buffers/docs/proto3#json

The protobuf definition of inventory.DeviceUnderTest is part of
https://chromium.googlesource.com/infra/infra/+/refs/heads/master/go/src/infra/libs/skylab/inventory/device.proto`)
		c.Flags.BoolVar(&c.tail, "tail", false, "Wait for the deployment task to complete.")

		c.Flags.BoolVar(&c.mcsv, "m", false, `interpret the specs file as a CSV of labstation descriptions.`)

		return c
	},
}

type addLabstationRun struct {
	subcommands.CommandRunBase
	authFlags    authcli.Flags
	envFlags     skycmdlib.EnvFlags
	newSpecsFile string
	tail         bool
	mcsv         bool
}

// Run implements the subcommands.CommandRun interface.
func (c *addLabstationRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *addLabstationRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	var specs []*inventory.DeviceUnderTest
	var err error
	if len(args) > 0 {
		return cmdlib.NewUsageError(c.Flags, "unexpected positional args: %s", args)
	}

	ctx := cli.GetContext(a, c, env)
	hc, err := cmdlib.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return err
	}
	e := c.envFlags.Env()
	ic := fleet.NewInventoryPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.AdminService,
		Options: site.DefaultPRPCOptions,
	})

	if c.mcsv {
		specs, err = userinput.GetMCSVSpecs(c.newSpecsFile)
		if err != nil {
			return err
		}
		specs, err = setOsTypeLabstation(specs)
		if err != nil {
			return err
		}
	} else {
		s, err := c.getSpecs(a)
		if err != nil {
			return err
		}
		specs = []*inventory.DeviceUnderTest{s}
	}

	// successfully do nothing if there's nothing to do
	if len(specs) == 0 {
		return fmt.Errorf("no specs given")
	}

	for _, spec := range specs {
		setIgnoredID(spec)
	}

	// Duplicate traffic to UFS (in experiment)
	ufsClient := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UFSService,
		Options: site.UFSPRPCOptions,
	})
	fmt.Fprintf(a.GetOut(), "####### TESTING with ufs service: %s #######\n", e.UFSService)
	if err := c.deployToUFS(ctx, ufsClient, specs); err != nil {
		fmt.Fprintf(a.GetOut(), "%s\n", err.Error())
		fmt.Fprintf(a.GetOut(), "####### The above error is NOT FATAL #######\n")
	} else {
		fmt.Fprintf(a.GetOut(), "Successfully deploy the following labstations to UFS:\n")
		for _, spec := range specs {
			fmt.Fprintf(a.GetOut(), "\t%s\n", spec.GetCommon().GetHostname())
		}
		fmt.Fprintf(a.GetOut(), "####### Finish TESTING #######\n")
	}
	fmt.Fprintf(a.GetOut(), "\n")

	deploymentID, err := c.triggerDeploy(ctx, ic, specs)
	if err != nil {
		return err
	}
	ds, err := ic.GetDeploymentStatus(ctx, &fleet.GetDeploymentStatusRequest{DeploymentId: deploymentID})
	if err != nil {
		return err
	}
	if err := printDeploymentStatus(a.GetOut(), deploymentID, ds); err != nil {
		return err
	}

	if c.tail {
		return tailDeployment(ctx, a.GetOut(), ic, deploymentID, ds)
	}
	return nil
}

const (
	addLabstationHelpText = `* All [PLACEHOLDER] values must be replaced with real values, or those fields
	must be deleted.`

	addLabstationInitialSpecs = `{
	"common": {
		"environment": "ENVIRONMENT_PROD",
		"hostname": "[PLACEHOLDER] Required: unqualified hostname of the host",
		"id": "[IGNORED]. Do not edit (crbug.com/950553). ID is auto-generated.",
		"labels": {
			"board": "[PLACEHOLDER] board of the labstation (roughly identifies the portage overlay the OS images come from)",
			"selfServePools": [
				"labstation_main"
			],
			"model": "[PLACEHOLDER] model of the labstation (roughly identifies the labstation hardware variant)",
			"osType": "OS_TYPE_LABSTATION"
		}
  }
}`
)

// getSpecs parses the DeviceUnderTest from specsFile, or from the user.
//
// If c.specsFile is provided, it is parsed.
// If c.specsFile is "", getSpecs() obtains the specs interactively from the user.
func (c *addLabstationRun) getSpecs(a subcommands.Application) (*inventory.DeviceUnderTest, error) {
	if c.newSpecsFile != "" {
		return parseSpecsFile(c.newSpecsFile)
	}
	template := mustParseSpec(addLabstationInitialSpecs)
	specs, err := userinput.GetDeviceSpecs(template, addLabstationHelpText, userinput.CLIPrompt(a.GetOut(), os.Stdin, true), ensureNoPlaceholderValues)
	if err != nil {
		return nil, err
	}
	return specs, nil
}

// deployToUFS kicks off the inventory updates to UFS
func (c *addLabstationRun) deployToUFS(ctx context.Context, ufsClient ufsAPI.FleetClient, specs []*inventory.DeviceUnderTest) error {
	// Set the namespace to os in context metadata for UFS api call
	newCtx := skycmdlib.SetupContext(ctx, ufsUtil.OSNamespace)
	for _, spec := range specs {
		_, labstationsToAdd, _, err := invV2Api.ImportFromV1DutSpecs([]*inventory.CommonDeviceSpecs{spec.GetCommon()})
		if len(labstationsToAdd) != 1 {
			return errors.Reason("Cannot parse lab config from the labstation %s's spec", spec.GetCommon().GetHostname()).Err()
		}
		mlse := ufsUtil.LabstationToLSE(labstationsToAdd[0].GetLabstation(), "", nil)
		_, err = ufsClient.CreateMachineLSE(newCtx, &ufsAPI.CreateMachineLSERequest{
			MachineLSE:   mlse,
			MachineLSEId: mlse.GetName(),
		})
		if err != nil {
			return errors.Annotate(err, "fail to add host %s to UFS inventory system", mlse.GetName()).Err()
		}
	}
	return nil
}

// triggerDeploy kicks off a DeployDut attempt via crosskylabadmin.
//
// This function returns the deployment task ID for the attempt.
func (c *addLabstationRun) triggerDeploy(ctx context.Context, ic fleet.InventoryClient, specs []*inventory.DeviceUnderTest) (string, error) {
	serialized, err := serializeMany(specs)
	if err != nil {
		return "", errors.Annotate(err, "trigger deploy").Err()
	}

	resp, err := ic.DeployDut(ctx, &fleet.DeployDutRequest{
		NewSpecs: serialized,
		Actions: &fleet.DutDeploymentActions{
			SetupLabstation:          true,
			RunPreDeployVerification: true,
		},
		Options: &fleet.DutDeploymentOptions{
			AssignServoPortIfMissing: false,
		},
	})
	if err != nil {
		return "", errors.Annotate(err, "trigger deploy").Err()
	}
	return resp.GetDeploymentId(), nil
}

func setOsTypeLabstation(labstations []*inventory.DeviceUnderTest) ([]*inventory.DeviceUnderTest, error) {
	for _, lab := range labstations {
		osTypeLabstation := inventory.SchedulableLabels_OS_TYPE_LABSTATION
		lab.Common.Labels.OsType = &osTypeLabstation
	}
	return labstations, nil
}
