// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"context"
	"fmt"
	"time"

	skycmdlib "infra/cmd/skylab/internal/cmd/cmdlib"
	"infra/cmd/skylab/internal/site"
	"infra/cmdsupport/cmdlib"
	"infra/libs/skylab/swarming"
	"infra/libs/skylab/worker"

	"github.com/google/uuid"
	"go.chromium.org/luci/auth/client/authcli"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
)

const defaultTaskPriority = 25

// TaskCreator creates Swarming tasks
type TaskCreator struct {
	Client      *swarming.Client
	Environment site.Environment
	// Session is an ID that is used to mark tasks and for tracking all of the tasks created in a logical session.
	session string
}

// TaskInfo contains information of the created task.
type TaskInfo struct {
	// ID of the created task in the Swarming.
	ID string
	// TaskURL provides the URL to the created task in Swarming.
	TaskURL string
}

// NewTaskCreator creates and initialize the TaskCreator.
func NewTaskCreator(ctx context.Context, authFlags *authcli.Flags, envFlags skycmdlib.EnvFlags) (*TaskCreator, error) {
	h, err := cmdlib.NewHTTPClient(ctx, authFlags)
	if err != nil {
		return nil, errors.Annotate(err, "failed to create TaskCreator").Err()
	}
	env := envFlags.Env()
	client, err := swarming.NewClient(h, env.SwarmingService)
	if err != nil {
		return nil, errors.Annotate(err, "failed to create TaskCreator").Err()
	}

	tc := &TaskCreator{
		Client:      client,
		Environment: env,
		session:     uuid.New().String(),
	}
	return tc, nil
}

// IsSwarmingTaskErr returns true if the given error is because of a swarming task failure
func IsSwarmingTaskErr(e error) bool {
	_, ok := e.(swarmingTaskError)
	return ok
}

type swarmingTaskError struct {
	err error
}

func (e swarmingTaskError) Error() string {
	return e.err.Error()
}

// Set it to 2 hours to allow deploy to finish
const deployTaskExecutionTimeout = 7200

// Set it 5 hours, to allow any kicked off deploy tasks to finish
const deployTaskExpirationTimeout = 18000

// Set deploy task as the highest priority to avoid the case that a scheduled repair job is run before a scheduled deployment task
const deployTaskPriority = 24

// DeployTask creates deploy task for a particular DUT
//
// The deployment task's parameters are hardcoded by the system instead of users now.
// TODO: Call DeployTask in add-dut and update-dut directly instead of calling crosskylabadmin.
func (tc *TaskCreator) DeployTask(ctx context.Context, dutID, actions string) (taskID string, err error) {
	r := tc.getDeployTaskRequest(dutID, actions)
	ctx, cf := context.WithTimeout(ctx, 60*time.Second)
	defer cf()
	resp, err := tc.Client.CreateTask(ctx, r)
	if err != nil {
		return "", swarmingTaskError{err}
	}
	return resp.TaskId, nil
}

func (tc *TaskCreator) getDeployTaskRequest(dutID, actions string) *swarming_api.SwarmingRpcsNewTaskRequest {
	c := worker.Command{
		TaskName: "deploy",
		Actions:  actions,
	}
	c.Config(tc.Environment.Wrapped())
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: deployTaskExpirationTimeout,
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command:              c.Args(),
			Dimensions:           dimsWithDUTID(dutID),
			ExecutionTimeoutSecs: deployTaskExecutionTimeout,
			// We never want tasks deduplicated with earlier tasks.
			Idempotent: false,
		},
		WaitForCapacity: true,
	}}
	return &swarming_api.SwarmingRpcsNewTaskRequest{
		Name:           "deploy",
		Tags:           tc.combineTags("deploy", c.LogDogAnnotationURL, []string{fmt.Sprintf("deploy_task:%s", dutID)}),
		TaskSlices:     slices,
		Priority:       deployTaskPriority,
		ServiceAccount: tc.Environment.ServiceAccount,
	}
}

func (tc *TaskCreator) dutNameToBotID(ctx context.Context, host string) (string, error) {
	return tc.Client.DutNameToBotID(ctx, host)
}

// getLeaseCommand provides bash command to set dut state and run loop to keep DUT busy
//
// DUT state will be set as 'needs_repair'.
func getLeaseCommand(updateDutState bool) []string {
	if updateDutState {
		return []string{"/bin/sh", "-c", `/opt/infra-tools/skylab_swarming_worker -task-name set_needs_repair; while true; do sleep 60; echo Zzz...; done`}
	}
	return []string{"/bin/sh", "-c", `while true; do sleep 60; echo Zzz...; done`}
}

// sessionTag return admin session tag for swarming.
func (tc *TaskCreator) sessionTag() string {
	return fmt.Sprintf("admin_session:%s", tc.session)
}

// taskURL generates URL to the task in swarming.
func (tc *TaskCreator) taskURL(id string) string {
	return swarming.TaskURL(tc.Environment.SwarmingService, id)
}

func dimsWithDUTID(dutID string) []*swarming_api.SwarmingRpcsStringPair {
	return []*swarming_api.SwarmingRpcsStringPair{
		{Key: "pool", Value: swarming.SkylabPool},
		{Key: "dut_id", Value: dutID},
	}
}

func (tc *TaskCreator) combineTags(toolName, logDogURL string, customTags []string) []string {
	tags := []string{
		fmt.Sprintf("skylab-tool:%s", toolName),
		fmt.Sprintf("luci_project:%s", tc.Environment.LUCIProject),
		fmt.Sprintf("pool:%s", swarming.SkylabPool),
		tc.sessionTag(),
	}
	if logDogURL != "" {
		// log_location is required to see the logs in the swarming
		tags = append(tags, fmt.Sprintf("log_location:%s", logDogURL))
	}
	return append(tags, customTags...)
}
