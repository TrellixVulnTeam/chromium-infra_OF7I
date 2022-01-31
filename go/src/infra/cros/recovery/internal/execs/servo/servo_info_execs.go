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

// servoVerifyMainCcdCr50Exec checks whether or not the servo device contains
// CCD-CR50.
func servoVerifyMainCcdCr50Exec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	sType, err := WrappedServoType(ctx, args)
	if err != nil {
		return errors.Annotate(err, "servo verify main ccd cr50").Err()
	}
	ccdCr50 := "ccd_cr50"
	s, err := mainServoDeviceHelper(sType.String())
	if err != nil {
		return errors.Annotate(err, "servo verify main ccd cr50").Err()
	} else if s != ccdCr50 {
		return errors.Reason("servo verify main ccd cr50: servo type is %q does not match %q", s, ccdCr50).Err()
	}
	return nil
}

func init() {
	execs.Register("servo_servod_port_present", servoVerifyPortNumberExec)
	execs.Register("is_servo_main_ccd_cr50", servoVerifyMainCcdCr50Exec)
}
