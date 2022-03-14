// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/cros/firmware"
	"infra/cros/recovery/internal/execs"
)

const (
	// Read the AP firmware and dump the sections that we're interested in.
	readAndDumpAPFirmwareCmd = `mkdir /tmp/verify_firmware;` +
		`cd /tmp/verify_firmware; ` +
		`for section in VBLOCK_A VBLOCK_B FW_MAIN_A FW_MAIN_B; ` +
		`do flashrom -p host -r -i $section:$section; ` +
		`done`
	// Verify the firmware blocks A and B.
	verifyFirmwareCmd = `vbutil_firmware --verify /tmp/verify_firmware/VBLOCK_%s` +
		` --signpubkey /usr/share/vboot/devkeys/root_key.vbpubk` +
		` --fv /tmp/verify_firmware/FW_MAIN_%s`
	// remove the firmware related files we created before.
	removeFirmwareFileCmd = `rm -rf /tmp/verify_firmware`
)

// isFirmwareInGoodState confirms that a host's firmware is in a good state.
//
// For DUTs that run firmware tests, it's possible that the firmware on the DUT can get corrupted.
// This verify action checks whether it appears that firmware should be re-flashed using servo.
func isFirmwareInGoodState(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	_, err := r(ctx, time.Minute, readAndDumpAPFirmwareCmd)
	if err != nil {
		return errors.Annotate(err, "firmware in good state").Err()
	}
	defer func() { r(ctx, time.Minute, removeFirmwareFileCmd) }()
	for _, val := range []string{"A", "B"} {
		_, err := r(ctx, time.Minute, fmt.Sprintf(verifyFirmwareCmd, val, val))
		if err != nil {
			return errors.Annotate(err, "firmware in good state: firmware %s is in a bad state", val).Err()
		}
	}
	return nil
}

// isOnRWFirmwareStableVersionExec confirms that the DUT is currently running the stable version based on its specification.
func isOnRWFirmwareStableVersionExec(ctx context.Context, info *execs.ExecInfo) error {
	sv, err := info.Versioner().Cros(ctx, info.RunArgs.DUT.Name)
	if err != nil {
		return errors.Annotate(err, "on rw firmware stable version").Err()
	}
	err = matchCrosSystemValueToExpectation(ctx, info.DefaultRunner(), "fwid", sv.FwVersion)
	return errors.Annotate(err, "on rw firmware stable version").Err()
}

// isRWFirmwareStableVersionAvailableExec confirms the stable firmware is up to date with the available firmware.
func isRWFirmwareStableVersionAvailableExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	sv, err := info.Versioner().Cros(ctx, info.RunArgs.DUT.Name)
	if err != nil {
		return errors.Annotate(err, "rw firmware stable version available").Err()
	}
	modelFirmware, err := ReadFirmwareManifest(ctx, r, info.RunArgs.DUT.Model)
	if err != nil {
		return errors.Annotate(err, "rw firmware stable version available").Err()
	}
	availableVersion, err := modelFirmware.AvailableRWFirmware()
	if err != nil {
		return errors.Annotate(err, "rw firmware stable version available").Err()
	}
	stableVersion := sv.FwVersion
	if availableVersion != stableVersion {
		return errors.Reason("rw firmware stable version not available, expected %q, found %q", availableVersion, stableVersion).Err()
	}
	return nil
}

// runFirmwareUpdaterExec run firmware process on the host to flash firmware from installed OS.
//
// Default mode used is autoupdate.
// To reboot device by the end please provide `reboot:by_servo` or `reboot:by_host`.
func runFirmwareUpdaterExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.NewRunner(info.RunArgs.DUT.Name)
	am := info.GetActionArgs(ctx)
	req := &firmware.FirmwareUpdaterRequest{
		// Options for the mode are: autoupdate, recovery, factory.
		Mode:  am.AsString(ctx, "mode", "autoupdate"),
		Force: am.AsBool(ctx, "force", false),
	}
	info.NewLogger().Debug("Run firmware update: request to run with %q mode.", req.Mode)
	if err := firmware.RunFirmwareUpdater(ctx, req, run, info.NewLogger()); err != nil {
		return errors.Annotate(err, "run firmware update").Err()
	}
	switch am.AsString(ctx, "reboot", "") {
	case "by_servo":
		info.NewLogger().Debug("Start DUT reset by servo.")
		if err := info.NewServod().Set(ctx, "power_state", "reset"); err != nil {
			return errors.Annotate(err, "run firmware update: reboot by servo").Err()
		}
	case "by_host":
		info.NewLogger().Debug("Start DUT reset by host.")
		if _, err := run(ctx, time.Minute, "reboot && exit"); err != nil {
			return errors.Annotate(err, "run firmware update: reboot by host").Err()
		}
	}
	return nil
}

// runDisableWriteProtectExec disables software-controlled write-protect.
//
// ChromeOS devices have 'host' and 'ec' FPROMs, provide by 'fprom:ec'.
func runDisableFPROMWriteProtectExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.NewRunner(info.RunArgs.DUT.Name)
	am := info.GetActionArgs(ctx)
	fprom := am.AsString(ctx, "fprom", "")
	err := firmware.DisableWriteProtect(ctx, run, info.NewLogger(), info.ActionTimeout, fprom)
	return errors.Annotate(err, "disable fprom: %q write-protect", fprom).Err()
}

func hasDevSignedFirmwareExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.NewRunner(info.RunArgs.DUT.Name)
	if keys, err := firmware.ReadFirmwareKeysFromHost(ctx, run, info.NewLogger()); err != nil {
		return errors.Annotate(err, "has dev signed firmware").Err()
	} else if firmware.IsDevKeys(keys, info.NewLogger()) {
		return nil
	}
	return errors.Reason("has dev signed firmware: dev signed key not found").Err()
}

func init() {
	execs.Register("cros_is_firmware_in_good_state", isFirmwareInGoodState)
	execs.Register("cros_is_on_rw_firmware_stable_verion", isOnRWFirmwareStableVersionExec)
	execs.Register("cros_is_rw_firmware_stable_version_available", isRWFirmwareStableVersionAvailableExec)
	execs.Register("cros_has_dev_signed_firmware", hasDevSignedFirmwareExec)
	execs.Register("cros_run_firmware_update", runFirmwareUpdaterExec)
	execs.Register("cros_disable_fprom_write_protect", runDisableFPROMWriteProtectExec)
}
