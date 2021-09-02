// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"infra/cmdsupport/cmdlib"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/errors"
)

// UpdateDUTCmd is the command that updates fields for a satlab DUT.
var UpdateDUTCmd = &subcommands.Command{
	UsageLine: "dut [options ...]",
	ShortDesc: "Update a Satlab DUT",
	CommandRun: func() subcommands.CommandRun {
		c := &updateDUT{}
		registerUpdateShivasFlags(c)
		return c
	},
}

// UpdateDUT is the 'satlab update dut' command. Its fields are the command line arguments.
type updateDUT struct {
	shivasUpdateDUT
}

// Run is the main entrypoint to 'satlab update dut'.
func (c *updateDUT) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

// InnerRun is the implementation of 'satlab update dut'.
func (c *updateDUT) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	return errors.New("not implemented")
}
