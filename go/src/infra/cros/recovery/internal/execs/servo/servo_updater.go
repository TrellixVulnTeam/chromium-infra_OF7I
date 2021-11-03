// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"fmt"
	"strings"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

const (
	// // This is command to check whether the 'print' option for servo
	// // updater utility is available on the servo host. This option was
	// // added in R90.
	// servoUpdaterPrintCheckCMD = "servo_updater --help | grep print"

	// This is the command to get servo firmware version for requested
	// board and channel.
	latestVersionCMD = "servo_updater -p -b %q -c %s | grep firmware"

	// This character separates out the fields from the output of
	// servo updater command line utility. For example, an output is
	// "firmware: servo_v4_v2.4.58-c37246f9c". The splitter is
	// required to separate out the string after ':'.
	servoUpdaterOutputSplitter = ":"

	// // This command fetches the path to the latest available firmware
	// // on host.
	// latestFirmwareVersionCMD = "realpath /usr/share/servo_updater/firmware/%s.bin"
)

// Checks whether the servo update is required for the passed servo device.
func needsUpdate(ctx context.Context, runner execs.Runner, device *tlw.ServoTopologyItem, channel tlw.ServoFirmwareChannel, board string) bool {
	if isVersionOutdated(ctx, runner, device, channel, board) {
		log.Debug(ctx, "Needs Update: version is outdated, update needed.")
		return true
	}
	log.Debug(ctx, "Needs Update: Device Type is %q, update not needed.", device.Type)
	return false
}

// Checks whether the servo version is outdated for the passed servo device.
func isVersionOutdated(ctx context.Context, runner execs.Runner, device *tlw.ServoTopologyItem, channel tlw.ServoFirmwareChannel, board string) bool {
	cVersion := device.FwVersion
	log.Debug(ctx, "Is Version Outdated: current version is %q", cVersion)
	if cVersion == "" {
		return true
	}
	lVersion := latestVersionFromUpdater(ctx, runner, channel, board)

	log.Debug(ctx, "Is Version Outdated: latest version is %q", lVersion)
	// In LABPACK, if lVersion is empty, we raise an
	// exception. However, we really only care whether latest version
	// is available (i.e. is non-empty string), so that we can compare
	// it with current version.
	return cVersion != lVersion
}

// Get latest available version from the servo_updater command.
func latestVersionFromUpdater(ctx context.Context, runner execs.Runner, channel tlw.ServoFirmwareChannel, board string) string {
	result, err := runner(ctx, fmt.Sprintf(latestVersionCMD, board, strings.ToLower(string(channel))))
	// An example result is "firmware: servo_v4_v2.4.58-c37246f9c". We
	// need to parse-out the string after ":" here, because that is
	// the firmware version value we are looking for.
	if err == nil {
		resultComponents := strings.Split(result, servoUpdaterOutputSplitter)
		if len(resultComponents) == 2 {
			return strings.TrimSpace(resultComponents[len(resultComponents)-1])
		}
	}
	// If any error was encountered while determining the later
	// version, it is sufficient to return an empty string because
	// when this returned value is compared with the current version,
	// a mismatch will be detected and firmware update check will
	// return an error from the function that calls this current
	// function.
	return ""
}
