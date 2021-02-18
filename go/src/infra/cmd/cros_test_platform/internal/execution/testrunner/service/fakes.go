// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"context"
	"fmt"
	"infra/cmd/cros_test_platform/internal/execution/types"
	"infra/libs/skylab/request"

	"github.com/golang/protobuf/jsonpb"
	"github.com/google/uuid"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/luci/common/data/stringset"
)

// StubClient implements a noop Client with "reasonable" default behavior for
// all methods.
type StubClient struct{}

// Ensure we implement the promised interface.
var _ Client = StubClient{}

// ValidateArgs implements Client interface.
func (c StubClient) ValidateArgs(context.Context, *request.Args) (bool, []types.TaskDimKeyVal, error) {
	return true, nil, nil
}

// LaunchTask implements Client interface.
func (c StubClient) LaunchTask(context.Context, *request.Args) (TaskReference, error) {
	return TaskReference(unguessableString()), nil
}

// FetchResults implements Client interface.
func (c StubClient) FetchResults(context.Context, TaskReference) (*FetchResultsResponse, error) {
	return &FetchResultsResponse{
		Result:    &skylab_test_runner.Result{},
		LifeCycle: test_platform.TaskState_LIFE_CYCLE_COMPLETED,
	}, nil
}

// SwarmingTaskID implements Client interface.
func (c StubClient) SwarmingTaskID(TaskReference) string {
	return unguessableString()
}

// URL implements Client interface.
func (c StubClient) URL(TaskReference) string {
	return unguessableString()
}

// Hard-coding strings returned from the stub/fake Client implementations in
// unit-test expectations makes for unreadable, brittle tests.
//
// Make this practice impossible by returning an unstable string.
func unguessableString() string {
	return uuid.New().String()
}

// BotsAwareFakeClient implements a fake Client which is aware of the available
// bots.
//
// BotsAwareFakeClient implements ValidateArgs to be consistent with available
// bots, buts stubs out the other methods.
type BotsAwareFakeClient struct {
	StubClient

	// List of dimensions for each available bot.
	// Dimensions for each bot must be supplied as "key:value" strings.
	Bots []stringset.Set
}

// NewBotsAwareFakeClient creates a new BotsAwareFakeClient with some defaults.
func NewBotsAwareFakeClient(bots ...stringset.Set) BotsAwareFakeClient {
	// Required on all test_runner bots.
	for _, b := range bots {
		b.Add("pool:ChromeOSSkylab")
	}
	return BotsAwareFakeClient{
		Bots: bots,
	}
}

// ValidateArgs implements Client interface.
func (c BotsAwareFakeClient) ValidateArgs(ctx context.Context, args *request.Args) (bool, []types.TaskDimKeyVal, error) {
	s, err := args.StaticDimensions()
	if err != nil {
		panic(fmt.Sprintf("Failed to obtain static dimensions from %v: %s", args, err))
	}

	var rejected []types.TaskDimKeyVal
	ds := make(stringset.Set)
	for _, kv := range s {
		rejected = append(rejected, types.TaskDimKeyVal{Key: kv.Key, Val: kv.Value})
		ds.Add(fmt.Sprintf("%s:%s", kv.Key, kv.Value))
	}

	for _, b := range c.Bots {
		if ds.Difference(b).Len() == 0 {
			return true, nil, nil
		}
	}
	return false, rejected, nil
}

// StubClientWithCannedResults is a stub Client that always returns canned
// result for the FetchResults method.
type StubClientWithCannedResults struct {
	StubClient
	CannedResponses []FetchResultsResponse
}

// Ensure we implement the promised interface.
var _ Client = &StubClientWithCannedResults{}

// FetchResults implements Client interface.
func (c *StubClientWithCannedResults) FetchResults(ctx context.Context, t TaskReference) (*FetchResultsResponse, error) {
	if len(c.CannedResponses) == 0 {
		panic("ran out of canned responses!")
	}
	r := c.CannedResponses[0]
	c.CannedResponses = c.CannedResponses[1:]
	return &r, nil
}

// NewStubClientWithCannedIncompleteTasks returns a new StubWithCannedResultsClient
// where all tasks are deemed incomplete with the given lifeCycle.
//
// In particular, this means that no detailed test_runner response is available
// in the response for FetchResults.
func NewStubClientWithCannedIncompleteTasks(lifeCycle test_platform.TaskState_LifeCycle) *StubClientWithCannedResults {
	return &StubClientWithCannedResults{
		CannedResponses: repeatManyTimes(FetchResultsResponse{
			LifeCycle: lifeCycle,
			Result:    nil,
		}),
	}
}

// NewStubClientWithSuccessfulTasks returns a new StubWithCannedResultsClient
// where all tasks are deemed to have completed successfully.
func NewStubClientWithSuccessfulTasks() *StubClientWithCannedResults {
	return &StubClientWithCannedResults{
		CannedResponses: repeatManyTimes(FetchResultsResponse{
			LifeCycle: test_platform.TaskState_LIFE_CYCLE_COMPLETED,
			Result: &skylab_test_runner.Result{
				Harness: &skylab_test_runner.Result_AutotestResult{
					AutotestResult: &skylab_test_runner.Result_Autotest{
						Incomplete: false,
						TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
							{
								Name:    unguessableString(),
								Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS,
							},
						},
					},
				},
			},
		}),
	}
}

// NewStubClientWithFailedTasks returns a new StubWithCannedResultsClient
// where all tasks are deemed to have completed, but unsuccessfully.
func NewStubClientWithFailedTasks() *StubClientWithCannedResults {
	return &StubClientWithCannedResults{
		CannedResponses: repeatManyTimes(FetchResultsResponse{
			LifeCycle: test_platform.TaskState_LIFE_CYCLE_COMPLETED,
			Result: &skylab_test_runner.Result{
				Harness: &skylab_test_runner.Result_AutotestResult{
					AutotestResult: &skylab_test_runner.Result_Autotest{
						Incomplete: false,
						TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
							{
								Name:    unguessableString(),
								Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_FAIL,
							},
						},
					},
				},
			},
		}),
	}
}

var oughtToBeEnoughForAnybody = 64 // *10*1024 if rumours are to be believed.

func repeatManyTimes(r FetchResultsResponse) []FetchResultsResponse {
	rs := make([]FetchResultsResponse, oughtToBeEnoughForAnybody)
	for i := range rs {
		rs[i] = deepCopyFetchResultsResponse(r)
	}
	return rs
}

func deepCopyFetchResultsResponse(r FetchResultsResponse) FetchResultsResponse {
	m := jsonpb.Marshaler{}
	resp := FetchResultsResponse{LifeCycle: r.LifeCycle}
	if r.Result == nil {
		return resp
	}

	resp.Result = &skylab_test_runner.Result{}
	d, err := m.MarshalToString(r.Result)
	if err != nil {
		panic(fmt.Sprintf("Error when copying canned response: %s", err))
	}
	if err := jsonpb.UnmarshalString(d, resp.Result); err != nil {
		panic(fmt.Sprintf("Error when copying canned response: %s; marshalled result: %s", err, d))
	}
	return resp
}

// CallCountingClientWrapper is a Client wrapper that additionally counts the
// number of times each Client method is called.
type CallCountingClientWrapper struct {
	// Name the wrapped client to avoid accidental forwarding of method calls
	// without counting.
	Client Client

	// CallCounts is a POD value that contains the number of each time each method
	// in Client was called.
	CallCounts struct {
		ValidateArgs   int
		LaunchTask     int
		FetchResults   int
		SwarmingTaskID int
		URL            int
	}
}

// Ensure we implement the promised interface.
var _ Client = &CallCountingClientWrapper{Client: StubClient{}}

// ValidateArgs implements Client interface.
func (c *CallCountingClientWrapper) ValidateArgs(ctx context.Context, args *request.Args) (bool, []types.TaskDimKeyVal, error) {
	c.CallCounts.ValidateArgs++
	return c.Client.ValidateArgs(ctx, args)
}

// LaunchTask implements Client interface.
func (c *CallCountingClientWrapper) LaunchTask(ctx context.Context, args *request.Args) (TaskReference, error) {
	c.CallCounts.LaunchTask++
	return c.Client.LaunchTask(ctx, args)
}

// FetchResults implements Client interface.
func (c *CallCountingClientWrapper) FetchResults(ctx context.Context, t TaskReference) (*FetchResultsResponse, error) {
	c.CallCounts.FetchResults++
	return c.Client.FetchResults(ctx, t)
}

// SwarmingTaskID implements Client interface.
func (c *CallCountingClientWrapper) SwarmingTaskID(t TaskReference) string {
	c.CallCounts.SwarmingTaskID++
	return c.Client.SwarmingTaskID(t)
}

// URL implements Client interface.
func (c *CallCountingClientWrapper) URL(t TaskReference) string {
	c.CallCounts.URL++
	return c.Client.URL(t)
}

// ArgsCollectingClientWrapper collects arguments provided to the Client method
// calls before forwarding them to the wrapped Client.
type ArgsCollectingClientWrapper struct {
	// Name the wrapped client to avoid accidental forwarding of method calls
	// without counting.
	Client Client

	// Calls is a POD that contains the arguments to all Client method calls
	// through a ArgsCollectingClientWrapper.
	Calls struct {
		ValidateArgs []struct {
			Args *request.Args
		}
		LaunchTask []struct {
			Args *request.Args
		}
		FetchResults []struct {
			T TaskReference
		}
		SwarmingTaskID []struct {
			T TaskReference
		}
		URL []struct {
			T TaskReference
		}
	}
}

// Ensure we implement the promised interface.
var _ Client = &ArgsCollectingClientWrapper{Client: StubClient{}}

// ValidateArgs implements Client interface.
func (c *ArgsCollectingClientWrapper) ValidateArgs(ctx context.Context, args *request.Args) (bool, []types.TaskDimKeyVal, error) {
	c.Calls.ValidateArgs = append(c.Calls.ValidateArgs, struct {
		Args *request.Args
	}{
		Args: args,
	})
	return c.Client.ValidateArgs(ctx, args)
}

// LaunchTask implements Client interface.
func (c *ArgsCollectingClientWrapper) LaunchTask(ctx context.Context, args *request.Args) (TaskReference, error) {
	c.Calls.LaunchTask = append(c.Calls.LaunchTask, struct {
		Args *request.Args
	}{
		Args: args,
	})
	return c.Client.LaunchTask(ctx, args)
}

// FetchResults implements Client interface.
func (c *ArgsCollectingClientWrapper) FetchResults(ctx context.Context, t TaskReference) (*FetchResultsResponse, error) {
	c.Calls.FetchResults = append(c.Calls.FetchResults, struct {
		T TaskReference
	}{
		T: t,
	})
	return c.Client.FetchResults(ctx, t)
}

// SwarmingTaskID implements Client interface.
func (c *ArgsCollectingClientWrapper) SwarmingTaskID(t TaskReference) string {
	c.Calls.SwarmingTaskID = append(c.Calls.SwarmingTaskID, struct {
		T TaskReference
	}{
		T: t,
	})
	return c.Client.SwarmingTaskID(t)
}

// URL implements Client interface.
func (c *ArgsCollectingClientWrapper) URL(t TaskReference) string {
	c.Calls.URL = append(c.Calls.URL, struct {
		T TaskReference
	}{
		T: t,
	})
	return c.Client.URL(t)
}
