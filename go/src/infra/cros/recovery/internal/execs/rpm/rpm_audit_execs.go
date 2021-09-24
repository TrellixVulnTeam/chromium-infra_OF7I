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

func rpmAuditExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	// TODO: Add support for device with battery.
	if err := rpmPowerOffExec(ctx, args, actionArgs); err != nil {
		return errors.Annotate(err, "rpm audit").Err()
	}
	if err := cros.IsPingable(ctx, args, args.ResourceName, 2); err == nil {
		args.DUT.RPMOutlet.State = tlw.RPMStateWrongConfig
		return errors.Reason("rpm audit: resource still sshable").Err()
	}
	if err := rpmPowerOnExec(ctx, args, actionArgs); err != nil {
		return errors.Annotate(err, "rpm audit").Err()
	}
	if err := cros.WaitUntilSSHable(ctx, args, args.ResourceName, cros.NormalBootingTime); err != nil {
		args.DUT.RPMOutlet.State = tlw.RPMStateWrongConfig
		return errors.Annotate(err, "rpm audit: resource did not booted").Err()
	}
	args.DUT.RPMOutlet.State = tlw.RPMStateWorking
	return nil
}

func init() {
	execs.Register("rpm_audit", rpmAuditExec)
}
