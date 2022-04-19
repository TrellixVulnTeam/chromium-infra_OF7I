// Copyright 2021 The Chromium OS Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package topology

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/servo"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
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

	// This character delimits the prefix that represents base-name of
	// servo hub path. For example, given the root servo base name
	// '1-3.2.1', the prefix '1-3.2' represents the servo hub. The
	// character '.' delineates this prefix within the complete base
	// name.
	servoTailSplitter = "."

	// This command finds the complete path of files named 'serial'
	serialFindCMD = "find %s/* -name serial"
)

// Mapping of various vid-pid values to servo types.
var vidPidServoTypes = map[string]string{
	"18d1:501b": servo.SERVO_V4,
	"18d1:520d": servo.SERVO_V4P1,
	"18d1:5014": servo.CCD_CR50,
	"18d1:504a": servo.CCD_GSC,
	"18d1:501a": servo.SERVO_MICRO,
	"18d1:5041": servo.C2D2,
	"18d1:5020": servo.SWEETBERRY,
}

// GetRootServo fetches the ServoTopologyItem representing the
// root-servo for a given servo serial number.
func GetRootServo(ctx context.Context, runner execs.Runner, servoSerial string) (*tlw.ServoTopologyItem, error) {
	devicePath, err := GetRootServoPath(ctx, runner, servoSerial)
	if err != nil {
		return nil, errors.Annotate(err, "get root servo").Err()
	}
	return readDeviceInfo(ctx, runner, devicePath), nil
}

// readDeviceInfo retrieves the ServoTopologyItem structure
// representing the servo for the passed devicePath
func readDeviceInfo(ctx context.Context, runner execs.Runner, devicePath string) *tlw.ServoTopologyItem {
	var err error
	servo := &tlw.ServoTopologyItem{
		SysfsPath: devicePath,
	}
	servo.Serial, err = readServoFs(ctx, runner, devicePath, serialNumberFileName)
	if err != nil {
		log.Debugf(ctx, "Read Device Info: %q", err)
	}
	vidPid, err := fsReadVidPid(ctx, runner, devicePath)
	if err != nil {
		log.Debugf(ctx, "Read Device Info: %q", err)
	}
	if servo.Type, err = convertVidPidToServoType(vidPid); err != nil {
		log.Debugf(ctx, "Read Device Info: %q", err)
	}
	servo.UsbHubPort, err = readServoFs(ctx, runner, devicePath, servoHubFileName)
	if err != nil {
		log.Debugf(ctx, "Read Device Info: %q", err)
	}
	servo.FwVersion, err = readServoFs(ctx, runner, devicePath, configurationFileName)
	if err != nil {
		log.Debugf(ctx, "Read Device Info: %q", err)
	}
	servo.SysfsProduct, err = readServoFs(ctx, runner, devicePath, productFileName)
	if err != nil {
		log.Debugf(ctx, "Read Device Info: %q", err)
	}
	log.Debugf(ctx, "Read Device Info: servo %q", ConvertServoTopologyItemToString(servo))
	return servo
}

// RereadServoFwVersion rereads the servo's firmware inside servo toplogy to the current version
// based on the configuration file.
func RereadServoFwVersion(ctx context.Context, runner execs.Runner, servo *tlw.ServoTopologyItem) error {
	if fwVersion, err := readServoFs(ctx, runner, servo.SysfsPath, configurationFileName); err != nil {
		return errors.Annotate(err, "reread servo fw version").Err()
	} else {
		log.Debugf(ctx, "Reread servo device %q firmware to be %q", servo.Type, fwVersion)
		servo.FwVersion = fwVersion
		return nil
	}
}

// GetRootServoPath gets the path of root servo on servo host.
func GetRootServoPath(ctx context.Context, runner execs.Runner, servoSerial string) (string, error) {
	servoPath, err := runner(ctx, time.Minute, fmt.Sprintf(servodtoolDeviceUSBPathCMD, servoSerial))
	if err != nil {
		return "", errors.Annotate(err, "get root servo path: servo not detected").Err()
	}
	if len(servoPath) < minServoPathLength {
		return "", errors.Reason("get root servo path: servo not detected, servo path is empty").Err()
	}
	return servoPath, nil
}

// GetServoUsbDevnum returns the current usb devnum of servo.
func GetServoUsbDevnum(ctx context.Context, runner execs.Runner, servoSerial string) (string, error) {
	rootServoPath, err := GetRootServoPath(ctx, runner, servoSerial)
	if err != nil {
		return "", errors.Annotate(err, "get servo usb devnum").Err()
	}
	devnum, err := runner(ctx, time.Minute, fmt.Sprintf("cat %s/devnum", rootServoPath))
	if err != nil {
		return "", errors.Annotate(err, "get servo usb devnum").Err()
	}
	return devnum, nil
}

// Read servo information from a file contained within file-system
// path to servo. filename is the name of the file from which the
// information is to be read. servoPath is the complete path to the
// directory that contains servo details.
func readServoFs(ctx context.Context, runner execs.Runner, servoPath string, filename string) (string, error) {
	fullPath := filepath.Join(servoPath, filename)
	v, err := runner(ctx, 20*time.Second, fmt.Sprintf(fileReadCmd, fullPath))
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
func convertVidPidToServoType(vidPid string) (string, error) {
	if vidPid == "" {
		return "", errors.Reason("convert vid pid to servo type: vidPid is empty").Err()
	}
	deviceType, ok := vidPidServoTypes[vidPid]
	if !ok {
		return "", errors.Reason("convert vid pid to servo type: servo type for vidPid %q does not exist", vidPid).Err()
	}
	return deviceType, nil
}

// RetrieveServoTopology retries the servo topology consisting of root servo and servo
// children on a host.
func RetrieveServoTopology(ctx context.Context, runner execs.Runner, servoSerial string) (*tlw.ServoTopology, error) {
	servoTopology := &tlw.ServoTopology{}
	devices, err := ListOfDevices(ctx, runner, servoSerial)
	if err != nil {
		errors.Annotate(err, "retrieve servo topology").Err()
	}
	for _, d := range devices {
		if IsItemGood(ctx, d) {
			if d.Serial == servoSerial {
				servoTopology.Root = d
			} else {
				servoTopology.Children = append(servoTopology.Children, d)
			}
		} else {
			log.Infof(ctx, "Retrieve Servo Topology: %q is missing some data", d)
		}
	}
	return servoTopology, nil
}

// ListOfDevices returns a slice of ServoTopologyItem objects that
// represent all the servo devices connected to the servo host.
func ListOfDevices(ctx context.Context, runner execs.Runner, servoSerial string) ([]*tlw.ServoTopologyItem, error) {
	devices := []*tlw.ServoTopologyItem{}
	servoHub, err := findServoHub(ctx, runner, servoSerial)
	if err != nil {
		return nil, errors.Annotate(err, "retrieve servo topology").Err()
	}
	// Find all serial files of devices under servo-hub. Each device
	// has to have a serial number.
	v, err := runner(ctx, time.Minute, fmt.Sprintf(serialFindCMD, servoHub))
	if err != nil {
		return nil, errors.Annotate(err, "retrieve servo topology").Err()
	}
	for _, device := range strings.Split(v, "\n") {
		devices = append(devices, readDeviceInfo(ctx, runner, filepath.Dir(device)))
	}
	return devices, nil
}

// Find the servo hub path. This is the directory that contains
// details within subdirectries about all available servo devices.
// The root servo is connected directly to the servo-hub. To find
// other servos connected to the hub we need find the path to the
// servo-hub. The servod-tool always return direct path to the servo,
// such as:
//    /sys/bus/usb/devices/1-3.2.1
//    base path:  /sys/bus/usb/devices/
//    root-servo:  1-3.2.1
// The alternative path to the same root servo is
// '/sys/bus/usb/devices/1-3.2/1-3.2.1/' where '1-3.2' is path to
// servo-hub. To extract path to servo-hub, remove last digit of the
// port where root servo connects to the servo-hub.
//    base path:  /sys/bus/usb/devices/
//    servo-hub:  1-3.2
//    root-servo: .1
// Later we will join only base path with servo-hub.
func findServoHub(ctx context.Context, runner execs.Runner, servoSerial string) (string, error) {
	rootServoPath, err := GetRootServoPath(ctx, runner, servoSerial)
	if err != nil {
		return "", errors.Annotate(err, "find servo hub").Err()
	}
	basePath, servoTail := filepath.Dir(rootServoPath), filepath.Base(rootServoPath)
	log.Debugf(ctx, "Find Servo Hub: basePath %q, servoTail %q", basePath, servoTail)
	servoHubTail := strings.Split(servoTail, servoTailSplitter)
	return filepath.Join(basePath, strings.Join(servoHubTail[:len(servoHubTail)-1], servoTailSplitter)), nil
}

// ConvertServoTopologyItemToString returns a string representation of
// ServoTopologyItem instance.
func ConvertServoTopologyItemToString(c *tlw.ServoTopologyItem) string {
	return fmt.Sprintf("deviceType %q, product %q, serial %q, hub %q, path %q, version %q", c.Type, c.SysfsProduct, c.Serial, c.UsbHubPort, c.SysfsPath, c.FwVersion)
}

// IsItemGood checks whether a ServoTopologyItem has
// minimum required data.
func IsItemGood(ctx context.Context, c *tlw.ServoTopologyItem) bool {
	return c.Serial != "" && c.Type != "" && c.UsbHubPort != ""
}

// Devices creates and returns a slice of the servo devices in servo topology.
// if there is a filteredBoard, then only return the slice of topology
// item that contains that one filtered board.
func Devices(c *tlw.ServoTopology, filteredBoard string) []*tlw.ServoTopologyItem {
	devices := []*tlw.ServoTopologyItem{}
	// Servo device root.
	if filteredBoard == "" || filteredBoard == c.Root.Type {
		devices = append(devices, c.Root)
	}
	// Servo device children.
	for _, d := range c.Children {
		if filteredBoard == "" || filteredBoard == d.Type {
			devices = append(devices, d)
		}
	}
	return devices
}
