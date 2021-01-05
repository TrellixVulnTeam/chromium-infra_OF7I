// Copyright 2017 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	ds "go.chromium.org/luci/gae/service/datastore"
	tq "go.chromium.org/luci/gae/service/taskqueue"
	"go.chromium.org/luci/grpc/grpcutil"

	admin "infra/tricium/api/admin/v1"
	"infra/tricium/appengine/common"
	"infra/tricium/appengine/common/config"
)

// LauncherServer represents the Tricium pRPC Launcher server.
type launcherServer struct{}

// Launch processes one launch request.
//
// Specifically, this is responsible for generating the workflow, and enqueuing
// subsequent requests to trigger initial workers and update datastore.
func (r *launcherServer) Launch(c context.Context, req *admin.LaunchRequest) (res *admin.LaunchResponse, err error) {
	defer func() {
		err = grpcutil.GRPCifyAndLogErr(c, err)
	}()
	logging.Fields{
		"runID": req.RunId,
	}.Infof(c, "Request received.")
	if err := validateLaunchRequest(req); err != nil {
		return nil, errors.Annotate(err, "invalid request").
			Tag(grpcutil.InvalidArgumentTag).Err()
	}
	if err := launch(c, req, config.LuciConfigServer, common.PubsubServer); err != nil {
		return nil, err
	}
	return &admin.LaunchResponse{}, nil
}

// validateLaunchRequest returns an error if the request is invalid.
func validateLaunchRequest(req *admin.LaunchRequest) error {
	if req.RunId == 0 {
		return errors.New("missing run ID")
	}
	if req.Project == "" {
		return errors.New("missing project")
	}
	if req.GitUrl == "" {
		return errors.New("missing git URL")
	}
	if req.GitRef == "" {
		return errors.New("missing git ref")
	}
	if len(req.Files) == 0 {
		return errors.New("missing files to analyze")
	}
	return nil
}

func launch(c context.Context, req *admin.LaunchRequest, cp config.ProviderAPI, pubsub common.PubSubAPI) error {
	// Guard checking if there is already a stored workflow for the run ID
	// in the request; if so stop here.
	w := &config.Workflow{ID: req.RunId}
	if err := ds.Get(c, w); err != ds.ErrNoSuchEntity {
		logging.Infof(c, "Launch request for already-launched workflow")
		return nil
	}

	// Generate workflow and convert to string.
	sc, err := cp.GetServiceConfig(c)
	if err != nil {
		return errors.Annotate(err, "failed to get service config").Err()
	}
	pc, err := cp.GetProjectConfig(c, req.Project)
	if err != nil {
		return errors.Annotate(err, "failed to get project config").Err()
	}

	logging.Fields{
		"runID":   req.RunId,
		"project": req.Project,
	}.Infof(c, "About to generate workflow.")

	wf, err := config.Generate(sc, pc, req.Files, req.GitRef, req.GitUrl)
	if err != nil {
		// Generate may fail if there are non-recipe-based analyzers.
		// To help ease the transition, just log a warning and abort.
		logging.Fields{
			"project": req.Project,
		}.Warningf(c, "Workflow generation failed, %s.", err)
		return nil
	}
	// If there is nothing to run, abort and log a warning.
	if len(wf.Functions) == 0 {
		logging.Fields{
			"project": req.Project,
		}.Warningf(c, "Workflow had no functions to run.")
		return nil
	}

	// Set up pubsub for worker completion notification.
	err = pubsub.Setup(c)
	if err != nil {
		return errors.Annotate(err, "failed to setup pubsub for workflow").Err()
	}
	wfb, err := proto.Marshal(wf)
	if err != nil {
		return errors.Annotate(err, "failed to marshal workflow proto").Err()
	}

	// Prepare workflow config entry to store.
	wfConfig := &config.Workflow{
		ID:                 req.RunId,
		SerializedWorkflow: wfb,
	}

	// Prepare workflow launched request.
	b, err := proto.Marshal(&admin.WorkflowLaunchedRequest{RunId: req.RunId})
	if err != nil {
		return errors.Annotate(err, "failed to marshal trigger request proto").Err()
	}
	wfTask := tq.NewPOSTTask("/tracker/internal/workflow-launched", nil)
	wfTask.Payload = b

	// Prepare trigger requests for root workers.
	wTasks := []*tq.Task{}
	for _, worker := range wf.RootWorkers() {
		b, err := proto.Marshal(&admin.TriggerRequest{
			RunId:  req.RunId,
			Worker: worker,
		})
		if err != nil {
			return errors.Annotate(err, "failed to encode driver request").Err()
		}
		t := tq.NewPOSTTask("/driver/internal/trigger", nil)
		t.Payload = b
		wTasks = append(wTasks, t)
	}
	return ds.RunInTransaction(c, func(c context.Context) (err error) {
		// Store workflow config.
		if err = ds.Put(c, wfConfig); err != nil {
			return errors.Annotate(err, "failed to store workflow").Err()
		}
		// Run the below two operations in parallel.
		done := make(chan error)
		defer func() {
			if err2 := <-done; err2 != nil {
				err = err2
			}
		}()
		go func() {
			// Mark workflow as launched. Processing of this request needs the
			// stored workflow config.
			if err := tq.Add(c, common.TrackerQueue, wfTask); err != nil {
				done <- errors.Annotate(err, "failed to enqueue workflow launched track request").Err()
			}
			done <- nil
		}()
		// Trigger root workers. Processing of this request needs the stored
		// workflow config.
		if err := tq.Add(c, common.DriverQueue, wTasks...); err != nil {
			return errors.Annotate(err, "failed to enqueue trigger request(s) for root worker(s)").Err()
		}
		return nil
	}, nil)
}
