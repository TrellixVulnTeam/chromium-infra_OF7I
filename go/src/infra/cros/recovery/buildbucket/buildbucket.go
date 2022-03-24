// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package buildbucket

import (
	"context"

	"go.chromium.org/luci/auth/client/authcli"
	"go.chromium.org/luci/grpc/prpc"

	"infra/cros/recovery/tasknames"
	"infra/libs/skylab/buildbucket"
	"infra/libs/skylab/buildbucket/labpack"
)

// Client is a buildbucket client.
type Client = buildbucket.Client

// NewLabpackClient creates a new client directed at the "labpack" chromeos builder.
func NewLabpackClient(ctx context.Context, authFlags authcli.Flags, prpcOptions *prpc.Options) (buildbucket.Client, error) {
	return buildbucket.NewClient(ctx, authFlags, prpcOptions, "chromeos", "labpack", "labpack")
}

// ScheduleBuilder schedules a build using the specified task name (e.g. Deploy).
func ScheduleBuilder(ctx context.Context, bc Client, host string, taskName tasknames.TaskName, adminService string, unifiedFleetService string) (int64, error) {
	p := &labpack.Params{
		UnitName:       host,
		TaskName:       string(taskName),
		EnableRecovery: true,
		AdminService:   adminService,
		// NOTE: We use the UFS service, not the Inventory service here.
		InventoryService: unifiedFleetService,
		NoStepper:        false,
		NoMetrics:        false,
		UpdateInventory:  true,
		// TODO(gregorynisbet): Pass config file to labpack task.
		Configuration: "",
	}
	taskID, err := labpack.ScheduleTask(ctx, bc, p)
	if err != nil {
		return 0, err
	}
	return taskID, nil
}
