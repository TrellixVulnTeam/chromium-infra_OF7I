// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package buildbucket provides a buildbucket Client with helper methods to schedule the tasks.
package buildbucket

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/auth/client/authcli"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/prpc"

	structbuilder "google.golang.org/protobuf/types/known/structpb"
)

const defaultTaskPriority = 24

// ScheduleLabpackTaskParams includes the parameters necessary to schedule a labpack task.
type ScheduleLabpackTaskParams struct {
	UnitName         string
	ExpectedDUTState string
	Props            *structbuilder.Struct
	ExtraTags        []string
	// TODO(gregorynisbet): Support map[string]string as dims value.
	ExtraDims map[string]string
	// Bullder custom fields. If not provide default values will be used
	BuilderName    string
	BuilderProject string
	BuilderBucket  string
}

// Client provides helper methods to interact with buildbucket builds.
type Client interface {
	// ScheduleLabpackTask schedules a labpack task.
	ScheduleLabpackTask(ctx context.Context, params *ScheduleLabpackTaskParams) (int64, error)
	// BuildURL constructs the URL to a build with the given ID.
	BuildURL(buildID int64) string
}

// ClientImpl is the implementation of the Client interface.
type clientImpl struct {
	client    buildbucket_pb.BuildsClient
	builderID *buildbucket_pb.BuilderID
}

// NewClient returns a new client to interact with buildbucket builds.
//
// Deprecated:
//   This function takes authcli.Flags parameters directly instead of an http.Client.
//   This is too inflexible when it comes to authentication strategies.
//   Replace calls to this function with calls to NewClient2 and then rename NewClient2 to NewClient.
//
func NewClient(ctx context.Context, f authcli.Flags, options *prpc.Options, project string, bucket string, builder string) (Client, error) {
	hc, err := NewHTTPClient(ctx, &f)
	if err != nil {
		return nil, errors.Annotate(err, "new buildbucket client").Err()
	}
	return NewClient2(ctx, hc, options, project, bucket, builder)
}

// NewClient2 returns a new client to interact with buildbucket builds.
// TODO(gregorynisbet): Replace calls to NewClient with calls to NewClient2, and then rename NewClient2 to NewClient.
func NewClient2(ctx context.Context, hc *http.Client, options *prpc.Options, project string, bucket string, builder string) (Client, error) {
	if hc == nil {
		return nil, errors.Reason("buildbucket client cannot be created from nil http.Client").Err()
	}
	pClient := &prpc.Client{
		C:       hc,
		Host:    "cr-buildbucket.appspot.com",
		Options: options,
	}

	return &clientImpl{
		client: buildbucket_pb.NewBuildsPRPCClient(pClient),
		builderID: &buildbucket_pb.BuilderID{
			Project: project,
			Bucket:  bucket,
			Builder: builder,
		},
	}, nil
}

// NewHTTPClient returns an HTTP client with authentication set up.
func NewHTTPClient(ctx context.Context, f *authcli.Flags) (*http.Client, error) {
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

// ScheduleLabpackTask creates new task in build bucket with labpack.
func (c *clientImpl) ScheduleLabpackTask(ctx context.Context, params *ScheduleLabpackTaskParams) (int64, error) {
	if params == nil {
		return 0, errors.Reason("ScheduleLabpackTask: params cannot be nil").Err()
	}
	dims := make(map[string]string)
	dims["id"] = "crossk-" + params.UnitName
	if params.ExpectedDUTState != "" {
		dims["dut_state"] = params.ExpectedDUTState
	}
	for key, value := range params.ExtraDims {
		if _, ok := dims[key]; !ok {
			dims[key] = value
		}
	}

	tags := []string{
		fmt.Sprintf("dut-name:%s", params.UnitName),
	}
	tags = append(tags, params.ExtraTags...)

	tagPairs, err := splitTagPairs(tags)
	if err != nil {
		return -1, err
	}
	b := &buildbucket_pb.BuilderID{
		Project: "chromeos",
		Bucket:  "labpack_runner",
		Builder: "labpack_builder",
	}
	if params.BuilderName != "" {
		b.Builder = params.BuilderName
	}
	if params.BuilderProject != "" {
		b.Project = params.BuilderProject
	}
	if params.BuilderBucket != "" {
		b.Bucket = params.BuilderBucket
	}
	bbReq := &buildbucket_pb.ScheduleBuildRequest{
		Builder:    b,
		Properties: params.Props,
		Tags:       tagPairs,
		Dimensions: bbDimensions(dims),
		Priority:   defaultTaskPriority,
	}

	build, err := c.client.ScheduleBuild(ctx, bbReq)
	if err != nil {
		return -1, err
	}
	return build.Id, nil
}

// BuildURLFmt is the format of a build URL.
const BuildURLFmt = "https://ci.chromium.org/p/%s/builders/%s/%s/b%d"

// BuildURL constructs the URL to a build with the given ID.
func (c *clientImpl) BuildURL(buildID int64) string {
	return fmt.Sprintf(BuildURLFmt, c.builderID.Project, c.builderID.Bucket, c.builderID.Builder, buildID)
}

func splitTagPairs(tags []string) ([]*buildbucket_pb.StringPair, error) {
	ret := make([]*buildbucket_pb.StringPair, 0, len(tags))
	for _, t := range tags {
		p := strings.SplitN(t, ":", 2)
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

// bbDimensions converts a map of dimensions to a slice of
// *buildbucket_pb.RequestedDimension.
func bbDimensions(dims map[string]string) []*buildbucket_pb.RequestedDimension {
	ret := make([]*buildbucket_pb.RequestedDimension, len(dims))
	i := 0
	for key, value := range dims {
		ret[i] = &buildbucket_pb.RequestedDimension{
			Key:   strings.Trim(key, " "),
			Value: strings.Trim(value, " "),
		}
		i++
	}
	return ret
}
