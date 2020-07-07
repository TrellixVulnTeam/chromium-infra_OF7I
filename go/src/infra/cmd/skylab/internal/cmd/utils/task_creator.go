// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"context"
	"fmt"
	"strings"
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

// RepairTask creates admin_repair task for particular DUT
func (tc *TaskCreator) RepairTask(ctx context.Context, host string, expirationSec int) (taskID string, err error) {
	dims, err := tc.dimsWithBotID(ctx, host)
	if err != nil {
		return "", errors.Annotate(err, "failed to get dimensions for %s", host).Err()
	}
	c := worker.Command{
		TaskName: "admin_repair",
	}
	c.Config(tc.Environment.Wrapped())
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: int64(expirationSec),
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command:              c.Args(),
			Dimensions:           dims,
			ExecutionTimeoutSecs: 5400,
		},
		WaitForCapacity: true,
	}}
	r := &swarming_api.SwarmingRpcsNewTaskRequest{
		Name:           "admin_repair",
		Tags:           tc.combineTags("repair", c.LogDogAnnotationURL, nil),
		TaskSlices:     slices,
		Priority:       defaultTaskPriority,
		ServiceAccount: tc.Environment.ServiceAccount,
	}
	ctx, cf := context.WithTimeout(ctx, 60*time.Second)
	defer cf()
	resp, err := tc.Client.CreateTask(ctx, r)
	if err != nil {
		return "", errors.Annotate(err, "failed to create task").Err()
	}
	return resp.TaskId, nil
}

// VerifyTask creates admin_verify task for particular DUT.
func (tc *TaskCreator) VerifyTask(ctx context.Context, host string, expirationSec int) (TaskInfo, error) {
	dims, err := tc.dimsWithBotID(ctx, host)
	if err != nil {
		return TaskInfo{}, errors.Annotate(err, "failed to get dimensions for %s", host).Err()
	}
	c := worker.Command{
		TaskName: "admin_verify",
	}
	c.Config(tc.Environment.Wrapped())
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: int64(expirationSec),
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command:              c.Args(),
			Dimensions:           dims,
			ExecutionTimeoutSecs: 5400,
		},
		WaitForCapacity: true,
	}}
	r := &swarming_api.SwarmingRpcsNewTaskRequest{
		Name:           "admin_verify",
		Tags:           tc.combineTags("verify", c.LogDogAnnotationURL, nil),
		TaskSlices:     slices,
		Priority:       defaultTaskPriority,
		ServiceAccount: tc.Environment.ServiceAccount,
	}
	ctx, cf := context.WithTimeout(ctx, 60*time.Second)
	defer cf()
	resp, err := tc.Client.CreateTask(ctx, r)
	if err != nil {
		return TaskInfo{}, errors.Annotate(err, "failed to create task").Err()
	}
	task := TaskInfo{
		ID:      resp.TaskId,
		TaskURL: tc.taskURL(resp.TaskId),
	}
	return task, nil
}

// AuditTask creates admin_audit task for particular DUT.
func (tc *TaskCreator) AuditTask(ctx context.Context, host, actions string, expirationSec int) (TaskInfo, error) {
	dims, err := tc.dimsWithBotID(ctx, host)
	if err != nil {
		return TaskInfo{}, errors.Annotate(err, "failed to get dimensions for %s", host).Err()
	}
	c := worker.Command{
		TaskName: "admin_audit",
		Actions:  actions,
	}
	c.Config(tc.Environment.Wrapped())
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: int64(expirationSec),
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command:              c.Args(),
			Dimensions:           dims,
			ExecutionTimeoutSecs: 5400,
		},
		WaitForCapacity: true,
	}}
	r := &swarming_api.SwarmingRpcsNewTaskRequest{
		Name:           "admin_audit",
		Tags:           tc.combineTags("audit", c.LogDogAnnotationURL, nil),
		TaskSlices:     slices,
		Priority:       defaultTaskPriority,
		ServiceAccount: tc.Environment.ServiceAccount,
	}
	ctx, cf := context.WithTimeout(ctx, 60*time.Second)
	defer cf()
	resp, err := tc.Client.CreateTask(ctx, r)
	if err != nil {
		return TaskInfo{}, errors.Annotate(err, "failed to create task").Err()
	}
	task := TaskInfo{
		ID:      resp.TaskId,
		TaskURL: tc.taskURL(resp.TaskId),
	}
	return task, nil
}

// LeaseByHostnameTask creates lease_task for particular DUT
func (tc *TaskCreator) LeaseByHostnameTask(ctx context.Context, host string, durationSec int, reason string) (taskID string, err error) {
	dims, err := tc.dimsWithBotID(ctx, host)
	if err != nil {
		return "", errors.Annotate(err, "failed to get dimensions for %s", host).Err()
	}
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: 10 * 60,
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command:              getLeaseCommand(),
			Dimensions:           dims,
			ExecutionTimeoutSecs: int64(durationSec),
		},
	}}
	r := &swarming_api.SwarmingRpcsNewTaskRequest{
		Name: "lease task",
		Tags: tc.combineTags("lease", "",
			[]string{
				// This quota account specifier is only relevant for DUTs that are
				// in the prod skylab DUT_POOL_QUOTA pool; it is irrelevant and
				// harmless otherwise.
				"qs_account:leases",
				"lease-by:hostname",
				fmt.Sprintf("dut-name:%s", host),
				fmt.Sprintf("lease-reason:%s", reason),
			}),
		TaskSlices:     slices,
		Priority:       15,
		ServiceAccount: tc.Environment.ServiceAccount,
	}
	ctx, cf := context.WithTimeout(ctx, 60*time.Second)
	defer cf()
	resp, err := tc.Client.CreateTask(ctx, r)
	if err != nil {
		return "", errors.Annotate(err, "failed to create task").Err()
	}
	return resp.TaskId, nil
}

// LeaseByModelTask creates a lease_task targeted at a particular model and dimensions
func (tc *TaskCreator) LeaseByModelTask(ctx context.Context, model string, dims map[string]string, durationSec int, reason string) (taskID string, err error) {
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: 10 * 60,
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command: getLeaseCommand(),
			Dimensions: appendUniqueDimensions([]*swarming_api.SwarmingRpcsStringPair{
				{Key: "pool", Value: swarming.SkylabPool},
				{Key: "label-model", Value: model},
				// We need to make sure we don't disturb DUT_POOL_CTS, so for now by-model leases
				// can only target DUT_POOL_QUOTA.
				{Key: "label-pool", Value: "DUT_POOL_QUOTA"},
				// Getting an unhealthy DUT is a horrible user experience, so we make sure
				// that only ready DUTs are leasable by model.
				{Key: "dut_state", Value: "ready"},
			}, convertDimensions(dims)...),
			ExecutionTimeoutSecs: int64(durationSec),
		},
	}}
	r := &swarming_api.SwarmingRpcsNewTaskRequest{
		Name: "lease task",
		Tags: appendUniqueTags(
			tc.combineTags("lease", "",
				[]string{
					// This quota account specifier is only relevant for DUTs that are
					// in the prod skylab DUT_POOL_QUOTA pool; it is irrelevant and
					// harmless otherwise.
					"qs_account:leases",
					"lease-by:model",
					fmt.Sprintf("model:%s", model),
					fmt.Sprintf("lease-reason:%s", reason),
				}),
			convertTags(dims)...),
		TaskSlices:     slices,
		Priority:       15,
		ServiceAccount: tc.Environment.ServiceAccount,
	}
	ctx, cf := context.WithTimeout(ctx, 60*time.Second)
	defer cf()
	resp, err := tc.Client.CreateTask(ctx, r)
	if err != nil {
		return "", errors.Annotate(err, "lease by model task").Err()
	}
	return resp.TaskId, nil
}

// LeaseByBoardTask creates a lease_task targeted at a particular board and dimensions.
func (tc *TaskCreator) LeaseByBoardTask(ctx context.Context, board string, dims map[string]string, durationSec int, reason string) (taskID string, err error) {
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: 600,
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command: getLeaseCommand(),
			Dimensions: appendUniqueDimensions([]*swarming_api.SwarmingRpcsStringPair{
				{Key: "pool", Value: swarming.SkylabPool},
				{Key: "label-board", Value: board},
				// We need to make sure we don't disturb DUT_POOL_CTS, so for now by-model leases
				// can only target DUT_POOL_QUOTA.
				{Key: "label-pool", Value: "DUT_POOL_QUOTA"},
				// Getting an unhealthy DUT is a horrible user experience, so we make sure
				// that only ready DUTs are leasable by model.
				{Key: "dut_state", Value: "ready"},
			}, convertDimensions(dims)...),
			ExecutionTimeoutSecs: int64(durationSec),
		},
	}}
	r := &swarming_api.SwarmingRpcsNewTaskRequest{
		Name: "lease task",
		Tags: appendUniqueTags(
			tc.combineTags("lease", "",
				[]string{
					// This quota account specifier is only relevant for DUTs that are
					// in the prod skylab DUT_POOL_QUOTA pool; it is irrelevant and
					// harmless otherwise.
					"qs_account:leases",
					"lease-by:model",
					"board:" + board,
					"lease-reason:" + reason,
				}),
			convertTags(dims)...),
		TaskSlices:     slices,
		Priority:       15,
		ServiceAccount: tc.Environment.ServiceAccount,
	}
	ctx, cf := context.WithTimeout(ctx, 60*time.Second)
	defer cf()
	resp, err := tc.Client.CreateTask(ctx, r)
	if err != nil {
		return "", errors.Annotate(err, "lease by board task").Err()
	}
	return resp.TaskId, nil
}

// convertDimensions takes a map and converts it into a string pair.
func convertDimensions(m map[string]string) []*swarming_api.SwarmingRpcsStringPair {
	var out []*swarming_api.SwarmingRpcsStringPair
	for k, v := range m {
		out = append(out, &swarming_api.SwarmingRpcsStringPair{Key: k, Value: v})
	}
	return out
}

// appendUniqueDimensions takes a base []*swarming_api.SwarmingRpcsStringPair and an arbitrary
// number of key-value pairs and appends them onto the first slice.
func appendUniqueDimensions(first []*swarming_api.SwarmingRpcsStringPair, rest ...*swarming_api.SwarmingRpcsStringPair) []*swarming_api.SwarmingRpcsStringPair {
	seen := make(map[string]bool)
	for _, item := range first {
		seen[item.Key] = true
	}

	for _, item := range rest {
		if seen[item.Key] {
			continue
		}
		first = append(first, item)
		seen[item.Key] = true
	}
	return first
}

// convertTags takes a map and converts it to a slice of strings in
// swarming dimension format.
func convertTags(m map[string]string) []string {
	var out []string
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s:%s", k, v))
	}
	return out
}

func (tc *TaskCreator) dutNameToBotID(ctx context.Context, host string) (string, error) {
	return tc.Client.DutNameToBotID(ctx, host)
}

// getLeaseCommand provides bash command to set dut state and run loop to keep DUT busy
//
// DUT state will be set as 'needs_repair'
func getLeaseCommand() []string {
	return []string{"/bin/sh", "-c", `/opt/infra-tools/skylab_swarming_worker -task-name set_needs_repair; while true; do sleep 60; echo Zzz...; done`}
}

// appendUniqueTags takes a []string and adds items to it if they're unique
func appendUniqueTags(first []string, rest ...string) []string {
	seen := make(map[string]bool)
	for _, item := range first {
		key := getTagPrefix(item)
		seen[key] = true
	}
	for _, item := range rest {
		key := getTagPrefix(item)
		if seen[key] {
			continue
		}
		first = append(first, item)
		seen[key] = true
	}
	return first
}

// getTagPrefix gets the key in a string of the form key:value.
func getTagPrefix(s string) string {
	delimIdx := strings.Index(s, ":")
	if delimIdx == -1 {
		return ""
	}
	return s[0:delimIdx]
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
	return swarming.TaskListURLForTags(tc.Environment.SwarmingService, tags)
}

// taskURL generates URL to the task in swarming.
func (tc *TaskCreator) taskURL(id string) string {
	return swarming.TaskURL(tc.Environment.SwarmingService, id)
}

func (tc *TaskCreator) dimsWithBotID(ctx context.Context, host string) ([]*swarming_api.SwarmingRpcsStringPair, error) {
	id, err := tc.dutNameToBotID(ctx, host)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get bot ID for %s", host).Err()
	}
	return []*swarming_api.SwarmingRpcsStringPair{
		{Key: "pool", Value: swarming.SkylabPool},
		{Key: "id", Value: id},
	}, nil
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
