// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpm

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// hasRpmInfoExec verifies if rpm info is present for DUT.
func hasRpmInfoExec(ctx context.Context, info *execs.ExecInfo) error {
	if r := info.RunArgs.DUT.RPMOutlet; r != nil {
		// TODO(otabek@): set fixed number to check and add accept argument value.
		if r.GetHostname() != "" && r.GetOutlet() != "" {
			return nil
		}
		r.State = tlw.RPMOutlet_MISSING_CONFIG
	}
	return errors.Reason("has rpm info: not present or incorrect").Err()
}

// rpmPowerCycleExec performs power cycle the device by RPM.
func rpmPowerCycleExec(ctx context.Context, info *execs.ExecInfo) error {
	if err := info.RPMAction(ctx, info.RunArgs.DUT.Name, info.RunArgs.DUT.RPMOutlet, tlw.RunRPMActionRequest_CYCLE); err != nil {
		return errors.Annotate(err, "rpm power cycle").Err()
	}
	log.Debugf(ctx, "RPM power cycle finished with success.")
	return nil
}

// rpmPowerOffExec performs power off the device by RPM.
func rpmPowerOffExec(ctx context.Context, info *execs.ExecInfo) error {
	if err := info.RPMAction(ctx, info.RunArgs.DUT.Name, info.RunArgs.DUT.RPMOutlet, tlw.RunRPMActionRequest_OFF); err != nil {
		return errors.Annotate(err, "rpm power off").Err()
	}
	log.Debugf(ctx, "RPM power OFF finished with success.")
	return nil
}

// rpmPowerOffExec performs power on the device by RPM.
func rpmPowerOnExec(ctx context.Context, info *execs.ExecInfo) error {
	if err := info.RPMAction(ctx, info.RunArgs.DUT.Name, info.RunArgs.DUT.RPMOutlet, tlw.RunRPMActionRequest_ON); err != nil {
		return errors.Annotate(err, "rpm power on").Err()
	}
	log.Debugf(ctx, "RPM power ON finished with success.")
	return nil
}

func init() {
	execs.Register("has_rpm_info", hasRpmInfoExec)
	execs.Register("rpm_power_cycle", rpmPowerCycleExec)
	execs.Register("rpm_power_off", rpmPowerOffExec)
	execs.Register("rpm_power_on", rpmPowerOnExec)
}
