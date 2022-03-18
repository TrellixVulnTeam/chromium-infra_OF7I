// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package attacheddevicehost

import (
	"fmt"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/cmdhelp"
	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

// DeleteAttachedDeviceHostCmd deletes the attached device host for a given name.
var DeleteAttachedDeviceHostCmd = &subcommands.Command{
	UsageLine:  "attached-device-host ...",
	ShortDesc:  "Delete an attached device host",
	LongDesc:   cmdhelp.DeleteADHText,
	CommandRun: deleteADHCommandRun,
}

// DeleteADHCmd is an alias to DeleteAttachedDeviceHostCmd
var DeleteADHCmd = &subcommands.Command{
	UsageLine:  "adh ...",
	ShortDesc:  "Delete an attached device host",
	LongDesc:   cmdhelp.DeleteADHText,
	CommandRun: deleteADHCommandRun,
}

func deleteADHCommandRun() subcommands.CommandRun {
	c := &deleteAttachedDeviceHost{}
	c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
	c.envFlags.Register(&c.Flags)
	c.commonFlags.Register(&c.Flags)
	return c
}

type deleteAttachedDeviceHost struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
}

func (c *deleteAttachedDeviceHost) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *deleteAttachedDeviceHost) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	if _, err = utils.PrintExistingAttachedDeviceHost(ctx, ic, args[0]); err != nil {
		return err
	}

	prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
	if prompt != nil && !prompt(fmt.Sprintf("Are you sure you want to delete the attached device host: %s. ", args[0])) {
		return nil
	}

	_, err = ic.DeleteMachineLSE(ctx, &ufsAPI.DeleteMachineLSERequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineLSECollection, args[0]),
	})
	if err == nil {
		fmt.Fprintln(a.GetOut(), args[0], "is deleted successfully.")
		return nil
	}
	return err
}

func (c *deleteAttachedDeviceHost) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the attached device host name to be deleted.")
	}
	return nil
}
