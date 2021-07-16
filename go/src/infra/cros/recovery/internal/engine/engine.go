// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package engine provides struts and functionality of recovery engine.
// For more details please read go/paris-recovery-engine.
package engine

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/planpb"
)

// recoveryEngine holds info required for running a recovery plan.
type recoveryEngine struct {
	planName string
	plan     *planpb.Plan
	args     *execs.RunArgs
}

// Error tag to track error with request to start critical actions over.
var startOverTag = errors.BoolTag{Key: errors.NewTagKey("start-over")}

// Run runs the recovery plan.
func Run(ctx context.Context, planName string, plan *planpb.Plan, args *execs.RunArgs) error {
	r := &recoveryEngine{
		planName: planName,
		plan:     plan,
		args:     args,
	}
	defer r.close()
	return r.runPlan(ctx)
}

// close free up used resources.
func (r *recoveryEngine) close() {
	// TODO(otabek@): Close the caches.
}

// runPlan executes recovery plan with critical-actions.
func (r *recoveryEngine) runPlan(ctx context.Context) error {
	log.Info(ctx, "Plan %q: started", r.planName)
	log.Debug(ctx, "\n%s", r.describe())
	for {
		if err := r.runActions(ctx, r.plan.GetCriticalActions(), true); err != nil {
			if startOverTag.In(err) {
				log.Info(ctx, "Plan %q: received request to start over.", r.planName)
				r.resetCacheAfterSuccessfulRecoveryAction()
				continue
			}
			if r.plan.GetAllowFail() {
				log.Info(ctx, "Plan %q: failed with error: %s.", r.planName, err)
				log.Info(ctx, "Plan %q: is allowed to fail, continue.", r.planName)
				return nil
			}
			return errors.Annotate(err, "run plan %q", r.planName).Err()
		}
		break
	}
	log.Info(ctx, "Plan %q: finished successfully.", r.planName)
	return nil
}

// runActions runs actions in order.
// Execution steps:
// 1) Check action's result in cache.
// 2) Check if the action is applicable based on conditions. Skip if any fail.
// 3) Run dependencies of the action. Fail if any fails.
// 4) Run action exec function. Fail if any fail.
func (r *recoveryEngine) runActions(ctx context.Context, actions []string, canUseRecovery bool) error {
	for _, actionName := range actions {
		if err, ok := r.actionResultFromCache(actionName); ok {
			if err == nil {
				log.Info(ctx, "Action %q: pass (cached).", actionName)
				continue
			}
			a := r.getAction(actionName)
			if a.GetAllowFailAfterRecovery() {
				log.Info(ctx, "Action %q: fail (cached). Error: %s", actionName, err)
				log.Debug(ctx, "Action %q: error ignored as action is allowed to fail.", actionName)
				continue
			}
			return errors.Annotate(err, "run action %q: (cached)", actionName).Err()
		}
		if !r.isActionApplicable(ctx, actionName) {
			log.Info(ctx, "Action %q: one of conditions failed, skipping...", actionName)
			continue
		}
		if err := r.runDependencies(ctx, actionName, canUseRecovery); err != nil {
			return errors.Annotate(err, "run action %q", actionName).Err()
		}
		if err := r.runActionExec(ctx, actionName, canUseRecovery); err != nil {
			if startOverTag.In(err) {
				return errors.Annotate(err, "run action %q", actionName).Err()
			}
			a := r.getAction(actionName)
			if a.GetAllowFailAfterRecovery() {
				log.Info(ctx, "Action %q: fail. Error: %s", actionName, err)
				log.Debug(ctx, "Action %q: error ignored as action is allowed to fail.", actionName)
			} else {
				return errors.Annotate(err, "run action %q", actionName).Err()
			}
		} else {
			log.Info(ctx, "Action %q: finished successfully.", actionName)
		}
	}
	return nil
}

// runActionExec runs action's exec function and initiates recovery flow if exec fails.
// The recover flow start only if it is allowed by canUseRecovery.
func (r *recoveryEngine) runActionExec(ctx context.Context, actionName string, canUseRecovery bool) error {
	a := r.getAction(actionName)
	if err := execs.Run(ctx, a.ExecName, r.args); err != nil {
		if canUseRecovery && len(a.GetRecoveryActions()) > 0 {
			if rErr := r.runRecoveries(ctx, actionName); rErr != nil {
				return errors.Annotate(rErr, "run action %q exec", actionName).Err()
			}
			log.Info(ctx, "Run action %q exec: no recoveries left to try", actionName)
		}
		// Cache the action error only after running recoveries.
		// If no recoveries were run, we still cache the action.
		r.cacheActionResult(actionName, err)
		return errors.Annotate(err, "run action %q exec", actionName).Err()
	}
	r.cacheActionResult(actionName, nil)
	return nil
}

// isActionApplicable checks if action is applicable based on condition actions.
func (r *recoveryEngine) isActionApplicable(ctx context.Context, actionName string) bool {
	a := r.getAction(actionName)
	if err := r.runActions(ctx, a.GetConditions(), false); err != nil {
		log.Debug(ctx, "Action %q: conditions fails. Error: %s", actionName, err)
		return false
	}
	return true
}

// runDependencies runs action's dependencies.
func (r *recoveryEngine) runDependencies(ctx context.Context, actionName string, canUseRecovery bool) error {
	a := r.getAction(actionName)
	err := r.runActions(ctx, a.GetDependencies(), canUseRecovery)
	return errors.Annotate(err, "run dependencies").Err()
}

// runRecoveries runs action's recoveries.
// Recovery actions are expected to fail. If recovery action fails then next will be attempted.
// Finishes with nil if no recovery action provided or nether succeeded.
// Finishes with start-over request if any recovery succeeded.
// Recovery action will skip if used before.
func (r *recoveryEngine) runRecoveries(ctx context.Context, actionName string) error {
	a := r.getAction(actionName)
	for _, recoveryName := range a.GetRecoveryActions() {
		if r.isRecoveryUsed(ctx, actionName, recoveryName) {
			// Engine allows to use each recovery action only once in scope of the action.
			continue
		}
		if err := r.runActions(ctx, []string{recoveryName}, false); err != nil {
			log.Debug(ctx, "Recovery action %q: fail. Error: %s ", recoveryName, err)
			r.registerRecoveryUsage(actionName, recoveryName, err)
			continue
		}
		r.registerRecoveryUsage(actionName, recoveryName, nil)
		log.Info(ctx, "Recovery action %q: request to start-over.", recoveryName)
		return errors.Reason("recovery action %q: request to start over", recoveryName).Tag(startOverTag).Err()
	}
	return nil
}

// getAction finds and provides action from the plan collection.
func (r *recoveryEngine) getAction(name string) *planpb.Action {
	if a, ok := r.plan.Actions[name]; ok {
		return a
	}
	// If we reach this place then we have issues with plan validation logic.
	panic(fmt.Sprintf("action %q not found in the plan", name))
}

// describe describes the plan details with critical actions.
func (r *recoveryEngine) describe() string {
	d := fmt.Sprintf("Plan %q, AllowFail: %v ", r.planName, r.plan.AllowFail)
	if len(r.plan.GetCriticalActions()) > 0 {
		prefix := "\n "
		d += fmt.Sprintf("%sCritical-actions:", prefix)
		for i, a := range r.plan.GetCriticalActions() {
			d += fmt.Sprintf("%s %d: %s", prefix, i, r.describeAction(a, prefix+"  "))
		}
	} else {
		d += "\n No critical-actions"
	}
	return d
}

// describeAction describes the action structure recursively.
func (r *recoveryEngine) describeAction(actionName string, prefix string) string {
	a := r.getAction(actionName)
	ap := fmt.Sprintf("Action %q, AllowFailAfterRecovery: %v, RunControl: %v",
		actionName, a.GetAllowFailAfterRecovery(), a.GetRunControl())
	if len(a.GetConditions()) > 0 {
		ap += fmt.Sprintf("%sConditions:", prefix)
		for i, d := range a.GetConditions() {
			ap += fmt.Sprintf("%s%d: %s", prefix, i, r.describeAction(d, prefix+"  "))
		}
	}
	ap += fmt.Sprintf("%sExec: %s", prefix, r.describeActionExec(actionName))
	if len(a.GetDependencies()) > 0 {
		ap += fmt.Sprintf("%sDependencies:", prefix)
		for i, d := range a.GetDependencies() {
			ap += fmt.Sprintf("%s%d: %s", prefix, i, r.describeAction(d, prefix+"  "))
		}
	}
	if len(a.GetRecoveryActions()) > 0 {
		ap += fmt.Sprintf("%sRecoveryActions:", prefix)
		for i, d := range a.GetRecoveryActions() {
			ap += fmt.Sprintf("%s%d: %s", prefix, i, r.describeAction(d, prefix+"  "))
		}
	}
	return ap
}

// describeActionExec describes the action's exec function with details.
func (r *recoveryEngine) describeActionExec(actionName string) string {
	a := r.getAction(actionName)
	er := a.GetExecName()
	if len(a.GetExecExtraArgs()) > 0 {
		er += fmt.Sprintf(", Args: %s", a.GetExecExtraArgs())
	}
	return er
}

// actionResultFromCache reads action's result from cache.
func (r *recoveryEngine) actionResultFromCache(actionName string) (ar error, ok bool) {
	// TODO(otabek@): Read from action results cache
	return nil, false
}

// cacheActionResult sets action's result to the cache.
func (r *recoveryEngine) cacheActionResult(actionName string, err error) {
	// TODO(otabek@): Set result to the action result cache based on run-control.
}

// resetCacheAfterSuccessfulRecoveryAction resets cache for actions
// with run-control=RERUN_AFTER_RECOVERY.
func (r *recoveryEngine) resetCacheAfterSuccessfulRecoveryAction() {
	// TODO(otabek@): Implement reset based on recovery engine design.
}

// isRecoveryUsed checks if recovery action is used in plan or action level scope.
func (r *recoveryEngine) isRecoveryUsed(ctx context.Context, actionName, recoveryName string) bool {
	// TODO(otabek@): Read action results cache and recovery usage cache.
	// TODO(otabek@): Update recovery usage cache if response found in action results cache.
	return false
}

// registerRecoveryUsage sets recovery action usage to the cache.
func (r *recoveryEngine) registerRecoveryUsage(actionName, recoveryName string, err error) {
	// TODO(otabek@): Set result to the action result cache based on run-control.
	// TODO(otabek@): Update recovery usage cache.
}
