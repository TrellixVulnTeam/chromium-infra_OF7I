// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package bb provides a buildbucket Client with helper methods to schedule the tasks.
package bb

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

	"infra/cmd/mallet/internal/site"
)

const defaultTaskPriority = 24

// Client provides helper methods to interact with buildbucket builds.
type Client struct {
	client    buildbucket_pb.BuildsClient
	builderID *buildbucket_pb.BuilderID
}

// NewClient returns a new client to interact with buildbucket builds.
func NewClient(ctx context.Context, authFlags authcli.Flags) (*Client, error) {
	hClient, err := newHTTPClient(ctx, &authFlags)
	if err != nil {
		return nil, err
	}

	pClient := &prpc.Client{
		C:       hClient,
		Host:    "cr-buildbucket.appspot.com",
		Options: site.DefaultPRPCOptions,
	}

	return &Client{
		client: buildbucket_pb.NewBuildsPRPCClient(pClient),
		builderID: &buildbucket_pb.BuilderID{
			Project: "chromeos",
			Bucket:  "labpack",
			Builder: "labpack",
		},
	}, nil
}

// newHTTPClient returns an HTTP client with authentication set up.
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

// ScheduleLabpackTask creates new task in build bucket with labpack.
func (c *Client) ScheduleLabpackTask(ctx context.Context, unit string, props *structbuilder.Struct) (int64, error) {
	dims := make(map[string]string)
	dims["id"] = "crossk-" + unit

	tags := []string{
		fmt.Sprintf("dut-name:%s", unit),
	}
	tagPairs, err := splitTagPairs(tags)
	if err != nil {
		return -1, err
	}

	bbReq := &buildbucket_pb.ScheduleBuildRequest{
		Builder: &buildbucket_pb.BuilderID{
			Project: "chromeos",
			Bucket:  "labpack_runner",
			Builder: "labpack_builder",
		},
		Properties: props,
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

// BuildURL constructs the URL to a build with the given ID.
func (c *Client) BuildURL(buildID int64) string {
	return fmt.Sprintf("https://ci.chromium.org/p/%s/builders/%s/%s/b%d",
		c.builderID.Project, c.builderID.Bucket, c.builderID.Builder, buildID)
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
