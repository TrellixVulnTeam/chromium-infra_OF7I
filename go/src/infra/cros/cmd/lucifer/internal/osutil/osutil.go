// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package osutil contains high-level utility functions for operating system
// functionality.
package osutil

import (
	"context"
	"log"
	"os/exec"
	"path/filepath"
)

// RunResult contains information about process run via RunWithAbort.
type RunResult struct {
	// Started is true if the process was started successfully.
	Started bool
	Aborted bool
	// ExitStatus only makes sense if Started is true.
	ExitStatus int
}

// RunWithAbort runs an exec.Cmd with context cancellation/aborting.
// The command will have been waited for when this function returns.
//
// This function returns an error if the command failed to start.
// This function always returns a valid RunResult, even in case of errors.
func RunWithAbort(ctx context.Context, cmd *exec.Cmd) (RunResult, error) {
	r := RunResult{}
	name := filepath.Base(cmd.Path)
	if err := cmd.Start(); err != nil {
		return r, err
	}
	r.Started = true
	exited := make(chan struct{})
	go func() {
		if err := cmd.Wait(); err != nil {
			r.ExitStatus = getCmdExitStatus(err)
		}
		close(exited)
	}()
	select {
	case <-ctx.Done():
		log.Printf("Aborting command %s", name)
		r.Aborted = true
		terminate(cmd, exited)
	case <-exited:
	}
	return r, nil
}

// sigkill sends SIGKILL to a command.
func sigkill(cmd *exec.Cmd) {
	name := filepath.Base(cmd.Path)
	log.Printf("SIGKILLing %s", name)
	if err := cmd.Process.Kill(); err != nil {
		// Something has gone really wrong, blow up.
		log.Panicf("Failed to SIGKILL ¯\\_(ツ)_/¯")
	}
}
