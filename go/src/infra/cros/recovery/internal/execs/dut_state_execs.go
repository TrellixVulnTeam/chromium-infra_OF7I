// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"

	"infra/cros/dutstate"
)

// dutStateReadyActionExec sets dut-state as ready.
func dutStateReadyActionExec(ctx context.Context, args *RunArgs) error {
	args.DUT.State = dutstate.Ready
	return nil
}

// dutStateRepairFailedActionExec sets dut-state as repair_failed.
func dutStateRepairFailedActionExec(ctx context.Context, args *RunArgs) error {
	args.DUT.State = dutstate.RepairFailed
	return nil
}

// dutStateNeedsDeployActionExec sets dut-state as needs_deploy.
func dutStateNeedsDeployActionExec(ctx context.Context, args *RunArgs) error {
	args.DUT.State = dutstate.NeedsDeploy
	return nil
}

// TODO(otabek@): Add execs for other states.

func init() {
	execMap["dut_state_ready"] = dutStateReadyActionExec
	execMap["dut_state_repair_failed"] = dutStateRepairFailedActionExec
	execMap["dut_state_needs_deploy"] = dutStateNeedsDeployActionExec
}
