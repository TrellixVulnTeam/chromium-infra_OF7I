// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/servo"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/execs/servo/topology"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/retry"
	"infra/cros/recovery/tlw"
)

const (
	// This is the command to get servo firmware version for requested
	// board and channel.
	latestVersionCMD = "servo_updater -p -b %q -c %s | grep firmware"

	// This character separates out the fields from the output of
	// servo updater command line utility. For example, an output is
	// "firmware: servo_v4_v2.4.58-c37246f9c". The splitter is
	// required to separate out the string after ':'.
	servoUpdaterOutputSplitter = ":"

	// This command fetches the path to the latest available firmware on host.
	// latestFirmwareVersionCMD = "realpath /usr/share/servo_updater/firmware/%s.bin"

	// Commands to update FW for servo. Always reboot servo after update.
	fwUpdateCmdTail         = `-b %s -s %s -c %s --reboot`
	fwUpdateCmd             = `servo_updater ` + fwUpdateCmdTail
	fwUpdateForContainerCmd = `python /update_servo_firmware.py ` + fwUpdateCmdTail
	fwUpdateForceCmdTail    = ` --force `

	// fwUpdaterTimeout is the max time it allows for the firmware update command to execute before fail.
	fwUpdaterTimeout = 2 * time.Minute
)

var (
	// Map for all the supported servo devices that are capable of updating its servo firmware.
	updatableServoNames = map[string]bool{
		servo.SERVO_V4:    true,
		servo.SERVO_V4P1:  true,
		servo.SERVO_MICRO: true,
		servo.C2D2:        true,
		servo.SWEETBERRY:  true,
	}
	// servoUpdateIssueMessages is the list of known, tracking issue related to servo_updater.
	servoUpdateIssueMessages = []string{"Configuration not set"}
)

// Checks whether the servo update is required for the passed servo device.
func needsUpdate(ctx context.Context, runner execs.Runner, device *tlw.ServoTopologyItem, channel tlw.ServoFirmwareChannel) bool {
	if !updatableServoNames[device.Type] {
		log.Debugf(ctx, "Needs Update: servo type %q cannot be updated", device.Type)
		return false
	}
	if isVersionOutdated(ctx, runner, device, channel) {
		log.Debugf(ctx, "Needs Update: version is outdated, update needed.")
		return true
	}
	log.Debugf(ctx, "Needs Update: Device Type is %q, update not needed.", device.Type)
	return false
}

// Checks whether the servo version is outdated for the passed servo device.
func isVersionOutdated(ctx context.Context, runner execs.Runner, device *tlw.ServoTopologyItem, channel tlw.ServoFirmwareChannel) bool {
	cVersion := device.FwVersion
	log.Debugf(ctx, "Is Version Outdated: current version is %q", cVersion)
	if cVersion == "" {
		return true
	}
	lVersion := latestVersionFromUpdater(ctx, runner, channel, device.Type)
	log.Debugf(ctx, "Is Version Outdated: latest version is %q", lVersion)
	// In LABPACK, if lVersion is empty, we raise an
	// exception. However, we really only care whether latest version
	// is available (i.e. is non-empty string), so that we can compare
	// it with current version.
	if cVersion != lVersion {
		log.Infof(ctx, "Servo %q version mismatch %q != %q", device.Type, cVersion, lVersion)
		return true
	}
	return false
}

// Get latest available version from the servo_updater command.
func latestVersionFromUpdater(ctx context.Context, runner execs.Runner, channel tlw.ServoFirmwareChannel, board string) string {
	result, err := runner(ctx, time.Minute, fmt.Sprintf(latestVersionCMD, board, strings.ToLower(string(channel))))
	// An example result is "firmware: servo_v4_v2.4.58-c37246f9c". We
	// need to parse-out the string after ":" here, because that is
	// the firmware version value we are looking for.
	if err == nil {
		resultComponents := strings.Split(result, servoUpdaterOutputSplitter)
		if len(resultComponents) == 2 {
			return strings.TrimSpace(resultComponents[len(resultComponents)-1])
		}
	} else {
		log.Debugf(ctx, "Latest Version From Updater: encountered error while determining latest version %q", err)
	}
	// If any error was encountered while determining the later
	// version, it is sufficient to return an empty string because
	// when this returned value is compared with the current version,
	// a mismatch will be detected and firmware update check will
	// return an error from the function that calls this current
	// function.
	return ""
}

const (
	// Commands to kill active servo_updater
	killActiveUpdatersCmd = `ps aux | grep -ie [s]ervo_updater |grep "%s" | awk '{print $2}' | xargs kill -9`
)

// KillActiveUpdaterProcesses kills any active servo_updater processes running on the host.
func KillActiveUpdaterProcesses(ctx context.Context, r execs.Runner, timeout time.Duration, deviceSerial string) error {
	cmd := fmt.Sprintf(killActiveUpdatersCmd, deviceSerial)
	if _, err := r(ctx, timeout, cmd); err != nil {
		log.Debugf(ctx, "Fail to kill active update process")
		return errors.Annotate(err, "kill active update process").Err()
	}
	return nil
}

// createServoDeviceFwUpdateCmd returns the specific servo device update command that is unique to different type of the servo board(device).
func createServoDeviceFwUpdateCmd(useContainer bool, device *tlw.ServoTopologyItem, forceUpdate bool, channel tlw.ServoFirmwareChannel) string {
	var cmd string
	if useContainer {
		cmd = fwUpdateForContainerCmd
	} else {
		cmd = fwUpdateCmd
	}
	cmd = fmt.Sprintf(cmd, device.Type, device.Serial, channel)
	if forceUpdate {
		cmd += fwUpdateForceCmdTail
	}
	return cmd
}

// updateServoDeviceFW is the detailed execution of the process of updating one particular servo board(device).
func updateServoDeviceFW(ctx context.Context, r execs.Runner, useContainer bool, device *tlw.ServoTopologyItem, forceUpdate bool, ignoreVersion bool, channel tlw.ServoFirmwareChannel) error {
	if !ignoreVersion {
		if needsUpdate(ctx, r, device, channel) {
			log.Debugf(ctx, "This servo %q: still need update", device.Type)
		} else {
			log.Debugf(ctx, "This servo %q: does not need update", device.Type)
			return nil
		}
	} else {
		log.Debugf(ctx, "Version check for %q skipped by request!", device.Type)
	}
	updateCmd := createServoDeviceFwUpdateCmd(useContainer, device, forceUpdate, channel)
	if forceUpdate {
		log.Debugf(ctx, "Try to update servo fw of the %q with force.", device.Type)
	} else {
		log.Debugf(ctx, "Try to update servo fw of the %q.", device.Type)
	}
	if !useContainer {
		// Kill any active updater processes before return.
		defer func() {
			if err := KillActiveUpdaterProcesses(ctx, r, 30*time.Second, device.Serial); err != nil {
				log.Debugf(ctx, "Kill active update process fail: %s", err)
			}
		}()
	}
	// Perform servo device firmware update.
	if fwUpdateStdOut, err := r(ctx, fwUpdaterTimeout, updateCmd); err != nil {
		return errors.Annotate(err, "update servo device fw").Err()
	} else {
		log.Debugf(ctx, "Servo firmware update of %q finished with output: %s", device.Type, fwUpdateStdOut)
		log.Infof(ctx, "Servo firmware update of %q finished.", device.Type)
		return nil
	}
}

// runUpdateServoDeviceFwAttempt will update the specific servo board(device) based on the condition specified by the parameter once.
//
// if there is no error returned, then the update of this specific device is successful.
// if there is error returned, then the update is not successful.
func runUpdateServoDeviceFwAttempt(ctx context.Context, r execs.Runner, device *tlw.ServoTopologyItem, req FwUpdaterRequest) error {
	err := updateServoDeviceFW(ctx, r, req.UseContainer, device, req.ForceUpdate, req.IgnoreVersion, req.FirmwareChannel)
	if err != nil {
		errMsg := err.Error()
		log.Debugf(ctx, `(Not critical) fail to update %s; %s`, device.Type, errMsg)
		for _, issueMessage := range servoUpdateIssueMessages {
			if strings.Contains(errMsg, issueMessage) {
				return errors.Annotate(err, "run update servo device fw attempt: issue with servo_updater detected").Err()
			}
		}
	}
	// This will reread the servo fw version after the update attempt
	// so the topology item is up to date for the check "needsUpdate()".
	// Since after the servo update fw, the configuration file taking a little bit
	// time to populate, so retry it 3 times
	readServoFwErr := retry.LimitCount(ctx, 3, time.Second, func() error {
		return topology.RereadServoFwVersion(ctx, r, device)
	}, "re-read servo device fw version")
	if readServoFwErr != nil {
		return errors.Annotate(err, "run update servo device fw attempt").Err()
	}
	if req.IgnoreVersion || !needsUpdate(ctx, r, device, req.FirmwareChannel) {
		log.Infof(ctx, "Servo %q firmware updated successfully", device.Type)
		return nil
	}
	return errors.Reason("run update servo device fw attempt: the servo device still require updates").Err()
}

// FwUpdaterRequest is the request struct for updating the servo firmware of current DUT's servo.
type FwUpdaterRequest struct {
	// Whether the current DUT's ServoHost is using container servod.
	UseContainer bool
	// Firmware channel of the servo for the current DUT.
	FirmwareChannel tlw.ServoFirmwareChannel
	// Count of attempts to update servo firmware.
	TryAttemptCount int
	// Try force update again if the first fw update attempt failed.
	TryForceUpdateAfterFail bool
	// Force to update the servo fw using the force update command.
	ForceUpdate bool
	// Do not check the version on the device.
	IgnoreVersion bool
}

// UpdateDevicesServoFw will try to update every board (servo devices) within the servo topology.
//
// @return: slice of the board (servo device) that didn't update successfully.
func UpdateDevicesServoFw(ctx context.Context, r execs.Runner, req FwUpdaterRequest, devices []*tlw.ServoTopologyItem) []*tlw.ServoTopologyItem {
	failDevices := []*tlw.ServoTopologyItem{}
	for _, device := range devices {
		if !topology.IsItemGood(ctx, device) {
			log.Debugf(ctx, "%s does not have minimum required data to update its firmware", device.Type)
			continue
		}
		if !updatableServoNames[device.Type] {
			log.Debugf(ctx, "%s is not supportive of servo firmware update.", device.Type)
			continue
		}
		updateErr := retry.LimitCount(ctx, req.TryAttemptCount, 0, func() error {
			return runUpdateServoDeviceFwAttempt(ctx, r, device, req)
		}, fmt.Sprintf("update %s's servo firmware", device.Type))
		if updateErr == nil {
			log.Infof(ctx, "%s servo firmware updated successfully.", device.Type)
			continue
		}
		// Normal update attempt failed.
		if req.TryForceUpdateAfterFail {
			newReq := req
			newReq.ForceUpdate = true
			if err := runUpdateServoDeviceFwAttempt(ctx, r, device, newReq); err == nil {
				log.Infof(ctx, "%s servo firmware force-updated successfully.", device.Type)
				continue
			}
		}
		// Normal and force update attempt both failed.
		log.Infof(ctx, "Fail update firmware for %s", device.Type)
		failDevices = append(failDevices, device)
	}
	return failDevices
}
