// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build linux

// Package osutil contains high-level utility functions for operating system
// functionality.
package osutil

import (
	"context"
	"log"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func runWithAbortImpl(ctx context.Context, cmd *exec.Cmd) (RunResult, error) {
	r := RunResult{}
	name := filepath.Base(cmd.Path)
	if err := cmd.Start(); err != nil {
		return r, err
	}
	r.Started = true
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
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

// killTimeout is the duration between sending SIGTERM and SIGKILL
// when a process is aborted.
const killTimeout = 6 * time.Second

// terminate terminates a command using SIGTERM and then SIGKILL.
// exited is a channel that is closed when the command is waited for.
// The command will have been waited for when this function returns.
func terminate(cmd *exec.Cmd, exited <-chan struct{}) {
	name := filepath.Base(cmd.Path)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("Failed to SIGTERM command %s: %s", name, err)
	}
	select {
	case <-time.After(killTimeout):
		sigkill(cmd)
		<-exited
	case <-exited:
	}
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
