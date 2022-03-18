// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package attacheddevicemachine

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

// DeleteAttachedDeviceMachineCmd deletes the attached device machine for a given name.
var DeleteAttachedDeviceMachineCmd = &subcommands.Command{
	UsageLine:  "attached-device-machine ...",
	ShortDesc:  "Delete an attached device machine (Hardware asset: Android Phone, iPad, etc.)",
	LongDesc:   cmdhelp.DeleteADMText,
	CommandRun: deleteADMCommandRun,
}

// DeleteADMCmd is an alias to DeleteAttachedDeviceMachineCmd
var DeleteADMCmd = &subcommands.Command{
	UsageLine:  "adm ...",
	ShortDesc:  "Delete an attached device machine (Hardware asset: Android Phone, iPad, etc.)",
	LongDesc:   cmdhelp.DeleteADMText,
	CommandRun: deleteADMCommandRun,
}

func deleteADMCommandRun() subcommands.CommandRun {
	c := &deleteAttachedDeviceMachine{}
	c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
	c.envFlags.Register(&c.Flags)
	c.commonFlags.Register(&c.Flags)
	return c
}

type deleteAttachedDeviceMachine struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags
}

func (c *deleteAttachedDeviceMachine) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *deleteAttachedDeviceMachine) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
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

	if _, err = utils.PrintExistingAttachedDeviceMachine(ctx, ic, args[0]); err != nil {
		return err
	}

	prompt := utils.CLIPrompt(a.GetOut(), os.Stdin, false)
	if prompt != nil && !prompt(fmt.Sprintf("Are you sure you want to delete the attached device machine: %s. ", args[0])) {
		return nil
	}

	_, err = ic.DeleteMachine(ctx, &ufsAPI.DeleteMachineRequest{
		Name: ufsUtil.AddPrefix(ufsUtil.MachineCollection, args[0]),
	})
	if err == nil {
		fmt.Fprintln(a.GetOut(), args[0], "is deleted successfully.")
		return nil
	}
	return err
}

func (c *deleteAttachedDeviceMachine) validateArgs() error {
	if c.Flags.NArg() == 0 {
		return cmdlib.NewUsageError(c.Flags, "Please provide the attached device machine name to be deleted.")
	}
	return nil
}
