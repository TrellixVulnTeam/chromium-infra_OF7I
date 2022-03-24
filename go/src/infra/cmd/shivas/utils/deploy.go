// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"context"
	"fmt"

	"infra/cmd/shivas/site"
	"infra/cros/recovery/tasknames"
	"infra/libs/skylab/buildbucket"
	"infra/libs/skylab/buildbucket/labpack"
)

// ScheduleDeployTask schedules a deploy task by Buildbucket for PARIS.
func ScheduleDeployTask(ctx context.Context, bc buildbucket.Client, e site.Environment, unit string) error {
	p := &labpack.Params{
		UnitName:       unit,
		TaskName:       string(tasknames.Deploy),
		EnableRecovery: true,
		AdminService:   e.AdminService,
		// NOTE: We use the UFS service, not the Inventory service here.
		InventoryService: e.UnifiedFleetService,
		UpdateInventory:  true,
	}
	taskID, err := labpack.ScheduleTask(ctx, bc, p)
	if err != nil {
		return err
	}
	fmt.Printf("Triggered Deploy task for DUT %s. Follow the deploy job at %s\n", p.UnitName, bc.BuildURL(taskID))
	return nil
}
