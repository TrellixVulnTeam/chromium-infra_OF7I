// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build !windows

package osutil

import (
	"log"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func getCmdExitStatus(err error) int {
	if ee, ok := err.(*exec.ExitError); ok {
		// Cannot use ee.ProcessState.ExitCode() as ExitCode is not supported until go1.12.
		// https://golang.org/pkg/os/#ProcessState.ExitCode
		// But chroot has go version 1.11.2.
		if status, ok := ee.ProcessState.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
		return -1
	}
	return -1
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
