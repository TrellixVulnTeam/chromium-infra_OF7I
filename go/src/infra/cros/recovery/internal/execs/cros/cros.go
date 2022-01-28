// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/retry"
)

const (
	// Values presented as the string of the hex without 0x to match
	// representation in sysfs (idVendor/idProduct).

	// Servo's DUT side HUB vendor id
	SERVO_DUT_HUB_VID = "04b4"
	// Servo's DUT side HUB product id
	SERVO_DUT_HUB_PID = "6502"
	// Servo's DUT side NIC vendor id
	SERVO_DUT_NIC_VID = "0bda"
	// Servo's DUT side NIC product id
	SERVO_DUT_NIC_PID = "8153"
)

const (
	// Time to wait a rebooting ChromeOS, in seconds.
	NormalBootingTime = 150
	// Command to extract release builder path from device.
	extactReleaseBuilderPathCommand = "cat /etc/lsb-release | grep CHROMEOS_RELEASE_BUILDER_PATH"
)

// releaseBuildPath reads release build path from lsb-release.
func releaseBuildPath(ctx context.Context, run execs.Runner) (string, error) {
	// lsb-release is set of key=value so we need extract right part from it.
	//  Example: CHROMEOS_RELEASE_BUILDER_PATH=board-release/R99-9999.99.99
	output, err := run(ctx, time.Minute, extactReleaseBuilderPathCommand)
	if err != nil {
		return "", errors.Annotate(err, "release build path").Err()
	}
	log.Debug(ctx, "Read value: %q.", output)
	p, err := regexp.Compile("CHROMEOS_RELEASE_BUILDER_PATH=([\\w\\W]*)")
	if err != nil {
		return "", errors.Annotate(err, "release build path").Err()
	}
	parts := p.FindStringSubmatch(output)
	if len(parts) < 2 {
		return "", errors.Reason("release build path: fail to read value from %s", output).Err()
	}
	return strings.TrimSpace(parts[1]), nil
}

const (
	extactReleaseBoardCommand = "cat /etc/lsb-release | grep CHROMEOS_RELEASE_BOARD"
	releaseBoardRegexp        = `CHROMEOS_RELEASE_BOARD=(\S+)`
)

// ReleaseBoard reads release board info from lsb-release.
func ReleaseBoard(ctx context.Context, r execs.Runner) (string, error) {
	output, err := r(ctx, time.Minute, extactReleaseBoardCommand)
	if err != nil {
		return "", errors.Annotate(err, "release board").Err()
	}
	compiledRegexp, err := regexp.Compile(releaseBoardRegexp)
	if err != nil {
		return "", errors.Annotate(err, "release board").Err()
	}
	matches := compiledRegexp.FindStringSubmatch(output)
	if len(matches) != 2 {
		return "", errors.Reason("release board: cannot find chromeos release board information").Err()
	}
	board := matches[1]
	log.Debug(ctx, "Release board: %q.", board)
	return board, nil
}

// uptime returns uptime of resource.
func uptime(ctx context.Context, run execs.Runner) (*time.Duration, error) {
	// Received value represent two parts where the first value represents the total number
	// of seconds the system has been up and the second value is the sum of how much time
	// each core has spent idle, in seconds. We are looking
	//  E.g.: 683503.88 1003324.85
	// Consequently, the second value may be greater than the overall system uptime on systems with multiple cores.
	out, err := run(ctx, time.Minute, "cat /proc/uptime")
	if err != nil {
		return nil, errors.Annotate(err, "uptime").Err()
	}
	log.Debug(ctx, "Read value: %q.", out)
	p, err := regexp.Compile("([\\d.]{6,})")
	if err != nil {
		return nil, errors.Annotate(err, "uptime").Err()
	}
	parts := p.FindStringSubmatch(out)
	if len(parts) < 2 {
		return nil, errors.Reason("uptime: fail to read value from %s", out).Err()
	}
	// Direct parse to duration.
	// Example: 683503.88s -> 189h51m43.88s
	dur, err := time.ParseDuration(fmt.Sprintf("%ss", parts[1]))
	return &dur, errors.Annotate(err, "get uptime").Err()
}

// IsPingable checks whether the resource is pingable
func IsPingable(ctx context.Context, args *execs.RunArgs, resourceName string, count int) error {
	return args.Access.Ping(ctx, resourceName, count)
}

const (
	pingAttemptInteval = 5 * time.Second
	sshAttemptInteval  = 10 * time.Second
)

// WaitUntilPingable waiting resource to be pingable.
func WaitUntilPingable(ctx context.Context, args *execs.RunArgs, resourceName string, waitTime time.Duration, count int) error {
	return retry.WithTimeout(ctx, pingAttemptInteval, waitTime, func() error {
		return IsPingable(ctx, args, resourceName, count)
	}, "wait to ping")
}

// IsSSHable checks whether the resource is sshable
func IsSSHable(ctx context.Context, run execs.Runner) error {
	_, err := run(ctx, time.Minute, "true")
	return errors.Annotate(err, "is sshable").Err()
}

// WaitUntilSSHable waiting resource to be sshable.
func WaitUntilSSHable(ctx context.Context, run execs.Runner, waitTime time.Duration) error {
	return retry.WithTimeout(ctx, sshAttemptInteval, waitTime, func() error {
		return IsSSHable(ctx, run)
	}, "wait to ssh access")
}

// matchCrosSystemValueToExpectation reads value from crossystem and compared to expected value.
func matchCrosSystemValueToExpectation(ctx context.Context, run execs.Runner, subcommand string, expectedValue string) error {
	out, err := run(ctx, time.Minute, "crossystem "+subcommand)
	if err != nil {
		return errors.Annotate(err, "match crossystem value to expectation: fail read %s", subcommand).Err()
	}
	actualValue := strings.TrimSpace(out)
	if actualValue != expectedValue {
		return errors.Reason("match crossystem value to expectation: %q, found: %q", expectedValue, actualValue).Err()
	}
	return nil
}

// IsPathExist checks if a given path exists or not.
// Raise error if the path does not exist.
func IsPathExist(ctx context.Context, args *execs.RunArgs, path string) error {
	path = strings.ReplaceAll(path, "\\", "\\\\")
	path = strings.ReplaceAll(path, "$", `\$`)
	path = strings.ReplaceAll(path, `"`, `\"`)
	path = strings.ReplaceAll(path, "`", `\`+"`")
	r := args.NewRunner(args.ResourceName)
	_, err := r(ctx, time.Minute, fmt.Sprintf(`test -e "%s"`, path))
	if err != nil {
		return errors.Annotate(err, "path exist").Err()
	}
	return nil
}

// pathHasEnoughValue is a helper function that checks the given path's free disk space / inodes is no less than the min disk space /indoes specified.
func pathHasEnoughValue(ctx context.Context, args *execs.RunArgs, dutName string, path string, typeOfSpace string, minSpaceNeeded float64) error {
	if err := IsPathExist(ctx, args, path); err != nil {
		return errors.Annotate(err, "path has enough value: %s: path: %q not exist", typeOfSpace, path).Err()
	}
	var cmd string
	if typeOfSpace == "disk space" {
		oneMB := math.Pow(10, 6)
		log.Info(ctx, "Checking for >= %f (GB/inodes) of %s under %s on machine %s", minSpaceNeeded, typeOfSpace, path, dutName)
		cmd = fmt.Sprintf(`df -PB %.f %s | tail -1`, oneMB, path)
	} else {
		// checking typeOfSpace == "inodes"
		cmd = fmt.Sprintf(`df -Pi %s | tail -1`, path)
	}
	r := args.NewRunner(dutName)
	output, err := r(ctx, time.Minute, cmd)
	if err != nil {
		return errors.Annotate(err, "path has enough value: %s", typeOfSpace).Err()
	}
	outputList := strings.Fields(output)
	free, err := strconv.ParseFloat(outputList[3], 64)
	if err != nil {
		log.Error(ctx, err.Error())
		return errors.Annotate(err, "path has enough value: %s", typeOfSpace).Err()
	}
	if typeOfSpace == "diskspace" {
		mbPerGB := math.Pow(10, 3)
		free = float64(free) / mbPerGB
	}
	if free < minSpaceNeeded {
		return errors.Reason("path has enough value: %s: Not enough free %s on %s - %f (GB/inodes) free, want %f (GB/inodes)", typeOfSpace, typeOfSpace, path, free, minSpaceNeeded).Err()
	}
	log.Info(ctx, "Found %f (GB/inodes) >= %f (GB/inodes) of %s under %s on machine %s", free, minSpaceNeeded, typeOfSpace, path, dutName)
	return nil
}

// hasOnlySingleLine determines if the given string is only one single line.
func hasOnlySingleLine(ctx context.Context, s string) bool {
	if s == "" {
		log.Debug(ctx, "The string is empty")
		return false
	}
	lines := strings.Split(s, "\n")
	if len(lines) != 1 {
		log.Debug(ctx, "Found %d lines in the string.", len(lines))
		return false
	}
	return true
}

const (
	// findFilePathByContentCmdGlob find the file path by the content.
	// ex: grep -l xxx $(find /xxx/xxxx -maxdepth 1 -name xxx)
	findFilePathByContentCmdGlob = "grep -l %s $(find %s -maxdepth 1 -name %s)"
)

// FindSingleUsbDeviceFSDir find the common parent directory where the unique device with VID and PID is enumerated by file system.
//
//   1) Get path to the unique idVendor file with VID
//   2) Get path to the unique idProduct file with PID
//   3) Get directions of both files and compare them
//
// @param basePath: Path to the directory where to look for the device.
// @param vid: Vendor ID of the looking device.
// @param pid: Product ID of the looking device.
//
// @returns: path to the folder of the device.
func FindSingleUsbDeviceFSDir(ctx context.Context, r execs.Runner, basePath string, vid string, pid string) (string, error) {
	if basePath == "" {
		return "", errors.Reason("find single usb device file system directory: basePath is not provided").Err()
	}
	basePath += "*/"
	// find vid path:
	vidPath, err := r(ctx, time.Minute, fmt.Sprintf(findFilePathByContentCmdGlob, vid, basePath, "idVendor"))
	if err != nil {
		return "", errors.Annotate(err, "find single usb device file system directory").Err()
	} else if !hasOnlySingleLine(ctx, vidPath) {
		return "", errors.Reason("find single usb device file system directory: found more then one device with required VID: %s", vid).Err()
	}
	// find pid path:
	pidPath, err := r(ctx, time.Minute, fmt.Sprintf(findFilePathByContentCmdGlob, pid, basePath, "idProduct"))
	if err != nil {
		return "", errors.Annotate(err, "find single usb device file system directory").Err()
	} else if !hasOnlySingleLine(ctx, pidPath) {
		return "", errors.Reason("find single usb device file system directory: found more then one device with required PID: %s", pid).Err()
	}
	// If both files locates matched then we found our device.
	commDirCmd := fmt.Sprintf("LC_ALL=C comm -12 <(dirname %s) <(dirname %s)", vidPath, pidPath)
	commDir, err := r(ctx, time.Minute, commDirCmd)
	if err != nil {
		return "", errors.Annotate(err, "find single usb device file system directory").Err()
	} else if commDir == "" || commDir == "." {
		return "", errors.Reason("find single usb device file system directory: directory not found").Err()
	}
	return commDir, nil
}

const (
	// macAddressFileUnderNetFolderOfThePathGlob find NIC address from the nic path.
	// start finding the file name that contains both the /net/ and /address/ under the nic path folder.
	macAddressFileUnderNetFolderOfThePathGlob = "find %s/ | grep /net/ | grep /address"
	// Regex string to validate that MAC address is valid.
	// example of a correct format MAC address: f4:f5:e8:50:d1:cf
	macAddressVerifyRegexp = `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`
)

// ServoNICMacAddress read servo NIC mac address visible from DUT side.
//
// @param nic_path: Path to network device on the host
func ServoNICMacAddress(ctx context.Context, r execs.Runner, nicPath string) (string, error) {
	findNICAddressFileCmd := fmt.Sprintf(macAddressFileUnderNetFolderOfThePathGlob, nicPath)
	nicAddressFile, err := r(ctx, time.Minute, findNICAddressFileCmd)
	if err != nil {
		return "", errors.Annotate(err, "servo nic mac address").Err()
	} else if !hasOnlySingleLine(ctx, nicAddressFile) {
		return "", errors.Reason("servo nic mac address: found more then one nic address file").Err()
	}
	log.Info(ctx, "Found servo NIC address file: %q", nicAddressFile)
	macAddress, err := r(ctx, time.Minute, fmt.Sprintf("cat %s", nicAddressFile))
	if err != nil {
		return "", errors.Annotate(err, "servo nic mac address").Err()
	}
	macAddressRegexp, err := regexp.Compile(macAddressVerifyRegexp)
	if err != nil {
		return "", errors.Annotate(err, "servo nic mac address: regular expression for correct mac address cannot compile").Err()
	}
	if !macAddressRegexp.MatchString(macAddress) {
		log.Info(ctx, "Incorrect format of the servo nic mac address: %s", macAddress)
		return "", errors.Reason("servo nic mac address: incorrect format mac address found").Err()
	}
	log.Info(ctx, "Servo NIC MAC address visible from DUT: %s", macAddress)
	return macAddress, nil
}
