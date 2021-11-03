// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package execs provides collection of execution functions for actions and ability to execute them.
package execs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/logger"
	"infra/cros/recovery/logger/metrics"
	"infra/cros/recovery/tlw"
)

const (
	// This character separates the name and values for extra
	// arguments defined for actions.
	DefaultSplitter = ":"
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

// CloserFunc is a function that updates an action and is safe to use in a defer block.
type CloserFunc = func(context.Context, error)

// NewMetric creates a new metric. Neither the action nor the closer function that NewMetrics returns will
// ever be nil.
// TODO(gregorynisbet): Consider adding a time parameter.
func (a *RunArgs) NewMetric(ctx context.Context, kind string) (*metrics.Action, CloserFunc) {
	startTime := time.Now()
	action := &metrics.Action{
		ActionKind: kind,
		StartTime:  startTime,
	}
	return createMetric(ctx, a.Metrics, action)
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

// 127: linux command line error of command not found.
var SSHErrorCLINotFound = errors.BoolTag{Key: errors.NewTagKey("ssh_error_cli_not_found")}

// other linux error tag.
var GeneralError = errors.BoolTag{Key: errors.NewTagKey("general_error")}

// internal error tag.
var SSHErrorInternal = errors.BoolTag{Key: errors.NewTagKey("ssh_error_internal")}

// -1: fail to create ssh session.
var FailToCreateSSHErrorInternal = errors.BoolTag{Key: errors.NewTagKey("fail_to_create_ssh_error_internal")}

// -2: session is down, but the server sends no confirmation of the exit status.
var NoExitStatusErrorInternal = errors.BoolTag{Key: errors.NewTagKey("no_exit_status_error_internal")}

// other internal error tag.
var OtherErrorInternal = errors.BoolTag{Key: errors.NewTagKey("other_error_internal")}

// Runner defines the type for a function that will execute a command
// on a host, and returns the result as a single line.
type Runner func(context.Context, string) (string, error)

// NewRunner returns a function of type Runner that executes a command
// on a host and returns the results as a single line. This function
// defines the specific host on which the command will be
// executed. Examples of such specific hosts can be the DUT, or the
// servo-host etc.
func (args *RunArgs) NewRunner(host string) Runner {
	runner := func(ctx context.Context, cmd string) (string, error) {
		r := args.Access.Run(ctx, host, cmd)
		exitCode := r.ExitCode
		if exitCode != 0 {
			errAnnotator := errors.Reason("runner: command %q completed with exit code %q", cmd, r.ExitCode)
			// different kinds of internal errors
			if exitCode < 0 {
				errAnnotator.Tag(SSHErrorInternal)
				if exitCode == -1 {
					errAnnotator.Tag(FailToCreateSSHErrorInternal)
				} else if exitCode == -2 {
					errAnnotator.Tag(NoExitStatusErrorInternal)
				} else if exitCode == -3 {
					errAnnotator.Tag(OtherErrorInternal)
				}
				// general linux errors
			} else if exitCode == 127 {
				errAnnotator.Tag(SSHErrorCLINotFound)
			} else {
				errAnnotator.Tag(GeneralError)
			}
			return "", errAnnotator.Err()
		}
		return strings.TrimSpace(r.Stdout), nil
	}
	return runner
}

// ParseActionArgs parses the action arguments using the splitter, and
// returns a map of the key and values. If any mal-formed action
// arguments are found their value is set to empty string in the map.
func ParseActionArgs(ctx context.Context, actionArgs []string, splitter string) map[string]string {
	argsMap := make(map[string]string)
	for _, a := range actionArgs {
		actionArg := strings.Split(a, splitter)
		log.Debug(ctx, "Parse Action Args: action arg %q", a)
		if len(actionArg) != 2 {
			log.Debug(ctx, "Parse Action Args: malformed action arg %q", a)
			argsMap[actionArg[0]] = ""
		} else {
			log.Debug(ctx, "Parse Action Args: k: %q, v: %q", actionArg[0], actionArg[1])
			argsMap[actionArg[0]] = actionArg[1]
		}
	}
	return argsMap
}

// CreateMetric creates a metric with an actionKind, and a startTime.
// It returns an action and a closer function.
//
// Intended usage:
//
//  err is managed by the containing function.
//
//  Note that it is necessary to explicitly defer evaluation of err to the
//  end of the function.
//
//  action, closer := createMetric(ctx, ...)
//  if closer != nil {
//    defer func() {
//      closer(ctx, err)
//    }
//  }
//
func createMetric(ctx context.Context, m metrics.Metrics, action *metrics.Action) (*metrics.Action, func(context.Context, error)) {
	if m == nil {
		return nil, nil
	}
	a, err := m.Create(ctx, action)
	if err != nil {
		log.Error(ctx, err.Error())
	}
	closer := func(ctx context.Context, e error) {
		if m == nil {
			return
		}
		if a == nil {
			return
		}
		// TODO(gregorynisbet): Consider strategies for multiple fail reasons.
		if e != nil {
			a.FailReason = e.Error()
		}
		_, err := m.Update(ctx, a)
		if err != nil {
			log.Error(ctx, err.Error())
		}
	}
	return a, closer
}
