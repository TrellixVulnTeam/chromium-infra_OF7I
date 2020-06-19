// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"net/url"

	ds "go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/server/router"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"

	"infra/appengine/cr-audit-commits/app/rules"
)

// Scheduler is the periodic task that:
//
//   - Determines the concrete ref for every audit configuration configured.
//   - Creates a new RepoState entry for any new refs.
//   - Schedules an audit task for each active ref in the appropriate queue.
func Scheduler(rc *router.Context) {
	ctx, resp := rc.Context, rc.Writer

	// Create a new cloud task client
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		logging.WithError(err).Errorf(ctx, "Could not create cloud task client due to %s", err.Error())
	}
	defer client.Close()

	for configName, config := range rules.GetRuleMap() {
		var refConfigs []*rules.RefConfig
		var err error
		if config.DynamicRefFunction != nil {
			refConfigs, err = config.DynamicRefFunction(ctx, *config)
			if err != nil {
				logging.WithError(err).Errorf(ctx, "Could not determine the concrete ref for %s due to %s", configName, err.Error())
				RefAuditsDue.Add(ctx, 1, false)
				continue
			}
		} else {
			refConfigs = []*rules.RefConfig{config}
		}
		for _, refConfig := range refConfigs {
			state := &rules.RepoState{RepoURL: refConfig.RepoURL()}
			err = ds.Get(ctx, state)
			switch err {
			case ds.ErrNoSuchEntity:
				state.ConfigName = configName
				state.Metadata = refConfig.Metadata
				state.BranchName = refConfig.BranchName
				state.LastKnownCommit = refConfig.StartingCommit
				if err = ds.Put(ctx, state); err != nil {
					logging.WithError(err).Errorf(ctx, "Could not save ref state for %s due to %s", configName, err.Error())
					RefAuditsDue.Add(ctx, 1, false)
					continue
				}
			case nil:
				break
			default:
				http.Error(resp, err.Error(), 500)
				return
			}

			// Build the Task payload.
			req := &taskspb.CreateTaskRequest{
				Parent: "projects/cr-audit-commits/locations/us-central1/queues/default",
				Task: &taskspb.Task{
					MessageType: &taskspb.Task_AppEngineHttpRequest{
						AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
							HttpMethod:  taskspb.HttpMethod_GET,
							RelativeUri: fmt.Sprintf("/_task/auditor?refUrl=%s", url.QueryEscape(refConfig.RepoURL())),
						},
					},
				},
			}

			_, err := client.CreateTask(ctx, req)
			if err != nil {
				logging.WithError(err).Errorf(ctx, "Could not schedule audit for %s due to %s", refConfig.RepoURL(), err.Error())
				RefAuditsDue.Add(ctx, 1, false)
				continue
			}
			RefAuditsDue.Add(ctx, 1, true)
		}
	}
}
