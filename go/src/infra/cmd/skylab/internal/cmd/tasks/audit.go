// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"fmt"
	"strings"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/cmd/utils"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
)

// Audit subcommand: Audit hosts.
var Audit = &subcommands.Command{
	UsageLine: "audit [HOST...]",
	ShortDesc: "create audit tasks",
	LongDesc: `Create audit tasks.

This command does not wait for the tasks to start running.
By default all actions runnings. To run specified action provide it via flags.`,
	CommandRun: func() subcommands.CommandRun {
		c := &auditRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.expirationMins, "expiration-mins", 10, "The expiration minutes of the repair request.")
		c.Flags.BoolVar(&c.runVerifyServoUSB, "run-verify-servo-usb", false, "Run the verifier for servo usb drive.")
		c.Flags.BoolVar(&c.runVerifyDUTStorage, "run-verify-dut-storage", false, "Run the verifier for DUT storage.")
		c.Flags.BoolVar(&c.runVerifyServoFw, "run-verify-servo-fw", false, "Run the verifier for servo firmware update.")
		return c
	},
}

type auditRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  skycmdlib.EnvFlags

	expirationMins      int
	runVerifyServoUSB   bool
	runVerifyDUTStorage bool
	runVerifyServoFw    bool
}

func (c *auditRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *auditRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if err := c.validateArgs(args); err != nil {
		return err
	}
	actions, err := c.actions()
	if err != nil {
		return err
	}

	ctx := cli.GetContext(a, c, env)
	creator, err := utils.NewTaskCreator(ctx, &c.authFlags, c.envFlags)
	if err != nil {
		return err
	}

	successMap := make(map[string]utils.TaskInfo)
	errorMap := make(map[string]error)
	for _, host := range args {
		dutName := skycmdlib.FixSuspiciousHostname(host)
		if dutName != host {
			fmt.Fprintf(a.GetErr(), "correcting (%s) to (%s)\n", host, dutName)
		}
		task, err := creator.AuditTask(ctx, dutName, actions, c.expirationMins*60)
		if err != nil {
			errorMap[dutName] = err
		} else {
			successMap[dutName] = task
		}
	}
	if len(errorMap) > 0 {
		fmt.Fprintln(a.GetOut(), "\n### Failed to create ###")
		for host, err := range errorMap {
			fmt.Fprintf(a.GetOut(), "%s: %s\n", host, err.Error())
		}
	}
	if len(successMap) > 0 {
		fmt.Fprintf(a.GetOut(), "\n### Successful created - %d ###\n", len(successMap))
		for host, task := range successMap {
			fmt.Fprintf(a.GetOut(), "Created Swarming task %s for host %s\n", task.TaskURL, host)
		}
		if len(successMap) > 1 {
			fmt.Fprintln(a.GetOut(), "\n### Batch tasks URL ###")
			fmt.Fprintln(a.GetOut(), creator.SessionTasksURL())
		}
	}
	return nil
}

func (c *auditRun) validateArgs(args []string) error {
	if c.expirationMins >= dayInMinutes {
		return cmdlib.NewUsageError(c.Flags, "Expiration minutes (%d minutes) cannot exceed 1 day [%d minutes]", c.expirationMins, dayInMinutes)
	}
	if len(args) == 0 {
		return errors.Reason("at least one host has to provided").Err()
	}
	return nil
}

// Collect actions to run. If we do not by specified action or all of them if no action specified.
func (c *auditRun) actions() (string, error) {
	var a []string
	if !(c.runVerifyDUTStorage || c.runVerifyServoUSB || c.runVerifyServoFw) {
		c.runVerifyDUTStorage = true
		c.runVerifyServoUSB = true
		c.runVerifyServoFw = true
	}
	if c.runVerifyDUTStorage {
		a = append(a, "verify-dut-storage")
	}
	if c.runVerifyServoUSB {
		a = append(a, "verify-servo-usb-drive")
	}
	if c.runVerifyServoFw {
		a = append(a, "verify-servo-fw")
	}
	if len(a) == 0 {
		return "", errors.Reason("No actions to run").Err()
	}
	return strings.Join(a, ","), nil
}
