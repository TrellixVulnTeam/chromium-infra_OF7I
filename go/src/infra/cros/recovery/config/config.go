// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"encoding/json"
	"io"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/log"
)

// ExecsExist function to check if exec is exit.
type ExecsExist func(execName string) bool

// Load performs loading the configuration source with data validation.
func Load(ctx context.Context, r io.Reader, execsExit ExecsExist) (*Configuration, error) {
	log.Debugf(ctx, "Load configuration: started.")
	if r == nil {
		return nil, errors.Reason("load configuration: reader is not provided").Err()
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Annotate(err, "load configuration").Err()
	}
	if len(data) == 0 {
		return nil, errors.Reason("load configuration: is empty").Err()
	}
	c := &Configuration{}
	if err := json.Unmarshal(data, c); err != nil {
		return nil, errors.Annotate(err, "load configuration").Err()
	}
	if execsExit == nil {
		log.Infof(ctx, "Load configuration: validation skipped!")
	} else {
		c, err = Validate(ctx, c, execsExit)
		if err != nil {
			return nil, errors.Annotate(err, "load configuration").Err()
		}
	}
	log.Debugf(ctx, "Load configuration: finished successfully.")
	return c, nil
}

// Validate validate configuration before usage.
//
// The validater is also fix missed adjusted actions.
func Validate(ctx context.Context, c *Configuration, execsExist ExecsExist) (*Configuration, error) {
	if c == nil {
		return c, nil
	}
	for pName, p := range c.GetPlans() {
		createMissingActions(p, p.GetCriticalActions())
		for _, a := range p.GetActions() {
			createMissingActions(p, a.GetConditions())
			createMissingActions(p, a.GetDependencies())
			createMissingActions(p, a.GetRecoveryActions())
		}
		if err := setAndVerifyExecs(p, execsExist); err != nil {
			return nil, errors.Annotate(err, "load configuration").Err()
		}
		// Check for cycle in dependency.
		if err := verifyPlanAcyclic(p); err != nil {
			return nil, errors.Annotate(err, "load configuration: of %q", pName).Err()
		}
	}
	return c, nil
}

// createMissingActions creates missing actions to the plan.
func createMissingActions(p *Plan, actions []string) {
	if p.GetActions() == nil {
		p.Actions = make(map[string]*Action)
	}
	for _, a := range actions {
		if _, ok := p.GetActions()[a]; !ok {
			p.GetActions()[a] = &Action{}
		}
	}
}

// Check the plans critical action for present of connection to avoid infinity loop running of recovery engine.
func verifyPlanAcyclic(plan *Plan) error {
	visited := map[string]bool{}
	var verifyAction func(string) error
	// ReferenceName stands for each action's type of dependency list.
	// ReferenceName can be either one of the three: condition, dependency, recovery.
	verifyDependActions := func(referenceName string, currentSetOfActions []string) error {
		for _, actionName := range currentSetOfActions {
			if _, ok := plan.Actions[actionName]; ok {
				if err := verifyAction(actionName); err != nil {
					return errors.Annotate(err, "check %q from %s", actionName, referenceName).Err()
				}
			}
		}
		return nil
	}
	// Verify the current Action in the current layer of the DFS.
	verifyAction = func(actionName string) error {
		if visited[actionName] {
			return errors.Reason("found loop").Err()
		}
		visited[actionName] = true
		if err := verifyDependActions("condition", plan.Actions[actionName].GetConditions()); err != nil {
			return err
		}
		if err := verifyDependActions("dependency", plan.Actions[actionName].GetDependencies()); err != nil {
			return err
		}
		if err := verifyDependActions("recovery", plan.Actions[actionName].GetRecoveryActions()); err != nil {
			return err
		}
		visited[actionName] = false
		return nil
	}
	for _, eachActionName := range plan.GetCriticalActions() {
		if _, ok := visited[eachActionName]; !ok {
			return verifyAction(eachActionName)
		}
	}
	return nil
}

// setAndVerifyExecs sets exec-name if missing and validate whether exec is present
// in recovery-lib.
func setAndVerifyExecs(p *Plan, execsExist ExecsExist) error {
	for an, a := range p.GetActions() {
		if a.GetExecName() == "" {
			a.ExecName = an
		}
		if !execsExist(a.GetExecName()) {
			return errors.Reason("exec %q is not exist", a.GetExecName()).Err()
		}
	}
	return nil
}
