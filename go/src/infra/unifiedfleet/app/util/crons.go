// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

var CronJobNames = map[string]string{
	"mainBQCron":                 "ufs.dumper",
	"changeEventToBQCron":        "ufs.change_event.BqDump",
	"snapshotToBQCron":           "ufs.snapshot_msg.BqDump",
	"networkConfigToBQCron":      "ufs.cros_network.dump",
	"hartSyncCron":               "ufs.sync_devices.sync",
	"droneQueenSyncCron":         "ufs.push_to_drone_queen",
	"InventoryMetricsReportCron": "ufs.report_inventory",
}
