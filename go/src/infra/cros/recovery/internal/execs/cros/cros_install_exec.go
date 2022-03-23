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
	"infra/cros/recovery/internal/execs/metrics"
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
	log.Debugf(ctx, "Servo OS Install Repair: needSink :%t", needSink)
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
		log.Infof(ctx, "servo os install repair: servo type is neither V4, or V4P1, no need to switch power-deliver to sink.")
	}
	// Step 3. Turn power on
	if err := servod.Set(ctx, "power_state", "on"); err != nil {
		return errors.Annotate(err, "servo OS install repair").Err()
	}
	// Next: Clear TPM
	tpmRecoveryCmd := "chromeos-tpm-recovery"
	if _, err := run(ctx, info.ActionTimeout, tpmRecoveryCmd); err != nil {
		log.Debugf(ctx, "servo OS install repair: (non-critical) error %q with command %q.", tpmRecoveryCmd)
	}
	// Now: install image.
	if _, err := run(ctx, info.ActionTimeout, "chromeos-install", "--yes"); err != nil {
		return errors.Annotate(err, "servo OS install repair").Err()
	}
	// 120 seconds is the default halt timeout from LABPACK, and we
	// are continuing with it here.
	argsMap := info.GetActionArgs(ctx)
	haltTimeout := argsMap.AsDuration(ctx, "halt_timeout", 120, time.Second)
	log.Debugf(ctx, "Servo OS Install Repair: using halt timeout : %s", haltTimeout)
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

// verifyBootInRecoveryModeExec verify that device can boot in recovery mode and reboot to normal mode again.
func verifyBootInRecoveryModeExec(ctx context.Context, info *execs.ExecInfo) error {
	am := info.GetActionArgs(ctx)
	dut := info.RunArgs.DUT
	dutRun := info.NewRunner(dut.Name)
	dutBackgroundRun := info.NewBackgroundRunner(dut.Name)
	dutPing := info.NewPinger(dut.Name)
	servod := info.NewServod()
	// Flag to notice when device booted and sshable.
	var successBooted bool
	callback := func(_ context.Context) error {
		successBooted = true
		return nil
	}
	req := &cros.BootInRecoveryRequest{
		DUT:          dut,
		BootTimeout:  am.AsDuration(ctx, "boot_timeout", 480, time.Second),
		BootInterval: am.AsDuration(ctx, "boot_interval", 10, time.Second),
		// Register that device booted and sshable.
		Callback:            callback,
		HaltTimeout:         am.AsDuration(ctx, "halt_timeout", 120, time.Second),
		IgnoreRebootFailure: am.AsBool(ctx, "ignore_reboot_failure", false),
	}
	if err := cros.BootInRecoveryMode(ctx, req, dutRun, dutBackgroundRun, dutPing, servod, info.NewLogger()); err != nil {
		return errors.Annotate(err, "verify boot in recovery mode").Err()
	}
	if !successBooted {
		return errors.Reason("verify boot in recovery mode: did not booted").Err()
	}
	return nil
}

// isTimeToForceDownloadImageToUsbKeyExec verifies if we want to force download image to usbkey.
//
// @params: actionArgs should be in the format of:
// Ex: ["task_name:xxx", "repair_failed_count:1", "repair_failed_interval:10"]
func isTimeToForceDownloadImageToUsbKeyExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	taskName := argsMap.AsString(ctx, "task_name", "")
	repairFailedCountTarget := argsMap.AsInt(ctx, "repair_failed_count", -1)
	repairFailedInterval := argsMap.AsInt(ctx, "repair_failed_interval", 10)
	repairFailedCount, err := metrics.CountFailedRepairFromMetrics(ctx, taskName, info)
	if err != nil {
		return errors.Annotate(err, "is time to force download image to usbkey").Err()
	}
	// The previous repair task was successful, and the user didn't specify
	// when repair_failed_count == 0 to flash usbkey image.
	if repairFailedCount == 0 && repairFailedCountTarget != 0 {
		return errors.Reason("is time to force download image to usbkey: the number of failed repair is 0, will not force to install os iamge").Err()
	}
	if repairFailedCount == repairFailedCountTarget || repairFailedCount%repairFailedInterval == 0 {
		log.Infof(ctx, "Required re-download image to usbkey as a previous repair failed. Fail count: %d", repairFailedCount)
		return nil
	}
	return errors.Reason("is time to force download image to usbkey: Fail count: %d", repairFailedCount).Err()
}

func init() {
	execs.Register("cros_dev_mode_boot_from_servo_usb_drive", devModeBootFromServoUSBDriveExec)
	execs.Register("cros_run_chromeos_install_command_after_boot_usbdrive", runChromeosInstallCommandWhenBootFromUSBDriveExec)
	execs.Register("os_install_repair", osInstallRepairExec)
	execs.Register("cros_verify_boot_in_recovery_mode", verifyBootInRecoveryModeExec)
	execs.Register("cros_is_time_to_force_download_image_to_usbkey", isTimeToForceDownloadImageToUsbKeyExec)
}
