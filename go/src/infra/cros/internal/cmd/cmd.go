// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package cmd provides support for running commands.
package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
)

// CommandRunner is the common interface for this module.
type CommandRunner interface {
	RunCommand(ctx context.Context, stdoutBuf, stderrBuf *bytes.Buffer, dir, name string, args ...string) error
}

// RealCommandRunner actually runs commands.
type RealCommandRunner struct{}

// RunCommand runs a command.
func (c RealCommandRunner) RunCommand(ctx context.Context, stdoutBuf, stderrBuf *bytes.Buffer, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	cmd.Dir = dir
	return cmd.Run()
}

// FakeCommandRunner does not actually run commands.
// It is used for testing.
type FakeCommandRunner struct {
	Stdout      string
	Stderr      string
	ExpectedCmd []string
	ExpectedDir string
	FailCommand bool
	FailError   string
}

// RunCommand runs a command (not actually).
func (c FakeCommandRunner) RunCommand(ctx context.Context, stdoutBuf, stderrBuf *bytes.Buffer, dir, name string, args ...string) error {
	stdoutBuf.WriteString(c.Stdout)
	stderrBuf.WriteString(c.Stderr)
	cmd := append([]string{name}, args...)
	if len(c.ExpectedCmd) > 0 {
		if !reflect.DeepEqual(cmd, c.ExpectedCmd) {
			expectedCmd := strings.Join(c.ExpectedCmd, " ")
			actualCmd := strings.Join(cmd, " ")
			return fmt.Errorf("wrong cmd; expected %s got %s", expectedCmd, actualCmd)
		}
	}
	if c.ExpectedDir != "" {
		if dir != c.ExpectedDir {
			return fmt.Errorf("wrong cmd dir; expected %s got %s", c.ExpectedDir, dir)
		}
	}
	if c.FailCommand {
		return fmt.Errorf(c.FailError)
	}
	return nil
}

// FakeCommandRunnerMulti provides multiple command runners.
type FakeCommandRunnerMulti struct {
	run            int
	CommandRunners []FakeCommandRunner
}

// RunCommand runs a command (not actually).
func (c *FakeCommandRunnerMulti) RunCommand(ctx context.Context, stdoutBuf, stderrBuf *bytes.Buffer, dir, name string, args ...string) error {
	if c.run >= len(c.CommandRunners) {
		return fmt.Errorf("unexpected cmd")
	}
	err := c.CommandRunners[c.run].RunCommand(ctx, stdoutBuf, stderrBuf, dir, name, args...)
	c.run++
	return err
}
