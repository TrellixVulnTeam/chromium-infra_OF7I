// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/cros"
	"infra/cros/recovery/internal/execs"
)

// servoHostIsLabstationExec confirms the servo host is a labstation
func servoHostIsLabstationExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	expected := argsMap.AsString(ctx, "board", "labstation")
	run := info.NewRunner(info.RunArgs.DUT.ServoHost.Name)
	log := info.NewLogger()
	board, err := cros.ReleaseBoard(ctx, run, log)
	if err != nil {
		return errors.Annotate(err, "servo host is labstation").Err()
	}
	if !strings.Contains(board, expected) {
		return errors.Reason("servo host is not labstation").Err()
	}
	return nil
}

// servoUsesServodContainerExec checks if the servo uses a servod-container.
func servoUsesServodContainerExec(ctx context.Context, info *execs.ExecInfo) error {
	if !IsContainerizedServoHost(ctx, info.RunArgs.DUT.ServoHost) {
		return errors.Reason("servo not using servod container").Err()
	}
	return nil
}

func init() {
	execs.Register("servo_host_is_labstation", servoHostIsLabstationExec)
	execs.Register("servo_uses_servod_container", servoUsesServodContainerExec)
}
