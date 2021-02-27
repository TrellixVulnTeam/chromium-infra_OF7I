// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cron

import (
	"go.chromium.org/luci/common/tsmon/field"
	"go.chromium.org/luci/common/tsmon/metric"
)

var (
	dumpInventorySnapshotTick = metric.NewCounter(
		"chromeos/inventory/dump_lab_config_snapshot",
		"dumpInventorySnapshot attempt",
		nil,
		field.Bool("success"), // If the attempt succeed
	)

	dumpInventoryDeviceSnapshotTick = metric.NewCounter(
		"chromeos/inventory/dump_ufs_lab_config_snapshot",
		"dumpInventoryDeviceSnapshot attempt",
		nil,
		field.Bool("success"), // If the attempt succeed
	)

	dumpInventoryDutStateSnapshotTick = metric.NewCounter(
		"chromeos/inventory/dump_ufs_state_config_snapshot",
		"dumpInventoryDutStateSnapshot attempt",
		nil,
		field.Bool("success"), // If the attempt succeed
	)
)
