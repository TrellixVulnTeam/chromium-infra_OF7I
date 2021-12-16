// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
	cvv0 "go.chromium.org/luci/cv/api/v0"
	cvv1 "go.chromium.org/luci/cv/api/v1"
	"go.chromium.org/luci/server/router"
	"google.golang.org/protobuf/encoding/protojson"

	"infra/appengine/weetbix/internal/cv"
	ctlpb "infra/appengine/weetbix/internal/ingestion/control/proto"
	pb "infra/appengine/weetbix/proto/v1"
)

const (
	// TODO(chanli@@) Removing the hosts after CVPubSub and GetRun RPC added them.
	// Host name of buildbucket.
	bbHost = "cr-buildbucket.appspot.com"

	chromiumProject = "chromium"
)

var (
	cvRunCounter = metric.NewCounter(
		"weetbix/cv_pubsub/runs",
		"The number of CV runs received by Weetbix from PubSub",
		nil,
		// "success", "transient-failure" or "permanent-failure".
		field.String("status"))

	runIDRe = regexp.MustCompile(`^projects/(.*)/runs/(.*)$`)
)

// CVRunPubSubHandler accepts and processes CV Pub/Sub messages.
func CVRunPubSubHandler(ctx *router.Context) {
	status := "unknown"
	defer func() {
		// Closure for late binding.
		cvRunCounter.Add(ctx.Context, 1, status)
	}()
	processed, err := cvPubSubHandlerImpl(ctx.Context, ctx.Request)

	switch {
	case err != nil:
		errors.Log(ctx.Context, errors.Annotate(err, "handling cv pubsub event").Err())
		status = processErr(ctx, err)
		return
	case !processed:
		status = "ignored"
	default:
		status = "success"
	}
	ctx.Writer.WriteHeader(http.StatusOK)
}

func cvPubSubHandlerImpl(ctx context.Context, request *http.Request) (processed bool, err error) {
	psRun, err := extractPubSubRun(request)
	if err != nil {
		return false, errors.Annotate(err, "failed to extract run").Err()
	}

	project, runID, err := parseRunID(psRun.Id)
	if err != nil {
		return false, errors.Annotate(err, "failed to parse run ID").Err()
	}
	if project != chromiumProject {
		// Received a non-chromium run, ignore it.
		return false, nil
	}

	run, err := getRun(ctx, psRun)
	switch {
	case err != nil:
		return false, errors.Annotate(err, "failed to get run").Err()
	case run.GetCreateTime() == nil:
		return false, errors.New("could not get create time for the run")
	case run.GetMode() != "FULL_RUN":
		// Not a FULL_RUN, so the CL under test would not be submitted.
		// Since we're only dealing with Chromium try results for now, ignore the
		// run.
		return false, nil
	}

	pr := &ctlpb.PresubmitResult{
		PresubmitRunId: &pb.PresubmitRunId{
			System: "luci-cv",
			Id:     fmt.Sprintf("%s/%s", project, runID),
		},
		PresubmitRunSucceeded: run.Status == cvv0.Run_SUCCEEDED,
		CreationTime:          run.CreateTime,
	}

	// Schedule ResultIngestion tasks for each build.
	var buildIDs []string
	for _, tj := range run.Tryjobs {
		b := tj.GetResult().GetBuildbucket()
		if b == nil {
			// Non build-bucket result.
			continue
		}

		buildIDs = append(buildIDs, buildID(bbHost, b.Id))
	}

	if err := JoinPresubmitResult(ctx, project, buildIDs, pr); err != nil {
		return true, errors.Annotate(err, "joining presubmit results").Err()
	}

	return true, nil
}

func extractPubSubRun(r *http.Request) (*cvv1.PubSubRun, error) {
	var msg pubsubMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		return nil, errors.Annotate(err, "could not decode cv pubsub message").Err()
	}

	var run cvv1.PubSubRun
	err := protojson.Unmarshal(msg.Message.Data, &run)
	if err != nil {
		return nil, errors.Annotate(err, "could not parse cv pubsub message data").Err()
	}
	return &run, nil
}

func parseRunID(runID string) (project string, run string, err error) {
	m := runIDRe.FindStringSubmatch(runID)
	if m == nil {
		return "", "", errors.Reason("run ID does not match %s", runIDRe).Err()
	}
	return m[1], m[2], nil
}

// getRun gets the full Run message by make a GetRun RPC to CV.
//
// Currently we're calling cv.v0.Runs.GetRun, and should switch to v1 when it's
// ready to use.
func getRun(ctx context.Context, psRun *cvv1.PubSubRun) (*cvv0.Run, error) {
	c, err := cv.NewClient(ctx, psRun.Hostname)
	if err != nil {
		return nil, errors.Annotate(err, "failed to create cv client").Err()
	}
	req := &cvv0.GetRunRequest{
		Id: psRun.Id,
	}
	return c.GetRun(ctx, req)
}
