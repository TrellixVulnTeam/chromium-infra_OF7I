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

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/localtlw/servod"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

const (
	// This is the servod control for obtaining ppdut5 bus voltage in
	// millivolts.
	servodPPDut5Cmd = "ppdut5_mv"
)

// GetUSBDrivePathOnDut finds and returns the path of USB drive on a DUT.
func GetUSBDrivePathOnDut(ctx context.Context, args *execs.RunArgs) (string, error) {
	// switch USB on servo multiplexer to the DUT-side
	if _, err := ServodCallSet(ctx, args, servod.ImageUsbkeyDirection, servod.ImageUsbkeyTowardsDUT); err != nil {
		return "", errors.Annotate(err, "get usb drive path on dut: could not switch USB to DUT").Err()
	}
	run := args.NewRunner(args.DUT.Name)
	// A detection delay is required when attaching this USB drive to DUT
	time.Sleep(usbDetectionDelay * time.Second)
	if out, err := run(ctx, "ls /dev/sd[a-z]"); err != nil {
		return "", errors.Annotate(err, "get usb drive path on dut").Err()
	} else {
		for _, p := range strings.Split(out, "\n") {
			dtOut, dtErr := run(ctx, fmt.Sprintf(". /usr/share/misc/chromeos-common.sh; get_device_type %s", p))
			if dtErr != nil {
				return "", errors.Annotate(dtErr, "get usb drive path on dut: could not check %q", p).Err()
			}
			if dtOut == "USB" {
				if _, fErr := run(ctx, fmt.Sprintf("fdisk -l %s", p)); fErr == nil {
					return p, nil
				} else {
					log.Debug(ctx, "Get USB-drive path on dut: checked candidate usb drive path %q and found it incorrect.", p)
				}
			}
		}
		log.Debug(ctx, "Get USB-drive path on dut: did not find any valid USB drive path on the DUT.")
	}
	return "", errors.Reason("get usb drive path on dut: did not find any USB Drive connected to the DUT as we checked that DUT is up").Err()
}

const (
	// servoTypeCmd is the servod command for getting the servo type information.
	servoTypeCmd = "servo_type"
)

// GetServoType finds and returns the servo type of the DUT's servo.
func GetServoType(ctx context.Context, args *execs.RunArgs) (string, error) {
	servoType, err := servodGetString(ctx, args, servoTypeCmd)
	if err != nil {
		return "", errors.Annotate(err, "get servo type").Err()
	}
	if servoType == "" {
		return "", errors.Reason("get servo type: servo type is empty").Err()
	}
	return servoType, nil
}

// MainServoDevice returns the main servo device.
//
// For example, if the servo_type value returned by servod is
// 'servo_v4_with_ccd_cr50', then the function with extract the
// substring 'ccd_cr50' and will return it. The logic is to extract
// and return the substring between '_with_' and '_and_', if such a
// substring is present in the servo_type value.
//
// A few more examples are:
// servo_type: "servo_v3", returned value: "servo_v3"
// servo_type: "servo_v4", returned value: "servo_v4"
// servo_type: "servo_v4_with_ccd_cr50", returned value: "ccd_cr50"
// servo_type: "servo_v4_with_servo_micro_and_ccd_cr50", returned value: "servo_micro"
func MainServoDevice(ctx context.Context, args *execs.RunArgs) (string, error) {
	servoType, err := GetServoType(ctx, args)
	if err != nil {
		return "", errors.Annotate(err, "main servo device").Err()
	}
	s, err := mainServoDeviceHelper(servoType)
	if err != nil {
		return "", errors.Annotate(err, "main servo device").Err()
	}
	return s, nil
}

// mainServoDeviceHelper extracts the main servo device from the
// servoType string.
//
// This is the central logic for finding the main device, and has been
// extracted out to make unit-tests simple.
func mainServoDeviceHelper(servoType string) (string, error) {
	s1 := strings.Split(servoType, "_with_")
	s2 := strings.Split(s1[len(s1)-1], "_and_")[0]
	if len(s2) == 0 {
		return s2, errors.Reason("main servo device helper: main device not found").Err()
	}
	return s2, nil
}

// IsContainerizedServoHost checks if the servohost is using servod container.
func IsContainerizedServoHost(ctx context.Context, servoHost *tlw.ServoHost) bool {
	if servoHost == nil || servoHost.ContainerName == "" {
		return false
	}
	log.Debug(ctx, "Servo uses servod container with the name: %s", servoHost.ContainerName)
	return true
}
