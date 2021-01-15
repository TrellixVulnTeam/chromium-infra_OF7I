// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package service implements a skylab.Client using calls to BuildBucket.
package service

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/config"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	"go.chromium.org/luci/auth"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/genproto/protobuf/field_mask"

	"infra/libs/skylab/request"
	"infra/libs/skylab/swarming"
)

// TaskReference is an implementation-independent way to identify test_runner tasks.
type TaskReference string

// NewTaskReference creates a unique task reference.
func NewTaskReference() TaskReference {
	return TaskReference(uuid.New().String())
}

// FetchResultsResponse is an implementation-independent container for
// information about running and finished tasks.
type FetchResultsResponse struct {
	Result    *skylab_test_runner.Result
	LifeCycle test_platform.TaskState_LifeCycle
}

// Client defines an interface used to interact with test_runner as a service.
type Client interface {
	// ValidateArgs validates that a test_runner build can be created with
	// the give arguments.
	ValidateArgs(context.Context, *request.Args) (bool, map[string]string, error)

	// LaunchTask creates a new test_runner task with the given arguments.
	LaunchTask(context.Context, *request.Args) (TaskReference, error)

	// FetchResults fetches results for a previously launched test_runner task.
	FetchResults(context.Context, TaskReference) (*FetchResultsResponse, error)

	// SwarmingTaskID returns the swarming task ID for a test_runner build.
	SwarmingTaskID(TaskReference) string

	// URL returns a canonical URL for a test_runner build.
	URL(TaskReference) string
}

type task struct {
	bbID           int64
	swarmingTaskID string
}

// clientImpl is a concrete Client implementation to interact with
// test_runner as a service.
type clientImpl struct {
	swarmingClient swarmingClient
	bbClient       buildbucket_pb.BuildsClient
	builder        *buildbucket_pb.BuilderID
	knownTasks     map[TaskReference]*task
}

// Ensure we satisfy the promised interface.
var _ Client = &clientImpl{}

type swarmingClient interface {
	BotExists(context.Context, []*swarming_api.SwarmingRpcsStringPair) (bool, error)
}

// NewClient creates a concrete instance of a Client.
func NewClient(ctx context.Context, cfg *config.Config) (Client, error) {
	sc, err := newSwarmingClient(ctx, cfg.SkylabSwarming)
	if err != nil {
		return nil, errors.Annotate(err, "create test_runner service client").Err()
	}
	bbc, err := newBBClient(ctx, cfg.TestRunner.Buildbucket)
	if err != nil {
		return nil, errors.Annotate(err, "create test_runner service client").Err()
	}
	return &clientImpl{
		swarmingClient: sc,
		bbClient:       bbc,
		builder: &buildbucket_pb.BuilderID{
			Project: cfg.TestRunner.Buildbucket.Project,
			Bucket:  cfg.TestRunner.Buildbucket.Bucket,
			Builder: cfg.TestRunner.Buildbucket.Builder,
		},
		knownTasks: make(map[TaskReference]*task),
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

// TODO(crbug.com/1115207): dedupe with swarmingHTTPClient.
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

func newSwarmingClient(ctx context.Context, c *config.Config_Swarming) (*swarming.Client, error) {
	logging.Infof(ctx, "Creating swarming client from config %v", c)
	hClient, err := swarmingHTTPClient(ctx, c.AuthJsonPath)
	if err != nil {
		return nil, err
	}

	client, err := swarming.NewClient(hClient, c.Server)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// TODO(crbug.com/1115207): dedupe with httpClient
func swarmingHTTPClient(ctx context.Context, authJSONPath string) (*http.Client, error) {
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

// ValidateArgs checks whether this test has dependencies satisfied by
// at least one Skylab bot.
//
// Any changes to this implementation should be also reflected in
// rawSwarmingSkylabClient.ValidateArgs
// TODO(crbug.com/1033287): Remove the rawSwarmingSkylabClient implementation.
func (c *clientImpl) ValidateArgs(ctx context.Context, args *request.Args) (botExists bool, rejectedTaskDims map[string]string, err error) {
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
		logging.Warningf(ctx, "Dependency validation failed for %s: no bot exists with dimensions: %v", args.TestRunnerRequest.GetTest().GetAutotest().GetName(), rejectedTaskDims)
	}
	return
}

// LaunchTask sends an RPC request to start the task.
func (c *clientImpl) LaunchTask(ctx context.Context, args *request.Args) (TaskReference, error) {
	req, err := args.NewBBRequest(c.builder)
	if err != nil {
		return "", errors.Annotate(err, "launch task for %s", args.TestRunnerRequest.GetTest().GetAutotest().GetName()).Err()
	}
	resp, err := c.bbClient.ScheduleBuild(ctx, req)
	if err != nil {
		return "", errors.Annotate(err, "launch task for %s", args.TestRunnerRequest.GetTest().GetAutotest().GetName()).Err()
	}
	tr := NewTaskReference()
	c.knownTasks[tr] = &task{
		bbID: resp.Id,
	}
	return tr, nil
}

// getBuildFieldMask is the list of buildbucket fields that are needed.
var getBuildFieldMask = []string{
	"id",
	"infra.swarming.task_id",
	// Build details are parsed from the build's output properties.
	"output.properties",
	// Build status is used to determine whether the build is complete.
	"status",
}

// FetchResults fetches the latest state and results of the given task.
func (c *clientImpl) FetchResults(ctx context.Context, t TaskReference) (*FetchResultsResponse, error) {
	task, ok := c.knownTasks[t]
	if !ok {
		return nil, errors.Reason("fetch results: could not find task among launched tasks").Err()
	}
	req := &buildbucket_pb.GetBuildRequest{
		Id:     task.bbID,
		Fields: &field_mask.FieldMask{Paths: getBuildFieldMask},
	}
	b, err := c.bbClient.GetBuild(ctx, req)
	if err != nil {
		return nil, errors.Annotate(err, "fetch results for build %d", task.bbID).Err()
	}

	task.swarmingTaskID = b.GetInfra().GetSwarming().GetTaskId()

	lc := bbStatusToLifeCycle[b.Status]
	if !lifeCyclesWithResults[lc] {
		return &FetchResultsResponse{LifeCycle: lc}, nil
	}

	res, err := extractResult(b)
	if err != nil {
		return nil, errors.Annotate(err, "fetch results for build %d", task.bbID).Err()
	}

	return &FetchResultsResponse{
		Result:    res,
		LifeCycle: lc,
	}, nil
}

// lifeCyclesWithResults lists all task states which have a chance of producing
// test cases results. E.g. this excludes killed tasks.
var lifeCyclesWithResults = map[test_platform.TaskState_LifeCycle]bool{
	test_platform.TaskState_LIFE_CYCLE_COMPLETED: true,
}

var bbStatusToLifeCycle = map[buildbucket_pb.Status]test_platform.TaskState_LifeCycle{
	buildbucket_pb.Status_SCHEDULED:     test_platform.TaskState_LIFE_CYCLE_PENDING,
	buildbucket_pb.Status_STARTED:       test_platform.TaskState_LIFE_CYCLE_RUNNING,
	buildbucket_pb.Status_SUCCESS:       test_platform.TaskState_LIFE_CYCLE_COMPLETED,
	buildbucket_pb.Status_FAILURE:       test_platform.TaskState_LIFE_CYCLE_COMPLETED,
	buildbucket_pb.Status_INFRA_FAILURE: test_platform.TaskState_LIFE_CYCLE_ABORTED,
	buildbucket_pb.Status_CANCELED:      test_platform.TaskState_LIFE_CYCLE_CANCELLED,
}

func extractResult(from *buildbucket_pb.Build) (*skylab_test_runner.Result, error) {
	op := from.GetOutput().GetProperties().GetFields()
	if op == nil {
		return nil, fmt.Errorf("extract results from build %d: missing output properties", from.Id)
	}
	cr := op["compressed_result"].GetStringValue()
	if cr == "" {
		return nil, fmt.Errorf("extract results from build %d: missing result output property", from.Id)
	}
	pb, err := decompress(cr)
	if err != nil {
		return nil, errors.Annotate(err, "extract results from build %d", from.Id).Err()
	}
	var r skylab_test_runner.Result
	if err := proto.Unmarshal(pb, &r); err != nil {
		return nil, errors.Annotate(err, "extract results from build %d", from.Id).Err()
	}
	return &r, nil
}

func decompress(from string) ([]byte, error) {
	bs, err := base64.StdEncoding.DecodeString(from)
	if err != nil {
		return nil, errors.Annotate(err, "decompress").Err()
	}
	reader, err := zlib.NewReader(bytes.NewReader(bs))
	if err != nil {
		return nil, errors.Annotate(err, "decompress").Err()
	}
	bs, err = ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.Annotate(err, "decompress").Err()
	}
	return bs, nil
}

// URL is the Buildbucket URL of the task.
func (c *clientImpl) URL(t TaskReference) string {
	return fmt.Sprintf("https://ci.chromium.org/p/%s/builders/%s/%s/b%d",
		c.builder.Project, c.builder.Bucket, c.builder.Builder, c.knownTasks[t].bbID)
}

// SwarmingTaskID is the Swarming ID of the underlying task.
func (c *clientImpl) SwarmingTaskID(t TaskReference) string {
	return c.knownTasks[t].swarmingTaskID
}
