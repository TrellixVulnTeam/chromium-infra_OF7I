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
func setServoStateExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	stateMap := execs.ParseActionArgs(ctx, actionArgs, ":")
	servoStateString, existed := stateMap["state"]
	if !existed {
		return errors.Reason("set servo state: missing servo state information in the argument").Err()
	}
	servoStateString = strings.TrimSpace(servoStateString)
	if servoStateString == "" {
		return errors.Reason("set servo state: the servo state string is empty").Err()
	}
	if servoStateString != strings.ToUpper(servoStateString) {
		return errors.Reason("set servo state: the servo state string is in wrong format").Err()
	}
	log.Info(ctx, "Previous servo state: %s", args.DUT.ServoHost.Servo.State)
	args.DUT.ServoHost.Servo.State = tlw.ServoState(servoStateString)
	log.Info(ctx, "Set servo state to be: %s", servoStateString)
	return nil
}

func init() {
	execs.Register("servo_set_servo_state", setServoStateExec)
}
