// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package swarming defines an interface for interacting with swarming.
package swarming

import (
	"context"
	"net/http"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/luci/auth"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
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
	swarmingClient Client
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
func (c *rawSwarmingSkylabClient) ValidateArgs(ctx context.Context, args *request.Args) (botExists bool, rejectedTaskDims map[string]string, err error) {
	dims, err := args.StaticDimensions()
	if err != nil {
		err = errors.Annotate(err, "validate dependencies").Err()
		return
	}
	botExists, err = c.swarmingClient.BotExists(ctx, dims)
	if err != nil {
		err = errors.Annotate(err, "validate dependencies").Err()
		return
	}
	if !botExists {
		rejectedTaskDims = map[string]string{}
		for _, dim := range dims {
			rejectedTaskDims[dim.Key] = dim.Value
		}
		logging.Warningf(ctx, "Dependency validation failed for %s: no bot exists with dimensions: %v", args.Cmd.TaskName, rejectedTaskDims)
	}
	return
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

// URL is the Swarming URL of the task.
func (c *rawSwarmingSkylabClient) URL(t skylab.TaskReference) string {
	return c.swarmingClient.GetTaskURL(string(t))
}

// SwarmingTaskID is the Swarming ID of the task.
func (c *rawSwarmingSkylabClient) SwarmingTaskID(t skylab.TaskReference) string {
	return string(t)
}
