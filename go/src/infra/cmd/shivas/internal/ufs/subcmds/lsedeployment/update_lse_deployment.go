// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package lsedeployment

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// UpdateMachineLSEDeploymentCmd create/update a machine lse deployment record by the given info
var UpdateMachineLSEDeploymentCmd = &subcommands.Command{
	UsageLine: "host-deployment ...",
	ShortDesc: "Create/Update a deployment record",
	LongDesc: `Create/Update a deployment record.

Example:

shivas update host-deployment -serial serial1 -host host2 -deployment-id id3

Create/Update a deployment record.`,
	CommandRun: func() subcommands.CommandRun {
		c := &updateMachineLSEDeployment{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.commonFlags.Register(&c.Flags)

		c.Flags.StringVar(&c.serial, "serial", "", "the serial of the deployment record to create/update")
		c.Flags.StringVar(&c.host, "host", "", "the hostname of the deployment record to update")
		c.Flags.StringVar(&c.deploymentID, "deployment-id", "", "the deployment identifier of the deployment record to update")
		c.Flags.StringVar(&c.deploymentEnv, "deployment-env", "", "the deployment env of the deployment record to update."+cmdhelp.DeploymentEnvFilterHelpText)
		c.Flags.BoolVar(&c.noHostYet, "no-host-yet", false, "If true, the host will be reset to no-host-yet-<serial>")
		return c
	},
}

type updateMachineLSEDeployment struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	serial        string
	host          string
	deploymentID  string
	deploymentEnv string
	noHostYet     bool
}

func (c *updateMachineLSEDeployment) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *updateMachineLSEDeployment) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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
		fmt.Printf("Using UFS service %s\n", e.UnifiedFleetService)
	}
	ic := ufsAPI.NewFleetPRPCClient(&prpc.Client{
		C:       hc,
		Host:    e.UnifiedFleetService,
		Options: site.DefaultPRPCOptions,
	})

	dr := c.parseArgs()
	utils.PrintExistingLSEDeploymentRecord(ctx, ic, dr.SerialNumber)
	res, err := ic.UpdateMachineLSEDeployment(ctx, &ufsAPI.UpdateMachineLSEDeploymentRequest{
		MachineLseDeployment: dr,
		UpdateMask: utils.GetUpdateMask(&c.Flags, map[string]string{
			"deployment-id":  "deployment_identifier",
			"deployment-env": "deployment_env",
			"host":           "hostname",
			"no-host-yet":    "hostname",
		}),
	})
	if err != nil {
		return err
	}
	res.SerialNumber = ufsUtil.RemovePrefix(res.SerialNumber)
	fmt.Println("The deployment record after update:")
	utils.PrintProtoJSON(res, !utils.NoEmitMode(false))
	return nil
}

func (c *updateMachineLSEDeployment) parseArgs() *ufspb.MachineLSEDeployment {
	var dr *ufspb.MachineLSEDeployment
	if c.noHostYet {
		dr = ufsUtil.FormatDeploymentRecord("", c.serial)
	} else if c.host != "" {
		dr = ufsUtil.FormatDeploymentRecord(c.host, c.serial)
	} else {
		dr = &ufspb.MachineLSEDeployment{
			SerialNumber: c.serial,
		}
	}
	if c.deploymentEnv == utils.ClearFieldValue {
		dr.DeploymentIdentifier = ""
	} else if c.deploymentID != "" {
		dr.DeploymentIdentifier = c.deploymentID
	}
	if c.deploymentEnv == utils.ClearFieldValue {
		dr.DeploymentEnv = ufsUtil.ToUFSDeploymentEnv("")
	} else {
		dr.DeploymentEnv = ufsUtil.ToUFSDeploymentEnv(c.deploymentEnv)
	}
	return dr
}

func (c *updateMachineLSEDeployment) validateArgs() error {
	if c.serial == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-serial' is required.")
	}
	if c.host == "" && c.deploymentID == "" && !c.noHostYet && c.deploymentEnv == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\nNothing to update. Please provide any field to update")
	}
	if c.deploymentEnv != "" && c.deploymentEnv != utils.ClearFieldValue && ufsUtil.ToUFSDeploymentEnv(c.deploymentEnv) == ufspb.DeploymentEnv_DEPLOYMENTENV_UNDEFINED {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\ndeployment-env is neither empty nor a valid env. Please check the help msg.")
	}
	return nil
}
