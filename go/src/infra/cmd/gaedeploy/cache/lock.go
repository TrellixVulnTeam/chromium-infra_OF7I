// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cache

import (
	"context"
	"math/rand"
	"time"

	"github.com/danjacques/gofslock/fslock"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
)

// lockFS grabs a lock file and returns a function that releases it.
func lockFS(ctx context.Context, path string, giveUpTimeout time.Duration) (unlock func() error, err error) {
	ctx, cancel := context.WithTimeout(ctx, giveUpTimeout)
	defer cancel()

	attempt := 0

	l := fslock.L{
		Path: path,
		Block: fslock.Blocker(func() error {
			attempt++
			delay := 5*time.Second + time.Duration(rand.Int63n(int64(5*time.Second)))
			logging.Warningf(ctx, "Failed to grab FS lock on attempt %d, retrying after %s...", attempt, delay)
			tr := clock.Sleep(ctx, delay)
			return tr.Err
		}),
	}

	handle, err := l.Lock()
	if err != nil {
		return nil, err
	}

	return handle.Unlock, nil
}
