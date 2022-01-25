// Copyright 2021 The Chromium OS Authors. All rights reserved.  Use
// of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// servoVerifyPortNumberExec verifies that the servo host attached to
// the DUT has a port number configured for running servod daemon on
// the servo host.
func servoVerifyPortNumberExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if args.DUT != nil && args.DUT.ServoHost != nil && args.DUT.ServoHost.ServodPort > 9000 {
		log.Debug(ctx, "Servo Verify Port Number Exec: %d", args.DUT.ServoHost.ServodPort)
		return nil
	}
	return errors.Reason("servo verify port number: port number is not available").Err()
}

// servoVerifyV4Exec verifies whether the servo attached to the servo
// host if of type V4.
func servoVerifyV4Exec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	sType, err := WrappedServoType(ctx, args)
	if err != nil {
		log.Debug(ctx, "Servo Verify V4: could not determine the servo type")
		return errors.Annotate(err, "servo verify v4").Err()
	}
	if !sType.IsV4() {
		log.Debug(ctx, "Servo Verify V4: servo type is neither V4, or V4P1.")
		return errors.Reason("servo verify v4: servo type %q is not V4.", sType).Err()
	}
	return nil
}

// servoVerifyV4TypeCExec verifies whether the type of a V4 servo is
// type-c or not.
//
// This applies only to servo V4.
func servoVerifyV4TypeCExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	dutConnectionTypeControl := "root.dut_connection_type"
	connectionType, err := servodGetString(ctx, args, dutConnectionTypeControl)
	if err != nil {
		return errors.Annotate(err, "servo verify v4 type c").Err()
	}
	if connectionType == "" {
		return errors.Reason("servo verify v4 type c: value of control %q is empty.", dutConnectionTypeControl).Err()
	}
	if connectionType != "type-c" {
		return errors.Reason("servo verify v4 type c: connection type %q does not correspond to type-c.", connectionType).Err()
	}
	return nil
}

func init() {
	execs.Register("servo_servod_port_present", servoVerifyPortNumberExec)
	execs.Register("servo_v4", servoVerifyV4Exec)
	execs.Register("servo_v4_type_c", servoVerifyV4TypeCExec)
}
