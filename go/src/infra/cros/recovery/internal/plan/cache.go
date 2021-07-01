// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package plan

import (
	"fmt"
	"log"
)

// runCache holds cache per actions for running plan.
// The structure will holds caches for running plan with actions.
type runCache struct {
	// actions associates action names to errors.
	// An action that failed is associated with a non-nil error.
	// An action that succeeded is associated with nil.
	actions map[string]error
	// Tracking usage of recoveries under each action.
	// Key represents two names "action_name|recovery_name".
	// An recovery that failed is associated with a non-nil error.
	// An recovery that succeeded is associated with nil.
	// The goal of cache it to track usage to prevent looping recovery-verify
	// for cases when recovery is not cacheable.
	recoveries map[string]error
}

// newCache creates new runCache.
// runCache need to be created for each plan run separate.
// runCache contains two caches:
// 1) Action cache (global): associates action to execution result of it.
//    The key of the cache is action's name.
//    If failed: associated with a non-nil error.
//    If succeeded: associated with nil.
// 2) Recovery cache (local): associates recovery action in scope of the parent action (action with recoveries collection) to executed result of it.
//    The key of the cache is action's name + recovery's name ('action_name|recovery_name').
//    Goal is prevent recevery-verify loop when recovery set.
//    If failed: associated with a non-nil error.
//    If succeeded: associated with nil.
func newCache() *runCache {
	return &runCache{
		actions:    make(map[string]error),
		recoveries: make(map[string]error),
	}
}

// close clears the cache in effective way.
// Please read https://github.com/golang/go/issues/20138 for more details.
func (c *runCache) close() {
	for k := range c.actions {
		delete(c.actions, k)
	}
	for k := range c.recoveries {
		delete(c.recoveries, k)
	}
}

// Cache action result.
// If failed: set non-nil error as failure reason.
// If succeeded: set nil as success result.
func (c *runCache) cacheAction(a *Action, e error) {
	if a.AllowCache {
		c.actions[a.Name] = e
	}
}

// Get action error per action.
// A non-nil error associated with fail result and nil with success.
func (c *runCache) getActionError(a *Action) (err error, ok bool) {
	err, ok = c.actions[a.Name]
	return
}

// Reset action and its dependencies results from cache.
func (c *runCache) resetForAction(a *Action) {
	log.Printf("Reset cache for action %q", a.Name)
	delete(c.actions, a.Name)
	for _, dep := range a.Dependencies {
		c.resetForAction(dep)
	}
}

// Cache recovery action result.
// If failed: set non-nil error as failure reason.
// If succeeded: set nil as success result.
func (c *runCache) cacheRecovery(a *Action, recovery *Action, e error) {
	c.recoveries[c.createRecoveryKey(a, recovery)] = e
	// Caching in case recovery used for another action.
	c.cacheAction(recovery, e)
}

// Check if recovery action was used at all or for the action.
// The first check is whether a recovery exists in the action cache, if not, check in the recovery cache.
func (c *runCache) isRecoveryUsed(a *Action, recovery *Action) bool {
	// If recovery was used at all.
	// Recovery may not exist in action cache due to it's not cacheable globally.
	if _, ok := c.getActionError(recovery); ok {
		return ok
	}
	// Checking if recovery was used in scope of the action.
	_, ok := c.recoveries[c.createRecoveryKey(a, recovery)]
	return ok
}

// createRecoveryKey creates key for recoveries map.
// The key contains two actions names.
// Example: 'action1|action2'.
func (c *runCache) createRecoveryKey(a *Action, recovery *Action) string {
	return fmt.Sprintf("%s|%s", a.Name, recovery.Name)
}
