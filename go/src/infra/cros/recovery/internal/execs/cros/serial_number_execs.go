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
	readSerialNumberCommand = "vpd -g serial_number"
)

// updateSerialNumberToInvExec updates serial number in DUT-info.
func updateSerialNumberToInvExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	sn, err := run(ctx, time.Minute, readSerialNumberCommand)
	if err != nil {
		return errors.Annotate(err, "update serial number in DUT-info").Err()
	}
	if sn == "" {
		return errors.Reason("update serial number in DUT-info: is empty").Err()
	}
	log.Debugf(ctx, "Update serial_number %q in DUT-info.", sn)
	info.RunArgs.DUT.SerialNumber = sn
	return nil
}

// matchSerialNumberToInvExec matches serial number from the resource to value in the Inventory.
func matchSerialNumberToInvExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	actualSerialNumber, err := run(ctx, time.Minute, readSerialNumberCommand)
	if err != nil {
		return errors.Annotate(err, "match serial number to inventory").Err()
	}
	expectedSerialNumber := info.RunArgs.DUT.SerialNumber
	if actualSerialNumber != expectedSerialNumber {
		return errors.Reason("match serial number to inventory: failed, expected: %q, but got %q", expectedSerialNumber, actualSerialNumber).Err()
	}
	return nil
}

func init() {
	execs.Register("cros_update_serial_number_inventory", updateSerialNumberToInvExec)
	execs.Register("cros_match_serial_number_inventory", matchSerialNumberToInvExec)
}
