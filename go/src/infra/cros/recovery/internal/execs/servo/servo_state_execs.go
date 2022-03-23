// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// setServoStateExec sets the servo state of the servo of the DUT from the actionArgs argument.
//
// @actionArgs: the list of the string that contains the servo state information.
// It should only contain one string in the format of: "state:x"
// x must be all capatalized and matched one of the record in the predefined servo state.
func setServoStateExec(ctx context.Context, info *execs.ExecInfo) error {
	args := info.GetActionArgs(ctx)
	newState := strings.ToUpper(args.AsString(ctx, "state", ""))
	if newState == "" {
		return errors.Reason("set servo state: state is not provided").Err()
	}
	// Verify if servo is supported.
	// If servo is not supported the report failure.
	if d := info.RunArgs.DUT; d == nil || d.ServoHost == nil || d.ServoHost.Servo == nil {
		return errors.Reason("set servo state: servo is not supported").Err()
	}
	log.Debugf(ctx, "Previous servo state: %s", info.RunArgs.DUT.ServoHost.Servo.State)
	info.RunArgs.DUT.ServoHost.Servo.State = tlw.ServoState(newState)
	log.Infof(ctx, "Set servo state to be: %s", newState)
	return nil
}

// matchStateExec confirms the servo state is the same as the passed in argument from the configuration.
//
// format of the actionArgs: ["state:xxx"] where xxx is one of the predefined servo state.
func matchStateExec(ctx context.Context, info *execs.ExecInfo) error {
	args := info.GetActionArgs(ctx)
	expectedState := strings.ToLower(args.AsString(ctx, "state", ""))
	if expectedState == "" {
		return errors.Reason("match state: the servo state string is empty").Err()
	}
	// Update after DUT migrated to proto representation.
	if d := info.RunArgs.DUT; d != nil && d.ServoHost != nil {
		if sh := d.ServoHost; sh != nil && sh.Servo != nil {
			currentState := strings.ToLower(string(sh.Servo.State))
			if currentState != expectedState {
				return errors.Reason("match state: servo state mismatch: expected: %q, but got %q", expectedState, currentState).Err()
			}
			// The states matched.
			return nil
		}
	}
	return errors.Reason("match state: current servo state is unknown").Err()
}

func init() {
	execs.Register("servo_set_servo_state", setServoStateExec)
	execs.Register("servo_match_state", matchStateExec)
}
