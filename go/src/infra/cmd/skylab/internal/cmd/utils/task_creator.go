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

	"go.chromium.org/luci/auth/client/authcli"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
)

// TaskCreator creates Swarming tasks
type TaskCreator struct {
	Client      *swarming.Client
	Environment site.Environment
}

// NewTaskCreator creates and initialize the TaskCreator.
func NewTaskCreator(ctx context.Context, authFlags *authcli.Flags, envFlags skycmdlib.EnvFlags) (*TaskCreator, error) {
	h, err := cmdlib.NewHTTPClient(ctx, authFlags)
	if err != nil {
		return nil, errors.Annotate(err, "failed to create TaskCreator").Err()
	}
	env := envFlags.Env()
	client, err := swarming.New(ctx, h, env.SwarmingService)
	if err != nil {
		return nil, errors.Annotate(err, "failed to create TaskCreator").Err()
	}

	tc := &TaskCreator{
		Client:      client,
		Environment: env,
	}
	return tc, nil
}

// RepairTask creates admin_repair task for particular DUT
func (tc *TaskCreator) RepairTask(ctx context.Context, host string, customTags []string, expirationSec int) (taskID string, err error) {
	id, err := tc.dutNameToBotID(ctx, host)
	if err != nil {
		return "", errors.Annotate(err, "fail to get bot ID for %s", host).Err()
	}
	c := worker.Command{
		TaskName: "admin_repair",
	}
	c.Config(tc.Environment.Wrapped())
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: int64(expirationSec),
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command: c.Args(),
			Dimensions: []*swarming_api.SwarmingRpcsStringPair{
				{Key: "pool", Value: "ChromeOSSkylab"},
				{Key: "id", Value: id},
			},
			ExecutionTimeoutSecs: 5400,
		},
		WaitForCapacity: true,
	}}
	tags := []string{
		fmt.Sprintf("log_location:%s", c.LogDogAnnotationURL),
		fmt.Sprintf("luci_project:%s", tc.Environment.LUCIProject),
		"pool:ChromeOSSkylab",
		"skylab-tool:repair",
	}
	tags = append(tags, customTags...)
	r := &swarming_api.SwarmingRpcsNewTaskRequest{
		Name:           "admin_repair",
		Tags:           tags,
		TaskSlices:     slices,
		Priority:       25,
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
func (tc *TaskCreator) VerifyTask(ctx context.Context, host string, expirationSec int) (taskID string, err error) {
	id, err := tc.dutNameToBotID(ctx, host)
	if err != nil {
		return "", errors.Annotate(err, "fail to get bot ID for %s", host).Err()
	}
	c := worker.Command{
		TaskName: "admin_verify",
	}
	c.Config(tc.Environment.Wrapped())
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: int64(expirationSec),
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command: c.Args(),
			Dimensions: []*swarming_api.SwarmingRpcsStringPair{
				{Key: "pool", Value: "ChromeOSSkylab"},
				{Key: "id", Value: id},
			},
			ExecutionTimeoutSecs: 5400,
		},
		WaitForCapacity: true,
	}}
	r := &swarming_api.SwarmingRpcsNewTaskRequest{
		Name: "admin_verify",
		Tags: []string{
			fmt.Sprintf("log_location:%s", c.LogDogAnnotationURL),
			fmt.Sprintf("luci_project:%s", tc.Environment.LUCIProject),
			"pool:ChromeOSSkylab",
			"skylab-tool:verify",
		},
		TaskSlices:     slices,
		Priority:       25,
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

// LeaseByHostnameTask creates lease_task for particular DUT
func (tc *TaskCreator) LeaseByHostnameTask(ctx context.Context, host string, durationSec int, reason string) (taskID string, err error) {
	id, err := tc.dutNameToBotID(ctx, host)
	if err != nil {
		return "", err
	}
	slices := []*swarming_api.SwarmingRpcsTaskSlice{{
		ExpirationSecs: 10 * 60,
		Properties: &swarming_api.SwarmingRpcsTaskProperties{
			Command: getLeaseCommand(),
			Dimensions: []*swarming_api.SwarmingRpcsStringPair{
				{Key: "pool", Value: "ChromeOSSkylab"},
				{Key: "id", Value: id},
			},
			ExecutionTimeoutSecs: int64(durationSec),
		},
	}}
	r := &swarming_api.SwarmingRpcsNewTaskRequest{
		Name: "lease task",
		Tags: []string{
			"pool:ChromeOSSkylab",
			"skylab-tool:lease",
			// This quota account specifier is only relevant for DUTs that are
			// in the prod skylab DUT_POOL_QUOTA pool; it is irrelevant and
			// harmless otherwise.
			"qs_account:leases",
			"lease-by:hostname",
			fmt.Sprintf("dut-name:%s", host),
			fmt.Sprintf("lease-reason:%s", reason),
		},
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
				{Key: "pool", Value: "ChromeOSSkylab"},
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
		Tags: appendUniqueTags([]string{
			"pool:ChromeOSSkylab",
			"skylab-tool:lease",
			// This quota account specifier is only relevant for DUTs that are
			// in the prod skylab DUT_POOL_QUOTA pool; it is irrelevant and
			// harmless otherwise.
			"qs_account:leases",
			"lease-by:model",
			fmt.Sprintf("model:%s", model),
			fmt.Sprintf("lease-reason:%s", reason),
		}, convertTags(dims)...),
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
	dims := []*swarming_api.SwarmingRpcsStringPair{
		{Key: "pool", Value: "ChromeOSSkylab"},
		{Key: "dut_name", Value: host},
	}
	ids, err := tc.Client.GetBotIDs(ctx, dims)
	switch {
	case err != nil:
		return "", errors.Annotate(err, "failed to find bot").Err()
	case len(ids) == 0:
		return "", errors.Reason("not found any bot with dut_name: %v", host).Err()
	case len(ids) > 1:
		return "", errors.Reason("more that one bot with dut_name: %v", host).Err()
	}
	return ids[0], nil
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
