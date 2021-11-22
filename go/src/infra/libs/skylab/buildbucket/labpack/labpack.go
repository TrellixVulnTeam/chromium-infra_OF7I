// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package labpack

import (
	"context"

	"infra/libs/skylab/buildbucket"

	structbuilder "google.golang.org/protobuf/types/known/structpb"
)

// Params are the parameters to the labpack job.
type Params struct {
	// UnitName is the DUT or similar that we are scheduling against.
	// For example, a DUT hostname is a valid UnitName.
	UnitName string
	// TaskName is used to drive the recovery process, e.g. "labstation_deploy".
	TaskName string
	// Whether recovery actions are enabled or not.
	EnableRecovery bool
	// Hostname of the admin service.
	AdminService string
	// Hostname of the inventory service.
	InventoryService string
	// Whether to update the inventory or not when the task is finished.
	UpdateInventory bool
	// NoStepper determines whether the log stepper things.
	NoStepper bool
	// NoMetrics determines whether metrics recording (Karte) is in effect.
	NoMetrics bool
	// Configuration is a base64-encoded string of the job config.
	Configuration string
}

// AsMap takes the parameters and flattens it into a map with string keys.
func (p *Params) AsMap() map[string]interface{} {
	return map[string]interface{}{
		"unit_name":         p.UnitName,
		"task_name":         p.TaskName,
		"enable_recovery":   p.EnableRecovery,
		"admin_service":     p.AdminService,
		"inventory_service": p.InventoryService,
		"update_inventory":  p.UpdateInventory,
		"no_stepper":        p.NoStepper,
		"no_metrics":        p.NoMetrics,
		"configuration":     p.Configuration,
	}
}

// ScheduleTask schedules a buildbucket task.
func ScheduleTask(ctx context.Context, client buildbucket.Client, params *Params) (int64, error) {
	props, err := structbuilder.NewStruct(params.AsMap())
	if err != nil {
		return 0, err
	}
	taskID, err := client.ScheduleLabpackTask(ctx, params.UnitName, props)
	if err != nil {
		return 0, err
	}
	return taskID, nil
}
