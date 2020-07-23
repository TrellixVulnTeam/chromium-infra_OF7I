// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package swarming defines an interface for interacting with swarming.
package swarming

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/luci/auth"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/isolated"
	"go.chromium.org/luci/common/isolatedclient"
	"go.chromium.org/luci/common/logging"

	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/cmd/cros_test_platform/internal/execution/isolate/getter"
	"infra/cmd/cros_test_platform/internal/execution/skylab"
	"infra/libs/skylab/request"
	"infra/libs/skylab/swarming"
)

// Client defines an interface used to interact with a swarming service.
type Client interface {
	GetTaskURL(taskID string) string
	BotExists(context.Context, []*swarming_api.SwarmingRpcsStringPair) (bool, error)
	CreateTask(context.Context, *swarming_api.SwarmingRpcsNewTaskRequest) (*swarming_api.SwarmingRpcsTaskRequestMetadata, error)
	GetResults(ctx context.Context, IDs []string) ([]*swarming_api.SwarmingRpcsTaskResult, error)
}

// swarming.Client is the reference implementation of the Client interface.
var _ Client = &swarming.Client{}

type rawSwarmingSkylabClient struct {
	isolateGetter  isolate.GetterFactory
	swarmingClient Client
}

// NewSkylabClient creates a new skylab.Client.
func NewSkylabClient(ctx context.Context, cfg *config.Config) (skylab.Client, error) {
	sc, err := NewClient(ctx, cfg.SkylabSwarming)
	if err != nil {
		return nil, errors.Annotate(err, "create Skylab client").Err()
	}
	return &rawSwarmingSkylabClient{
		isolateGetter:  getterFactory(cfg.SkylabIsolate),
		swarmingClient: sc,
	}, nil
}

func getterFactory(conf *config.Config_Isolate) isolate.GetterFactory {
	return func(ctx context.Context, server string) (isolate.Getter, error) {
		hClient, err := httpClient(ctx, conf.AuthJsonPath)
		if err != nil {
			return nil, err
		}

		isolateClient := isolatedclient.NewClient(server, isolatedclient.WithAuthClient(hClient))

		return getter.New(isolateClient), nil
	}
}

func httpClient(ctx context.Context, authJSONPath string) (*http.Client, error) {
	options := auth.Options{
		ServiceAccountJSONPath: authJSONPath,
		Scopes:                 []string{auth.OAuthScopeEmail},
	}
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, options)
	h, err := a.Client()
	if err != nil {
		return nil, errors.Annotate(err, "create http client").Err()
	}
	return h, nil
}

// NewClient creates a new Client.
func NewClient(ctx context.Context, c *config.Config_Swarming) (*swarming.Client, error) {
	logging.Infof(ctx, "Creating swarming client from config %v", c)
	hClient, err := httpClient(ctx, c.AuthJsonPath)
	if err != nil {
		return nil, err
	}

	client, err := swarming.NewClient(hClient, c.Server)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// ValidateArgs checks whether this test has dependencies satisfied by
// at least one Skylab bot.
//
// Any changes to this implementation should be also reflected in
// bbSkylabClient.ValidateArgs
// TODO(crbug.com/1033287): Remove this implementation.
func (c *rawSwarmingSkylabClient) ValidateArgs(ctx context.Context, args *request.Args) (bool, error) {
	dims, err := args.StaticDimensions()
	if err != nil {
		return false, errors.Annotate(err, "validate dependencies").Err()
	}
	exists, err := c.swarmingClient.BotExists(ctx, dims)
	if err != nil {
		return false, errors.Annotate(err, "validate dependencies").Err()
	}
	if !exists {
		var ds []string
		for _, dim := range dims {
			ds = append(ds, fmt.Sprintf("%+v", dim))
		}
		logging.Warningf(ctx, "Dependency validation failed for %s: no bot exists with dimensions: %s", args.Cmd.TaskName, strings.Join(ds, ", "))

	}
	return exists, nil
}

// LaunchTask sends an RPC request to start the task.
func (c *rawSwarmingSkylabClient) LaunchTask(ctx context.Context, args *request.Args) (skylab.TaskReference, error) {
	req, err := args.SwarmingNewTaskRequest()
	if err != nil {
		return "", errors.Annotate(err, "launch attempt for %s", args.Cmd.TaskName).Err()
	}
	resp, err := c.swarmingClient.CreateTask(ctx, req)
	if err != nil {
		return "", errors.Annotate(err, "launch attempt for %s", args.Cmd.TaskName).Err()
	}
	return skylab.TaskReference(resp.TaskId), nil
}

// FetchResults fetches the latest state and results of the given task.
func (c *rawSwarmingSkylabClient) FetchResults(ctx context.Context, t skylab.TaskReference) (*skylab.FetchResultsResponse, error) {
	id := string(t)
	results, err := c.swarmingClient.GetResults(ctx, []string{id})
	if err != nil {
		return nil, errors.Annotate(err, "fetch results for task %s", t).Err()
	}
	result, err := unpackResult(results, id)
	if err != nil {
		return nil, errors.Annotate(err, "fetch results for task %s", t).Err()
	}

	lc, err := asLifeCycle(result.State)
	if err != nil {
		return nil, errors.Annotate(err, "fetch results for task %s", t).Err()
	}

	if !skylab.LifeCyclesWithResults[lc] {
		return &skylab.FetchResultsResponse{LifeCycle: lc}, nil
	}

	r, err := extractResult(ctx, result, c.isolateGetter)
	if err != nil {
		logging.Infof(ctx, "failed to fetch autotest results for task %s due to error '%s', treating its results as incomplete (failure)", id, err.Error())
		return &skylab.FetchResultsResponse{LifeCycle: lc}, nil
	}

	return &skylab.FetchResultsResponse{
		LifeCycle: lc,
		Result:    r,
	}, nil
}

// URL is the Swarming URL of the task.
func (c *rawSwarmingSkylabClient) URL(t skylab.TaskReference) string {
	return c.swarmingClient.GetTaskURL(string(t))
}

// SwarmingTaskID is the Swarming ID of the task.
func (c *rawSwarmingSkylabClient) SwarmingTaskID(t skylab.TaskReference) string {
	return string(t)
}

func unpackResult(results []*swarming_api.SwarmingRpcsTaskResult, taskID string) (*swarming_api.SwarmingRpcsTaskResult, error) {
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

func extractResult(ctx context.Context, sResult *swarming_api.SwarmingRpcsTaskResult, gf isolate.GetterFactory) (*skylab_test_runner.Result, error) {
	if sResult == nil {
		return nil, errors.Reason("get result: nil swarming result").Err()
	}

	taskID := sResult.TaskId
	outputRef := sResult.OutputsRef
	if outputRef == nil {
		return nil, fmt.Errorf("get result for task %s: task has no output ref", taskID)
	}

	getter, err := gf(ctx, outputRef.Isolatedserver)
	if err != nil {
		return nil, errors.Annotate(err, "get result").Err()
	}

	logging.Infof(ctx, "fetching result for task %s from isolate ref %+v", taskID, outputRef)
	content, err := getter.GetFile(ctx, isolated.HexDigest(outputRef.Isolated), resultsFileName)
	if err != nil {
		return nil, errors.Annotate(err, "get result for task %s", taskID).Err()
	}

	var r skylab_test_runner.Result

	err = jsonpb.Unmarshal(bytes.NewReader(content), &r)
	if err != nil {
		return nil, errors.Annotate(err, "get result for task %s", taskID).Err()
	}

	return &r, nil
}
