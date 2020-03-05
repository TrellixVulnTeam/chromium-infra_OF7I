// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package bb implements a skylab.Client using calls to BuildBucket.
package bb

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/luci/auth"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cmd/cros_test_platform/internal/execution/skylab"
	"infra/cmd/cros_test_platform/internal/execution/swarming"
	"infra/libs/skylab/request"
)

type build struct {
	bbID int64
}

type bbSkylabClient struct {
	swarmingClient swarming.Client
	bbClient       buildbucket_pb.BuildsClient
	knownBuilds    map[skylab.TaskReference]build
}

// NewSkylabClient creates a new skylab.Client.
func NewSkylabClient(ctx context.Context, cfg *config.Config) (skylab.Client, error) {
	sc, err := swarming.NewClient(ctx, cfg.SkylabSwarming)
	if err != nil {
		return nil, errors.Annotate(err, "create Skylab client").Err()
	}
	bbc, err := newBBClient(ctx, cfg.TestRunner.Buildbucket)
	if err != nil {
		return nil, errors.Annotate(err, "create Skylab client").Err()
	}
	return &bbSkylabClient{
		swarmingClient: sc,
		bbClient:       bbc,
		knownBuilds:    make(map[skylab.TaskReference]build),
	}, nil
}

func newBBClient(ctx context.Context, cfg *config.Config_Buildbucket) (buildbucket_pb.BuildsClient, error) {
	hClient, err := httpClient(ctx)
	if err != nil {
		return nil, errors.Annotate(err, "create buildbucket client").Err()
	}
	pClient := &prpc.Client{
		C:    hClient,
		Host: cfg.Host,
	}
	return buildbucket_pb.NewBuildsPRPCClient(pClient), nil
}

// TODO(crbug.com/1058585): dedupe with swarming.httpClient.
func httpClient(ctx context.Context) (*http.Client, error) {
	a := auth.NewAuthenticator(ctx, auth.SilentLogin, auth.Options{
		Scopes: []string{auth.OAuthScopeEmail},
	})
	h, err := a.Client()
	if err != nil {
		return nil, errors.Annotate(err, "create http client").Err()
	}
	return h, nil
}

// ValidateArgs checks whether this test has dependencies satisfied by
// at least one Skylab bot.
//
// Any changes to this implementation should be also reflected in
// rawSwarmingSkylabClient.ValidateArgs
// TODO(crbug.com/1033287): Remove the rawSwarmingSkylabClient implementation.
func (c *bbSkylabClient) ValidateArgs(ctx context.Context, args *request.Args) (bool, error) {
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
		logging.Warningf(ctx, "Dependency validation failed for %s: no bot exists with dimensions: %s", args.TestRunnerRequest.GetTest().GetAutotest().GetName(), strings.Join(ds, ", "))
	}
	return exists, nil
}

// LaunchTask sends an RPC request to start the task.
func (c *bbSkylabClient) LaunchTask(ctx context.Context, args *request.Args) (skylab.TaskReference, error) {
	req, err := args.NewBBRequest()
	if err != nil {
		return "", errors.Annotate(err, "launch task for %s", args.TestRunnerRequest.GetTest().GetAutotest().GetName()).Err()
	}
	resp, err := c.bbClient.ScheduleBuild(ctx, req)
	if err != nil {
		return "", errors.Annotate(err, "launch task for %s", args.TestRunnerRequest.GetTest().GetAutotest().GetName()).Err()
	}
	tr := skylab.NewTaskReference()
	c.knownBuilds[tr] = build{
		bbID: resp.Id,
	}
	return tr, nil
}

// FetchResults stub.
func (c *bbSkylabClient) FetchResults(ctx context.Context, t skylab.TaskReference) (*skylab.FetchResultsResponse, error) {
	panic("Not yet implemented.")
}

// URL stub.
func (c *bbSkylabClient) URL(t skylab.TaskReference) string {
	panic("Not yet implemented.")
}

// SwarmingTaskID stub.
func (c *bbSkylabClient) SwarmingTaskID(t skylab.TaskReference) string {
	panic("Not yet implemented.")
}
