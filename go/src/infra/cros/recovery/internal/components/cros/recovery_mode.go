// Copyright 2022 The Chromium OS Authors. All rights reserved.  Use
// of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/components/servo"
	"infra/cros/recovery/logger"
	"infra/cros/recovery/tlw"
)

// BootInRecoveryRequest holds info to boot device in recovery mode.
type BootInRecoveryRequest struct {
	DUT *tlw.Dut
	// Booting time value to verify when device booted and available for SSH.
	BootTimeout  time.Duration
	BootInterval time.Duration
	// Call function to cal after device booted in recovery mode.
	Callback func(context.Context) error
	// rebbot after recovery
	HaltTimeout         time.Duration
	IgnoreRebootFailure bool
}

// BootInRecoveryMode perform boot device in recovery mode.
//
// Boot in recovery mode performed by RO firmware and in some cases required stopPD negotiation.
// Please specify callback function to perform needed actions when device booted in recovery mode.
func BootInRecoveryMode(ctx context.Context, req *BootInRecoveryRequest, dutRun, dutBackgroundRun components.Runner, dutPing components.Pinger, servod components.Servod, log logger.Logger) error {
	needSink, err := RecoveryModeRequiredPDOff(ctx, dutRun, dutPing, servod, req.DUT)
	if err != nil {
		return errors.Annotate(err, "boot in recovery mode").Err()
	}
	log.Debugf("Servo OS Install Repair: needSink :%t", needSink)
	// Turn power off.
	if err := servo.SetPowerState(ctx, servod, servo.PowerStateValueOFF); err != nil {
		return errors.Annotate(err, "boot in recovery mode").Err()
	}
	// Next:Boot in recovery mode. The steps are:
	// Step 1. Switch the USB to DUT on the servo multiplexer
	if err := servo.UpdateUSBVisibility(ctx, servo.USBVisibleDUT, servod); err != nil {
		return errors.Annotate(err, "boot in recovery mode").Err()
	}
	// Step 2. For servo V4, switch power delivery to sink mode. c.f.:
	// crbug.com/1129165.
	if needSink {
		if err := servo.SetPDRole(ctx, servod, servo.PD_OFF, false); err != nil {
			return errors.Annotate(err, "boot in recovery mode").Err()
		}
	} else {
		log.Infof("Boot in recovery mode: servo type is neither V4, or V4P1, no need to switch power-deliver to sink.")
	}
	closing := func() error {
		// Register turn off for the DUT if at the end.
		// All errors just logging as the action to clean up the state.
		if _, err := dutBackgroundRun(ctx, req.HaltTimeout, "halt"); err != nil {
			return errors.Annotate(err, "boot in recovery mode").Err()
		}
		if err := servo.SetPowerState(ctx, servod, servo.PowerStateValueOFF); err != nil {
			return errors.Annotate(err, "boot in recovery mode").Err()
		}
		if err := servo.UpdateUSBVisibility(ctx, servo.USBVisibleOff, servod); err != nil {
			return errors.Annotate(err, "boot in recovery mode").Err()
		}
		if needSink {
			if err := servo.SetPDRole(ctx, servod, servo.PD_ON, false); err != nil {
				return errors.Annotate(err, "boot in recovery mode").Err()
			}
		}
		if err := servo.SetPowerState(ctx, servod, servo.PowerStateValueON); err != nil {
			return errors.Annotate(err, "boot in recovery mode").Err()
		}
		return nil
	}
	if req.IgnoreRebootFailure {
		defer func() {
			if err := closing(); err != nil {
				log.Debugf("Boot in recovery mode: %s", err)
			}
		}()
	}
	// Step 3. Turn power on
	if err := servo.SetPowerState(ctx, servod, servo.PowerStateValueRecoveryMode); err != nil {
		return errors.Annotate(err, "boot in recovery mode").Err()
	}
	if err := WaitUntilSSHable(ctx, req.BootTimeout, req.BootInterval, dutRun, log); err != nil {
		return errors.Annotate(err, "boot in recovery mode").Err()
	}
	if err := IsBootedFromExternalStorage(ctx, dutRun, log); err != nil {
		return errors.Annotate(err, "boot in recovery mode").Err()
	}
	log.Debugf("Device successful booted in recovery mode from USB-drive")
	if req.Callback != nil {
		log.Debugf("Boot in recovery mode: passing control to call back")
		if err := req.Callback(ctx); err != nil {
			return errors.Annotate(err, "boot in recovery mode: callback").Err()
		}
	}
	if !req.IgnoreRebootFailure {
		return closing()
	}
	return nil
}
