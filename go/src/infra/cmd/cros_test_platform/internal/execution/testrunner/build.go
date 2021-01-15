// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package testrunner exposes a way to interact with test_runner builds.
package testrunner

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/common"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	trservice "infra/cmd/cros_test_platform/internal/execution/testrunner/service"
	"infra/libs/skylab/request"
)

// ArgsGenerator is used to generate arguments to buildbucket / swarming.
type ArgsGenerator interface {
	// Generate arguments for swarming task or buildbucket build.
	GenerateArgs(ctx context.Context) (request.Args, error)
	// Check the internal consistency of of the generator arguments.
	CheckConsistency() error
}

// InvalidDependencies tag indicates that an error was caused because
// swarming dependencies for a task were invalid.
var InvalidDependencies = errors.BoolTag{Key: errors.NewTagKey("invalid test dependencies")}

// ValidateDependencies checks whether this test has dependencies satisfied by
// at least one Skylab bot.
//
// Returns nil if the dependencies are valid and satisfiable.
// Returns an error tagged with InvalidDependencies tag if provided dependencies
// are invalid.
// Optionally returns a map of the unsatisfiable dependencies.
//
// Errors encountered in dependency validation are returned as generic errors.
func ValidateDependencies(ctx context.Context, c trservice.Client, argsGenerator ArgsGenerator) (map[string]string, error) {
	if err := argsGenerator.CheckConsistency(); err != nil {
		logging.Warningf(ctx, "Dependency validation failed: %s.", err)
		return nil, InvalidDependencies.Apply(err)
	}

	args, err := argsGenerator.GenerateArgs(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "validate dependencies").Err()
	}
	ok, rejected, err := c.ValidateArgs(ctx, &args)
	if err != nil {
		return nil, errors.Annotate(err, "validate dependencies").Err()
	}
	if !ok {
		return rejected, errors.Reason("no swarming bots with requested dimensions").Tag(InvalidDependencies).Err()
	}
	return nil, nil
}

// Build represents an individual test_runner build.
type Build struct {
	argsGenerator  ArgsGenerator
	args           request.Args
	result         *skylab_test_runner.Result
	lifeCycle      test_platform.TaskState_LifeCycle
	swarmingTaskID string
	taskReference  trservice.TaskReference
	url            string
}

// NewBuild creates a new test_runner build.
func NewBuild(ctx context.Context, c trservice.Client, argsGenerator ArgsGenerator) (*Build, error) {
	t := &Build{argsGenerator: argsGenerator}
	args, err := t.argsGenerator.GenerateArgs(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "new task for %s", t.name()).Err()
	}
	ref, err := c.LaunchTask(ctx, &args)
	if err != nil {
		return nil, errors.Annotate(err, "new task for %s", t.name()).Err()
	}
	t.args = args
	t.taskReference = ref
	t.lifeCycle = test_platform.TaskState_LIFE_CYCLE_PENDING
	t.url = c.URL(ref)
	logging.Infof(ctx, "Launched attempt for %s as task %s", t.name(), t.url)
	return t, nil
}

// name is the build name as it is displayed in the UI.
func (b *Build) name() string {
	return b.args.Cmd.TaskName
}

func (b *Build) autotestResult() *skylab_test_runner.Result_Autotest {
	return b.result.GetAutotestResult()
}

// LifeCyclesWithResults lists all task states which have a chance of producing
// test cases results. E.g. this excludes killed tasks.
var LifeCyclesWithResults = map[test_platform.TaskState_LifeCycle]bool{
	test_platform.TaskState_LIFE_CYCLE_COMPLETED: true,
}

// The life cycles that are not final.
var transientLifeCycles = map[test_platform.TaskState_LifeCycle]bool{
	test_platform.TaskState_LIFE_CYCLE_PENDING: true,
	test_platform.TaskState_LIFE_CYCLE_RUNNING: true,
}

// Completed returns whether the current build is complete.
func (b *Build) Completed() bool {
	return !transientLifeCycles[b.lifeCycle]
}

// verdict aggregates the information about test cases contained in a task into
// a single verdict.
func (b *Build) verdict() test_platform.TaskState_Verdict {
	if !b.Completed() {
		return test_platform.TaskState_VERDICT_UNSPECIFIED
	}
	if b.autotestResult() == nil {
		return test_platform.TaskState_VERDICT_UNSPECIFIED
	}
	if b.autotestResult().Incomplete {
		return test_platform.TaskState_VERDICT_FAILED
	}

	// By default (if no test cases ran), then there is no verdict.
	verdict := test_platform.TaskState_VERDICT_NO_VERDICT
	for _, c := range b.autotestResult().GetTestCases() {
		switch c.Verdict {
		case skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL:
			// Any case failing means the flat verdict is a failure.
			return test_platform.TaskState_VERDICT_FAILED
		case skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS:
			// Otherwise, at least 1 passing verdict means a pass.
			verdict = test_platform.TaskState_VERDICT_PASSED
		default: // VERDICT_UNDEFINED and VERDICT_NO_VERDICT
			// Treat as no-op and do not affect flat verdict.
		}
	}
	return verdict
}

// Refresh fetches and updates the state of the given build from the test_runner
// service.
func (b *Build) Refresh(ctx context.Context, c trservice.Client) error {
	resp, err := c.FetchResults(ctx, b.taskReference)

	if err != nil {
		return errors.Annotate(err, "refresh task").Err()
	}

	b.swarmingTaskID = c.SwarmingTaskID(b.taskReference)
	b.lifeCycle = resp.LifeCycle

	// The build is still running.
	if transientLifeCycles[b.lifeCycle] {
		return nil
	}

	b.result = resp.Result
	// If the autotest result is missing, treat the build as incomplete.
	if b.autotestResult() == nil {
		b.result = &skylab_test_runner.Result{
			Harness: &skylab_test_runner.Result_AutotestResult{
				AutotestResult: &skylab_test_runner.Result_Autotest{Incomplete: true},
			},
		}
	}

	return nil
}

var liftTestCaseRunnerVerdict = map[skylab_test_runner.Result_Autotest_TestCase_Verdict]test_platform.TaskState_Verdict{
	skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS: test_platform.TaskState_VERDICT_PASSED,
	skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL: test_platform.TaskState_VERDICT_FAILED,
}

// testCases unpacks test cases contained in the results of a build.
func (b *Build) testCases() []*steps.ExecuteResponse_TaskResult_TestCaseResult {
	tcs := b.autotestResult().GetTestCases()
	if len(tcs) == 0 {
		// Prefer a nil over an empty slice since it's the proto default.
		return nil
	}
	ret := make([]*steps.ExecuteResponse_TaskResult_TestCaseResult, len(tcs))
	for i, tc := range tcs {
		ret[i] = &steps.ExecuteResponse_TaskResult_TestCaseResult{
			Name:                 tc.Name,
			Verdict:              liftTestCaseRunnerVerdict[tc.Verdict],
			HumanReadableSummary: tc.HumanReadableSummary,
		}
	}
	return ret
}

// Result constructs a TaskResults out of the last known state of the Build.
//
// In order to get the latest result, FetchResult needs to be called first.
func (b *Build) Result() *steps.ExecuteResponse_TaskResult {
	r := &steps.ExecuteResponse_TaskResult{
		Name: b.name(),
		State: &test_platform.TaskState{
			LifeCycle: b.lifeCycle,
			Verdict:   b.verdict(),
		},
		TaskUrl:   b.url,
		TestCases: b.testCases(),
	}
	if ld := b.result.GetLogData(); ld != nil {
		r.LogData = proto.Clone(ld).(*common.TaskLogData)
		// Clients use r.LogUrl to link to logs as it pre-dates the introduction
		// of r.LogData.StainlessUrl
		r.LogUrl = r.LogData.StainlessUrl
	}
	return r
}

// Retry creates a new build to retry the current build.
//
// Retry does not check whether the current build is complete.
func (b *Build) Retry(ctx context.Context, c trservice.Client) (*Build, error) {
	return NewBuild(ctx, c, b.argsGenerator)
}

// TaskURL returns the URL to the buildbucket build for this task.
func (b *Build) TaskURL() string {
	return b.url
}
