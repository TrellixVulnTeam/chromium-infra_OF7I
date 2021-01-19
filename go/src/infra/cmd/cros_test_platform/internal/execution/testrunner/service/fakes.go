// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"context"
	"fmt"
	"infra/libs/skylab/request"

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
func (c StubClient) ValidateArgs(context.Context, *request.Args) (bool, map[string]string, error) {
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
func (c BotsAwareFakeClient) ValidateArgs(ctx context.Context, args *request.Args) (bool, map[string]string, error) {
	s, err := args.StaticDimensions()
	if err != nil {
		panic(fmt.Sprintf("Failed to obtain static dimensions from %v: %s", args, err))
	}

	rejected := make(map[string]string)
	ds := make(stringset.Set)
	for _, kv := range s {
		rejected[kv.Key] = kv.Value
		ds.Add(fmt.Sprintf("%s:%s", kv.Key, kv.Value))
	}

	for _, b := range c.Bots {
		if ds.Difference(b).Len() == 0 {
			return true, nil, nil
		}
	}
	return false, rejected, nil
}

// ClientMethodCallCounts is a POD that contains the number of each time each
// method in Client was called.
type ClientMethodCallCounts struct {
	ValidateArgs   int
	LaunchTask     int
	FetchResults   int
	SwarmingTaskID int
	URL            int
}

// ClientCallCountingWrapper is a Client wrapper that additionally counts the
// number of times each Client method is called.
type ClientCallCountingWrapper struct {
	// Name the wrapped client to avoid accidental forwarding of method calls
	// without counting.
	Client Client

	counts ClientMethodCallCounts
}

// Ensure we implement the promised interface.
var _ Client = ClientCallCountingWrapper{Client: StubClient{}}

// ValidateArgs implements Client interface.
func (c ClientCallCountingWrapper) ValidateArgs(ctx context.Context, args *request.Args) (bool, map[string]string, error) {
	c.counts.ValidateArgs++
	return c.Client.ValidateArgs(ctx, args)
}

// LaunchTask implements Client interface.
func (c ClientCallCountingWrapper) LaunchTask(ctx context.Context, args *request.Args) (TaskReference, error) {
	c.counts.LaunchTask++
	return c.Client.LaunchTask(ctx, args)
}

// FetchResults implements Client interface.
func (c ClientCallCountingWrapper) FetchResults(ctx context.Context, t TaskReference) (*FetchResultsResponse, error) {
	c.counts.FetchResults++
	return c.Client.FetchResults(ctx, t)
}

// SwarmingTaskID implements Client interface.
func (c ClientCallCountingWrapper) SwarmingTaskID(t TaskReference) string {
	c.counts.SwarmingTaskID++
	return c.Client.SwarmingTaskID(t)
}

// URL implements Client interface.
func (c ClientCallCountingWrapper) URL(t TaskReference) string {
	c.counts.URL++
	return c.Client.URL(t)
}

// MethodCallCounts the number of times the Client methods have been called so
// far on the wrapped Client.
func (c ClientCallCountingWrapper) MethodCallCounts() ClientMethodCallCounts {
	return c.counts
}
