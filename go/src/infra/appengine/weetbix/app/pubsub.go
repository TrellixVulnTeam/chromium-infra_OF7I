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
	"go.chromium.org/luci/common/retry/transient"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	"go.chromium.org/luci/server/router"

	"infra/appengine/weetbix/internal/tasks/taskspb"
)

var (
	buildCounter = metric.NewCounter(
		"weetbix/buildbucket_pubsub/builds",
		"The number of buildbucket builds received by Weetbix from PubSub",
		nil,
		// "success", "transient-failure" or "permanent-failure".
		field.String("status"))
)

// chromiumCIBucket is the bucket for chromium ci builds in bb v1 format.
const chromiumCIBucket = "luci.chromium.ci"

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

	err := pubSubHandlerImpl(ctx.Context, ctx.Request)
	if err != nil {
		errors.Log(ctx.Context, errors.Annotate(err, "handling buildbucket pubsub event").Err())
		if transient.Tag.In(err) {
			// Transient errors are 500 so that PubSub retries them.
			status = "transient-failure"
			ctx.Writer.WriteHeader(http.StatusInternalServerError)
		} else {
			// Permentant failures are 200s so that PubSub does not retry them.
			status = "permanent-failure"
			ctx.Writer.WriteHeader(http.StatusOK)
		}
		return
	}
	status = "success"
	ctx.Writer.WriteHeader(http.StatusOK)
}

func pubSubHandlerImpl(ctx context.Context, request *http.Request) error {
	build, err := extractBuild(request)

	switch {
	case err != nil:
		return errors.Annotate(err, "failed to extract build").Err()

	case build == nil:
		// Ignore.
		return nil

	default:
		//TODO(chanli) enqueue a test result ingestion task.
		return nil
	}
}

func extractBuild(r *http.Request) (*taskspb.Build, error) {
	// Sent by pubsub.
	// This struct is just convenient for unwrapping the json message.
	// See https://source.chromium.org/chromium/infra/infra/+/main:luci/appengine/components/components/pubsub.py;l=178;drc=78ce3aa55a2e5f77dc05517ef3ec377b3f36dc6e.
	var msg struct {
		Message struct {
			Data []byte
		}
		Attributes map[string]interface{}
	}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		return nil, errors.Annotate(err, "could not decode message").Err()
	}

	var message struct {
		Build    bbv1.LegacyApiCommonBuildMessage
		Hostname string
	}
	switch err := json.Unmarshal(msg.Message.Data, &message); {
	case err != nil:
		return nil, errors.Annotate(err, "could not parse pubsub message data").Err()
	case message.Build.Bucket != chromiumCIBucket:
		// Received a non-chromium-ci build, ignore it.
		return nil, nil
	case message.Build.Status != bbv1.StatusCompleted:
		// Received build that hasn't completed yet, ignore it.
		return nil, nil
	}

	return &taskspb.Build{
		Id:   message.Build.Id,
		Host: message.Hostname,
	}, nil
}
