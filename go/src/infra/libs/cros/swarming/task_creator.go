// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"infra/cmdsupport/cmdlib"
	sw "infra/libs/skylab/swarming"

	"github.com/google/uuid"
	"go.chromium.org/luci/auth/client/authcli"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
)

const defaultTaskPriority = 25

// TaskCreator creates Swarming tasks
type TaskCreator struct {
	// Client is Swarming API Client
	client *sw.Client
	// SwarmingService is a path to teh Swarming API
	swarmingService string
	// Session is an ID that is used to mark tasks and for tracking all of the tasks created in a logical session.
	session     string
	LUCIProject string
}

// TaskInfo contains information of the created task.
type TaskInfo struct {
	// ID of the created task in the Swarming.
	ID string
	// TaskURL provides the URL to the created task in Swarming.
	TaskURL string
}

// NewTaskCreator creates and initialize the TaskCreator.
func NewTaskCreator(ctx context.Context, authFlags *authcli.Flags, swarmingService string) (*TaskCreator, error) {
	h, err := cmdlib.NewHTTPClient(ctx, authFlags)
	if err != nil {
		return nil, errors.Annotate(err, "failed to create TaskCreator").Err()
	}
	client, err := sw.NewClient(h, swarmingService)
	if err != nil {
		return nil, errors.Annotate(err, "failed to create TaskCreator").Err()
	}

	tc := &TaskCreator{
		client:          client,
		swarmingService: swarmingService,
		session:         uuid.New().String(),
		LUCIProject:     "chromeos",
	}
	return tc, nil
}

// ReserveDUT schedule task to change DUT state to reserved
func (tc *TaskCreator) ReserveDUT(ctx context.Context, host string) (*TaskInfo, error) {
	return tc.schedule(ctx, tc.reserveDUTRequest(host))
}

// ReserveDUTRequest creates task request to change DUT state to reserved
func (tc *TaskCreator) reserveDUTRequest(host string) *swarming_api.SwarmingRpcsNewTaskRequest {
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: 2 * 60 * 60,
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command: changeDUTStateCommand("set_reserved"),
			Dimensions: []*swarming_api.SwarmingRpcsStringPair{
				{Key: "pool", Value: sw.SkylabPool},
				{Key: "id", Value: dutNameToBotID(host)},
			},
			ExecutionTimeoutSecs: int64(5 * 60),
		},
	}}
	return &swarming_api.SwarmingRpcsNewTaskRequest{
		Name: "Reserve",
		Tags: tc.combineTags("Reserve", "",
			[]string{
				fmt.Sprintf("dut-name:%s", host),
			}),
		TaskSlices: slices,
		Priority:   defaultTaskPriority,
	}
}

// Schedule registers task in the swarming
func (tc *TaskCreator) schedule(ctx context.Context, req *swarming_api.SwarmingRpcsNewTaskRequest) (*TaskInfo, error) {
	ctx, cf := context.WithTimeout(ctx, 60*time.Second)
	defer cf()
	resp, err := tc.client.CreateTask(ctx, req)
	if err != nil {
		return nil, errors.Annotate(err, "failed to create task").Err()
	}
	return &TaskInfo{
		ID:      resp.TaskId,
		TaskURL: tc.taskURL(resp.TaskId),
	}, nil
}

// taskURL generates URL to the task in swarming.
func (tc *TaskCreator) taskURL(id string) string {
	return sw.TaskURL(tc.swarmingService, id)
}

// sessionTag return admin session tag for swarming.
func (tc *TaskCreator) sessionTag() string {
	return fmt.Sprintf("admin_session:%s", tc.session)
}

// SessionTasksURL gets URL to see all created tasks belong to the session.
func (tc *TaskCreator) SessionTasksURL() string {
	tags := []string{
		tc.sessionTag(),
	}
	return sw.TaskListURLForTags(tc.swarmingService, tags)
}

func changeDUTStateCommand(task string) []string {
	return []string{
		"/bin/sh",
		"-c",
		fmt.Sprintf("/opt/infra-tools/skylab_swarming_worker -task-name %s; echo Zzz...; do sleep 180", task),
	}
}

func dutNameToBotID(hostname string) string {
	if strings.HasSuffix(hostname, ".cros") {
		hostname = strings.TrimSuffix(hostname, ".cros")
	}
	if !strings.HasPrefix(hostname, "crossk-") {
		return "crossk-" + hostname
	}
	return hostname
}

func (tc *TaskCreator) combineTags(toolName, logDogURL string, customTags []string) []string {
	tags := []string{
		fmt.Sprintf("skylab-tool:%s", toolName),
		fmt.Sprintf("luci_project:%s", tc.LUCIProject),
		fmt.Sprintf("pool:%s", sw.SkylabPool),
		tc.sessionTag(),
	}
	if logDogURL != "" {
		// log_location is required to see the logs in the swarming
		tags = append(tags, fmt.Sprintf("log_location:%s", logDogURL))
	}
	return append(tags, customTags...)
}

// PrintResults prints results of the task creation.
func (tc *TaskCreator) PrintResults(wr io.Writer, successMap map[string]*TaskInfo, errorMap map[string]error) {
	if len(errorMap) > 0 {
		fmt.Fprintln(wr, "\n### Failed to create ###")
		for host, err := range errorMap {
			fmt.Fprintf(wr, "%s: %s\n", host, err.Error())
		}
	}
	if len(successMap) > 0 {
		fmt.Fprintf(wr, "\n### Successful created Swarming tasks - %d ###\n", len(successMap))
		for host, task := range successMap {
			fmt.Fprintf(wr, "%s: %s\n", host, task.TaskURL)
		}
		if len(successMap) > 1 {
			fmt.Fprintln(wr, "\n### Batch tasks URL ###")
			fmt.Fprintln(wr, tc.SessionTasksURL())
		}
	}
}
