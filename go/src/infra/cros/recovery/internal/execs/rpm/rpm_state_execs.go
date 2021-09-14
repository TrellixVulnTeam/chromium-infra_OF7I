// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpm

import (
	"context"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/tlw"
)

func rpmStateUnspecifiedExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	args.DUT.RPMOutlet.State = tlw.RPMStateUnspecified
	return nil
}

func rpmStateMissingConfigExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	args.DUT.RPMOutlet.State = tlw.RPMStateMissingConfig
	return nil
}

func rpmStateWrongConfigExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	args.DUT.RPMOutlet.State = tlw.RPMStateWrongConfig
	return nil
}

func rpmStateWorkingExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	args.DUT.RPMOutlet.State = tlw.RPMStateWorking
	return nil
}

func init() {
	execs.Register("rpm_state_unspecified", rpmStateUnspecifiedExec)
	execs.Register("rpm_state_missing_config", rpmStateMissingConfigExec)
	execs.Register("rpm_state_wrong_config", rpmStateWrongConfigExec)
	execs.Register("rpm_state_working", rpmStateWorkingExec)
}
