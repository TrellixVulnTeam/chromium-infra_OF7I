// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package recovery provides ability to run recovery tasks against on the target units.
package recovery

import (
	"context"
	"encoding/json"
	"fmt"
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
	if args.Logger == nil {
		args.Logger = logger.NewLogger()
	}
	ctx = log.WithLogger(ctx, args.Logger)
	if !args.EnableRecovery {
		log.Info(ctx, "Recovery actions is blocker by run arguments.")
	}
	log.Info(ctx, "Run recovery for %q", args.UnitName)
	resources, err := retrieveResources(ctx, args)
	if err != nil {
		return errors.Annotate(err, "run recovery %q", args.UnitName).Err()
	}
	// Load Configuration.
	config, err := loadConfiguration(ctx, args)
	if err != nil {
		return errors.Annotate(err, "run recovery %q", args.UnitName).Err()
	}
	// Keep track of fail to run resources.
	var errs []error
	lastResourceIndex := len(resources) - 1
	for ir, resource := range resources {
		log.Info(ctx, "Resource %q: started", resource)
		dut, err := readResource(ctx, resource, args)
		if err != nil {
			return errors.Annotate(err, "run resource %q", resource).Err()
		}
		if err := runDUTPlans(ctx, dut, config, args); err != nil {
			errs = append(errs, err)
		}
		if err := updateInventory(ctx, dut, args); err != nil {
			return errors.Annotate(err, "run resource %q", resource).Err()
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

// retrieveResources retrieves a list of target resources.
func retrieveResources(ctx context.Context, args *RunArgs) (resources []string, err error) {
	if args.StepHandler != nil {
		var step logger.Step
		step, ctx = args.StepHandler.StartStep(ctx, fmt.Sprintf("Retrieve resources for %s", args.UnitName))
		defer step.Close(ctx, err)
	}
	if args.Logger != nil {
		args.Logger.IndentLogging()
		defer args.Logger.DedentLogging()
	}
	resources, err = args.Access.ListResourcesForUnit(ctx, args.UnitName)
	return resources, errors.Annotate(err, "retrieve resources").Err()
}

// loadConfiguration loads and verifies a configuration.
// If configuration is not provided by args then default is used.
func loadConfiguration(ctx context.Context, args *RunArgs) (config *planpb.Configuration, err error) {
	if args.StepHandler != nil {
		var step logger.Step
		step, ctx = args.StepHandler.StartStep(ctx, "Load Configuration")
		defer step.Close(ctx, err)
	}
	if args.Logger != nil {
		args.Logger.IndentLogging()
		defer args.Logger.DedentLogging()
	}
	cr := args.ConfigReader
	if cr == nil {
		// Get default configuration if not provided.
		cr = DefaultConfig()
	}
	if config, err = loader.LoadConfiguration(ctx, cr); err != nil {
		return nil, errors.Annotate(err, "load configuration").Err()
	}
	if len(config.GetPlans()) == 0 {
		return nil, errors.Reason("load configuration: no plans provided by configuration").Err()
	}
	return config, nil
}

// readResource reads single DUT info from inventory.
func readResource(ctx context.Context, resource string, args *RunArgs) (dut *tlw.Dut, err error) {
	if args.StepHandler != nil {
		var step logger.Step
		step, ctx = args.StepHandler.StartStep(ctx, fmt.Sprintf("Read %s from inventory", resource))
		defer step.Close(ctx, err)
	}
	if args.Logger != nil {
		args.Logger.IndentLogging()
		defer args.Logger.DedentLogging()
	}
	dut, err = args.Access.GetDut(ctx, resource)
	if err != nil {
		return nil, errors.Annotate(err, "read resource %q", resource).Err()
	}
	logDUTInfo(ctx, resource, dut, "DUT info from inventory")
	return dut, nil
}

// updateInventory updates updated DUT info back to inventory.
//
// Skip update if not enabled.
func updateInventory(ctx context.Context, dut *tlw.Dut, args *RunArgs) (err error) {
	if args.StepHandler != nil {
		var step logger.Step
		step, ctx = args.StepHandler.StartStep(ctx, fmt.Sprintf("Update %s in inventory", dut.Name))
		defer step.Close(ctx, err)
	}
	if args.Logger != nil {
		args.Logger.IndentLogging()
		defer args.Logger.DedentLogging()
	}
	logDUTInfo(ctx, dut.Name, dut, "updated DUT info")
	if args.EnableUpdateInventory {
		log.Info(ctx, "Update inventory %q: starting...", dut.Name)
		// Update DUT info in inventory in any case. When fail and when it passed
		if err := args.Access.UpdateDut(ctx, dut); err != nil {
			return errors.Annotate(err, "update inventory").Err()
		}
	} else {
		log.Info(ctx, "Update inventory %q: disabled.", dut.Name)
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

// runDUTPlans executes single DUT against task's plans.
func runDUTPlans(ctx context.Context, dut *tlw.Dut, config *planpb.Configuration, args *RunArgs) (err error) {
	if args.StepHandler != nil {
		var step logger.Step
		step, ctx = args.StepHandler.StartStep(ctx, fmt.Sprintf("Run DUT %q", dut.Name))
		defer step.Close(ctx, err)
	}
	if args.Logger != nil {
		args.Logger.IndentLogging()
		defer args.Logger.DedentLogging()
	}
	log.Info(ctx, "Run DUT %q: starting...", dut.Name)
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
		Logger:         args.Logger,
		StepHandler:    args.StepHandler,
	}
	// TODO(otabek@): Add closing plan logic.
	for _, planName := range planNames {
		if err = runDUTPlan(ctx, planName, dut, config, execArgs); err != nil {
			log.Debug(ctx, "Run DUT %q plans: finished with error: %s.", dut.Name, err)
		}
	}
	log.Info(ctx, "Run DUT %q plans: finished successfully.", dut.Name)
	return nil
}

// runDUTPlan runs simple plan against the DUT.
func runDUTPlan(ctx context.Context, planName string, dut *tlw.Dut, config *planpb.Configuration, execArgs *execs.RunArgs) (err error) {
	if execArgs.StepHandler != nil {
		var step logger.Step
		step, ctx = execArgs.StepHandler.StartStep(ctx, fmt.Sprintf("Run plan %q", planName))
		defer step.Close(ctx, err)
	}
	if execArgs.Logger != nil {
		execArgs.Logger.IndentLogging()
		defer execArgs.Logger.DedentLogging()
	}
	resources := collectResourcesForPlan(planName, dut)
	if len(resources) == 0 {
		log.Info(ctx, "Run plan %q: no resources found.", planName)
	}
	plan := config.GetPlans()[planName]
	for _, resource := range resources {
		execArgs.ResourceName = resource
		log.Info(ctx, "Run plan %q for %q: started", planName, resource)
		if err := engine.Run(ctx, planName, plan, execArgs); err != nil {
			log.Error(ctx, "Run plan %q for %q: fail. Error: %s", planName, resource, err)
			if plan.GetAllowFail() {
				log.Debug(ctx, "Run plan %q for %q: ignore error as allowed to fail.", planName, resource)
			} else {
				return errors.Annotate(err, "run plan %q for %q", planName, resource).Err()
			}
		}
	}
	return nil
}

// collectResourcesForPlan collect resource names for supported plan.
// Mostly we have one resource per plan by in some cases we can have more
// resources and then we will run the same plan for each resource.
func collectResourcesForPlan(planName string, dut *tlw.Dut) []string {
	switch planName {
	case PlanCrOSRepair, PlanCrOSDeploy, PlanLabstationRepair, PlanLabstationDeploy:
		if dut.Name != "" {
			return []string{dut.Name}
		}
	case PlanServoRepair:
		if dut.ServoHost != nil && dut.ServoHost.Name != "" {
			return []string{dut.ServoHost.Name}
		}
	case PlanBluetoothPeerRepair:
		var resources []string
		for _, bp := range dut.BluetoothPeerHosts {
			if bp.Name != "" {
				resources = append(resources, bp.Name)
			}
		}
		return resources
	case PlanChameleonRepair:
		if dut.ChameleonHost != nil && dut.ChameleonHost.Name != "" {
			return []string{dut.ChameleonHost.Name}
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
	// StepHandler provides option to report steps.
	StepHandler logger.StepHandler
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
	PlanCrOSRepair          = "cros_repair"
	PlanCrOSDeploy          = "cros_deploy"
	PlanLabstationRepair    = "labstation_repair"
	PlanLabstationDeploy    = "labstation_deploy"
	PlanServoRepair         = "servo_repair"
	PlanChameleonRepair     = "chameleon_repair"
	PlanBluetoothPeerRepair = "bluetooth_peer_repair"
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
		if dut.ServoHost != nil && dut.ServoHost.Name != "" {
			plans = append(plans, PlanServoRepair)
		}
		switch args.TaskName {
		case TaskNameDeploy:
			plans = append(plans, PlanCrOSDeploy)
		default:
			if dut.ChameleonHost != nil && dut.ChameleonHost.Name != "" {
				plans = append(plans, PlanChameleonRepair)
			}
			if len(dut.BluetoothPeerHosts) > 0 {
				plans = append(plans, PlanBluetoothPeerRepair)
			}
			plans = append(plans, PlanCrOSRepair)
		}
	}
	return plans
}
