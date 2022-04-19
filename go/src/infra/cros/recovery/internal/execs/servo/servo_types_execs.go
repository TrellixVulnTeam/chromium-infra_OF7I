// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	comp_servo "infra/cros/recovery/internal/components/servo"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// servoVerifyV3Exec verifies whether the servo attached to the servo
// host if of type V3.
func servoVerifyV3Exec(ctx context.Context, info *execs.ExecInfo) error {
	// We first check that the servo host is not a labstation. The
	// "-servo" suffix will exist only when the setup is for type V3,
	// (i.e. there is no labstation present).
	servoSuffix := "-servo"
	if !strings.Contains(info.RunArgs.DUT.ServoHost.Name, servoSuffix) {
		return errors.Reason("servo verify v3: servo hostname does not carry suffix %q, this is not V3.", servoSuffix).Err()
	}
	// We further verify that the version detected for servo indeed
	// matches that for "V3". This confirms that the entire setup is
	// correct.
	sType, err := WrappedServoType(ctx, info)
	if err != nil {
		log.Debugf(ctx, "Servo Verify V3: could not determine the servo type")
		return errors.Annotate(err, "servo verify v3").Err()
	}
	if sType.IsV3() {
		return nil
	}
	log.Debugf(ctx, "Servo Verify V3: servo type is not V3.")
	return errors.Reason("servo verify v3: servo type %q is not V3.", sType).Err()
}

// servoVerifyV4Exec verifies whether the servo attached to the servo
// host if of type V4.
func servoVerifyV4Exec(ctx context.Context, info *execs.ExecInfo) error {
	sType, err := WrappedServoType(ctx, info)
	if err != nil {
		log.Debugf(ctx, "Servo Verify V4: could not determine the servo type")
		return errors.Annotate(err, "servo verify v4").Err()
	}
	if !sType.IsV4() {
		log.Debugf(ctx, "Servo Verify V4: servo type is neither V4, or V4P1.")
		return errors.Reason("servo verify v4: servo type %q is not V4.", sType).Err()
	}
	return nil
}

// servoVerifyServoMicroExec verifies whether the servo attached to
// the servo host if of type servo micro.
func servoVerifyServoMicroExec(ctx context.Context, info *execs.ExecInfo) error {
	sType, err := WrappedServoType(ctx, info)
	if err != nil {
		log.Debugf(ctx, "Servo Verify Servo Micro: could not determine the servo type")
		return errors.Annotate(err, "servo verify servo micro").Err()
	}
	if !sType.IsMicro() {
		log.Debugf(ctx, "Servo Verify servo micro: servo type is not servo micro.")
		return errors.Reason("servo verify servo micro: servo type %q is not servo micro.", sType).Err()
	}
	return nil
}

// servoIsDualSetupConfiguredExec checks whether the servo device has
// been setup in dual mode.
func servoIsDualSetupConfiguredExec(ctx context.Context, info *execs.ExecInfo) error {
	if info.RunArgs.DUT != nil && info.RunArgs.DUT.ExtraAttributes != nil {
		if attrs, ok := info.RunArgs.DUT.ExtraAttributes[tlw.ExtraAttributeServoSetup]; ok {
			for _, a := range attrs {
				if a == tlw.ExtraAttributeServoSetupDual {
					log.Debugf(ctx, "Servo Is Dual Setup Configured: servo device is configured to be in dual-setup mode.")
					return nil
				}
			}
		}
	}
	return errors.Reason("servo is dual setup configured: servo device is not configured to be in dual-setup mode").Err()
}

// servoVerifyDualSetupExec verifies whether the servo attached to the
// servo host actually exhibits dual setup.
func servoVerifyDualSetupExec(ctx context.Context, info *execs.ExecInfo) error {
	sType, err := WrappedServoType(ctx, info)
	if err != nil {
		return errors.Annotate(err, "servo verify dual setup").Err()
	}
	if !sType.IsDualSetup() {
		return errors.Reason("servo verify dual setup: servo type %q is not dual setup.", sType).Err()
	}
	return nil
}

// servoVerifyServoCCDExec verifies whether the servo attached to
// the servo host if of type servo ccd.
func servoVerifyServoCCDExec(ctx context.Context, info *execs.ExecInfo) error {
	sType, err := WrappedServoType(ctx, info)
	if err != nil {
		log.Debugf(ctx, "Servo Verify Servo CCD: could not determine the servo type")
		return errors.Annotate(err, "servo verify servo type ccd").Err()
	}
	if !sType.IsCCD() {
		log.Debugf(ctx, "Servo Verify servo CCD: servo type is not servo ccd.")
		return errors.Reason("servo verify servo ccd: servo type %q is not servo ccd.", sType).Err()
	}
	return nil
}

// mainDeviceIsGSCExec checks whether or not the servo device is CR50 or TI50.
func mainDeviceIsGSCExec(ctx context.Context, info *execs.ExecInfo) error {
	sType, err := WrappedServoType(ctx, info)
	if err != nil {
		return errors.Annotate(err, "main devices is gsc").Err()
	}
	md, err := mainServoDeviceHelper(sType.String())
	if err != nil {
		return errors.Annotate(err, "main devices is gsc").Err()
	}
	switch md {
	case comp_servo.CCD_CR50:
		fallthrough
	case comp_servo.CCD_GSC:
		info.NewLogger().Debugf("Found main device: %q", md)
		return nil
	default:
		return errors.Reason("main devices is gsc: found %q does not match expectations", md).Err()
	}
}

func init() {
	execs.Register("is_servo_v3", servoVerifyV3Exec)
	execs.Register("is_servo_v4", servoVerifyV4Exec)
	execs.Register("is_servo_micro", servoVerifyServoMicroExec)
	execs.Register("is_dual_setup_configured", servoIsDualSetupConfiguredExec)
	execs.Register("is_dual_setup", servoVerifyDualSetupExec)
	execs.Register("is_servo_type_ccd", servoVerifyServoCCDExec)
	execs.Register("servo_main_device_is_gcs", mainDeviceIsGSCExec)
}
