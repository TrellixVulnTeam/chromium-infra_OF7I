// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// servoTriggerKernelPanicExec repairs a Chrome device by sending a system request to the kernel.
// By default, it sends 3 times the Alt+VolUp+x key combination (aka sysrq-x)
// will ask the kernel to panic itself and reboot while conserving
// the kernel logs in console Ramoops.
//
// Ramoops is an oops/panic logger that writes its logs to RAM before the system crashes.
// It works by logging oopses and panics in a circular buffer. Ramoops needs a system with persistent
// RAM so that the content of that area can survive after a restart.
//
// @params: actionArgs should be in the format of:
// Ex: ["count:x", "retry_interval:x"]
func servoTriggerKernelPanicExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	// Number of times to send the system request by pressing "Alt+VolUp+X"
	requestCount := argsMap.AsInt(ctx, "count", 3)
	// retryInterval is the timeout to for executing the sending request through servod command for every iteration.
	retryInterval := argsMap.AsDuration(ctx, "retry_interval", 2, time.Second)
	servod := info.NewServod()
	for i := 0; i < requestCount; i++ {
		// Simulate Alt VolumeUp X simultaneous press.
		// This key combination is the kernel system request (sysrq) X.
		if err := servod.Set(ctx, "sysrq_x", "tab"); err != nil {
			return errors.Annotate(err, "servo trigger kernel panic").Err()
		}
		log.Debugf(ctx, "Wait %v after servod sending system request to trigger kernel panic.", retryInterval)
		time.Sleep(retryInterval)
	}
	log.Debugf(ctx, "Rest the DUT via keyboard 'Alt+VolUp+X' successfully.")
	return nil
}

func init() {
	execs.Register("servo_trigger_kernel_panic", servoTriggerKernelPanicExec)
}
