// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"bytes"
	"context"
	"fmt"
	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/cmd/cros_test_platform/internal/execution/swarming"
	"infra/libs/skylab/request"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/isolated"
	"go.chromium.org/luci/common/logging"
)

// TaskReference is an opaque way to identify a task.
type TaskReference struct {
	swarmingClient swarming.Client
	isolateGetter  isolate.GetterFactory
	swarmingTaskID string
	url            string
}

// FetchResultsResponse is an implementation-independent container for
// information about running and finished tasks.
type FetchResultsResponse struct {
	Result    *skylab_test_runner.Result
	LifeCycle test_platform.TaskState_LifeCycle
}

// Client bundles local interfaces to various remote services used by Runner.
type Client struct {
	Swarming      swarming.Client
	IsolateGetter isolate.GetterFactory
}

// ValidateArgs checks whether this test has dependencies satisfied by
// at least one Skylab bot.
func (c *Client) ValidateArgs(ctx context.Context, args *request.Args) (bool, error) {
	dims, err := args.StaticDimensions()
	if err != nil {
		return false, errors.Annotate(err, "validate dependencies").Err()
	}
	exists, err := c.Swarming.BotExists(ctx, dims)
	if err != nil {
		return false, errors.Annotate(err, "validate dependencies").Err()
	}
	if !exists {
		logging.Warningf(ctx, "Dependency validation failed for %s: no bot exists with dimensions %+v.", args.Cmd.TaskName, dims)
	}
	return exists, nil
}

// LaunchTask sends an RPC request to start the task. It returns without
// waiting for the task to start.
func (c *Client) LaunchTask(ctx context.Context, args *request.Args) (*TaskReference, error) {
	req, err := args.SwarmingNewTaskRequest()
	if err != nil {
		return nil, errors.Annotate(err, "launch attempt for %s", args.Cmd.TaskName).Err()
	}
	resp, err := c.Swarming.CreateTask(ctx, req)
	if err != nil {
		return nil, errors.Annotate(err, "launch attempt for %s", args.Cmd.TaskName).Err()
	}
	return &TaskReference{
		isolateGetter:  c.IsolateGetter,
		swarmingClient: c.Swarming,
		swarmingTaskID: resp.TaskId,
		url:            c.Swarming.GetTaskURL(resp.TaskId),
	}, nil
}

// Tasks with these life cycles contain test results.
// E.g. this excludes killed tasks as they have no chance to produce results.
var lifeCyclesWithResults = map[test_platform.TaskState_LifeCycle]bool{
	test_platform.TaskState_LIFE_CYCLE_COMPLETED: true,
}

// FetchResults fetches the latest state and results of the given task.
func (t *TaskReference) FetchResults(ctx context.Context) (*FetchResultsResponse, error) {
	results, err := t.swarmingClient.GetResults(ctx, []string{t.swarmingTaskID})
	if err != nil {
		return nil, errors.Annotate(err, "fetch results for task %s", t.URL()).Err()
	}
	result, err := extractSingleResultWithID(results, t.swarmingTaskID)
	if err != nil {
		return nil, errors.Annotate(err, "fetch results for task %s", t.URL()).Err()
	}

	lc, err := swarming.AsLifeCycle(result.State)
	if err != nil {
		return nil, errors.Annotate(err, "fetch results for task %s", t.URL()).Err()
	}

	if !lifeCyclesWithResults[lc] {
		return &FetchResultsResponse{LifeCycle: lc}, nil
	}

	r, err := downloadTestRunnerResult(ctx, result, t.isolateGetter)
	if err != nil {
		// This error may happen if test task exited abnormally, so it is
		// handled similarly to a cancelled task rather than a Swarming error.
		logging.Debugf(ctx, "failed to fetch autotest results for task %s due to error '%s', treating its results as incomplete (failure)", t.URL(), err.Error())
		return &FetchResultsResponse{LifeCycle: lc}, nil
	}

	return &FetchResultsResponse{
		LifeCycle: lc,
		Result:    r,
	}, nil
}

// URL is the Swarming URL of the task.
func (t *TaskReference) URL() string {
	return t.url
}

// SwarmingTaskID is the Swarming ID of the task.
func (t *TaskReference) SwarmingTaskID() string {
	return t.swarmingTaskID
}

func extractSingleResultWithID(results []*swarming_api.SwarmingRpcsTaskResult, taskID string) (*swarming_api.SwarmingRpcsTaskResult, error) {
	if len(results) != 1 {
		return nil, errors.Reason("expected 1 result for task id %s, got %d", taskID, len(results)).Err()
	}

	result := results[0]
	if result.TaskId != taskID {
		return nil, errors.Reason("expected result for task id %s, got %s", taskID, result.TaskId).Err()
	}

	return result, nil
}

const resultsFileName = "results.json"

func downloadTestRunnerResult(ctx context.Context, sResult *swarming_api.SwarmingRpcsTaskResult, gf isolate.GetterFactory) (*skylab_test_runner.Result, error) {
	if sResult == nil {
		return nil, errors.Reason("download result: nil swarming result").Err()
	}

	taskID := sResult.TaskId
	outputRef := sResult.OutputsRef
	if outputRef == nil {
		return nil, fmt.Errorf("download result for task %s: task has no output ref", taskID)
	}

	getter, err := gf(ctx, outputRef.Isolatedserver)
	if err != nil {
		return nil, errors.Annotate(err, "download result for task %s", taskID).Err()
	}

	logging.Debugf(ctx, "fetching result for task %s from isolate ref %+v", taskID, outputRef)
	content, err := getter.GetFile(ctx, isolated.HexDigest(outputRef.Isolated), resultsFileName)
	if err != nil {
		return nil, errors.Annotate(err, "download result for task %s", taskID).Err()
	}

	var r skylab_test_runner.Result

	err = jsonpb.Unmarshal(bytes.NewReader(content), &r)
	if err != nil {
		return nil, errors.Annotate(err, "download result for task %s", taskID).Err()
	}

	return &r, nil
}
