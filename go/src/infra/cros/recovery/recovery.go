// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package recovery provides ability to run recovery tasks against on the target units.
package recovery

import (
	"context"
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
func Run(ctx context.Context, in *Input) error {
	if err := in.verify(); err != nil {
		return errors.Annotate(err, "run recovery: verify input").Err()
	}
	// Get resources involved.
	resources, err := in.Access.ListResourcesForUnit(ctx, in.UnitName)
	if err != nil {
		return errors.Annotate(err, "run recovery").Err()
	}
	// Keep track of fail to run resources.
	var errs []error
	for ir, resource := range resources {
		log.Printf("Resource %q: started", resource)
		dut, err := in.Access.GetDut(ctx, resource)
		if err != nil {
			return errors.Annotate(err, "run recovery %q", resource).Err()
		}
		log.Printf("Resource %q: received DUT %q info", resource, dut.Name)
		// TODO(otabek@): Generate list of plans based task name and DUT info.
		plans, err := config.LoadPlans([]string{"simple_plan"})
		if err != nil {
			return errors.Annotate(err, "run recovery %q", dut.Name).Err()
		}
		// Creating one run argument for each resource.
		ea := &execs.RunArgs{
			DUT:    dut,
			Access: in.Access,
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

// Input provides input arguments for recovery process.
type Input struct {
	Access tlw.Access
	// UnitName represents some device setup against which running some tests or task in the system.
	// The unit can be represented as a single DUT or group of the DUTs registered in inventory as single unit.
	UnitName string
}

func (in *Input) verify() error {
	if in == nil {
		return errors.Reason("input is empty").Err()
	} else if in.UnitName == "" {
		return errors.Reason("unit name is not provided").Err()
	} else if in.Access == nil {
		return errors.Reason("access point is not provided").Err()
	}
	return nil
}
