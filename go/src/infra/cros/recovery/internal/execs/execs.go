// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package execs provides collection of execution functions for actions and ability to execute them.
package execs

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/logger"
	"infra/cros/recovery/logger/metrics"
	"infra/cros/recovery/tlw"
)

// exec represents an execution function of the action.
// The single exec can be associated with one or more actions.
type ExecFunction func(ctx context.Context, args *RunArgs, actionArgs []string) error

var (
	// Map of known exec functions used by recovery engine.
	// Use Register() function to add to this map.
	knownExecMap = make(map[string]ExecFunction)
)

// Register registers new exec function to be used with recovery engine.
// We panic if a name is reused.
func Register(name string, f ExecFunction) {
	if _, ok := knownExecMap[name]; ok {
		panic(fmt.Sprintf("Register exec %q: already registered", name))
	}
	if f == nil {
		panic(fmt.Sprintf("register exec %q: exec function is not provided", name))
	}
	knownExecMap[name] = f
}

// RunArgs holds input arguments for an exec function.
type RunArgs struct {
	// Resource name targeted by plan.
	ResourceName string
	DUT          *tlw.Dut
	Access       tlw.Access
	// Logger prints message to the logs.
	Logger logger.Logger
	// Provide option to stop use steps.
	ShowSteps bool
	// Metrics records actions and observations.
	Metrics metrics.Metrics
	// EnableRecovery tells if recovery actions are enabled.
	EnableRecovery bool
}

// Run runs exec function provided by this package by name.
func Run(ctx context.Context, name string, args *RunArgs, actionArgs []string) error {
	e, ok := knownExecMap[name]
	if !ok {
		return errors.Reason("exec %q: not found", name).Err()
	}
	return e(ctx, args, actionArgs)
}

// Exist check if exec function with name is present.
func Exist(name string) bool {
	_, ok := knownExecMap[name]
	return ok
}

// Runner defines the type for a function that will execute a command
// on a host, and returns the result as a single line.
type Runner func(context.Context, string) (string, error)

// NewRunner returns a function of type Runner that executes a command
// on a host and returns the results as a single line. This function
// defines the specific host on which the command will be
// executed. Examples of such specific hosts can be the DUT, or the
// servo-host etc.
func (args RunArgs) NewRunner(host string) Runner {
	runner := func(ctx context.Context, cmd string) (string, error) {
		r := args.Access.Run(ctx, host, cmd)
		if r.ExitCode != 0 {
			return "", errors.Reason("runner: command %q completed with exit code %q", cmd, r.ExitCode).Err()
		}
		return strings.TrimSpace(r.Stdout), nil
	}
	return runner
}
