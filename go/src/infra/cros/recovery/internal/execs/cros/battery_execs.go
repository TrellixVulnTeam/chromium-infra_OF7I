// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/execs/cros/battery"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// auditBatteryExec confirms that it is able to audit battery info
// and mark the DUT if it needs replacement.
func auditBatteryExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	b, err := battery.ReadBatteryInfo(ctx, r)
	if err != nil {
		info.RunArgs.DUT.Battery.State = tlw.HardwareStateUnspecified
		return errors.Annotate(err, "audit battery: dut battery state cannot extracted").Err()
	}
	hardwareState := battery.DetermineHardwareStatus(ctx, b.FullChargeCapacity, b.FullChargeCapacityDesigned)
	log.Infof(ctx, "Battery hardware state: %s", hardwareState)
	if hardwareState == tlw.HardwareStateUnspecified {
		return errors.Reason("audit battery: dut battery did not detected or state cannot extracted").Err()
	}
	if hardwareState == tlw.HardwareStateNeedReplacement {
		log.Infof(ctx, "Detected issue with storage on the DUT.")
		info.RunArgs.DUT.Battery.State = tlw.HardwareStateNeedReplacement
	}
	return nil
}

func init() {
	execs.Register("cros_audit_battery", auditBatteryExec)
}
