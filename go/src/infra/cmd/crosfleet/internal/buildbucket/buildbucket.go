// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package buildbucket provides a Buildbucket client with helper methods for
// interacting with builds.
package buildbucket

import (
	"context"
	"fmt"
	"infra/cmd/crosfleet/internal/common"
	"infra/cmd/crosfleet/internal/site"
	"infra/cmdsupport/cmdlib"
	"io"
	"strings"
	"time"

	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/luci/auth/client/authcli"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var maxServiceVersion = &test_platform.ServiceVersion{
	CrosfleetTool: site.VersionNumber,
}

// addServiceVersion marshals the max service version proto to a JSON-encoded
// string, adds it to the given Buildbucket property map, and returns the
// property map.
func addServiceVersion(props map[string]interface{}) map[string]interface{} {
	props["$chromeos/service_version"] = map[string]interface{}{
		// Convert to protoreflect.ProtoMessage for easier type comparison.
		"version": maxServiceVersion.ProtoReflect().Interface(),
	}
	return props
}

// Client provides helper methods to interact with Buildbucket builds.
type Client struct {
	client    buildbucketpb.BuildsClient
	builderID *buildbucketpb.BuilderID
}

// NewClient returns a new client to interact with Buildbucket builds from the
// given builder.
func NewClient(ctx context.Context, builder *buildbucketpb.BuilderID, bbService string, authFlags authcli.Flags) (*Client, error) {
	httpClient, err := cmdlib.NewHTTPClient(ctx, &authFlags)
	if err != nil {
		return nil, err
	}

	prpcClient := &prpc.Client{
		C:       httpClient,
		Host:    bbService,
		Options: site.DefaultPRPCOptions,
	}

	return &Client{
		client:    buildbucketpb.NewBuildsPRPCClient(prpcClient),
		builderID: builder,
	}, nil
}

// NewClientForTesting returns a new client with only the builderID configured.
func NewClientForTesting(builder *buildbucketpb.BuilderID) *Client {
	return &Client{builderID: builder}
}

// ScheduleBuild schedules a new build (of the client's builder) with the given
// properties, tags, bot dimensions, and Buildbucket priority, and returns the
// ID of the scheduled build.
//
// Buildbucket requests take properties of type *structpb.Struct. To simplify
// the conversion from other data structures to Structs, ScheduleBuild accepts
// properties of type map[string]interface{}, where interface{} can be any of
// Go's basic types (bool, string, number type, byte, or rune), a proto message
// (in the form protoreflect.ProtoMessage), or a nested map[string]interface{}
// that fulfils the same requirements recursively.
//
// NOTE: Buildbucket priority is separate from internal swarming priority.
func (c *Client) ScheduleBuild(ctx context.Context, props map[string]interface{}, dims map[string]string, tags map[string]string, priority int32) (int64, error) {
	props = addServiceVersion(props)
	propStruct, err := common.MapToStruct(props)
	if err != nil {
		return 0, err
	}
	request := &buildbucketpb.ScheduleBuildRequest{
		Builder:    c.builderID,
		Properties: propStruct,
		Dimensions: bbDims(dims),
		Tags:       bbTags(tags),
		Priority:   priority,
	}
	build, err := c.client.ScheduleBuild(ctx, request)
	if err != nil {
		return 0, errors.Annotate(err, "schedule build").Err()
	}
	return build.GetId(), nil
}

// WaitForBuildStart polls Buildbucket to check the status of the build with
// the given ID, and returns the build once it has started the given step.
func (c *Client) WaitForBuildStepStart(ctx context.Context, id int64, stepName string) (*buildbucketpb.Build, error) {
	for {
		build, err := c.GetBuild(ctx, id)
		if err != nil {
			return nil, err
		}
		switch s := build.Status; s {
		case buildbucketpb.Status_SCHEDULED:
			time.Sleep(10 * time.Second)
		case buildbucketpb.Status_STARTED:
			// For the purposes of this function, it is sufficient just to check
			// that the step exists (i.e. the build has reached and started the
			// step), and ignore the step's current status, since we already
			// confirmed the build has the overall healthy status "started".
			if containsStep(build.GetSteps(), stepName) {
				return build, nil
			}
			time.Sleep(10 * time.Second)
		default:
			statusString := buildbucketpb.Status_name[int32(s)]
			buildSummary := ""
			if build.SummaryMarkdown != "" {
				buildSummary = fmt.Sprintf("Build summary: %s", build.SummaryMarkdown)
			}
			return nil, fmt.Errorf(`build finished in unexpected state %s
%s
For more details, please visit the build page at %s`, statusString, buildSummary, c.BuildURL(build.Id))
		}
	}
}

// getAllBuilds performs a SearchBuilds call with the given SearchBuildsRequest,
// and returns all the builds that were found. For searches that return many
// builds, this function avoids having to deal with search pagination logic.
// This function only searches builds from the Client's builder.
func (c *Client) getAllBuilds(ctx context.Context, searchBuildsRequest *buildbucketpb.SearchBuildsRequest) ([]*buildbucketpb.Build, error) {
	// The Client is only designed to interact with builds from its builderID,
	// so we set that builderID within this function to keep the caller's
	// request simple.
	searchBuildsRequest.Predicate.Builder = c.builderID
	// Reset the PageToken to ensure we start at the beginning of the results.
	searchBuildsRequest.PageToken = ""

	var allBuilds []*buildbucketpb.Build
	for {
		response, err := c.client.SearchBuilds(ctx, searchBuildsRequest)
		if err != nil {
			return nil, err
		}
		allBuilds = append(allBuilds, response.Builds...)
		if response.NextPageToken == "" {
			break
		}
		searchBuildsRequest.PageToken = response.NextPageToken
	}
	return allBuilds, nil
}

// GetAllBuildsForUser finds returns all builds created by the given user
// matching the given SearchBuildsRequest. This function expects the field
// "builds.*.created_by" to be included in the field mask of the given
// SearchBuildRequest.
func (c *Client) GetAllBuildsByUser(ctx context.Context, user string, searchBuildsRequest *buildbucketpb.SearchBuildsRequest) ([]*buildbucketpb.Build, error) {
	allBuilds, err := c.getAllBuilds(ctx, searchBuildsRequest)
	if err != nil {
		return nil, err
	}
	var buildsByUser []*buildbucketpb.Build
	for _, build := range allBuilds {
		if build.CreatedBy == fmt.Sprintf("user:%s", user) {
			buildsByUser = append(buildsByUser, build)
		}
	}
	return buildsByUser, nil
}

// CancelBuildsByUser cancels any pending or active build created after the
// given timestamp that was launched by the given user. An optional bot ID list
// can be given, which restricts cancellations to builds running on bots on the
// given list. The optional cancellation reason is used if not blank.
func (c *Client) CancelBuildsByUser(ctx context.Context, writer io.Writer, earliestCreateTime *timestamppb.Timestamp, user string, ids []string, reason string) error {
	if reason == "" {
		reason = "cancelled from crosfleet CLI"
	}

	fieldsMask := &field_mask.FieldMask{Paths: []string{
		"builds.*.created_by",
		"builds.*.id",
		"builds.*.status",
		"builds.*.infra",
	}}
	timeRange := &buildbucketpb.TimeRange{
		StartTime: earliestCreateTime,
	}
	scheduledBuilds, err := c.GetAllBuildsByUser(ctx, user, &buildbucketpb.SearchBuildsRequest{
		Predicate: &buildbucketpb.BuildPredicate{
			CreateTime: timeRange,
			Status:     buildbucketpb.Status_SCHEDULED,
		},
		Fields: fieldsMask,
	})
	if err != nil {
		return err
	}
	startedBuilds, err := c.GetAllBuildsByUser(ctx, user, &buildbucketpb.SearchBuildsRequest{
		Predicate: &buildbucketpb.BuildPredicate{
			CreateTime: timeRange,
			Status:     buildbucketpb.Status_STARTED,
		},
		Fields: fieldsMask,
	})
	if err != nil {
		return err
	}

	var buildsToCancel []*buildbucketpb.Build
	// Only filter the builds by bot ID if bot IDs were given.
	if len(ids) > 0 {
		for _, scheduledBuild := range scheduledBuilds {
			requestedBotID := FindDimValInRequestedDims("id", scheduledBuild)
			if idMatch(requestedBotID, ids) {
				buildsToCancel = append(buildsToCancel, scheduledBuild)
			}
		}
		for _, startedBuild := range startedBuilds {
			provisionedBotID := FindDimValInFinalDims("id", startedBuild)
			if idMatch(provisionedBotID, ids) {
				buildsToCancel = append(buildsToCancel, startedBuild)
			}
		}
	} else {
		buildsToCancel = append(scheduledBuilds, startedBuilds...)
	}
	if len(buildsToCancel) == 0 {
		fmt.Fprintf(writer, "No scheduled or active builds found that were launched by the current user (%s)\n", user)
		return nil
	}

	for _, build := range buildsToCancel {
		fmt.Fprintf(writer, "Canceling build at %s\n", c.BuildURL(build.Id))
		_, err := c.client.CancelBuild(ctx, &buildbucketpb.CancelBuildRequest{
			Id:              build.Id,
			SummaryMarkdown: reason,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetBuild gets a Buildbucket build by ID, with the given build fields
// populated. If no fields are given, all fields will be populated.
func (c *Client) GetBuild(ctx context.Context, ID int64, fields ...string) (*buildbucketpb.Build, error) {
	if len(fields) == 0 {
		fields = []string{"*"}
	}
	request := &buildbucketpb.GetBuildRequest{
		Id:     ID,
		Fields: &field_mask.FieldMask{Paths: fields},
	}
	build, err := c.client.GetBuild(ctx, request)
	if err != nil {
		return nil, errors.Annotate(err, "get build").Err()
	}
	return build, nil
}

// GetLatestGreenBuild gets the latest green build for the client's builder.
// To optimize runtime, this call is only configured to populate the build's ID
// and output properties. More fields can be added to the field mask if needed.
func (c *Client) GetLatestGreenBuild(ctx context.Context) (*buildbucketpb.Build, error) {
	searchBuildsRequest := &buildbucketpb.SearchBuildsRequest{
		Predicate: &buildbucketpb.BuildPredicate{
			Builder: c.builderID,
			Status:  buildbucketpb.Status_SUCCESS,
		},
		Fields: &field_mask.FieldMask{Paths: []string{
			"builds.*.id",
			"builds.*.output.properties",
		}},
	}
	// Avoid the getAllBuilds function since it scrolls through all pages of
	// the search result, and we only want the most recent build.
	response, err := c.client.SearchBuilds(ctx, searchBuildsRequest)
	if err != nil {
		return nil, err
	}
	if len(response.Builds) == 0 {
		return nil, fmt.Errorf("no green builds found for builder %s", c.builderID.Builder)
	}
	return response.Builds[0], nil
}

// BuildURL constructs the URL for the LUCI page of the build (of the client's
// builder) with the given ID.
func (c *Client) BuildURL(ID int64) string {
	return fmt.Sprintf(
		"https://ci.chromium.org/ui/p/%s/builders/%s/%s/b%d",
		c.builderID.Project, c.builderID.Bucket, c.builderID.Builder, ID)
}

// bbDims converts the given map[string]string of bot dimensions to the
// required []*buildbucketpb.RequestedDimension type for Buildbucket requests.
func bbDims(dims map[string]string) []*buildbucketpb.RequestedDimension {
	var bbDimList []*buildbucketpb.RequestedDimension
	for key, val := range dims {
		bbDimList = append(bbDimList, &buildbucketpb.RequestedDimension{
			Key:   strings.Trim(key, " "),
			Value: strings.Trim(val, " "),
		})
	}
	return bbDimList
}

// bbTags converts the given map[string]string of Buildbucket tags to the
// required []*buildbucketpb.StringPair type for Buildbucket requests.
func bbTags(tags map[string]string) []*buildbucketpb.StringPair {
	var bbTagList []*buildbucketpb.StringPair
	for key, val := range tags {
		bbTagList = append(bbTagList, &buildbucketpb.StringPair{
			Key:   strings.Trim(key, " "),
			Value: strings.Trim(val, " "),
		})
	}
	return bbTagList
}

// FindDimValInRequestedDims finds the given dimension value in the build's
// requested dimensions, which are of type []*buildbucketpb.RequestedDimension.
func FindDimValInRequestedDims(dim string, build *buildbucketpb.Build) string {
	requestedDims := build.GetInfra().GetBuildbucket().GetRequestedDimensions()
	for _, d := range requestedDims {
		if d.GetKey() == dim {
			return d.GetValue()
		}
	}
	return ""
}

// FindDimValInFinalDims finds the given dimension value in the build's
// requested dimensions, which are of type []*buildbucketpb.StringPair.
//
// Since the dimensions looped through are of different type here than in
// FindDimValInRequestedDims, some duplicate code is unfortunately required.
func FindDimValInFinalDims(dim string, build *buildbucketpb.Build) string {
	provisionedDims := build.GetInfra().GetSwarming().GetBotDimensions()
	for _, d := range provisionedDims {
		if d.GetKey() == dim {
			return d.GetValue()
		}
	}
	return ""
}

// idMatch returns true if the given ID to match is found in the given ID list.
func idMatch(idToMatch string, idList []string) bool {
	for _, id := range idList {
		if idToMatch == id {
			return true
		}
	}
	return false
}

// containsStep returns true if a step with the given name is found in the
// given slice of steps.
func containsStep(steps []*buildbucketpb.Step, stepName string) bool {
	for _, step := range steps {
		if step.Name == stepName {
			return true
		}
	}
	return false
}
