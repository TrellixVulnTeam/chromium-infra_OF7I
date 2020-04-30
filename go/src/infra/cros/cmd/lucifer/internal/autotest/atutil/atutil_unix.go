// Copyright 2018 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package atutil

import (
	"context"
	"io"
	"syscall"

	"github.com/pkg/errors"

	"infra/cros/cmd/lucifer/internal/autotest"
	"infra/cros/cmd/lucifer/internal/osutil"
)

// runTask runs an autoserv task.
//
// Result.TestsFailed is always zero.
func runTask(ctx context.Context, c autotest.Config, a *autotest.AutoservArgs, w io.Writer) (*Result, error) {
	r := &Result{}
	cmd := autotest.AutoservCommand(c, a)
	cmd.Stdout = w
	cmd.Stderr = w

	var err error
	r.RunResult, err = osutil.RunWithAbort(ctx, cmd)
	if err != nil {
		return r, err
	}
	if es, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
		r.Exit = es.ExitStatus()
	} else {
		return r, errors.New("RunAutoserv: failed to get exit status: unknown process state")
	}
	if r.Exit != 0 {
		return r, errors.Errorf("RunAutoserv: exited %d", r.Exit)
	}
	return r, nil
}
