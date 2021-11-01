// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package loader provides functionality to load configuration and verify it.
package loader

import (
	"context"
	"encoding/json"
	"io"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/planpb"
)

// TODO(otabek@): Add data validation for loaded config.
// 1) Looping actions

// LoadConfiguration performs loading the configuration source with data validation.
func LoadConfiguration(ctx context.Context, r io.Reader) (*planpb.Configuration, error) {
	log.Debug(ctx, "Load configuration: started.")
	if r == nil {
		return nil, errors.Reason("load configuration: reader is not provided").Err()
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Annotate(err, "load configuration").Err()
	}
	if len(data) == 0 {
		return nil, errors.Reason("load configuration: configuration is empty").Err()
	}
	config := planpb.Configuration{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.Annotate(err, "load configuration").Err()
	}
	for pName, p := range config.GetPlans() {
		createMissingActions(p, p.GetCriticalActions())
		for _, a := range p.GetActions() {
			createMissingActions(p, a.GetConditions())
			createMissingActions(p, a.GetDependencies())
			createMissingActions(p, a.GetRecoveryActions())
		}
		if err := setAndVerifyExecs(p); err != nil {
			return nil, errors.Annotate(err, "load configuration").Err()
		}
		// Check for cycle in dependency.
		if err := verifyPlanAcyclic(p); err != nil {
			return nil, errors.Annotate(err, "load configuration: of %q", pName).Err()
		}
	}
	log.Debug(ctx, "Load configuration: finished successfully.")
	return &config, nil
}

// Check the plans critical action for present of connection to avoid infinity loop running of recovery engine.
func verifyPlanAcyclic(plan *planpb.Plan) error {
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

// createMissingActions creates missing actions to the plan.
func createMissingActions(p *planpb.Plan, actions []string) {
	if p.GetActions() == nil {
		p.Actions = make(map[string]*planpb.Action)
	}
	for _, a := range actions {
		if _, ok := p.GetActions()[a]; !ok {
			p.GetActions()[a] = &planpb.Action{}
		}
	}
}

// execsExist is link to the function to check if exec function is present.
// Link created to create ability to override for local testing.
var execsExist = execs.Exist

// setAndVerifyExecs sets exec-name if missing and validate whether exec is present
// in recovery-lib.
func setAndVerifyExecs(p *planpb.Plan) error {
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
