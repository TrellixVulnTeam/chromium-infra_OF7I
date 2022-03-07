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
	"os"
	"time"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/luciexe/build"

	"infra/cros/recovery/internal/engine"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/loader"
	"infra/cros/recovery/internal/localtlw/localproxy"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/planpb"
	"infra/cros/recovery/logger"
	"infra/cros/recovery/logger/metrics"
	"infra/cros/recovery/tasknames"
	"infra/cros/recovery/tlw"
)

// Run runs the recovery tasks against the provided unit.
// Process includes:
//   - Verification of input data.
//   - Set logger.
//   - Collect DUTs info.
//   - Load execution plan for required task with verification.
//   - Send DUTs info to inventory.
func Run(ctx context.Context, args *RunArgs) (rErr error) {
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
	if args.ShowSteps {
		var step *build.Step
		step, ctx = build.StartStep(ctx, fmt.Sprintf("Start %s", args.TaskName))
		defer func() { step.End(err) }()
	}
	if args.Metrics == nil {
		log.Debug(ctx, "run: metrics is nil")
	} else {
		log.Debug(ctx, "run: metrics is non-nil")
		start := time.Now()
		// TODO(gregorynisbet): Create a helper function to make this more compact.
		defer (func() {
			stop := time.Now()
			status := metrics.ActionStatusUnspecified
			failReason := ""
			if rErr == nil {
				status = metrics.ActionStatusFail
			} else {
				status = metrics.ActionStatusSuccess
				failReason = rErr.Error()
			}
			// Keep this call up to date with NewMetric in execs.go.
			if args.Metrics != nil { // Guard against incorrectly setting up Karte client. See b:217746479 for details.
				_, mErr := args.Metrics.Create(
					ctx,
					&metrics.Action{
						ActionKind:     "run_recovery",
						StartTime:      start,
						StopTime:       stop,
						SwarmingTaskID: args.SwarmingTaskID,
						BuildbucketID:  args.BuildbucketID,
						Hostname:       args.UnitName,
						// TODO(gregorynisbet): add status and FailReason.
						Status:     status,
						FailReason: failReason,
					},
				)
				if mErr != nil {
					args.Logger.Error("Metrics error during teardown: %s", err)
				}
			}
		})()
	}
	// Close all created local proxies.
	defer func() {
		localproxy.ClosePool()
	}()
	// Keep track of failure to run resources.
	// If one resource fail we still will try to run another one.
	var errs []error
	for ir, resource := range resources {
		if ir != 0 {
			log.Debug(ctx, "Continue to the next resource.")
		}
		if err := runResource(ctx, resource, args); err != nil {
			errs = append(errs, errors.Annotate(err, "run recovery %q", resource).Err())
		}
	}
	if len(errs) > 0 {
		return errors.Annotate(errors.MultiError(errs), "run recovery").Err()
	}
	return nil
}

// runResource run single resource.
func runResource(ctx context.Context, resource string, args *RunArgs) (rErr error) {
	log.Info(ctx, "Resource %q: started", resource)
	if args.ShowSteps {
		var step *build.Step
		step, ctx = build.StartStep(ctx, fmt.Sprintf("Resource %q", resource))
		defer func() { step.End(rErr) }()
	}
	dut, err := readInventory(ctx, resource, args)
	if err != nil {
		return errors.Annotate(err, "run resource %q", resource).Err()
	}
	// Load Configuration.
	config, err := loadConfiguration(ctx, dut, args)
	if err != nil {
		return errors.Annotate(err, "run resource %q", args.UnitName).Err()
	}
	if err := runDUTPlans(ctx, dut, config, args); err != nil {
		return errors.Annotate(err, "run resource %q", resource).Err()
	}
	if err := updateInventory(ctx, dut, args); err != nil {
		return errors.Annotate(err, "run resource %q", resource).Err()
	}
	return nil
}

// retrieveResources retrieves a list of target resources.
func retrieveResources(ctx context.Context, args *RunArgs) (resources []string, err error) {
	if args.ShowSteps {
		var step *build.Step
		step, ctx = build.StartStep(ctx, fmt.Sprintf("Retrieve resources for %s", args.UnitName))
		defer func() { step.End(err) }()
	}
	if args.Logger != nil {
		args.Logger.IndentLogging()
		defer func() { args.Logger.DedentLogging() }()
	}
	resources, err = args.Access.ListResourcesForUnit(ctx, args.UnitName)
	return resources, errors.Annotate(err, "retrieve resources").Err()
}

// loadConfiguration loads and verifies a configuration.
// If configuration is not provided by args then default is used.
func loadConfiguration(ctx context.Context, dut *tlw.Dut, args *RunArgs) (config *planpb.Configuration, err error) {
	if args.ShowSteps {
		var step *build.Step
		step, ctx = build.StartStep(ctx, "Load configuration")
		defer func() { step.End(err) }()
	}
	if args.Logger != nil {
		args.Logger.IndentLogging()
		defer func() { args.Logger.DedentLogging() }()
	}
	cr := args.ConfigReader
	if cr == nil {
		if args.TaskName == tasknames.Custom {
			return nil, errors.Reason("load configuration: expected config to be provided for custom tasks").Err()
		}
		// Get default configuration if not provided.
		cr, err = defaultConfiguration(args.TaskName, dut.SetupType)
		if err != nil {
			return nil, errors.Annotate(err, "load configuration").Err()
		}
	}
	if c, err := parseConfiguration(ctx, cr); err != nil {
		return nil, errors.Annotate(err, "load configuration").Err()
	} else {
		return c, nil
	}
}

// ParsedDefaultConfiguration returns parsed default configuration for requested task and setup.
func ParsedDefaultConfiguration(ctx context.Context, tn tasknames.TaskName, ds tlw.DUTSetupType) (*planpb.Configuration, error) {
	if cr, err := defaultConfiguration(tn, ds); err != nil {
		return nil, errors.Annotate(err, "load configuration").Err()
	} else if c, err := parseConfiguration(ctx, cr); err != nil {
		return nil, errors.Annotate(err, "load configuration").Err()
	} else {
		return c, nil
	}
}

// parseConfiguration parses configuration to configuration proto instance.
func parseConfiguration(ctx context.Context, cr io.Reader) (config *planpb.Configuration, err error) {
	if c, err := loader.LoadConfiguration(ctx, cr); err != nil {
		return c, errors.Annotate(err, "parse configuration").Err()
	} else if len(c.GetPlans()) == 0 {
		return nil, errors.Reason("load configuration: no plans provided by configuration").Err()
	} else {
		return c, nil
	}
}

// defaultConfiguration provides configuration based on type of setup and task name.
func defaultConfiguration(tn tasknames.TaskName, ds tlw.DUTSetupType) (io.Reader, error) {
	switch tn {
	case tasknames.Recovery:
		switch ds {
		case tlw.DUTSetupTypeCros:
			return CrosRepairConfig(), nil
		case tlw.DUTSetupTypeLabstation:
			return LabstationRepairConfig(), nil
		default:
			return nil, errors.Reason("Setup type: %q is not supported for task: %q!", ds, tn).Err()
		}
	case tasknames.Deploy:
		switch ds {
		case tlw.DUTSetupTypeCros:
			return CrosDeployConfig(), nil
		case tlw.DUTSetupTypeLabstation:
			return LabstationDeployConfig(), nil
		default:
			return nil, errors.Reason("Setup type: %q is not supported for task: %q!", ds, tn).Err()
		}
	case tasknames.Custom:
		return nil, errors.Reason("Setup type: %q does not have default configuration for custom tasks", ds).Err()
	default:
		return nil, errors.Reason("TaskName: %q is not supported..", tn).Err()
	}
}

// readInventory reads single resource info from inventory.
func readInventory(ctx context.Context, resource string, args *RunArgs) (dut *tlw.Dut, err error) {
	if args.ShowSteps {
		step, _ := build.StartStep(ctx, "Read inventory")
		defer func() { step.End(err) }()
	}
	if args.Logger != nil {
		args.Logger.IndentLogging()
		defer func() { args.Logger.DedentLogging() }()
	}
	defer func() {
		if r := recover(); r != nil {
			log.Debug(ctx, "Read resource received panic!")
			err = errors.Reason("read resource panic: %v", r).Err()
		}
	}()
	dut, err = args.Access.GetDut(ctx, resource)
	if err != nil {
		return nil, errors.Annotate(err, "read inventory %q", resource).Err()
	}
	logDUTInfo(ctx, resource, dut, "DUT info from inventory")
	return dut, nil
}

// updateInventory updates updated DUT info back to inventory.
//
// Skip update if not enabled.
func updateInventory(ctx context.Context, dut *tlw.Dut, args *RunArgs) (rErr error) {
	if args.ShowSteps {
		step, _ := build.StartStep(ctx, "Update inventory")
		defer func() { step.End(rErr) }()
	}
	if args.Logger != nil {
		args.Logger.IndentLogging()
		defer func() { args.Logger.DedentLogging() }()
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
func runDUTPlans(ctx context.Context, dut *tlw.Dut, config *planpb.Configuration, args *RunArgs) error {
	if args.Logger != nil {
		args.Logger.IndentLogging()
		defer args.Logger.DedentLogging()
	}
	log.Info(ctx, "Run DUT %q: starting...", dut.Name)
	planNames := config.GetPlanNames()
	log.Debug(ctx, "Run DUT %q plans: will use %s.", dut.Name, planNames)
	hasClosingPlan := false
	for _, planName := range planNames {
		if planName == PlanClosing {
			// The Closing plan will be added by default and it i sok if it missed.
			hasClosingPlan = true
		}
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
		ShowSteps:      args.ShowSteps,
		Metrics:        args.Metrics,
		SwarmingTaskID: args.SwarmingTaskID,
		BuildbucketID:  args.BuildbucketID,
		LogRoot:        args.LogRoot,
		// JumpHost: -- We explicitly do NOT pass the jump host to execs directly.
	}
	// As port 22 to connect to the lab is closed and there is work around to
	// create proxy for local execution. Creating proxy for all resources used
	// for this devices. We need created all of them at the beginning as one
	// plan can have access to current resource or another one.
	// Always has to be empty for merge code
	if jumpHostForLocalProxy := args.DevJumpHost; jumpHostForLocalProxy != "" {
		for _, planName := range planNames {
			resources := collectResourcesForPlan(planName, execArgs.DUT)
			for _, resource := range resources {
				if sh := execArgs.DUT.ServoHost; sh != nil && sh.Name == resource && sh.ContainerName != "" {
					continue
				}
				if err := localproxy.RegHost(ctx, resource, jumpHostForLocalProxy); err != nil {
					return errors.Annotate(err, "run plans: create proxy for %q", resource).Err()
				}
			}
		}
	}
	defer func() {
		// If closing plan provided by configuration then we do not need run it here.
		if !hasClosingPlan {
			plan, ok := config.GetPlans()[PlanClosing]
			if !ok {
				log.Info(ctx, "Run plans: plan %q not found in configuration.", PlanClosing)
			} else {
				// Closing plan always allowed to fail.
				plan.AllowFail = true
				if err := runSinglePlan(ctx, PlanClosing, plan, execArgs); err != nil {
					log.Debug(ctx, "Run plans: plan %q for %q finished with error: %s", PlanClosing, dut.Name, err)
				} else {
					log.Debug(ctx, "Run plans: plan %q for %q finished successfully", PlanClosing, dut.Name)
				}
			}
		}
	}()
	for _, planName := range planNames {
		plan, ok := config.GetPlans()[planName]
		if !ok {
			return errors.Reason("run plans: plan %q: not found in configuration", planName).Err()
		}
		if err := runSinglePlan(ctx, planName, plan, execArgs); err != nil {
			return errors.Annotate(err, "run plans").Err()
		}
	}
	log.Info(ctx, "Run DUT %q plans: finished successfully.", dut.Name)
	return nil
}

// runSinglePlan run single plan for all resources associated with plan.
func runSinglePlan(ctx context.Context, planName string, plan *planpb.Plan, execArgs *execs.RunArgs) error {
	log.Info(ctx, "Run plan %q: starting...", planName)
	resources := collectResourcesForPlan(planName, execArgs.DUT)
	if len(resources) == 0 {
		log.Info(ctx, "Run plan %q: no resources found.", planName)
		return nil
	}
	for _, resource := range resources {
		if err := runDUTPlanPerResource(ctx, resource, planName, plan, execArgs); err != nil {
			log.Info(ctx, "Run %q plan for %s: finished with error: %s.", planName, resource, err)
			if plan.GetAllowFail() {
				log.Debug(ctx, "Run plan %q for %q: ignore error as allowed to fail.", planName, resource)
			} else {
				return errors.Annotate(err, "run plan %q", planName).Err()
			}
		}
	}
	return nil
}

// runDUTPlanPerResource runs a plan against the single resource of the DUT.
func runDUTPlanPerResource(ctx context.Context, resource, planName string, plan *planpb.Plan, execArgs *execs.RunArgs) (rErr error) {
	log.Info(ctx, "Run plan %q for %q: started", planName, resource)
	if execArgs.ShowSteps {
		var step *build.Step
		step, ctx = build.StartStep(ctx, fmt.Sprintf("Run plan %q for %q", planName, resource))
		defer func() { step.End(rErr) }()
	}
	if execArgs.Logger != nil {
		execArgs.Logger.IndentLogging()
		defer func() { execArgs.Logger.DedentLogging() }()
	}
	execArgs.ResourceName = resource
	if err := engine.Run(ctx, planName, plan, execArgs); err != nil {
		return errors.Annotate(err, "run plan %q for %q", planName, execArgs.ResourceName).Err()
	}
	log.Info(ctx, "Run plan %q for %s: finished successfully.", planName, execArgs.ResourceName)
	return nil
}

// collectResourcesForPlan collect resource names for supported plan.
// Mostly we have one resource per plan but in some cases we can have more
// resources and then we will run the same plan for each resource.
func collectResourcesForPlan(planName string, dut *tlw.Dut) []string {
	switch planName {
	case PlanCrOS, PlanClosing:
		if dut.Name != "" {
			return []string{dut.Name}
		}
	case PlanServo:
		if dut.ServoHost != nil {
			return []string{dut.ServoHost.Name}
		}
	case PlanBluetoothPeer:
		var resources []string
		for _, bp := range dut.BluetoothPeerHosts {
			resources = append(resources, bp.Name)
		}
		return resources
	case PlanChameleon:
		if dut.ChameleonHost != nil {
			return []string{dut.ChameleonHost.Name}
		}
	case PlanWifiRouter:
		var resources []string
		for _, router := range dut.WifiRouterHosts {
			resources = append(resources, router.GetName())
		}
		return resources
	}
	return nil
}

// RunArgs holds input arguments for recovery process.
//
// Keep this type up to date with internal/execs/execs.go:RunArgs .
// Also update recovery.go:runDUTPlans .
//
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
	// Option to use steps.
	ShowSteps bool
	// Metrics is the metrics sink and event search API.
	Metrics metrics.Metrics
	// TaskName used to drive the recovery process.
	TaskName tasknames.TaskName
	// EnableRecovery tells if recovery actions are enabled.
	EnableRecovery bool
	// EnableUpdateInventory tells if update inventory after finishing the plans is enabled.
	EnableUpdateInventory bool
	// SwarmingTaskID is the ID of the swarming task.
	SwarmingTaskID string
	// BuildbucketID is the ID of the buildbucket build
	BuildbucketID string
	// LogRoot is an absolute path to a directory.
	// All logs produced by actions or verifiers must be deposited there.
	LogRoot string
	// JumpHost is the host to use as a SSH proxy between ones dev environment and the lab,
	// if necessary. An empty JumpHost means do not use a jump host.
	DevJumpHost string
}

// verify verifies input arguments.
func (a *RunArgs) verify() error {
	switch {
	case a == nil:
		return errors.Reason("is empty").Err()
	case a.UnitName == "":
		return errors.Reason("unit name is not provided").Err()
	case a.Access == nil:
		return errors.Reason("access point is not provided").Err()
	case a.LogRoot == "":
		// TODO(gregorynisbet): Upgrade this to a real error.
		fmt.Fprintf(os.Stderr, "%s\n", "log root cannot be empty!\n")
	}
	fmt.Fprintf(os.Stderr, "log root is %q\n", a.LogRoot)
	return nil
}

// List of predefined plans.
const (
	PlanCrOS          = "cros"
	PlanServo         = "servo"
	PlanChameleon     = "chameleon"
	PlanBluetoothPeer = "bluetooth_peer"
	PlanWifiRouter    = "wifi_router"
	// That is final plan which will run always if present in configuration.
	// The goal is execution final step to clean up stages if something left
	// over in the devices.
	PlanClosing = "close"
)
