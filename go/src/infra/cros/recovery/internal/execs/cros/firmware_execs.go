// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"time"

	"go.chromium.org/luci/common/errors"

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
func isFirmwareInGoodState(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.NewRunner(args.ResourceName)
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
func isOnRWFirmwareStableVersionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	stableVersion := args.DUT.StableVersion.CrosFirmwareVersion
	err := matchCrosSystemValueToExpectation(ctx, args.NewRunner(args.ResourceName), "fwid", stableVersion)
	return errors.Annotate(err, "on rw firmware stable version").Err()
}

// isRWFirmwareStableVersionAvailableExec confirms the stable firmware is up to date with the available firmware.
func isRWFirmwareStableVersionAvailableExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.NewRunner(args.ResourceName)
	modelFirmware, err := ReadFirmwareManifest(ctx, r, args.DUT.Model)
	if err != nil {
		return errors.Annotate(err, "rw firmware stable version available").Err()
	}
	availableVersion, err := modelFirmware.AvailableRWFirmware()
	if err != nil {
		return errors.Annotate(err, "rw firmware stable version available").Err()
	}
	stableVersion := args.DUT.StableVersion.CrosFirmwareVersion
	if availableVersion != stableVersion {
		return errors.Reason("rw firmware stable version not available, expected %q, found %q", availableVersion, stableVersion).Err()
	}
	return nil
}

func init() {
	execs.Register("cros_is_firmware_in_good_state", isFirmwareInGoodState)
	execs.Register("cros_is_on_rw_firmware_stable_verion", isOnRWFirmwareStableVersionExec)
	execs.Register("cros_is_rw_firmware_stable_version_available", isRWFirmwareStableVersionAvailableExec)
}
