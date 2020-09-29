// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"go.chromium.org/luci/common/logging"
	ds "go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/server/router"

	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"

	"infra/appengine/cr-audit-commits/app/rules"
)

// Scheduler is the periodic task that:
//
//   - Determines the concrete ref for every audit configuration configured.
//   - Creates a new RepoState entry for any new refs.
//   - Schedules an audit task for each active ref in the appropriate queue.
func (a *app) Schedule(rc *router.Context) {
	ctx, resp := rc.Context, rc.Writer

	for configName, config := range configGet(ctx) {
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
				Parent: "projects/" + os.Getenv("GOOGLE_CLOUD_PROJECT") + "/locations/us-central1/queues/default",
				Task: &taskspb.Task{
					MessageType: &taskspb.Task_AppEngineHttpRequest{
						AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
							HttpMethod:  taskspb.HttpMethod_GET,
							RelativeUri: fmt.Sprintf("/_task/auditor?refUrl=%s", url.QueryEscape(refConfig.RepoURL())),
						},
					},
				},
			}

			_, err = a.cloudTasksClient.CreateTask(ctx, req)

			if err != nil {
				logging.WithError(err).Errorf(ctx, "Could not schedule audit for %s due to %s", refConfig.RepoURL(), err.Error())
				RefAuditsDue.Add(ctx, 1, false)
				http.Error(resp, err.Error(), http.StatusInternalServerError)

				continue // Should just return after setting response code, no?
			}
			RefAuditsDue.Add(ctx, 1, true)
		}
	}

	resp.WriteHeader(http.StatusOK)
}
