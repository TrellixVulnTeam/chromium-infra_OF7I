// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/errors"

	"infra/cmdsupport/cmdlib"
	"infra/cros/cmd/satlab/internal/commands"
	"infra/cros/cmd/satlab/internal/paths"
	"infra/cros/cmd/satlab/internal/site"
)

// DeleteDUTCmd is the implementation of the "satlab delete DUT" command.
var DeleteDUTCmd = &subcommands.Command{
	UsageLine: "dut [options ...]",
	ShortDesc: "Delete a Satlab DUT",
	CommandRun: func() subcommands.CommandRun {
		c := &deleteDUT{}
		registerShivasFlags(c)
		return c
	},
}

// DeleteDUT holds the arguments that are needed for the delete DUT command.
type deleteDUT struct {
	shivasDeleteDUT
	// Satlab-specific fields, if any exist, go here.
}

// Run attempts to delete a DUT and returns an exit status.
func (c *deleteDUT) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

// InnerRun is the implementation of the delete command.
func (c *deleteDUT) innerRun(a subcommands.Application, positionalArgs []string, env subcommands.Env) error {
	if c.commonFlags.SatlabID == "" {
		var err error
		c.commonFlags.SatlabID, err = commands.GetDockerHostBoxIdentifier()
		if err != nil {
			return errors.Annotate(err, "get dut").Err()
		}
	}

	// No flags need to be annotated with the satlab prefix for delete dut.
	// However, the positional arguments need to have the satlab prefix
	// prepended.
	for i, item := range positionalArgs {
		positionalArgs[i] = site.MaybePrepend(site.Satlab, c.commonFlags.SatlabID, item)
	}

	args := (&commands.CommandWithFlags{
		Commands:       []string{paths.ShivasCLI, "delete", "dut"},
		PositionalArgs: positionalArgs,
		Flags: map[string][]string{
			"namespace": {c.envFlags.Namespace},
		},
	}).ToCommand()
	command := exec.Command(args[0], args[1:]...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return errors.Annotate(
			err,
			fmt.Sprintf(
				"delete dut: running %s",
				strings.Join(args, " "),
			),
		).Err()
	}
	return nil
}
