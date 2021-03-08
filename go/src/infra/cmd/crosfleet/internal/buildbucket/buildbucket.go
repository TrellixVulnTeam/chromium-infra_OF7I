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
	SkylabTool:    site.VersionNumber,
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
func NewClient(ctx context.Context, builderInfo site.BuildbucketBuilderInfo, authFlags authcli.Flags) (*Client, error) {
	httpClient, err := cmdlib.NewHTTPClient(ctx, &authFlags)
	if err != nil {
		return nil, err
	}

	prpcClient := &prpc.Client{
		C:       httpClient,
		Host:    builderInfo.Host,
		Options: site.DefaultPRPCOptions,
	}

	return &Client{
		client:    buildbucketpb.NewBuildsPRPCClient(prpcClient),
		builderID: builderInfo.BuilderID,
	}, nil
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
// the given ID, and returns the build once it has started. Specific fields can
// be specified to populate on the the returned build; if none are specified,
// all build fields will be populated.
func (c *Client) WaitForBuildStart(ctx context.Context, id int64) (*buildbucketpb.Build, error) {
	for {
		build, err := c.GetBuild(ctx, id)
		if err != nil {
			return nil, err
		}
		switch s := build.Status; s {
		case buildbucketpb.Status_SCHEDULED:
			time.Sleep(10 * time.Second)
		case buildbucketpb.Status_STARTED:
			return build, nil
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

// CancelBuildWithBotID cancels any pending or active builds created after the
// given timestamp, with the given Swarming bot ID, if they were launched by
// the given user. The optional cancellation reason is used if not blank.
func (c *Client) CancelBuildWithBotID(ctx context.Context, id string, earliestCreateTime *timestamppb.Timestamp, user, reason string, writer io.Writer) error {
	if reason == "" {
		reason = "cancelled from crosfleet CLI"
	}

	searchBuildsRequest := &buildbucketpb.SearchBuildsRequest{
		Predicate: &buildbucketpb.BuildPredicate{
			Builder: c.builderID,
			CreateTime: &buildbucketpb.TimeRange{
				StartTime: earliestCreateTime,
			},
		},
		Fields: &field_mask.FieldMask{Paths: []string{
			"builds.*.created_by",
			"builds.*.id",
			"builds.*.status",
			"builds.*.infra",
		}},
		PageToken: "",
	}

	cancelled := 0
	for {
		response, err := c.client.SearchBuilds(ctx, searchBuildsRequest)
		if err != nil {
			return err
		}
		for _, build := range response.Builds {
			if !isUnfinishedBuildWithBotID(build, id) {
				continue
			}
			if build.CreatedBy != fmt.Sprintf("user:%s", user) {
				continue
			}
			fmt.Fprintf(writer, "Canceling build at %s for bot ID %s\n", c.BuildURL(build.Id), id)
			_, err := c.client.CancelBuild(ctx, &buildbucketpb.CancelBuildRequest{
				Id:              build.Id,
				SummaryMarkdown: reason,
			})
			if err != nil {
				return err
			}
			cancelled++
		}
		if response.NextPageToken == "" {
			break
		}
		searchBuildsRequest.PageToken = response.NextPageToken
	}

	if cancelled == 0 {
		fmt.Fprintf(writer, "No pending or active builds found with bot ID %s that were launched by the current user (%s)\n", id, user)
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

// isUnfinishedBuildWithBotID returns true if the build is either scheduled with
// the given bot ID as a requested dimension, or already started and
// provisioned with the given bot ID.
func isUnfinishedBuildWithBotID(build *buildbucketpb.Build, id string) bool {
	switch build.Status {
	case buildbucketpb.Status_SCHEDULED:
		if FindDimValInRequestedDims("id", build) == id {
			return true
		}
	case buildbucketpb.Status_STARTED:
		if FindDimValInFinalDims("id", build) == id {
			return true
		}
	}
	return false
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
