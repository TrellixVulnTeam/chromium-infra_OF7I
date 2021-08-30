// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"

	"infra/cros/dutstate"
	"infra/cros/recovery/internal/execs"
)

// dutStateReadyActionExec sets dut-state as ready.
func dutStateReadyActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	args.DUT.State = dutstate.Ready
	return nil
}

// dutStateRepairFailedActionExec sets dut-state as repair_failed.
func dutStateRepairFailedActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	args.DUT.State = dutstate.RepairFailed
	return nil
}

// dutStateNeedsDeployActionExec sets dut-state as needs_deploy.
func dutStateNeedsDeployActionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	args.DUT.State = dutstate.NeedsDeploy
	return nil
}

// TODO(otabek@): Add execs for other states.

func init() {
	execs.Register("dut_state_ready", dutStateReadyActionExec)
	execs.Register("dut_state_repair_failed", dutStateRepairFailedActionExec)
	execs.Register("dut_state_needs_deploy", dutStateNeedsDeployActionExec)
}
