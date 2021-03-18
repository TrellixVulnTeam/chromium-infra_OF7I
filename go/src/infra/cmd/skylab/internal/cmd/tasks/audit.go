// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"fmt"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
)

// Audit subcommand: Audit hosts.
var Audit = &subcommands.Command{
	UsageLine: "audit [flags...] [HOST...]",
	ShortDesc: "create audit tasks",
	LongDesc: `Create audit tasks.

This command does not wait for the tasks to start running.
By default no action runnings. Please specified action to run them via flags.`,
	CommandRun: func() subcommands.CommandRun {
		c := &auditRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.expirationMins, "expiration-mins", 10, "The expiration minutes of the audit request.")
		c.Flags.BoolVar(&c.runVerifyServoUSB, "run-verify-servo-usb", false, "Run the verifier for Servo USB drive.")
		c.Flags.BoolVar(&c.runVerifyDUTStorage, "run-verify-dut-storage", false, "Run the verifier for DUT storage.")
		c.Flags.BoolVar(&c.runVerifyServoFw, "run-verify-servo-fw", false, "Run the verifier for Servo firmware update.")
		c.Flags.BoolVar(&c.runFlashServoKeyboardMap, "run-flash-servo-keyboard-map", false, "Run the action to flash Servo keyboard map to the DUT.")
		c.Flags.BoolVar(&c.runVerifyDutMacaddr, "run-verify-dut-macaddr", false, "Run the verifier to check and cache mac address of DUT NIC to Servo.")
		return c
	},
}

type auditRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  skycmdlib.EnvFlags

	expirationMins           int
	runVerifyServoUSB        bool
	runVerifyDUTStorage      bool
	runVerifyServoFw         bool
	runFlashServoKeyboardMap bool
	runVerifyDutMacaddr      bool
}

func (c *auditRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *auditRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	hosts := args
	if len(args) == 0 {
		hosts = []string{"host1", "host2", "..."}
	}
	var options []string
	if c.runVerifyDUTStorage {
		options = append(options, "-dut-storage")
	}
	if c.runVerifyServoUSB {
		options = append(options, "-servo-usb")
	}
	if c.runVerifyServoFw {
		options = append(options, "-servo-fw")
	}
	if c.runFlashServoKeyboardMap {
		options = append(options, "-servo-keyboard")
	}
	if c.runVerifyDutMacaddr {
		options = append(options, "-dut-macaddr")
	}
	if len(options) == 0 {
		options = append(options, "[OPTIONS]")
	}
	return cmdlib.NewUsageError(
		c.Flags,
		"skylab audit has been removed! Please use:\n\nshivas audit-duts "+strings.Join(options, " ")+" "+strings.Join(hosts, " ")+"\n",
	)
}
