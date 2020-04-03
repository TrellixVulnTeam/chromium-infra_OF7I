// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"context"
	"fmt"
	"time"

	"infra/cmd/skylab/internal/site"
	"infra/libs/skylab/swarming"
	"infra/libs/skylab/worker"

	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
)

// TaskCreator creates Swarming tasks
type TaskCreator struct {
	Client      *swarming.Client
	Environment site.Environment
}

// RepairTask creates admin_repair task for particular DUT
func (tc *TaskCreator) RepairTask(ctx context.Context, host string, customTags []string, expirationSec int) (taskID string, err error) {
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
				{Key: "dut_name", Value: host},
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
