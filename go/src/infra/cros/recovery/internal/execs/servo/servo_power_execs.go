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
	"infra/cros/recovery/internal/retry"
)

const (
	// servodPdRoleCmd is the servod command for
	servodPdRoleCmd = "servo_pd_role"
	// servodPdRoleValueSnk is the one of the two values for servodPdRoleCmd
	// snk is the state that servo in normal mode and not passes power to DUT.
	servodPdRoleValueSnk = "snk"
	// servodPdRoleValueSrc is the one of the two values for servodPdRoleCmd
	// src is the state that servo in power delivery mode and passes power to the DUT.
	servodPdRoleValueSrc = "src"
)

// servoServodPdRoleToggleExec toggles the servod's "servo_pd_role" command from snk to src
// number of times based upon the actionArgs
//
// Ex. the actionArgs should be in the format of:
// ["retry_count:x", "wait_in_retry:x", "wait_before_retry:x"]
func servoServodPdRoleToggleExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	pdRoleToggleMap := execs.ParseActionArgs(ctx, actionArgs, execs.DefaultSplitter)
	retryCount := pdRoleToggleMap.AsInt(ctx, "retry_count", 1)
	waitInRetry := pdRoleToggleMap.AsInt(ctx, "wait_in_retry", 5)
	log.Debug(ctx, "The wait time for power restore in the middle of retry is being set to: %d", waitInRetry)
	waitBeforeRetry := pdRoleToggleMap.AsInt(ctx, "wait_before_retry", 1)
	log.Debug(ctx, "The wait time for power restore before retry is being set to: %d", waitBeforeRetry)
	// First setting the servod pd_role to the snk position.
	if _, err := ServodCallSet(ctx, args, servodPdRoleCmd, servodPdRoleValueSnk); err != nil {
		log.Debug(ctx, "Error setting the servo_pd_role: %q", err.Error())
	}
	time.Sleep(time.Duration(waitBeforeRetry) * time.Second)
	toggleErr := retry.LimitCount(ctx, retryCount, 0*time.Second, func() error {
		if _, err := ServodCallSet(ctx, args, servodPdRoleCmd, servodPdRoleValueSrc); err != nil {
			log.Debug(ctx, "Error setting the servo_pd_role: %q", err.Error())
		}
		// Waiting a few seconds as it can be change to snk if PD on servo has issue.
		time.Sleep(time.Duration(waitInRetry) * time.Second)
		if pdRoleValue, err := servodGetString(ctx, args, servodPdRoleCmd); err != nil {
			return errors.Annotate(err, "servod pd role toggle").Err()
		} else if pdRoleValue == servodPdRoleValueSrc {
			// log the main toggle action succeed.
			log.Debug(ctx, "Successfully toggle the servod: servo_pd_role value to src.")
			return nil
		} else {
			return errors.Reason("servod pd role toggle: did not successfully set it to src").Err()
		}
	}, "servod pd role toggle")
	return errors.Annotate(toggleErr, "servod pd role toggle").Err()
}

func init() {
	execs.Register("servo_servod_toggle_pd_role", servoServodPdRoleToggleExec)
}
