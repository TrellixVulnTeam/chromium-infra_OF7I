// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/errors"

	"infra/cmdsupport/cmdlib"
	"infra/cros/cmd/satlab/internal/commands"
	"infra/cros/cmd/satlab/internal/paths"
	"infra/cros/cmd/satlab/internal/site"
)

// GetDUTCmd is the implementation of "satlab get dut ...".
var GetDUTCmd = &subcommands.Command{
	UsageLine: "dut [options ...]",
	ShortDesc: "Get a Satlab DUT",
	CommandRun: func() subcommands.CommandRun {
		c := &getDUT{}
		registerGetShivasFlags(c)
		return c
	},
}

// GetDUT holds the arguments for "satlab get dut ...".
type getDUT struct {
	shivasGetDUT
	// Satlab-specific fields, if any exist, go here.
}

// Run runs the get DUT subcommand.
func (c *getDUT) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

// InnerRun runs the get command.
func (c *getDUT) innerRun(a subcommands.Application, positionalArgs []string, env subcommands.Env) error {
	// 'shivas get dut' will list all DUTs everywhere.
	// This command takes a while to execute and gives no immediate feedback, so provide an error message to the user.
	if len(positionalArgs) == 0 {
		// TODO(gregorynisbet): pick a default behavior for get DUT.
		return errors.New(`default "get dut" functionality not implemented`)
	}

	if c.commonFlags.SatlabID == "" {
		var err error
		c.commonFlags.SatlabID, err = commands.GetDockerHostBoxIdentifier()
		if err != nil {
			return errors.Annotate(err, "get dut").Err()
		}
	}

	// No flags need to be annotated with the satlab prefix for get dut.
	// However, the positional arguments need to have the satlab prefix
	// prepended.
	for i, item := range positionalArgs {
		positionalArgs[i] = site.MaybePrepend(site.Satlab, c.commonFlags.SatlabID, item)
	}
	flags := makeGetShivasFlags(c)
	args := (&commands.CommandWithFlags{
		Commands:       []string{paths.ShivasCLI, "get", "dut"},
		Flags:          flags,
		PositionalArgs: positionalArgs,
	}).ToCommand()
	command := exec.Command(args[0], args[1:]...)
	command.Stderr = os.Stderr
	out, err := command.Output()
	// TODO(gregorynisbet): switch to file descriptor in c or consider logging strategy.
	fmt.Printf("%s\n", string(out))
	return errors.Annotate(err, "get dut").Err()
}
