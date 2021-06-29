// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package plan

// runCache holds cache per actions for running plan.
// The structure will holds caches for running plan with actions.
type runCache struct {
	// actions associates action names to errors.
	// An action that failed is associated with a non-nil error.
	// An action that succeeded is associated with nil.
	actions map[string]error
}

// newRunCache creates new runCache.
func newRunCache() *runCache {
	return &runCache{
		actions: make(map[string]error),
	}
}

// close clears the cache in effective way.
// Please read https://github.com/golang/go/issues/20138 for more details.
func (c *runCache) close() {
	for k := range c.actions {
		delete(c.actions, k)
	}
}

// Cache result when action fail.
// The cache value will set an error as failure reason.
func (c *runCache) cacheAction(a *Action, e error) {
	if a.AllowCache {
		c.actions[a.Name] = e
	}
}

// Get action error per action.
// If action pass error will be nil.
func (c *runCache) getActionError(a *Action) (err error, ok bool) {
	err, ok = c.actions[a.Name]
	return
}
