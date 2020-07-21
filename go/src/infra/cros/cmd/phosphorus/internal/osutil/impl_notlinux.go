// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !linux

// Package osutil contains high-level utility functions for operating system
// functionality.
package osutil

import (
	"context"
	"os/exec"
)

func runWithAbortImpl(ctx context.Context, cmd *exec.Cmd) (RunResult, error) {
	panic("Not implemented for non-linux platforms")
}
