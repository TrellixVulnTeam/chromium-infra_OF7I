// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	b64 "encoding/base64"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/mallet/internal/site"
	"infra/cmdsupport/cmdlib"
	"infra/cros/recovery/tasknames"
	"infra/libs/skylab/buildbucket"
	"infra/libs/skylab/buildbucket/labpack"
	"infra/libs/skylab/swarming"
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
		c.Flags.BoolVar(&c.latest, "latest", false, "Use latest version of CIPD when scheduling. By default no.")
		c.Flags.StringVar(&c.adminSession, "admin-session", "", "Admin session used to group created tasks. By default generated.")
		return c
	},
}

// Client tag used for PARIS tasks scheduling.
const clientTag = "client:mallet"

type recoveryRun struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	onlyVerify   bool
	noStepper    bool
	configFile   string
	deployTask   bool
	updateUFS    bool
	latest       bool
	adminSession string
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
	hc, err := buildbucket.NewHTTPClient(ctx, &c.authFlags)
	if err != nil {
		return errors.Annotate(err, "recovery run").Err()
	}
	bc, err := buildbucket.NewClient2(ctx, hc, site.DefaultPRPCOptions, site.BBProject, site.MalletBucket, site.MalletBuilder)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.Reason("create recovery task: unit is not specified").Err()
	}
	// Admin session used to created common tag across created tasks.
	if c.adminSession == "" {
		c.adminSession = uuid.New().String()
	}
	sessionTag := fmt.Sprintf("admin-session:%s", c.adminSession)
	e := c.envFlags.Env()
	for _, unit := range args {
		var configuration string
		if c.configFile != "" {
			b, err := os.ReadFile(c.configFile)
			if err != nil {
				return errors.Annotate(err, "create recovery task: open configuration file").Err()
			}
			configuration = b64.StdEncoding.EncodeToString(b)
		}
		task := string(tasknames.Recovery)
		if c.deployTask {
			task = string(tasknames.Deploy)
		}

		v := labpack.CIPDProd
		if c.latest {
			v = labpack.CIPDLatest
		}
		taskID, err := labpack.ScheduleTask(
			ctx,
			bc,
			v,
			&labpack.Params{
				UnitName:         unit,
				TaskName:         task,
				EnableRecovery:   !c.onlyVerify,
				AdminService:     e.AdminService,
				InventoryService: e.UFSService,
				UpdateInventory:  c.updateUFS,
				NoStepper:        c.noStepper,
				NoMetrics:        true,
				Configuration:    configuration,
				ExtraTags: []string{
					sessionTag,
					fmt.Sprintf("task:%s", task),
					clientTag,
					fmt.Sprintf("version:%s", v),
				},
			},
		)
		if err != nil {
			return errors.Annotate(err, "create recovery task").Err()
		}
		fmt.Fprintf(a.GetOut(), "Created recovery task for %s: %s\n", unit, bc.BuildURL(taskID))
	}
	fmt.Fprintf(a.GetOut(), "Created tasks: %s\n", swarming.TaskListURLForTags(e.SwarmingService, []string{sessionTag}))
	return nil
}
