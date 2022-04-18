// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/maruel/subcommands"
	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cros/recovery/tasknames"
	"infra/libs/skylab/buildbucket"
	"infra/libs/skylab/buildbucket/labpack"
	"infra/libs/skylab/worker"
	"infra/libs/swarming"
)

type repairDuts struct {
	subcommands.CommandRunBase
	authFlags      authcli.Flags
	envFlags       site.EnvFlags
	expirationMins int
	onlyVerify     bool
	paris          bool
}

// RepairDutsCmd contains repair-duts command specification
var RepairDutsCmd = &subcommands.Command{
	UsageLine: "repair-duts",
	ShortDesc: "Repair the DUT by name",
	LongDesc: `Repair the DUT by name.
	./shivas repair <dut_name1> ...
	Schedule a swarming Repair task to the DUT to try to recover/verify it.`,
	CommandRun: func() subcommands.CommandRun {
		c := &repairDuts{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.BoolVar(&c.onlyVerify, "verify", false, "Run only verify actions.")
		c.Flags.IntVar(&c.expirationMins, "expiration-mins", 10, "The expiration minutes of the repair request.")
		c.Flags.BoolVar(&c.paris, "paris", false, "Use PARIS rather than legacy flow (dogfood).")
		return c
	},
}

// Run represent runner for reserve command
func (c *repairDuts) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *repairDuts) innerRun(a subcommands.Application, args []string, env subcommands.Env) (err error) {
	if len(args) == 0 {
		return errors.Reason("at least one hostname has to be provided").Err()
	}
	ctx := cli.GetContext(a, c, env)
	e := c.envFlags.Env()
	creator, err := swarming.NewTaskCreator(ctx, &c.authFlags, e.SwarmingService)
	if err != nil {
		return err
	}
	creator.LogdogService = e.LogdogService
	successMap := make(map[string]*swarming.TaskInfo)
	errorMap := make(map[string]error)
	var bc buildbucket.Client
	var sessionTag string
	if c.paris {
		var err error
		fmt.Fprintf(a.GetErr(), "Using PARIS flow for repair\n")
		bc, err = buildbucket.NewClient(ctx, c.authFlags, site.DefaultPRPCOptions, "chromeos", "labpack", "labpack")
		if err != nil {
			return err
		}
		sessionTag = fmt.Sprintf("admin-session:%s", uuid.New().String())
	}
	for _, host := range args {
		creator.GenerateLogdogTaskCode()

		cmd := &worker.Command{TaskName: c.taskName()}
		cmd.LogDogAnnotationURL = creator.LogdogURL()
		var taskInfo *swarming.TaskInfo
		var err error
		if c.paris {
			// Use PARIS.
			fmt.Fprintf(a.GetErr(), "Using PARIS for %q\n", host)
			taskInfo, err = scheduleRepairBuilder(ctx, bc, e, host, !c.onlyVerify, sessionTag)
		} else {
			// Legacy Flow, no PARIS.
			if c.onlyVerify {
				taskInfo, err = creator.VerifyTask(ctx, e.SwarmingServiceAccount, host, c.expirationMins*60, cmd.Args(), cmd.LogDogAnnotationURL)
			} else {
				taskInfo, err = creator.LegacyRepairTask(ctx, e.SwarmingServiceAccount, host, c.expirationMins*60, cmd.Args(), cmd.LogDogAnnotationURL)
			}
		}
		if err != nil {
			errorMap[host] = err
		} else {
			successMap[host] = taskInfo
		}

	}
	if c.paris {
		utils.PrintTasksBatchLink(a.GetOut(), e.SwarmingService, sessionTag)
	} else {
		creator.PrintResults(a.GetOut(), successMap, errorMap)
	}
	return nil
}

func (c *repairDuts) taskName() string {
	if c.onlyVerify {
		return "admin_verify"
	}
	return "admin_repair"
}

// ScheduleRepairBuilder schedules a labpack Buildbucket builder/recipe with the necessary arguments to run repair.
func scheduleRepairBuilder(ctx context.Context, bc buildbucket.Client, e site.Environment, host string, runRepair bool, adminSession string) (*swarming.TaskInfo, error) {
	p := &labpack.Params{
		UnitName:       host,
		TaskName:       string(tasknames.Recovery),
		EnableRecovery: runRepair,
		AdminService:   e.AdminService,
		// NOTE: We use the UFS service, not the Inventory service here.
		InventoryService: e.UnifiedFleetService,
		NoStepper:        false,
		NoMetrics:        false,
		UpdateInventory:  true,
		// TODO(gregorynisbet): Pass config file to labpack task.
		Configuration: "",
		ExtraTags: []string{
			adminSession,
		},
	}
	taskID, err := labpack.ScheduleTask(ctx, bc, labpack.CIPDProd, p)
	if err != nil {
		return nil, err
	}
	taskInfo := &swarming.TaskInfo{
		// Use an ID format that makes it extremely obvious that we're dealing with a
		// buildbucket invocation number rather than a swarming task.
		ID:      fmt.Sprintf("buildbucket:%d", taskID),
		TaskURL: bc.BuildURL(taskID),
	}
	return taskInfo, nil
}
