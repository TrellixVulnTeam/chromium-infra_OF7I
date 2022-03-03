// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/cros"
	"infra/cros/recovery/internal/execs"
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

func init() {
	execs.Register("cros_dev_mode_boot_from_servo_usb_drive", devModeBootFromServoUSBDriveExec)
	execs.Register("cros_run_chromeos_install_command_after_boot_usbdrive", runChromeosInstallCommandWhenBootFromUSBDriveExec)
}
