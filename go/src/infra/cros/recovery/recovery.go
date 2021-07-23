// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package recovery provides ability to run recovery tasks against on the target units.
package recovery

import (
	"context"
	"encoding/json"
	"io"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/engine"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/loader"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/planpb"
	"infra/cros/recovery/logger"
	"infra/cros/recovery/tlw"
)

// Run runs the recovery tasks against the provide unit.
// Process includes:
//   - Verification of input data.
//   - Set logger.
//   - Collect DUTs info.
//   - Load execution plan for required task with verification.
//   - Send DUTs info to inventory.
func Run(ctx context.Context, args *RunArgs) error {
	if err := args.verify(); err != nil {
		return errors.Annotate(err, "run recovery: verify input").Err()
	}
	if args.Logger != nil {
		ctx = log.WithLogger(ctx, args.Logger)
	}
	if !args.EnableRecovery {
		log.Info(ctx, "Recovery actions is blocker by run arguments.")
	}
	log.Info(ctx, "Run recovery for %q", args.UnitName)
	// Get resources involved.
	resources, err := args.Access.ListResourcesForUnit(ctx, args.UnitName)
	if err != nil {
		return errors.Annotate(err, "run recovery %q", args.UnitName).Err()
	}
	config, err := loader.LoadConfiguration(ctx, DefaultConfig())
	if err != nil {
		return errors.Annotate(err, "run recovery %q", args.UnitName).Err()
	}
	if len(config.GetPlans()) == 0 {
		return errors.Reason("run recovery %q: no plans provided by configuration", args.UnitName).Err()
	}
	// Keep track of fail to run resources.
	var errs []error
	lastResourceIndex := len(resources) - 1
	for ir, resource := range resources {
		log.Info(ctx, "Resource %q: started", resource)
		dut, err := args.Access.GetDut(ctx, resource)
		if err != nil {
			return errors.Annotate(err, "run recovery %q", resource).Err()
		}
		logDUTInfo(ctx, resource, dut, "DUT info from inventory")
		if err := runDUTPlans(ctx, dut, config, args); err != nil {
			errs = append(errs, err)
			log.Debug(ctx, "Resource %q: finished with error: %s.", resource, err)
		} else {
			log.Info(ctx, "Resource %q: finished successfully.", resource)
		}
		logDUTInfo(ctx, resource, dut, "updated DUT info")
		if args.EnableUpdateInventory {
			log.Info(ctx, "Resource %q: starting update DUT in inventory.", resource)
			// Update DUT info in inventory in any case. When fail and when it passed
			if err := args.Access.UpdateDut(ctx, dut); err != nil {
				return errors.Annotate(err, "run recovery %q", resource).Err()
			}
		} else {
			log.Info(ctx, "Resource %q: update inventory is disabled.", resource)
		}
		if ir != lastResourceIndex {
			log.Debug(ctx, "Continue to the next resource.")
		}
	}
	if len(errs) > 0 {
		return errors.Annotate(errors.MultiError(errs), "run recovery").Err()
	}
	return nil
}

func logDUTInfo(ctx context.Context, resource string, dut *tlw.Dut, msg string) {
	s, err := json.MarshalIndent(dut, "", "\t")
	if err != nil {
		log.Debug(ctx, "Resource %q: %s. Fail to print DUT info. Error: %s", resource, msg, err)
	} else {
		log.Debug(ctx, "Resource %q: %s \n%s", resource, msg, s)
	}
}

// runDUTPlans runs DUT's plans.
func runDUTPlans(ctx context.Context, dut *tlw.Dut, config *planpb.Configuration, args *RunArgs) error {
	planNames := dutPlans(dut, args)
	log.Debug(ctx, "Run DUT %q plans: will use %s.", dut.Name, planNames)
	for _, planName := range planNames {
		if _, ok := config.GetPlans()[planName]; !ok {
			return errors.Reason("run dut %q plans: plan %q not found in configuration", dut.Name, planName).Err()
		}
	}
	// Creating one run argument for each resource.
	execArgs := &execs.RunArgs{
		DUT:            dut,
		Access:         args.Access,
		EnableRecovery: args.EnableRecovery,
	}
	// TODO(otabek@): Add closing plan logic.
	lastPlanIndex := len(planNames) - 1
	for ip, planName := range planNames {
		plan := config.GetPlans()[planName]
		if err := engine.Run(ctx, planName, plan, execArgs); err != nil {
			log.Error(ctx, "Run DUT %q plans: plan %q fail. Error: %s", dut.Name, planName, err)
			if plan.GetAllowFail() {
				if ip == lastPlanIndex {
					log.Debug(ctx, "Ignore error as plan %q is allowed to fail.", planName)
				} else {
					log.Debug(ctx, "Continue to next plan as %q is allowed to fail.", planName)
				}
			} else {
				return errors.Annotate(err, "run dut %q plans", planName).Err()
			}
		}
	}
	return nil
}

// TaskName describes which flow/plans will be involved in the process.
type TaskName string

const (
	// Task used to run auto recovery/repair flow in the lab.
	// This task is default task used by the engine.
	TaskNameRecovery TaskName = "recovery"
	// Task used to prepare device to be used in the lab.
	TaskNameDeploy TaskName = "deploy"
)

// RunArgs holds input arguments for recovery process.
type RunArgs struct {
	// Access to the lab TLW layer.
	Access tlw.Access
	// UnitName represents some device setup against which running some tests or task in the system.
	// The unit can be represented as a single DUT or group of the DUTs registered in inventory as single unit.
	UnitName string
	// Provide access to read custom plans outside recovery. The default plans with actions will be ignored.
	ConfigReader io.Reader
	// Logger prints message to the logs.
	Logger logger.Logger
	// TaskName used to drive the recovery process.
	TaskName TaskName
	// EnableRecovery tells if recovery actions are enabled.
	EnableRecovery bool
	// EnableUpdateInventory tells if update inventory after finishing the plans is enabled.
	EnableUpdateInventory bool
}

// verify verifies input arguments.
func (a *RunArgs) verify() error {
	if a == nil {
		return errors.Reason("is empty").Err()
	} else if a.UnitName == "" {
		return errors.Reason("unit name is not provided").Err()
	} else if a.Access == nil {
		return errors.Reason("access point is not provided").Err()
	}
	return nil
}

// List of predefined plans.
// TODO(otabek@): Update list of plans and mapping with final version.
const (
	PlanCrOSRepair       = "cros_repair"
	PlanCrOSDeploy       = "cros_deploy"
	PlanCrOSAudit        = "cros_audit"
	PlanLabstationRepair = "labstation_repair"
	PlanLabstationDeploy = "labstation_deploy"
	PlanServoRepair      = "servo_repair"
)

// dutPlans creates list of plans for DUT.
// TODO(otabek@): Update list of plans and mapping with final version.
// Plans will chosen based on:
// 1) SetupType of DUT.
// 2) Peripherals information.
func dutPlans(dut *tlw.Dut, args *RunArgs) []string {
	// TODO(otabek@): Add logic to run simple action by request.
	// If the task was provide then use recovery as default task.
	var plans []string
	switch dut.SetupType {
	case tlw.DUTSetupTypeLabstation:
		switch args.TaskName {
		case TaskNameDeploy:
			plans = append(plans, PlanLabstationDeploy)
		default:
			plans = append(plans, PlanLabstationRepair)
		}
	default:
		if dut.ServoHost != nil {
			plans = append(plans, PlanServoRepair)
		}
		switch args.TaskName {
		case TaskNameDeploy:
			plans = append(plans, PlanCrOSDeploy)
		default:
			plans = append(plans, PlanCrOSRepair)
		}
	}
	return plans
}
