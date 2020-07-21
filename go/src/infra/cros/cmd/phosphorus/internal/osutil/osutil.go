// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package osutil contains high-level utility functions for operating system
// functionality.
package osutil

import (
	"context"
	"os/exec"
)

// RunResult contains information about process run via RunWithAbort.
type RunResult struct {
	// Started is true if the process was started successfully.
	Started bool
	Aborted bool
}

// RunWithAbort runs an exec.Cmd with context cancellation/aborting.
// The command will have been waited for when this function returns.
//
// This function returns an error if the command failed to start.
// This function always returns a valid RunResult, even in case of errors.
func RunWithAbort(ctx context.Context, cmd *exec.Cmd) (RunResult, error) {
	// Conditionally compiled platform-specific implementation.
	return runWithAbortImpl(ctx, cmd)
}
