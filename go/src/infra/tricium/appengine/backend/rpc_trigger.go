// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	ds "go.chromium.org/luci/gae/service/datastore"
	tq "go.chromium.org/luci/gae/service/taskqueue"
	"go.chromium.org/luci/grpc/grpcutil"

	admin "infra/tricium/api/admin/v1"
	"infra/tricium/appengine/common"
	"infra/tricium/appengine/common/config"
	"infra/tricium/appengine/common/gerrit"
	"infra/tricium/appengine/common/track"
)

// DriverServer represents the Tricium pRPC Driver server.
type driverServer struct{}

// Trigger triggers processes one trigger request to the Tricium driver.
func (*driverServer) Trigger(c context.Context, req *admin.TriggerRequest) (*admin.TriggerResponse, error) {
	if req.RunId == 0 {
		return nil, errors.New("missing run ID", grpcutil.InvalidArgumentTag)
	}
	if req.Worker == "" {
		return nil, errors.New("missing worker name", grpcutil.InvalidArgumentTag)
	}
	if req.IsolatedInputHash != "" {
		return nil, errors.New("isolated input hash in trigger request", grpcutil.InvalidArgumentTag)
	}
	if err := trigger(c, req, config.WorkflowCache, common.BuildbucketServer); err != nil {
		return nil, errors.Annotate(err, "failed to trigger worker").
			Tag(grpcutil.InternalTag).Err()
	}
	return &admin.TriggerResponse{}, nil
}

func trigger(c context.Context, req *admin.TriggerRequest, wp config.WorkflowCacheAPI, bb common.TaskServerAPI) error {
	workflow, err := wp.GetWorkflow(c, req.RunId)
	if err != nil {
		return errors.Annotate(err, "failed to read workflow config").Err()
	}
	worker, err := workflow.GetWorker(req.Worker)
	if err != nil {
		return errors.Annotate(err, "failed to get worker %q", req.Worker).Err()
	}
	patch := fetchPatchDetails(c, req.RunId)
	tags := getTags(c, req.Worker, req.RunId, patch)

	// Create PubSub userdata for trigger request.
	b, err := proto.Marshal(req)
	if err != nil {
		return errors.Annotate(err, "failed to marshal PubSub user data").Err()
	}
	userdata := base64.StdEncoding.EncodeToString(b)
	logging.Fields{
		"userdata": userdata,
	}.Infof(c, "PubSub userdata created.")

	result := &common.TriggerResult{}
	switch wi := worker.Impl.(type) {
	case *admin.Worker_Recipe:
		// Trigger worker.
		result, err = bb.Trigger(c, &common.TriggerParameters{
			Server:         workflow.BuildbucketServerHost,
			Worker:         worker,
			PubsubUserdata: userdata,
			Tags:           tags,
			Patch:          patch,
		})
		if err != nil {
			return errors.Annotate(err, "failed to call trigger on buildbucket API").Err()
		}
	case nil:
		return errors.Reason("missing Impl when isolating worker %s", worker.Name).Err()
	default:
		return errors.Reason("Impl.Impl has unexpected type %T", wi).Err()
	}
	// Mark worker as launched.
	b, err = proto.Marshal(&admin.WorkerLaunchedRequest{
		RunId:              req.RunId,
		Worker:             req.Worker,
		BuildbucketBuildId: result.BuildID,
	})
	if err != nil {
		return errors.Annotate(err, "failed to encode worker launched request").Err()
	}
	t := tq.NewPOSTTask("/tracker/internal/worker-launched", nil)
	t.Payload = b
	return tq.Add(c, common.TrackerQueue, t)
}

// getTags generates tags to send when triggering tasks via buildbucket.
//
// These tags can be used later when querying tasks, so
// any attribute of a job that we may want to query or filter
// by could be added as a tag.
func getTags(c context.Context, worker string, runID int64, patch common.PatchDetails) []string {
	function, platform, err := track.ExtractFunctionPlatform(worker)
	if err != nil {
		logging.WithError(err).Errorf(c, "Failed to split worker name: %s", worker)
		return nil
	}
	tags := []string{
		"function:" + function,
		"platform:" + platform,
		"run_id:" + strconv.FormatInt(runID, 10),
		"tricium:1",
	}
	if patch.GerritProject != "" {
		tags = append(tags,
			"gerrit_project:"+patch.GerritProject,
			"gerrit_change:"+patch.GerritChange,
			"gerrit_cl_number:"+patch.GerritCl,
			"gerrit_patch_set:"+patch.GerritPatch,
			fmt.Sprintf("buildset:patch/gerrit/%s/%s/%s", patch.GerritHost, patch.GerritCl, patch.GerritPatch),
		)
	}
	return tags
}

func fetchPatchDetails(c context.Context, runID int64) common.PatchDetails {
	var patch common.PatchDetails
	request := &track.AnalyzeRequest{ID: runID}
	if err := ds.Get(c, request); err != nil {
		logging.WithError(err).Errorf(c, "Failed to get request for run ID: %d", runID)
		return patch
	}
	patch.GitilesHost = request.GitURL
	patch.GitilesProject = request.Project
	if request.GerritProject != "" && request.GerritChange != "" {
		cl, p := gerrit.ExtractCLPatchSetNumbers(request.GitRef)
		patch.GerritHost = request.GerritHost
		patch.GerritProject = request.GerritProject
		patch.GerritChange = request.GerritChange
		patch.GerritCl = cl
		patch.GerritPatch = p
	}
	return patch
}
