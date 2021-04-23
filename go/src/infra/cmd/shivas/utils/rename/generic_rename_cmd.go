// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rename

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/shivas/site"
	"infra/cmdsupport/cmdlib"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
)

//TODO(anushruth): Refactor rename machine/nic/switch using this

// GenGenericRenameCmd creates a simple rename command for shivas. It takes a name of the
// resource to rename, renameFunc to call the rpc and printRes to print the result of the
// operation.
func GenGenericRenameCmd(kind string, rename RenameFunc, printRes PrintResFunc) *subcommands.Command {
	return &subcommands.Command{
		UsageLine: fmt.Sprintf("%s ...", kind),
		ShortDesc: fmt.Sprintf("Rename %s with new name", kind),
		LongDesc: fmt.Sprintf(`Rename %s with new name.

	Example:

	shivas rename %s -name {oldName} -new-name {newName}

	Renames the %s and prints the output in the user-specified format.

	WARNING: Ensure that renamed %s is working as intended.
	`, kind, kind, kind, kind),
		CommandRun: func() subcommands.CommandRun {
			c := &renameGeneric{}
			c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
			c.envFlags.Register(&c.Flags)
			c.commonFlags.Register(&c.Flags)
			c.Flags.StringVar(&c.name, "name", "", fmt.Sprintf("the name of the %s to rename", kind))
			c.Flags.StringVar(&c.newName, "new-name", "", fmt.Sprintf("the new name of the %s", kind))
			c.Flags.BoolVar(&c.json, "json", true, "enable to print the result of the rename")
			c.rename = rename
			c.printStdOut = printRes
			return c
		},
	}
}

// RenameFunc template to be used to call the rename API RPC.
type RenameFunc func(context.Context, ufsAPI.FleetClient, string, string) (interface{}, error)

// PrintResFunc template to be used to call the print results API.
type PrintResFunc func(proto.Message)

type renameGeneric struct {
	subcommands.CommandRunBase
	authFlags   authcli.Flags
	envFlags    site.EnvFlags
	commonFlags site.CommonFlags

	name    string
	newName string

	json bool

	// RPC and result printing fucntions
	rename      RenameFunc
	printStdOut PrintResFunc
}

func (c *renameGeneric) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *renameGeneric) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(); err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
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
	// Change  this  API if you want to reuse the command somewhere else.
	renamedResource, err := c.rename(ctx, ic, c.name, c.newName)
	if err != nil {
		return err
	}
	if c.json {
		c.printStdOut(renamedResource.(proto.Message))
	}
	if c.commonFlags.Verbose() {
		fmt.Printf("Renamed %s to %s\n", c.name, c.newName)
	}
	return nil
}

func (c *renameGeneric) validateArgs() error {
	if c.name == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-name' is required")
	}
	if c.newName == "" {
		return cmdlib.NewQuietUsageError(c.Flags, "Wrong usage!!\n'-new-name' is required")
	}
	return nil
}
