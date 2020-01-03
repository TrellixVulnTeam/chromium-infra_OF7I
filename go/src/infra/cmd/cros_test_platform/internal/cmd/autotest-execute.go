// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"fmt"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/common/errors"
)

// AutotestExecute subcommand: Run a set of enumerated tests against autotest backend.
var AutotestExecute = &subcommands.Command{
	UsageLine: "autotest-execute -input_json /path/to/input.json -output_json /path/to/output.json",
	ShortDesc: "Run a set of enumerated tests against autotest backend.",
	LongDesc:  `Run a set of enumerated tests against autotest backend.`,
	CommandRun: func() subcommands.CommandRun {
		c := &autotestExecuteRun{}
		c.addFlags()
		return c
	},
}

type autotestExecuteRun struct {
	commonExecuteRun
}

func (c *autotestExecuteRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.validateArgs(); err != nil {
		fmt.Fprintln(a.GetErr(), err.Error())
		c.Flags.Usage()
		return exitCode(err)
	}

	err := c.innerRun(a, args, env)
	if err != nil {
		fmt.Fprintf(a.GetErr(), "%s\n", err)
	}
	return exitCode(err)
}

func (c *autotestExecuteRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	return errors.Reason("autotest-execute is deprecated").Err()
}
