// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	bbv1 "go.chromium.org/luci/common/api/buildbucket/buildbucket/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/server/router"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/services/resultingester"
	"infra/appengine/weetbix/internal/tasks/taskspb"
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
	build, createTime, err := extractBuildAndCreateTime(request)

	switch {
	case err != nil:
		return errors.Annotate(err, "failed to extract build").Err()

	case build == nil:
		// Ignore.
		return nil

	default:
		task := &taskspb.IngestTestResults{
			Build:         build,
			PartitionTime: timestamppb.New(createTime),
		}
		return resultingester.Schedule(ctx, task)
	}
}

func extractBuildAndCreateTime(r *http.Request) (*taskspb.Build, time.Time, error) {
	var msg pubsubMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		return nil, time.Time{}, errors.Annotate(err, "could not decode buildbucket pubsub message").Err()
	}

	var message struct {
		Build    bbv1.LegacyApiCommonBuildMessage
		Hostname string
	}
	switch err := json.Unmarshal(msg.Message.Data, &message); {
	case err != nil:
		return nil, time.Time{}, errors.Annotate(err, "could not parse buildbucket pubsub message data").Err()
	case message.Build.Bucket != chromiumCIBucket:
		// Received a non-chromium-ci build, ignore it.
		return nil, time.Time{}, nil
	case message.Build.Status != bbv1.StatusCompleted:
		// Received build that hasn't completed yet, ignore it.
		return nil, time.Time{}, nil
	case message.Build.CreatedTs == 0:
		return nil, time.Time{}, errors.New("build did not have created timestamp specified")
	}

	createTime := bbv1.ParseTimestamp(message.Build.CreatedTs)
	build := &taskspb.Build{
		Id:   message.Build.Id,
		Host: message.Hostname,
	}
	return build, createTime, nil
}
