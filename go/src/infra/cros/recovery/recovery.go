// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package recovery provides ability to run recovery tasks against on the target units.
package recovery

import (
	"context"
	"io"
	"log"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/config"
	"infra/cros/recovery/internal/plan/execs"
	"infra/cros/recovery/tlw"
)

// Run runs the recovery tasks against the provide unit.
// Process includes:
//   - Verification of input data.
//   - Collect DUTs info.
//   - Load execution plan for required task with verification.
//   - Send DUTs info to inventory.
func Run(ctx context.Context, args *RunArgs) error {
	if err := args.verify(); err != nil {
		return errors.Annotate(err, "run recovery: verify input").Err()
	}
	// Get resources involved.
	resources, err := args.Access.ListResourcesForUnit(ctx, args.UnitName)
	if err != nil {
		return errors.Annotate(err, "run recovery").Err()
	}
	// Keep track of fail to run resources.
	var errs []error
	for ir, resource := range resources {
		log.Printf("Resource %q: started", resource)
		dut, err := args.Access.GetDut(ctx, resource)
		if err != nil {
			return errors.Annotate(err, "run recovery %q", resource).Err()
		}
		log.Printf("Resource %q: received DUT %q info", resource, dut.Name)
		// TODO(otabek@): Generate list of plans based task name and DUT info.
		plans, err := config.LoadPlans(ctx, dutPlans(dut), args.ConfigReader)
		if err != nil {
			return errors.Annotate(err, "run recovery %q", dut.Name).Err()
		}
		// Creating one run argument for each resource.
		ea := &execs.RunArgs{
			DUT:    dut,
			Access: args.Access,
		}
		for ip, p := range plans {
			if err := p.Run(ctx, ea); err != nil {
				log.Printf("Plan %q: fail. Error: %s", p.Name, err)
				if p.AllowFail {
					if ip == len(plans)-1 {
						log.Printf("Ignore error as plan %q is allowed to fail.", p.Name)
					} else {
						log.Printf("Continue to next plan as %q is allowed to fail.", p.Name)
					}
				} else {
					errs = append(errs, err)
					log.Printf("Resource %q: finished with error: %s.", dut.Name, err)
					if ir != len(resources)-1 {
						log.Printf("Continue to the next resource.")
					}
					break
				}
			}
		}
		log.Printf("Resource %q: finished successfully.", dut.Name)
	}
	// TODO(otabek@): Add logic to update DUT's info to inventory.
	if len(errs) > 0 {
		return errors.Annotate(errors.MultiError(errs), "run recovery").Err()
	}
	return nil
}

// RunArgs holds input arguments for recovery process.
type RunArgs struct {
	Access tlw.Access
	// UnitName represents some device setup against which running some tests or task in the system.
	// The unit can be represented as a single DUT or group of the DUTs registered in inventory as single unit.
	UnitName string
	// Provide access to read custom plans outside recovery. The default plans with actions will be ignored.
	ConfigReader io.Reader
}

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
	PlanRepairDUT        = "repair_dut"
	PlanRepairServo      = "repair_servo"
	PlanRepairLabstation = "repair_labstation"
	PlanRepairJetstream  = "repair_jetstream"
)

// dutPlans creates list of plans for DUT.
// TODO(otabek@): Update list of plans and mapping with final version.
// Plans will chosen based on:
// 1) SetupType of DUT.
// 2) Peripherals information.
func dutPlans(dut *tlw.Dut) []string {
	// TODO(otabek@): Add logic to run simple action by request.
	var plans []string
	switch dut.SetupType {
	case tlw.DUTSetupTypeLabstation:
		plans = append(plans, PlanRepairLabstation)
	case tlw.DUTSetupTypeJetstream:
		plans = append(plans, PlanRepairServo, PlanRepairJetstream)
	default:
		if dut.ServoHost != nil {
			plans = append(plans, PlanRepairServo)
		}
		plans = append(plans, PlanRepairDUT)
	}
	return plans
}
