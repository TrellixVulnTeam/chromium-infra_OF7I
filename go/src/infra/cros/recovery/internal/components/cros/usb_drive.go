// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/components/servo"
	"infra/cros/recovery/internal/retry"
	"infra/cros/recovery/logger"
)

// IsBootedFromExternalStorage verify that device has been booted from external storage.
func IsBootedFromExternalStorage(ctx context.Context, run components.Runner, log logger.Logger) error {
	bootStorage, err := run(ctx, time.Minute, "rootdev", "-s", "-d")
	if err != nil {
		return errors.Annotate(err, "booted from external storage").Err()
	} else if bootStorage == "" {
		return errors.Reason("booted from external storage: booted storage not detected").Err()
	}
	mainStorageCMD := ". /usr/sbin/write_gpt.sh; . /usr/share/misc/chromeos-common.sh; load_base_vars; get_fixed_dst_drive"
	mainStorage, err := run(ctx, time.Minute, mainStorageCMD)
	if err != nil {
		return errors.Annotate(err, "booted from external storage").Err()
	}
	// If main device is not detected then probably it can be dead or broken
	// but as we gt the boot device then it is external one.
	if mainStorage == "" || bootStorage != mainStorage {
		return nil
	}
	return errors.Reason("booted from external storage: booted from main storage").Err()
}

// BootFromServoUSBDriveInDevMode performs booting device from external storage when DUT is in DEV-mode.
// Device already should to be in DEV-mode and enabled to boot from USB-drive.
// The make to be able boot from USB-drive you need one of followed options:
// 1) Run enable_dev_usb_boot.
// 2) Set GBB with GBB_FLAG_FORCE_DEV_BOOT_USB flag.
//
// Steps:
// 1) Power off the host.
// 2) Trigger reboot by servo.
// 3) Perform ctrl+u by servo to try out boot from external storage.
func BootFromServoUSBDriveInDevMode(ctx context.Context, waitBootTimeout, waitBootInterval time.Duration, dutRun components.Runner, ping components.Pinger, servod components.Servod, log logger.Logger) error {
	if err := servo.UpdateUSBVisibility(ctx, servo.USBVisibleDUT, servod); err != nil {
		return errors.Annotate(err, "boot from servo usb drive in dev mode").Err()
	}
	if err := servod.Set(ctx, "power_state", "reset"); err != nil {
		return errors.Annotate(err, "boot from servo usb drive in dev mode").Err()
	}
	// Try to boot from UBS-drive so some period of time.
	err := retry.WithTimeout(ctx, waitBootInterval, waitBootTimeout, func() error {
		log.Debugf("Pressing ctrl+u")
		if err := servod.Set(ctx, "ctrl_u", "tab"); err != nil {
			return errors.Annotate(err, "wait for device boot").Err()
		}
		// Ping only once to safe time and do not miss the boot time frame.
		if err := IsPingable(ctx, 1, ping); err != nil {
			return errors.Annotate(err, "wait for device boot").Err()
		}
		log.Debugf("Device started booting!")
		return nil
	}, "wait to boot")
	if err != nil {
		return errors.Annotate(err, "boot from servo usb drive in dev mode").Err()
	}
	if err := WaitUntilSSHable(ctx, time.Minute, SSHRetryInteval, dutRun, log); err != nil {
		return errors.Annotate(err, "wait for device boot").Err()
	}
	// In some cases the device can boot from internal storage by multiple reasons.
	// Most prevident issues:
	// 1) Image on USB-drive is bad.
	// 2) Booting from USB-drive is not allowed.
	// 3) Device is not in DEV mode.
	if err := IsBootedFromExternalStorage(ctx, dutRun, log); err != nil {
		return errors.Annotate(err, "boot from servo usb drive in dev mode").Err()
	}
	return nil
}

// RunInstallOSCommand run chromeos-install command on the host.
func RunInstallOSCommand(ctx context.Context, timeout time.Duration, run components.Runner, log logger.Logger) error {
	out, err := run(ctx, timeout, "chromeos-install", "--yes")
	log.Debugf("Install OS:\n%s", out)
	return errors.Annotate(err, "install OS").Err()
}
