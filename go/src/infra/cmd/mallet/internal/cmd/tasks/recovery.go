// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	b64 "encoding/base64"
	"fmt"
	"os"

	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"
	structbuilder "google.golang.org/protobuf/types/known/structpb"

	"infra/cmd/mallet/internal/bb"
	"infra/cmd/mallet/internal/site"
	"infra/cmdsupport/cmdlib"
	"infra/cros/recovery"
)

// Recovery subcommand: Recovering the devices.
var Recovery = &subcommands.Command{
	UsageLine: "recovery",
	ShortDesc: "Recovery the DUT",
	LongDesc:  "Recovery the DUT.",
	CommandRun: func() subcommands.CommandRun {
		c := &recoveryRun{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.BoolVar(&c.onlyVerify, "only-verify", false, "Block recovery actions and run only verifiers.")
		c.Flags.StringVar(&c.configFile, "config", "", "Path to the custom json config file.")
		c.Flags.BoolVar(&c.noStepper, "no-stepper", false, "Block steper from using. This will prevent by using steps and you can only see logs.")
		c.Flags.BoolVar(&c.deployTask, "deploy", false, "Run deploy task. By default run recovery task.")
		c.Flags.BoolVar(&c.updateUFS, "update-ufs", false, "Update result to UFS. By default no.")
		return c
	},
}

type recoveryRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	onlyVerify bool
	noStepper  bool
	configFile string
	deployTask bool
	updateUFS  bool
}

func (c *recoveryRun) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		cmdlib.PrintError(a, err)
		return 1
	}
	return 0
}

func (c *recoveryRun) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	ctx := cli.GetContext(a, c, env)

	bc, err := bb.NewClient(ctx, c.authFlags, site.DefaultPRPCOptions, site.BBProject, site.MalletBucket, site.MalletBuilder)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.Reason("create recovery task: unit is not specified").Err()
	}
	unit := args[0]
	e := c.envFlags.Env()
	var configuration string
	if c.configFile != "" {
		b, err := os.ReadFile(c.configFile)
		if err != nil {
			return errors.Annotate(err, "create recovery task: open configuration file").Err()
		}
		configuration = b64.StdEncoding.EncodeToString(b)
	}
	task := string(recovery.TaskNameRecovery)
	if c.deployTask {
		task = string(recovery.TaskNameDeploy)
	}
	props, err := structbuilder.NewStruct(map[string]interface{}{
		"unit_name":         unit,
		"task_name":         task,
		"enable_recovery":   !c.onlyVerify,
		"admin_service":     e.AdminService,
		"inventory_service": e.UFSService,
		"update_inventory":  c.updateUFS,
		"no_stepper":        c.noStepper,
		"no_metrics":        true,
		"configuration":     configuration,
	})
	if err != nil {
		return errors.Annotate(err, "create recovery task").Err()
	}

	taskID, err := bc.ScheduleLabpackTask(ctx, unit, props)
	if err != nil {
		return errors.Annotate(err, "create recovery task").Err()
	}
	fmt.Fprintf(a.GetOut(), "Created recovery task for %s: %s\n", unit, bc.BuildURL(taskID))
	return nil
}
