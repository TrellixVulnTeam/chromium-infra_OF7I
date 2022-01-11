// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package chameleon

import (
	"context"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/tlw"
)

// setStateBrokenExec sets state as BROKEN.
func setStateBrokenExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	args.DUT.ChameleonHost.State = tlw.ChameleonStateBroken
	return nil
}

// setStateWorkingExec sets state as WORKING.
func setStateWorkingExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	args.DUT.ChameleonHost.State = tlw.ChameleonStateWorking
	return nil
}

func init() {
	execs.Register("chameleon_state_broken", setStateBrokenExec)
	execs.Register("chameleon_state_working", setStateWorkingExec)
}
