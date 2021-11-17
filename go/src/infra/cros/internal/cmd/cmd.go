// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package cmd provides support for running commands.
package cmd

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// CommandRunner is the common interface for this module.
type CommandRunner interface {
	RunCommand(ctx context.Context, stdoutBuf, stderrBuf io.Writer, dir, name string, args ...string) error
}

// RealCommandRunner actually runs commands.
type RealCommandRunner struct{}

// RunCommand runs a command.
func (c RealCommandRunner) RunCommand(ctx context.Context, stdoutBuf, stderrBuf io.Writer, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	cmd.Dir = dir
	return cmd.Run()
}

// FakeCommandRunner does not actually run commands.
// It is used for testing.
type FakeCommandRunner struct {
	Stdout string
	Stderr string
	// Only one of ExpectedCmd and ExpectedCmdPartial can be set.
	ExpectedCmd        []string
	ExpectedCmdPartial []string
	ExpectedDir        string
	FailCommand        bool
	FailError          error
}

// RunCommand runs a command (not actually).
func (c FakeCommandRunner) RunCommand(ctx context.Context, stdoutBuf, stderrBuf io.Writer, dir, name string, args ...string) error {
	stdoutBuf.Write([]byte(c.Stdout))
	stderrBuf.Write([]byte(c.Stderr))
	cmd := append([]string{name}, args...)
	if len(c.ExpectedCmd) > 0 && len(c.ExpectedCmdPartial) > 0 {
		return fmt.Errorf("ExpectedCmd and ExpectedCmdPartial cannot both be set")
	} else if len(c.ExpectedCmd) > 0 {
		if diff := cmp.Diff(c.ExpectedCmd, cmd); diff != "" {
			return fmt.Errorf("wrong cmd; (-want +got):\n%v", diff)
		}
	} else if len(c.ExpectedCmdPartial) > 0 {
		partialStr := strings.Join(c.ExpectedCmdPartial, " ")
		cmdStr := strings.Join(cmd, " ")
		if !strings.Contains(cmdStr, partialStr) {
			return fmt.Errorf("wrong cmd; %s did not contain %s", cmdStr, partialStr)
		}
	}
	if c.ExpectedDir != "" {
		if dir != c.ExpectedDir {
			return fmt.Errorf("wrong cmd dir; expected %s got %s", c.ExpectedDir, dir)
		}
	}
	if c.FailCommand {
		return c.FailError
	}
	return nil
}

// FakeCommandRunnerMulti provides multiple command runners.
type FakeCommandRunnerMulti struct {
	run            int
	CommandRunners []FakeCommandRunner
}

// RunCommand runs a command (not actually).
func (c *FakeCommandRunnerMulti) RunCommand(ctx context.Context, stdoutBuf, stderrBuf io.Writer, dir, name string, args ...string) error {
	if c.run >= len(c.CommandRunners) {
		return fmt.Errorf("unexpected cmd")
	}
	err := c.CommandRunners[c.run].RunCommand(ctx, stdoutBuf, stderrBuf, dir, name, args...)
	c.run++
	return err
}
