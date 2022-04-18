// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"context"
	"fmt"
	"io"

	"go.chromium.org/luci/common/errors"

	"infra/cmd/shivas/site"
	"infra/cros/recovery/tasknames"
	"infra/libs/skylab/buildbucket"
	"infra/libs/skylab/buildbucket/labpack"
	"infra/libs/skylab/swarming"
)

// ScheduleDeployTask schedules a deploy task by Buildbucket for PARIS.
func ScheduleDeployTask(ctx context.Context, bc buildbucket.Client, e site.Environment, unit, sessionTag string) error {
	if unit == "" {
		return errors.Reason("schedule deploy task: unit name is empty").Err()
	}
	p := &labpack.Params{
		UnitName:       unit,
		TaskName:       string(tasknames.Deploy),
		EnableRecovery: true,
		AdminService:   e.AdminService,
		// NOTE: We use the UFS service, not the Inventory service here.
		InventoryService: e.UnifiedFleetService,
		UpdateInventory:  true,
		ExtraTags: []string{
			sessionTag,
		},
	}
	taskID, err := labpack.ScheduleTask(ctx, bc, labpack.CIPDProd, p)
	if err != nil {
		return errors.Annotate(err, "schedule deploy task").Err()
	}
	fmt.Printf("Triggered Deploy task %s. Follow the deploy job at %s\n", p.UnitName, bc.BuildURL(taskID))
	return nil
}

// PrintTasksBatchLink prints batch link for scheduled tasks.
func PrintTasksBatchLink(wr io.Writer, swarmingService, commonTag string) {
	fmt.Fprintf(wr, "### Batch tasks URL ###\n")
	fmt.Fprintf(wr, "Created tasks: %s\n", TasksBatchLink(swarmingService, commonTag))
}

// TasksBatchLink created batch link to swarming for scheduled tasks.
func TasksBatchLink(swarmingService, commonTag string) string {
	return swarming.TaskListURLForTags(swarmingService, []string{commonTag})
}
