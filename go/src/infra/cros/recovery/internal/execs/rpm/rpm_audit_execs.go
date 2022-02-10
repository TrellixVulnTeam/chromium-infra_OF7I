// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpm

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/execs/cros"
	"infra/cros/recovery/tlw"
)

func rpmAuditExec(ctx context.Context, info *execs.ExecInfo) error {
	// TODO: Add support for device with battery.
	if err := rpmPowerOffExec(ctx, info); err != nil {
		return errors.Annotate(err, "rpm audit").Err()
	}
	if err := cros.IsPingable(ctx, info, info.RunArgs.ResourceName, 2); err == nil {
		info.RunArgs.DUT.RPMOutlet.State = tlw.RPMOutlet_WRONG_CONFIG
		return errors.Reason("rpm audit: resource still sshable").Err()
	}
	if err := rpmPowerOnExec(ctx, info); err != nil {
		return errors.Annotate(err, "rpm audit").Err()
	}
	if err := cros.WaitUntilSSHable(ctx, info.DefaultRunner(), cros.NormalBootingTime); err != nil {
		info.RunArgs.DUT.RPMOutlet.State = tlw.RPMOutlet_WRONG_CONFIG
		return errors.Annotate(err, "rpm audit: resource did not booted").Err()
	}
	info.RunArgs.DUT.RPMOutlet.State = tlw.RPMOutlet_WORKING
	return nil
}

func init() {
	execs.Register("rpm_audit", rpmAuditExec)
}
