// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package plan

import (
	"context"
	"fmt"
	"log"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/plan/execs"
)

// Action represents a recovery action, which can perform either a verification action or a repair action.
type Action struct {
	// Unique human readable short name of the action.
	// Please use names in lowercase and only alphanumeric symbols with underscore.
	// Names must be unique within a plan.
	// Example: dut_ping, dut_ssh, rpm_cycle, servod_echo.
	Name string
	// Name of the Exec function to use.
	// The name of action will be used if not provided.
	ExecName string
	// List of actions has to pass to allow start execution of this action.
	Dependencies []*Action
	// List of actions used to recover this action if it is fail.
	Recoveries []*Action
	// If set true then the action is allowed to fail without affecting the plan result.
	// If the action has recovery actions they will be tried and if action still fail it will be skipped.
	AllowFail bool
	// If set true then the action result will be cached and used for the next attempt.
	// If set false then action will run every time.
	AllowCache bool
}

// runAction represents recursive executable function to run single action with dependencies and recoveries.
func (a *Action) run(ctx context.Context, c *runCache, args *execs.RunArgs) error {
	if err := a.runDependencies(ctx, c, args); err != nil {
		c.cacheAction(a, err)
		return errors.Annotate(err, "run action %q", a.Name).Err()
	}
	if err := execs.Run(ctx, a.ExecName, args); err != nil {
		// TODO(otabek@): Add logic to try to recover the failure.
		c.cacheAction(a, err)
		return errors.Annotate(err, "run action %q", a.Name).Err()
	}
	c.cacheAction(a, nil)
	return nil
}

// runDependencies runs dependencies of the action.
// Method the first check the result of action from cache and if not exist then perform action.
func (a *Action) runDependencies(ctx context.Context, c *runCache, args *execs.RunArgs) error {
	for i, dep := range a.Dependencies {
		log.Printf("Run dependency %q: started.", dep.Name)
		if r, ok := c.getActionError(dep); ok {
			if r == nil {
				log.Printf("Dependency %q: pass (cached).", dep.Name)
				continue
			} else if dep.AllowFail {
				log.Printf("Dependency %q: fail (cached). Error: %s", dep.Name, r)
				dep.logAllowedFailFault(i, len(a.Dependencies))
				continue
			}
			return errors.Annotate(r, "run dependency %q: fail (cached)", dep.Name).Err()
		}
		if err := dep.run(ctx, c, args); err != nil {
			if dep.AllowFail {
				log.Printf("Dependency %q: fail. Error: %s", dep.Name, err)
				dep.logAllowedFailFault(i, len(a.Dependencies))
			} else {
				return errors.Annotate(err, "run dependency %q", dep.Name).Err()
			}
		} else {
			log.Printf("Dependency %q: finished successfully.", dep.Name)
		}
	}
	return nil
}

// logAllowedFailFault logs fault when action allowed to fail and AllowFail=true.
// Supported cases when have next action we report that we will proceed with next one or if we do not have next then just ignore result of this one.
func (a *Action) logAllowedFailFault(i, count int) {
	if i == count-1 {
		log.Printf("Ignore error as action %q is allowed to fail.", a.Name)
	} else {
		log.Printf("Continue to next action as %q is allowed to fail.", a.Name)
	}
}

// Describe describes the action structure recursively.
func (a *Action) Describe(prefix string) string {
	ap := fmt.Sprintf("Action %q, Exec: %s, AllowFail: %v, AllowCache: %v", a.Name, a.ExecName, a.AllowFail, a.AllowCache)
	if len(a.Dependencies) > 0 {
		ap += fmt.Sprintf("%sDependencies:", prefix)
		for i, d := range a.Dependencies {
			ap += fmt.Sprintf("%s%d: %s", prefix, i, d.Describe(prefix+"  "))
		}
	}
	if len(a.Recoveries) > 0 {
		ap += fmt.Sprintf("%sRecoveries:", prefix)
		for i, r := range a.Recoveries {
			ap += fmt.Sprintf("%s%d: %s", prefix, i, r.Describe(prefix+"  "))
		}
	}
	return ap
}
