// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package engine provides struts and functionality of recovery engine.
// For more details please read go/paris-recovery-engine.
package engine

import (
	"context"
	"fmt"
	"time"

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
	// Caches
	actionResultsCache map[string]error
	recoveryUsageCache map[recoveryUsageKey]error
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
	r.initCache()
	defer r.close()
	log.Debug(ctx, "Received plan %s\n%s", r.planName, r.describe())
	return r.runPlan(ctx)
}

// close free up used resources.
func (r *recoveryEngine) close() {
	if r.actionResultsCache != nil {
		r.actionResultsCache = nil
	}
	// TODO(otabek@): Close the caches.
}

// runPlan executes recovery plan with critical-actions.
func (r *recoveryEngine) runPlan(ctx context.Context) error {
	log.Info(ctx, "Plan %q: started", r.planName)
	for {
		if err := r.runActions(ctx, r.plan.GetCriticalActions(), r.args.EnableRecovery); err != nil {
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
func (r *recoveryEngine) runActions(ctx context.Context, actions []string, enableRecovery bool) error {
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
		if err := r.runDependencies(ctx, actionName, enableRecovery); err != nil {
			return errors.Annotate(err, "run action %q", actionName).Err()
		}
		if err := r.runActionExec(ctx, actionName, enableRecovery); err != nil {
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
// The recover flow start only recoveries is enabled.
func (r *recoveryEngine) runActionExec(ctx context.Context, actionName string, enableRecovery bool) error {
	a := r.getAction(actionName)
	if err := r.runActionExecWithTimeout(ctx, a); err != nil {
		if enableRecovery && len(a.GetRecoveryActions()) > 0 {
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

// Default time limit per action exec function.
const defaultExecTimeout = 60 * time.Second

func actionExecTimeout(a *planpb.Action) time.Duration {
	if a.ExecTimeout != nil {
		return a.ExecTimeout.AsDuration()
	}
	return defaultExecTimeout
}

// runActionExecWithTimeout runs action's exec function with timeout.
func (r *recoveryEngine) runActionExecWithTimeout(ctx context.Context, a *planpb.Action) error {
	timeout := actionExecTimeout(a)
	newCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cw := make(chan error, 1)
	go func() {
		err := execs.Run(newCtx, a.ExecName, r.args)
		cw <- err
	}()
	select {
	case err := <-cw:
		return errors.Annotate(err, "run exec %q with timeout %s", a.ExecName, timeout).Err()
	case <-newCtx.Done():
		log.Info(newCtx, "Run exec %q with timeout %s: excited timeout", a.ExecName, timeout)
		return errors.Reason("run exec %q with timeout %s: excited timeout", a.ExecName, timeout).Err()
	}
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
func (r *recoveryEngine) runDependencies(ctx context.Context, actionName string, enableRecovery bool) error {
	a := r.getAction(actionName)
	err := r.runActions(ctx, a.GetDependencies(), enableRecovery)
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
		if r.isRecoveryUsed(actionName, recoveryName) {
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

// initCache initializes cache on engine.
// The function extracted to supported testing.
func (r *recoveryEngine) initCache() {
	r.actionResultsCache = make(map[string]error, len(r.plan.GetActions()))
	r.recoveryUsageCache = make(map[recoveryUsageKey]error)
}

// actionResultFromCache reads action's result from cache.
func (r *recoveryEngine) actionResultFromCache(actionName string) (ar error, ok bool) {
	err, ok := r.actionResultsCache[actionName]
	return err, ok
}

// cacheActionResult sets action's result to the cache.
func (r *recoveryEngine) cacheActionResult(actionName string, err error) {
	switch r.getAction(actionName).GetRunControl() {
	case planpb.RunControl_RERUN_AFTER_RECOVERY, planpb.RunControl_RUN_ONCE:
		r.actionResultsCache[actionName] = err
	case planpb.RunControl_ALWAYS_RUN:
		// Do not cache the value
	}
}

// resetCacheAfterSuccessfulRecoveryAction resets cache for actions
// with run-control=RERUN_AFTER_RECOVERY.
func (r *recoveryEngine) resetCacheAfterSuccessfulRecoveryAction() {
	for name, a := range r.plan.GetActions() {
		if a.GetRunControl() == planpb.RunControl_RERUN_AFTER_RECOVERY {
			delete(r.actionResultsCache, name)
		}
	}
}

// isRecoveryUsed checks if recovery action is used in plan or action level scope.
func (r *recoveryEngine) isRecoveryUsed(actionName, recoveryName string) bool {
	k := recoveryUsageKey{
		action:   actionName,
		recovery: recoveryName,
	}
	// If the recovery has been used in previous actions then it can be in
	// the action result cache.
	if err, ok := r.actionResultsCache[recoveryName]; ok {
		r.recoveryUsageCache[k] = err
	}
	_, ok := r.recoveryUsageCache[k]
	return ok
}

// registerRecoveryUsage sets recovery action usage to the cache.
func (r *recoveryEngine) registerRecoveryUsage(actionName, recoveryName string, err error) {
	r.recoveryUsageCache[recoveryUsageKey{
		action:   actionName,
		recovery: recoveryName,
	}] = err
}

// recoveryUsageKey holds action and action's recovery name as key for recovery-usage cache.
type recoveryUsageKey struct {
	action   string
	recovery string
}
