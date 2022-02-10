// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/execs/cros"
)

const (
	labstationKeyWord = "labstation"
)

// servoHostIsLabstationExec confirms the servo host is a labstation
// TODO (yunzhiyu@): Revisit when we onboard dockers.
func servoHostIsLabstationExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.NewRunner(info.RunArgs.DUT.ServoHost.Name)
	board, err := cros.ReleaseBoard(ctx, r)
	if err != nil {
		return errors.Annotate(err, "servo host is labstation").Err()
	}
	if !strings.Contains(board, labstationKeyWord) {
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
