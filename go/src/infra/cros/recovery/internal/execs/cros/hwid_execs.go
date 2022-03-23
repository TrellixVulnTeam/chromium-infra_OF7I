// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	readHWIDCommand = "crossystem hwid"
)

// updateHWIDToInvExec read HWID from the resource and update DUT info.
func updateHWIDToInvExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	hwid, err := run(ctx, time.Minute, readHWIDCommand)
	if err != nil {
		return errors.Annotate(err, "update HWID in DUT-info").Err()
	}
	if hwid == "" {
		return errors.Reason("update HWID in DUT-info: is empty").Err()
	}
	log.Debugf(ctx, "Update HWID %q in DUT-info.", hwid)
	info.RunArgs.DUT.Hwid = hwid
	return nil
}

// matchHWIDToInvExec matches HWID from the resource to value in the Inventory.
func matchHWIDToInvExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	actualHWID, err := run(ctx, time.Minute, readHWIDCommand)
	if err != nil {
		return errors.Annotate(err, "match HWID to inventory").Err()
	}
	expectedHWID := info.RunArgs.DUT.Hwid
	if actualHWID != expectedHWID {
		return errors.Reason("match HWID to inventory: failed, expected: %q, but got %q", expectedHWID, actualHWID).Err()
	}
	return nil
}

func init() {
	execs.Register("cros_update_hwid_to_inventory", updateHWIDToInvExec)
	execs.Register("cros_match_hwid_to_inventory", matchHWIDToInvExec)
}
