// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"encoding/json"
	"net/http"

	bbv1 "go.chromium.org/luci/common/api/buildbucket/buildbucket/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/server/router"
	"google.golang.org/protobuf/types/known/timestamppb"

	ctlpb "infra/appengine/weetbix/internal/ingestion/control/proto"
)

const (
	// chromiumCIBucket is the bucket for chromium ci builds in bb v1 format.
	chromiumCIBucket = "luci.chromium.ci"
)

var (
	buildCounter = metric.NewCounter(
		"weetbix/buildbucket_pubsub/builds",
		"The number of buildbucket builds received by Weetbix from PubSub",
		nil,
		// "success", "ignored", "transient-failure" or "permanent-failure".
		field.String("status"))
)

// BuildbucketPubSubHandler accepts and process buildbucket Pub/Sub messages.
// As of Aug 2021, Weetbix subscribes to this Pub/Sub topic to get completed
// Chromium CI builds.
// For CQ builds, Weetbix uses CV Pub/Sub as the entrypoint.
func BuildbucketPubSubHandler(ctx *router.Context) {
	status := "unknown"
	defer func() {
		// Closure for late binding.
		buildCounter.Add(ctx.Context, 1, status)
	}()

	err := bbPubSubHandlerImpl(ctx.Context, ctx.Request)
	if err != nil {
		errors.Log(ctx.Context, errors.Annotate(err, "handling buildbucket pubsub event").Err())
		status = processErr(ctx, err)
		return
	}
	status = "success"
	ctx.Writer.WriteHeader(http.StatusOK)
}

func bbPubSubHandlerImpl(ctx context.Context, request *http.Request) error {
	build, err := extractBuild(request)

	switch {
	case err != nil:
		return errors.Annotate(err, "failed to extract build").Err()

	case build == nil:
		// Ignore.
		return nil

	default:
		if err := JoinBuildResult(ctx, build.project, build.id, build.isPresubmit, build.result); err != nil {
			return errors.Annotate(err, "joining build result").Err()
		}
		return nil
	}
}

type build struct {
	// project is the LUCI project containing the build.
	project string
	// id is the identity of the build. This is {hostname}/{build_id}.
	id string
	// isPresubmit is whether the build relates to a presubmit run.
	isPresubmit bool
	// result is information about the build to be passed
	// to ingestion.
	result *ctlpb.BuildResult
}

func extractBuild(r *http.Request) (*build, error) {
	var msg pubsubMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		return nil, errors.Annotate(err, "could not decode buildbucket pubsub message").Err()
	}

	var message struct {
		Build    bbv1.LegacyApiCommonBuildMessage
		Hostname string
	}
	switch err := json.Unmarshal(msg.Message.Data, &message); {
	case err != nil:
		return nil, errors.Annotate(err, "could not parse buildbucket pubsub message data").Err()
	case message.Build.Project != chromiumProject:
		// Received a non-chromium build, ignore it.
		return nil, nil
	case message.Build.Status != bbv1.StatusCompleted:
		// Received build that hasn't completed yet, ignore it.
		return nil, nil
	case message.Build.CreatedTs == 0:
		return nil, errors.New("build did not have created timestamp specified")
	}

	return &build{
		project:     message.Build.Project,
		id:          buildID(message.Hostname, message.Build.Id),
		isPresubmit: message.Build.Bucket != chromiumCIBucket,
		result: &ctlpb.BuildResult{
			CreationTime: timestamppb.New(bbv1.ParseTimestamp(message.Build.CreatedTs)),
			Id:           message.Build.Id,
			Host:         message.Hostname,
		},
	}, nil
}
