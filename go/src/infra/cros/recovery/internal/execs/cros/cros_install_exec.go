// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/cros"
	"infra/cros/recovery/internal/components/servo"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// Boot device from servo USB drive when device is in DEV mode.
func devModeBootFromServoUSBDriveExec(ctx context.Context, info *execs.ExecInfo) error {
	am := info.GetActionArgs(ctx)
	waitBootTimeout := am.AsDuration(ctx, "boot_timeout", 1, time.Second)
	waitBootInterval := am.AsDuration(ctx, "retry_interval", 1, time.Second)
	servod := info.NewServod()
	run := info.NewRunner(info.RunArgs.DUT.Name)
	ping := info.NewPinger(info.RunArgs.DUT.Name)
	err := cros.BootFromServoUSBDriveInDevMode(ctx, waitBootTimeout, waitBootInterval, run, ping, servod, info.NewLogger())
	return errors.Annotate(err, "dev-mode boot from servo usb-drive").Err()
}

// Install ChromeOS from servo USB drive when booted from it.
func runChromeosInstallCommandWhenBootFromUSBDriveExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	err := cros.RunInstallOSCommand(ctx, info.ActionTimeout, run, info.NewLogger())
	return errors.Annotate(err, "run install os after boot from USB-drive").Err()
}

// osInstallRepairExec re-installs a test image from USB, utilizing
// the features of servo to execute this repair.
func osInstallRepairExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.NewRunner(info.RunArgs.DUT.Name)
	servod := info.NewServod()
	needSink, err := cros.RecoveryModeRequiredPDOff(ctx, run, info.NewPinger(info.RunArgs.DUT.Name), servod, info.RunArgs.DUT)
	if err != nil {
		return errors.Annotate(err, "servo os install repair").Err()
	}
	log.Debug(ctx, "Servo OS Install Repair: needSink :%t", needSink)
	// Turn power off.
	if err := servod.Set(ctx, "power_state", "off"); err != nil {
		return errors.Annotate(err, "servo OS install repair").Err()
	}
	// Installation of the _image to servo-USB is being done through a
	// separate action that is set as a dependency for the current
	// action.

	// Next:Boot in recovery mode. The steps are:
	// Step 1. Switch the USB to DUT on the servo multiplexer
	if err := servo.UpdateUSBVisibility(ctx, servo.USBVisibleDUT, servod); err != nil {
		return errors.Annotate(err, "servo os install repair").Err()
	}
	// Step 2. For servo V4, switch power delivery to sink mode. c.f.:
	// crbug.com/1129165.
	if needSink {
		if err := servo.SetPDRole(ctx, servod, servo.PD_OFF, false); err != nil {
			return errors.Annotate(err, "servo OS install repair").Err()
		}
	} else {
		log.Info(ctx, "servo os install repair: servo type is neither V4, or V4P1, no need to switch power-deliver to sink.")
	}
	// Step 3. Turn power on
	if err := servod.Set(ctx, "power_state", "on"); err != nil {
		return errors.Annotate(err, "servo OS install repair").Err()
	}
	// Next: Clear TPM
	tpmRecoveryCmd := "chromeos-tpm-recovery"
	if _, err := run(ctx, info.ActionTimeout, tpmRecoveryCmd); err != nil {
		log.Debug(ctx, "servo OS install repair: (non-critical) error %q with command %q.", tpmRecoveryCmd)
	}
	// Now: install image.
	if _, err := run(ctx, info.ActionTimeout, "chromeos-install", "--yes"); err != nil {
		return errors.Annotate(err, "servo OS install repair").Err()
	}
	// 120 seconds is the default halt timeout from LABPACK, and we
	// are continuing with it here.
	argsMap := info.GetActionArgs(ctx)
	haltTimeout := argsMap.AsDuration(ctx, "halt_timeout", 120, time.Second)
	log.Debug(ctx, "Servo OS Install Repair: using halt timeout : %s", haltTimeout)
	// The halt command needs to be run in the background. For this to
	// succeed, the stdin, stdout and stderr are closed to allow ssh
	// session to terminate. This closely follows the logic in
	// LABPACK.
	runInBackground := info.NewBackgroundRunner(info.RunArgs.DUT.Name)
	if _, err := runInBackground(ctx, haltTimeout, "halt"); err != nil {
		return errors.Annotate(err, "servo OS install repair").Err()
	}
	if err := servod.Set(ctx, "power_state", "off"); err != nil {
		return errors.Annotate(err, "servo OS install repair").Err()
	}
	if err := servo.UpdateUSBVisibility(ctx, servo.USBVisibleOff, servod); err != nil {
		return errors.Annotate(err, "servo os install repair").Err()
	}
	if needSink {
		if err := servo.SetPDRole(ctx, servod, servo.PD_ON, false); err != nil {
			return errors.Annotate(err, "servo OS install repair").Err()
		}
	}
	if err := servod.Set(ctx, "power_state", "on"); err != nil {
		return errors.Annotate(err, "servo OS install repair").Err()
	}
	return nil
}

func init() {
	execs.Register("cros_dev_mode_boot_from_servo_usb_drive", devModeBootFromServoUSBDriveExec)
	execs.Register("cros_run_chromeos_install_command_after_boot_usbdrive", runChromeosInstallCommandWhenBootFromUSBDriveExec)
	execs.Register("os_install_repair", osInstallRepairExec)
}
