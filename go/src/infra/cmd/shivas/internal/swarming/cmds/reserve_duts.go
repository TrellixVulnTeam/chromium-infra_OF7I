// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/user"

	"github.com/google/uuid"
	"github.com/maruel/subcommands"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/common/cli"
	"go.chromium.org/luci/common/errors"

	"infra/cmd/shivas/site"
	"infra/cmd/shivas/utils"
	"infra/cros/recovery/config"
	"infra/cros/recovery/tasknames"
	"infra/libs/skylab/buildbucket"
	"infra/libs/skylab/buildbucket/labpack"
	"infra/libs/swarming"
)

type reserveDuts struct {
	subcommands.CommandRunBase
	authFlags authcli.Flags
	envFlags  site.EnvFlags

	legacy         bool
	expirationMins int
}

// ReserveDutsCmd contains reserve-dut command specification
var ReserveDutsCmd = &subcommands.Command{
	UsageLine: "reserve-duts",
	ShortDesc: "Reserve the DUT by name",
	LongDesc: `Reserve the DUT by name.
	./shivas reserve <dut_name>
	Schedule a swarming Reserve task to the DUT to set the state to RESERVED to prevent scheduling tasks and tests to the DUT.
	Reserved DUT does not have expiration time and can be changed by scheduling any admin task on it.`,
	CommandRun: func() subcommands.CommandRun {
		c := &reserveDuts{}
		c.authFlags.Register(&c.Flags, site.DefaultAuthOptions)
		c.envFlags.Register(&c.Flags)
		c.Flags.IntVar(&c.expirationMins, "expiration-mins", 120, "The expiration minutes of the repair request.")
		c.Flags.BoolVar(&c.legacy, "legacy", false, "Use legacy rather than paris flow.")
		return c
	},
}

// Run represent runner for reserve command
func (c *reserveDuts) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.innerRun(a, args, env); err != nil {
		fmt.Fprintf(a.GetErr(), "%s: %s\n", a.GetName(), err)
		return 1
	}
	return 0
}

func (c *reserveDuts) innerRun(a subcommands.Application, args []string, env subcommands.Env) error {
	if len(args) == 0 {
		return errors.Reason("at least one hostname has to be provided").Err()
	}
	user, err := user.Current()
	if err != nil {
		return err
	}
	ctx := cli.GetContext(a, c, env)
	e := c.envFlags.Env()
	creator, err := swarming.NewTaskCreator(ctx, &c.authFlags, e.SwarmingService)
	if err != nil {
		return err
	}

	successMap := make(map[string]*swarming.TaskInfo)
	errorMap := make(map[string]error)
	var bc buildbucket.Client
	var sessionTag string
	if !c.legacy {
		var err error
		fmt.Fprintf(a.GetErr(), "Using PARIS flow for repair\n")
		bc, err = buildbucket.NewClient(ctx, c.authFlags, site.DefaultPRPCOptions, "chromeos", "labpack", "labpack")
		if err != nil {
			return err
		}
		sessionTag = fmt.Sprintf("admin-session:%s", uuid.New().String())
	}
	for _, host := range args {
		// TODO(crbug/1128496): update state directly in the UFS without creating the swarming task
		var task *swarming.TaskInfo
		if c.legacy {
			// Use legacy
			task, err = creator.ReserveDUT(ctx, e.SwarmingServiceAccount, host, user.Username, c.expirationMins*60)
		} else {
			// Use PARIS.
			fmt.Fprintf(a.GetErr(), "Using PARIS for %q\n", host)
			task, err = scheduleReserveBuilder(ctx, bc, e, host, sessionTag)
		}
		if err != nil {
			errorMap[host] = err
		} else {
			successMap[host] = task
		}
	}
	if c.legacy {
		creator.PrintResults(a.GetOut(), successMap, errorMap)
	} else {
		utils.PrintTasksBatchLink(a.GetOut(), e.SwarmingService, sessionTag)
	}
	return nil
}

// ScheduleReserveBuilder schedules a labpack Buildbucket builder/recipe with the necessary arguments to run reserve.
func scheduleReserveBuilder(ctx context.Context, bc buildbucket.Client, e site.Environment, host string, adminSession string) (*swarming.TaskInfo, error) {
	rc := config.ReserveDutConfig()
	jsonByte, err := json.Marshal(rc)
	if err != nil {
		return nil, errors.Annotate(err, "scheduleReserveBuilder json err:").Err()
	}
	config := base64.StdEncoding.EncodeToString(jsonByte)
	// TODO(b/229896419): refactor to hide labpack.Params struct.
	p := &labpack.Params{
		UnitName:     host,
		TaskName:     string(tasknames.Custom),
		AdminService: e.AdminService,
		// NOTE: We use the UFS service, not the Inventory service here.
		InventoryService: e.UnifiedFleetService,
		NoStepper:        false,
		NoMetrics:        false,
		UpdateInventory:  true,
		Configuration:    config,
		ExtraTags: []string{
			adminSession,
			"task:reserve",
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
