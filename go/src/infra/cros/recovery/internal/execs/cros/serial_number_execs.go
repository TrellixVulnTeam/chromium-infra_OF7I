// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	readSerialNumberCommand = "vpd -g serial_number"
)

// updateSerialNumberToInvExec updates serial number in DUT-info.
func updateSerialNumberToInvExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, readSerialNumberCommand)
	if r.ExitCode != 0 {
		return errors.Reason("update serial number in DUT-info: failed with code: %d and %q", r.ExitCode, r.Stderr).Err()
	}
	sn := strings.TrimSpace(r.Stdout)
	if sn == "" {
		return errors.Reason("update serial number in DUT-info: is empty").Err()
	}
	log.Debug(ctx, "Update serial_number %q in DUT-info.", sn)
	args.DUT.SerialNumber = sn
	return nil
}

// matchSerialNumberToInvExec matches serial number from the resource to value in the Inventory.
func matchSerialNumberToInvExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, readSerialNumberCommand)
	if r.ExitCode != 0 {
		return errors.Reason("match serial number to inventory: failed with code: %d and %q", r.ExitCode, r.Stderr).Err()
	}
	expectedSerialNumber := args.DUT.SerialNumber
	actualSerialNumber := strings.TrimSpace(r.Stdout)
	if actualSerialNumber != expectedSerialNumber {
		return errors.Reason("match serial number to inventory: failed, expected: %q, but got %q", expectedSerialNumber, actualSerialNumber).Err()
	}
	return nil
}

func init() {
	execs.Register("cros_update_serial_number_inventory", updateSerialNumberToInvExec)
	execs.Register("cros_match_serial_number_inventory", matchSerialNumberToInvExec)
}
