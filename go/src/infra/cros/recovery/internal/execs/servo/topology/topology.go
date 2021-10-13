// Copyright 2021 The Chromium OS Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package topology

import (
	"context"
	"fmt"
	"path/filepath"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	// Prefix of the command to obtain usb-path to servo device
	servodtoolDeviceUSBPathCMD = "servodtool device -s %s usb-path"

	servoBasePath = "/sys/bus/usb/devices"

	minServoPathLength = len(servoBasePath + "/X")

	// Name of the file within servo-path that holds the serial number of servo
	serialNumberFileName = "serial"

	// Command to read the contents of a file on the file system.
	fileReadCmd = "cat %s"

	// Name of the file within servo-path that holds the servo hub info
	servoHubFileName = "devpath"

	// File containing Vendor ID
	vendorIdFileName = "idVendor"

	// File containing Product ID
	productIdFileName = "idProduct"

	// File containing Product
	productFileName = "product"

	// File containing Configuration
	configurationFileName = "configuration"

	// Various servo-types
	SERVO_V4_TYPE          = "servo_v4"
	SERVO_V4P1_TYPE        = "servo_v4p1"
	SERVO_CR50_TYPE        = "ccd_cr50"
	SERVO_C2D2_TYPE        = "c2d2"
	SERVO_SERVO_MICRO_TYPE = "servo_micro"
	SERVO_SWEETBERRY_TYPE  = "sweetberry"
)

// Mapping of various vid-pid values to servo types.
var vidPidServoTypes = map[string]string{
	"18d1:501b": SERVO_V4_TYPE,
	"18d1:520d": SERVO_V4P1_TYPE,
	"18d1:5014": SERVO_CR50_TYPE,
	"18d1:501a": SERVO_SERVO_MICRO_TYPE,
	"18d1:5041": SERVO_C2D2_TYPE,
	"18d1:5020": SERVO_SWEETBERRY_TYPE,
}

// ConnectedServo represents a servo device that is connected to a
// servo-host.
type ConnectedServo struct {
	// This is the complete path on the file system for the servo
	// device.
	path string
	// This is the type of product for servo, such as Servo V4.
	product string
	// This is the serial number for the servo device.
	serial string
	// This is the type of device for servo, such as servo_v4.
	deviceType string
	// This is the concatenation of vendor-ID and product-ID values
	// for the servo device.
	vidPid string
	// This is the complete hub path for a servo device.
	hubPath string
	// This is the version of servo device.
	version string
}

// String representation of ConnectedServo instance.
func (c *ConnectedServo) String() string {
	return fmt.Sprintf("path %q, product %q, serial %q, deviceType %q, vidPid %q, hubPath %q, version %q", c.path, c.product, c.serial, c.deviceType, c.vidPid, c.hubPath, c.version)
}

// IsGood checks whether a ConnectedServo has minimum required data.
func (c *ConnectedServo) IsGood(ctx context.Context) bool {
	return c.serial != "" && c.deviceType != "" && c.hubPath != ""
}

// GetRootServo fetches the root-servo for a given servo serial
// number.
func GetRootServo(ctx context.Context, runner execs.Runner, servoSerial string) (*ConnectedServo, error) {
	devicePath, err := getRootServoPath(ctx, runner, servoSerial)
	if err != nil {
		return nil, errors.Annotate(err, "get root servo").Err()
	}
	return readDeviceInfo(ctx, runner, devicePath), nil
}

// readDeviceInfo retrieves the ConnectedServo structure representing the
// servo for the passed devicePath
func readDeviceInfo(ctx context.Context, runner execs.Runner, devicePath string) *ConnectedServo {
	var err error
	servo := &ConnectedServo{
		path: devicePath,
	}
	servo.serial, err = readServoFs(ctx, runner, servo.path, serialNumberFileName)
	if err != nil {
		log.Debug(ctx, "Read Device Info: %q", err)
	}
	servo.vidPid, err = fsReadVidPid(ctx, runner, servo.path)
	if err != nil {
		log.Debug(ctx, "Read Device Info: %q", err)
	}
	if err = servo.convertVidPidToServoType(); err != nil {
		log.Debug(ctx, "Read Device Info: %q", err)
	}
	servo.hubPath, err = readServoFs(ctx, runner, servo.path, servoHubFileName)
	if err != nil {
		log.Debug(ctx, "Read Device Info: %q", err)
	}
	servo.version, err = readServoFs(ctx, runner, servo.path, configurationFileName)
	if err != nil {
		log.Debug(ctx, "Read Device Info: %q", err)
	}
	servo.product, err = readServoFs(ctx, runner, servo.path, productFileName)
	if err != nil {
		log.Debug(ctx, "Read Device Info: %q", err)
	}
	log.Debug(ctx, "Read Device Info: servoPath %q, servoSerial %q, servoVidPid %q, servoType %q, servoHub %q, servoVersion %q, servoProduct %q", servo.path, servo.serial, servo.vidPid, servo.deviceType, servo.hubPath, servo.version, servo.product)
	return servo
}

// Get the path of root servo on servo host.
func getRootServoPath(ctx context.Context, runner execs.Runner, servoSerial string) (string, error) {
	servoPath, err := runner(ctx, fmt.Sprintf(servodtoolDeviceUSBPathCMD, servoSerial))
	if err != nil {
		return "", errors.Annotate(err, "get root servo path: servo not detected").Err()
	}
	if len(servoPath) < minServoPathLength {
		return "", errors.Reason("get root servo path: servo not detected, servo path is empty").Err()
	}
	return servoPath, nil
}

// Read servo information from a file contained within file-system
// path to servo. filename is the name of the file from which the
// information is to be read. servoPath is the complete path to the
// directory that contains servo details.
func readServoFs(ctx context.Context, runner execs.Runner, servoPath string, filename string) (string, error) {
	fullPath := filepath.Join(servoPath, filename)
	v, err := runner(ctx, fmt.Sprintf(fileReadCmd, fullPath))
	if err != nil {
		return "", errors.Annotate(err, "read servo file %s", fullPath).Err()
	}
	// no need to check empty string here as to have empty value is ok here.
	return v, nil
}

// Read the vendor ID and product ID files from servo location on file
// system.
func fsReadVidPid(ctx context.Context, runner execs.Runner, servoPath string) (string, error) {
	vid, err := readServoFs(ctx, runner, servoPath, vendorIdFileName)
	if err != nil {
		return "", errors.Annotate(err, "fs read vid pid: could not read %q from %q", vendorIdFileName, servoPath).Err()
	}
	pid, err := readServoFs(ctx, runner, servoPath, productIdFileName)
	if err != nil {
		return "", errors.Annotate(err, "fs read vid pid: could not read %q from %q", productIdFileName, servoPath).Err()
	}
	return fmt.Sprintf("%s:%s", vid, pid), nil
}

// Get the value of servo type based on the a fixed map of vid-pid to
// serial type string.
func (c *ConnectedServo) convertVidPidToServoType() error {
	if c.vidPid == "" {
		return errors.Reason("convert vid pid to servo type: vidPid is empty").Err()
	}
	var ok bool
	c.deviceType, ok = vidPidServoTypes[c.vidPid]
	if !ok {
		return errors.Reason("convert vid pid to servo type: servo type for vidPid %q does not exist", c.vidPid).Err()
	}
	return nil
}
