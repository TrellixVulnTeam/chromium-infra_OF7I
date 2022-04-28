// Copyright 2020 The Chromium Authors. All rights reserved.
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

	"infra/cmd/shivas/site"
	"infra/libs/skylab/worker"
	"infra/libs/swarming"
)

const dayInMinutes = 24 * 60

type auditRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	expirationMins           int
	runVerifyServoUSB        bool
	runVerifyDUTStorage      bool
	runVerifyServoFw         bool
	runFlashServoKeyboardMap bool
	runVerifyDutMacaddr      bool
	runVerifyRpmConfig       bool

	actions string
}

// AuditDutsCmd contains audit-duts command specification
var AuditDutsCmd = &subcommands.Command{
	UsageLine: "audit-duts",
	ShortDesc: "Audit the DUT by name",
	LongDesc: `Audit the DUT by name.
	./shivas audit-duts -action1 -action2 <dut_name1> ...
	Schedule a swarming Audit task with required actions to the DUT to verify it.`,
	CommandRun: func() subcommands.CommandRun {
		c := &auditRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.BoolVar(&c.runVerifyServoUSB, "servo-usb", false, "Run the verifier for Servo USB drive.")
		c.Flags.BoolVar(&c.runVerifyDUTStorage, "dut-storage", false, "Run the verifier for DUT storage.")
		c.Flags.BoolVar(&c.runVerifyServoFw, "servo-fw", false, "Run the verifier for Servo firmware update.")
		c.Flags.BoolVar(&c.runFlashServoKeyboardMap, "servo-keyboard", false, "Run the action to flash Servo keyboard map to the DUT.")
		c.Flags.BoolVar(&c.runVerifyDutMacaddr, "dut-macaddr", false, "Run the verifier to check and cache mac address of DUT NIC to Servo.")
		c.Flags.BoolVar(&c.runVerifyRpmConfig, "rpm-config", false, "Run the verifier to check and cache mac address of DUT NIC to Servo.")
		c.Flags.IntVar(&c.expirationMins, "expiration-mins", 10, "The expiration minutes of the task request.")
		return c
	},
}

// Run represent runner for reserve command
func (c *auditRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *auditRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) (err error) {
	err = c.validateArgs(args)
	if err != nil {
		return errors.Annotate(err, "audit dut").Err()
	}
	ctx := cli.GetContext(a, c, env)
	e := c.envFlags.Env()
	creator, err := swarming.NewTaskCreator(ctx, &c.authFlags, e.SwarmingService)
	if err != nil {
		return errors.Annotate(err, "audit dut").Err()
	}
	creator.LogdogService = e.LogdogService
	successMap := make(map[string]*swarming.TaskInfo)
	errorMap := make(map[string]error)
	for _, host := range args {
		cmd := &worker.Command{
			TaskName: "admin_audit",
			Actions:  c.actions,
		}
		creator.GenerateLogdogTaskCode()
		cmd.LogDogAnnotationURL = creator.LogdogURL()
		task, err := creator.AuditTask(ctx, e.SwarmingServiceAccount, host, c.expirationMins*60, cmd.Args(), cmd.LogDogAnnotationURL)
		if err != nil {
			errorMap[host] = err
		} else {
			successMap[host] = task
		}
	}
	creator.PrintResults(a.GetOut(), successMap, errorMap)
	return nil
}

func (c *auditRun) validateArgs(args []string) (err error) {
	if c.expirationMins >= dayInMinutes {
		return errors.Reason("validate args: expiration minutes (%d minutes) cannot exceed 1 day [%d minutes]", c.expirationMins, dayInMinutes).Err()
	}
	if len(args) == 0 {
		return errors.Reason("validate args: at least one host has to provided").Err()
	}
	c.actions, err = c.collectActions()
	if err != nil {
		return errors.Annotate(err, "validate args").Err()
	}
	return nil
}

// collectActions presents logic to generate actions string to run audit task.
//
// At least one action has to be specified.
func (c *auditRun) collectActions() (string, error) {
	var a []string
	if c.runVerifyDUTStorage {
		a = append(a, "verify-dut-storage")
	}
	if c.runVerifyServoUSB {
		a = append(a, "verify-servo-usb-drive")
	}
	if c.runVerifyServoFw {
		a = append(a, "verify-servo-fw")
	}
	if c.runFlashServoKeyboardMap {
		a = append(a, "flash-servo-keyboard-map")
	}
	if c.runVerifyDutMacaddr {
		a = append(a, "verify-dut-macaddr")
	}
	if c.runVerifyRpmConfig {
		a = append(a, "verify-rpm-config")
	}
	if len(a) == 0 {
		return "", errors.Reason("collect actions: no actions was specified to run").Err()
	}
	return strings.Join(a, ","), nil
}
