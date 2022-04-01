// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/servo"
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
func servoServodPdRoleToggleExec(ctx context.Context, info *execs.ExecInfo) error {
	pdRoleToggleMap := info.GetActionArgs(ctx)
	retryCount := pdRoleToggleMap.AsInt(ctx, "retry_count", 1)
	waitInRetry := pdRoleToggleMap.AsInt(ctx, "wait_in_retry", 5)
	log.Debugf(ctx, "The wait time for power restore in the middle of retry is being set to: %d", waitInRetry)
	waitBeforeRetry := pdRoleToggleMap.AsInt(ctx, "wait_before_retry", 1)
	log.Debugf(ctx, "The wait time for power restore before retry is being set to: %d", waitBeforeRetry)
	// First setting the servod pd_role to the snk position.
	if err := info.NewServod().Set(ctx, servodPdRoleCmd, servodPdRoleValueSnk); err != nil {
		log.Debugf(ctx, "Error setting the servo_pd_role: %q", err.Error())
	}
	time.Sleep(time.Duration(waitBeforeRetry) * time.Second)
	toggleErr := retry.LimitCount(ctx, retryCount, 0*time.Second, func() error {
		if err := info.NewServod().Set(ctx, servodPdRoleCmd, servodPdRoleValueSrc); err != nil {
			log.Debugf(ctx, "Error setting the servo_pd_role: %q", err.Error())
		}
		// Waiting a few seconds as it can be change to snk if PD on servo has issue.
		time.Sleep(time.Duration(waitInRetry) * time.Second)
		if pdRoleValue, err := servodGetString(ctx, info.NewServod(), servodPdRoleCmd); err != nil {
			return errors.Annotate(err, "servod pd role toggle").Err()
		} else if pdRoleValue == servodPdRoleValueSrc {
			// log the main toggle action succeed.
			log.Debugf(ctx, "Successfully toggle the servod: servo_pd_role value to src.")
			return nil
		} else {
			return errors.Reason("servod pd role toggle: did not successfully set it to src").Err()
		}
	}, "servod pd role toggle")
	return errors.Annotate(toggleErr, "servod pd role toggle").Err()
}

// servoRecoverAcPowerExec recovers AC detection if AC is not detected.
//
// The fix based on toggle PD negotiating on EC level of DUT.
// Repair works only for the DUT which has EC and battery.
//
// @params: actionArgs should be in the format of:
// Ex: ["wait_timeout:x"]
func servoRecoverAcPowerExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	// Timeout to wait for recovering the ac power through ec. Default to be 120s.
	waitTimeout := argsMap.AsDuration(ctx, "wait_timeout", 120, time.Second)
	servod := info.NewServod()
	// Make sure ec is available and we can interact with that.
	if _, err := servodGetString(ctx, servod, "ec_board"); err != nil {
		log.Debugf(ctx, "Servo recover ac power: cannot get ec board with error: %s", err)
		// if EC is off it will fail to execute any EC command
		// to wake it up we do cold-reboot then we will have active ec connection for ~30 seconds.
		if err := servod.Set(ctx, "power_state", "reset"); err != nil {
			return errors.Annotate(err, "servo recover ac power").Err()
		}
	}
	if batteryIsCharging, err := servodGetBool(ctx, servod, "battery_is_charging"); err != nil {
		return errors.Annotate(err, "servo recover ac power").Err()
	} else if batteryIsCharging {
		log.Debugf(ctx, "Servo recover ac power: battery is charging")
		return nil
	}
	// Simple off-on not always working stable in all cases as source-sink not working too in another cases.
	// To cover more cases here we do both toggle to recover PD negotiation.
	// Source/sink switching CC lines to make DUT work as supplying or consuming power (between Rp and Rd).
	if err := servo.SetEcUartCmd(ctx, servod, "pd dualrole off", time.Second); err != nil {
		return errors.Annotate(err, "servo recover ac power").Err()
	}
	if err := servo.SetEcUartCmd(ctx, servod, "pd dualrole on", time.Second); err != nil {
		return errors.Annotate(err, "servo recover ac power").Err()
	}
	if err := servo.SetEcUartCmd(ctx, servod, "pd dualrole source", time.Second); err != nil {
		return errors.Annotate(err, "servo recover ac power").Err()
	}
	if err := servo.SetEcUartCmd(ctx, servod, "pd dualrole sink", time.Second); err != nil {
		return errors.Annotate(err, "servo recover ac power").Err()
	}
	// Wait to reinitialize PD negotiation and charge a little bit.
	log.Debugf(ctx, "Servo recover ac power: Wait %v", waitTimeout)
	time.Sleep(waitTimeout)
	if err := servod.Set(ctx, "power_state", "reset"); err != nil {
		return errors.Annotate(err, "servo recover ac power").Err()
	}
	if batteryIsCharging, err := servodGetBool(ctx, servod, "battery_is_charging"); err != nil {
		return errors.Annotate(err, "servo recover ac power").Err()
	} else if !batteryIsCharging {
		return errors.Annotate(err, "servo recover ac power: battery is not charging after recovery").Err()
	}
	return nil
}

// buildInPDSupportedExec verify if build in PD control is supported.
func buildInPDSupportedExec(ctx context.Context, info *execs.ExecInfo) error {
	servod := info.NewServod()
	pdControlSupported, err := servo.ServoSupportsBuiltInPDControl(ctx, servod)
	if err != nil {
		return errors.Annotate(err, "build in PD supported").Err()
	}
	if !pdControlSupported {
		return errors.Reason("build in PD supported").Err()
	}
	info.NewLogger().Debugf("Build in PD is supported!")
	return nil
}

func init() {
	execs.Register("servo_servod_toggle_pd_role", servoServodPdRoleToggleExec)
	execs.Register("servo_recover_ac_power", servoRecoverAcPowerExec)
	execs.Register("servo_build_in_pd_present", buildInPDSupportedExec)
}
