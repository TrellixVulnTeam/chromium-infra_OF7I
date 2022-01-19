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

func init() {
	execs.Register("servo_servod_port_present", servoVerifyPortNumberExec)
}
