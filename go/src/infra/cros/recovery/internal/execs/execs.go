// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package execs provides collection of execution functions for actions and ability to execute them.
package execs

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/logger"
	"infra/cros/recovery/logger/metrics"
	"infra/cros/recovery/tlw"
)

// ExecFunction represents an execution function of the action.
// The single exec can be associated with one or more actions.
type ExecFunction func(ctx context.Context, i *ExecInfo) error

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

// RunArgs holds plan input arguments.
//
// Keep this type up to date with recovery.go:RunArgs .
// Also update recovery.go:runDUTPlans .
//
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
	// SwarmingTaskID is the ID of the swarming task we're running under.
	SwarmingTaskID string
	// BuildbucketID is the ID of the buildbucket build we're running under.
	BuildbucketID string
	// LogRoot is an absolute path to a directory that contains logs.
	LogRoot string
}

// ExecInfo holds all data required to run exec.
// The struct created every time new for each exec run.
type ExecInfo struct {
	RunArgs *RunArgs
	// Name of exec.
	Name string
	// Extra arguments specified per action for exec.
	ActionArgs []string
	// Timeout specified per action.
	ActionTimeout time.Duration
}

// Run runs exec function provided by this package by name.
func Run(ctx context.Context, ei *ExecInfo) error {
	e, ok := knownExecMap[ei.Name]
	if !ok {
		return errors.Reason("exec %q: not found", ei.Name).Err()
	}
	return e(ctx, ei)
}

// Exist check if exec function with name is present.
func Exist(name string) bool {
	_, ok := knownExecMap[name]
	return ok
}
