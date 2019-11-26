// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package bb provides a buildbucket Client with helper methods for interacting
// with builds.
package bb

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/steps"
	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/genproto/protobuf/field_mask"

	"infra/cmd/skylab/internal/logutils"
	"infra/cmd/skylab/internal/site"
)

// NewClient returns a new client to interact with buildbucket builds.
func NewClient(ctx context.Context, env site.Environment, authFlags authcli.Flags) (*Client, error) {
	hClient, err := newHTTPClient(ctx, &authFlags)
	if err != nil {
		return nil, err
	}

	pClient := &prpc.Client{
		C:    hClient,
		Host: env.BuildbucketHost,
	}

	return &Client{
		client: buildbucket_pb.NewBuildsPRPCClient(pClient),
		env:    env,
	}, nil
}

// Client provides helper methods to interact with buildbucket builds.
type Client struct {
	client buildbucket_pb.BuildsClient
	env    site.Environment
}

// newHTTPClient returns an HTTP client with authentication set up.
//
// TODO(pprabhu) dedup with internal/cmd/common.go:newHTTPClient
func newHTTPClient(ctx context.Context, f *authcli.Flags) (*http.Client, error) {
	o, err := f.Options()
	if err != nil {
		return nil, errors.Annotate(err, "failed to get auth options").Err()
	}
	a := auth.NewAuthenticator(ctx, auth.OptionalLogin, o)
	c, err := a.Client()
	if err != nil {
		return nil, errors.Annotate(err, "failed to create HTTP client").Err()
	}
	return c, nil
}

// ScheduleLegacyBuild schedules a new cros_test_platform build.
//
// ScheduleLegacyBuild returns the buildbucket build ID for the scheduled build
// on success.
// ScheduleLegacyBuild does not wait for the scheduled build to start.
// ScheduleLegacyBuild uses the old "request" input property.
func (c *Client) ScheduleLegacyBuild(ctx context.Context, request *test_platform.Request, tags []string) (int64, error) {
	rs, err := requestToStructPB(request)
	if err != nil {
		return -1, err
	}
	props := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"request": rs,
		},
	}
	return c.scheduleBuildRaw(ctx, props, tags)
}

// ScheduleBuild schedules a new cros_test_platform build.
//
// ScheduleBuild returns the buildbucket build ID for the scheduled build on
// success.
// ScheduleBuild does not wait for the scheduled build to start.
func (c *Client) ScheduleBuild(ctx context.Context, requests map[string]*test_platform.Request, tags []string) (int64, error) {
	rs, err := requestsToStructPB(requests)
	if err != nil {
		return -1, err
	}
	props := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"requests": rs,
		},
	}
	return c.scheduleBuildRaw(ctx, props, tags)
}

// scheduleBuildRaw schedules a new cros_test_platform build for the given properties struct.
func (c *Client) scheduleBuildRaw(ctx context.Context, props *structpb.Struct, tags []string) (int64, error) {
	tagPairs, err := splitTagPairs(tags)
	if err != nil {
		return -1, err
	}

	bbReq := &buildbucket_pb.ScheduleBuildRequest{
		Builder: &buildbucket_pb.BuilderID{
			Project: c.env.BuildbucketProject,
			Bucket:  c.env.BuildbucketBucket,
			Builder: c.env.BuildbucketBuilder,
		},
		Properties: props,
		Tags:       tagPairs,
	}

	build, err := c.client.ScheduleBuild(ctx, bbReq)
	if err != nil {
		return -1, err
	}
	return build.Id, nil
}

// WaitForBuild waits for a buildbucket build and returns the response on build
// completion.
//
// WaitForBuild regularly logs output to stdout to pacify the logdog silence
// checker.
func (c *Client) WaitForBuild(ctx context.Context, ID int64) (*Build, error) {
	throttledLogger := logutils.NewThrottledInfoLogger(logging.Get(ctx), 10*time.Minute)
	progressMessage := fmt.Sprintf("Still waiting for result from %s", c.BuildURL(ID))
	for {
		build, err := c.GetBuild(ctx, ID)
		if err != nil {
			return nil, err
		}
		if isFinal(build.Status) {
			return build, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(15 * time.Second):
		}
		throttledLogger.MaybeLog(progressMessage)
	}
}

// Build contains selected state information from a fetched buildbucket Build.
type Build struct {
	ID     int64
	Status buildbucket_pb.Status
	// Tags strings of the form "key:value"
	Tags []string

	// Response may be nil if the output properties of the build do not contain
	// a response.
	Response *steps.ExecuteResponse

	// Responses may be nil if the output properties of the build do not contain
	// a responses field.
	Responses map[string]*steps.ExecuteResponse

	// Request may be nil if the output properties of the build do not contain a
	// request field.
	Request *test_platform.Request

	// Requests may be nil if the output properties of the build do not contain
	// a requests field.
	Requests map[string]*test_platform.Request

	// BackfillRequests may be nil if the output properties of the build do not
	// contain a backfill_requests field.
	BackfillRequests map[string]*test_platform.Request
}

// GetBuild gets a buildbucket build by ID.
func (c *Client) GetBuild(ctx context.Context, ID int64) (*Build, error) {
	req := &buildbucket_pb.GetBuildRequest{
		Id:     ID,
		Fields: &field_mask.FieldMask{Paths: getBuildFields},
	}
	build, err := c.client.GetBuild(ctx, req)
	if err != nil {
		return nil, errors.Annotate(err, "get build").Err()
	}
	return extractBuildData(build)
}

// SearchBuildsByTags searches for all buildbucket builds with the given tags.
//
// SearchBuildsByTags returns at most limit results.
func (c *Client) SearchBuildsByTags(ctx context.Context, limit int, tags ...string) ([]*Build, error) {
	if len(tags) == 0 {
		return nil, errors.Reason("must provide at least one tag").Err()
	}
	tps, err := splitTagPairs(tags)
	if err != nil {
		return nil, errors.Annotate(err, "search builds by tags").Err()
	}
	rawBuilds, err := c.searchRawBuilds(ctx, limit, &buildbucket_pb.BuildPredicate{Tags: tps})
	if err != nil {
		return nil, errors.Annotate(err, "search builds by tags").Err()
	}
	return extractBuildDataAll(rawBuilds)
}

// BuildURL constructs the URL to a build with the given ID.
func (c *Client) BuildURL(buildID int64) string {
	return fmt.Sprintf("https://ci.chromium.org/p/%s/builders/%s/%s/b%d",
		c.env.BuildbucketProject, c.env.BuildbucketBucket, c.env.BuildbucketBuilder, buildID)
}

func (c *Client) searchRawBuilds(ctx context.Context, limit int, predicate *buildbucket_pb.BuildPredicate) ([]*buildbucket_pb.Build, error) {
	rawBuilds := make([]*buildbucket_pb.Build, 0, limit)
	pageToken := ""
	// Each page request sets the same PageSize (limit) the SearchBuilds() rpc
	// requires the PageSize to be unchanged across page requests.
	// We could obtain more than limit results in this process, so only the
	// first limit results are returned at the end of this function.
	for {
		req := buildbucket_pb.SearchBuildsRequest{
			Predicate: predicate,
			Fields:    &field_mask.FieldMask{Paths: getSearchBuildsFields()},
			PageToken: pageToken,
			PageSize:  clipToInt32(limit),
		}
		resp, err := c.client.SearchBuilds(ctx, &req)
		if err != nil {
			return nil, errors.Annotate(err, "search raw builds").Err()
		}
		rawBuilds = append(rawBuilds, resp.GetBuilds()...)
		pageToken := resp.GetNextPageToken()
		if pageToken == "" || len(rawBuilds) >= limit {
			break
		}
	}
	// As noted above, the last paging call may accumulate some extra results.
	if limit < len(rawBuilds) {
		rawBuilds = rawBuilds[:limit]
	}
	return rawBuilds, nil
}

func clipToInt32(n int) int32 {
	if n <= math.MaxInt32 {
		return int32(n)
	}
	return math.MaxInt32
}

func splitTagPairs(tags []string) ([]*buildbucket_pb.StringPair, error) {
	ret := make([]*buildbucket_pb.StringPair, 0, len(tags))
	for _, t := range tags {
		p := strings.Split(t, ":")
		if len(p) != 2 {
			return nil, errors.Reason("malformed tag %s", t).Err()
		}
		ret = append(ret, &buildbucket_pb.StringPair{
			Key:   strings.Trim(p[0], " "),
			Value: strings.Trim(p[1], " "),
		})
	}
	return ret, nil
}

// getBuildFields is the list of buildbucket fields that are needed.
var getBuildFields = []string{
	"id",
	// Build details are parsed from the build's output properties.
	"output.properties",
	// Build status is used to determine whether the build is complete.
	"status",
	"tags",
}

func getSearchBuildsFields() []string {
	fs := make([]string, 0, len(getBuildFields))
	for _, f := range getBuildFields {
		fs = append(fs, fmt.Sprintf("builds.*.%s", f))
	}
	return fs
}

func extractBuildData(from *buildbucket_pb.Build) (*Build, error) {
	build := Build{
		ID:     from.Id,
		Status: from.GetStatus(),
	}

	build.Tags = make([]string, 0, len(from.GetTags()))
	for _, t := range from.GetTags() {
		build.Tags = append(build.Tags, fmt.Sprintf("%s:%s", t.Key, t.Value))
	}

	if op := from.GetOutput().GetProperties().GetFields(); op != nil {
		if rawResponse, ok := op["response"]; ok {
			response, err := structPBToExecuteResponse(rawResponse)
			if err != nil {
				return nil, errors.Annotate(err, "extractBuildData").Err()
			}
			build.Response = response
		}

		if raw, ok := op["responses"]; ok {
			r, err := structPBToResponses(raw)
			if err != nil {
				return nil, errors.Annotate(err, "extractBuildData").Err()
			}
			build.Responses = r
		}

		if reqValue, ok := op["request"]; ok {
			request, err := structPBToRequest(reqValue)
			if err != nil {
				return nil, errors.Annotate(err, "extractBuildData").Err()
			}
			build.Request = request
		}

		if raw, ok := op["requests"]; ok {
			r, err := structPBToRequests(raw)
			if err != nil {
				return nil, errors.Annotate(err, "extractBuildData").Err()
			}
			build.Requests = r
		}

		if raw, ok := op["backfills"]; ok {
			r, err := structPBToRequests(raw)
			if err != nil {
				return nil, errors.Annotate(err, "extractBuildData").Err()
			}
			build.BackfillRequests = r
		}
	}
	return &build, nil
}

func extractBuildDataAll(from []*buildbucket_pb.Build) ([]*Build, error) {
	builds := make([]*Build, len(from))
	for i, rb := range from {
		b, err := extractBuildData(rb)
		if err != nil {
			return nil, errors.Annotate(err, "search builds by tags").Err()
		}
		builds[i] = b
	}
	return builds, nil
}

func structPBToResponses(from *structpb.Value) (map[string]*steps.ExecuteResponse, error) {
	responses := make(map[string]*steps.ExecuteResponse)
	m, err := structPBStructToMap(from)
	if err != nil {
		return nil, errors.Annotate(err, "struct PB to responses").Err()
	}
	for k, v := range m {
		r, err := structPBToExecuteResponse(v)
		if err != nil {
			return nil, errors.Annotate(err, "struct PB to responses").Err()
		}
		responses[k] = r
	}
	return responses, nil
}

func structPBStructToMap(from *structpb.Value) (map[string]*structpb.Value, error) {
	switch m := from.Kind.(type) {
	case *structpb.Value_StructValue:
		if m.StructValue == nil {
			return nil, errors.Reason("struct has no fields").Err()
		}
		return m.StructValue.Fields, nil
	default:
		return nil, errors.Reason("not a Struct type").Err()
	}
}

func structPBToExecuteResponse(from *structpb.Value) (*steps.ExecuteResponse, error) {
	m := jsonpb.Marshaler{}
	json, err := m.MarshalToString(from)
	if err != nil {
		return nil, errors.Annotate(err, "structPBToExecuteResponse").Err()
	}
	response := &steps.ExecuteResponse{}
	if err := jsonpb.UnmarshalString(json, response); err != nil {
		return nil, errors.Annotate(err, "structPBToExecuteResponse").Err()
	}
	return response, nil
}

func structPBToRequests(from *structpb.Value) (map[string]*test_platform.Request, error) {
	requests := make(map[string]*test_platform.Request)
	m, err := structPBStructToMap(from)
	if err != nil {
		return nil, errors.Annotate(err, "struct PB to requests").Err()
	}
	for k, v := range m {
		r, err := structPBToRequest(v)
		if err != nil {
			return nil, errors.Annotate(err, "struct PB to requests").Err()
		}
		requests[k] = r
	}
	return requests, nil
}

func structPBToRequest(from *structpb.Value) (*test_platform.Request, error) {
	m := jsonpb.Marshaler{}
	json, err := m.MarshalToString(from)
	if err != nil {
		return nil, errors.Annotate(err, "structPBToExecuteRequest").Err()
	}
	request := &test_platform.Request{}
	if err := jsonpb.UnmarshalString(json, request); err != nil {
		return nil, errors.Annotate(err, "structPBToExecuteRequest").Err()
	}
	return request, nil
}

func requestToStructPB(from *test_platform.Request) (*structpb.Value, error) {
	m := jsonpb.Marshaler{}
	jsonStr, err := m.MarshalToString(from)
	if err != nil {
		return nil, err
	}
	reqStruct := &structpb.Struct{}
	if err := jsonpb.UnmarshalString(jsonStr, reqStruct); err != nil {
		return nil, err
	}
	return &structpb.Value{
		Kind: &structpb.Value_StructValue{StructValue: reqStruct},
	}, nil
}

func requestsToStructPB(from map[string]*test_platform.Request) (*structpb.Value, error) {
	fs := make(map[string]*structpb.Value)
	for k, r := range from {
		v, err := requestToStructPB(r)
		if err != nil {
			return nil, errors.Annotate(err, "requests to struct pb (%s)", k).Err()
		}
		fs[k] = v
	}
	return &structpb.Value{
		Kind: &structpb.Value_StructValue{
			StructValue: &structpb.Struct{
				Fields: fs,
			},
		},
	}, nil
}

func isFinal(status buildbucket_pb.Status) bool {
	return (status & buildbucket_pb.Status_ENDED_MASK) == buildbucket_pb.Status_ENDED_MASK
}
