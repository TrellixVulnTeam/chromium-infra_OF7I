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
	// Time to wait a rebooting ChromeOS, in seconds.
	NormalBootingTime = 150
	// Command to extract release builder path from device.
	extactReleaseBuilderPathCommand = "cat /etc/lsb-release | grep CHROMEOS_RELEASE_BUILDER_PATH"
)

// releaseBuildPath reads release build path from lsb-release.
func releaseBuildPath(ctx context.Context, resource string, args *execs.RunArgs) (string, error) {
	// lsb-release is set of key=value so we need extract right part from it.
	//  Example: CHROMEOS_RELEASE_BUILDER_PATH=board-release/R99-9999.99.99
	r := args.Access.Run(ctx, resource, extactReleaseBuilderPathCommand)
	if r.ExitCode != 0 {
		return "", errors.Reason("release build path: fail. exit:%d, %s", r.ExitCode, r.Stderr).Err()
	}
	log.Debug(ctx, "Read value: %q.", r.Stdout)
	p, err := regexp.Compile("CHROMEOS_RELEASE_BUILDER_PATH=([\\w\\W]*)")
	if err != nil {
		return "", errors.Annotate(err, "release build path").Err()
	}
	parts := p.FindStringSubmatch(r.Stdout)
	if len(parts) < 2 {
		return "", errors.Reason("release build path: fail to read value from %s", r.Stdout).Err()
	}
	return strings.TrimSpace(parts[1]), nil
}

// uptime returns uptime of resource.
func uptime(ctx context.Context, resource string, args *execs.RunArgs) (*time.Duration, error) {
	// Received value represent two parts where the first value represents the total number
	// of seconds the system has been up and the second value is the sum of how much time
	// each core has spent idle, in seconds. We are looking
	//  E.g.: 683503.88 1003324.85
	// Consequently, the second value may be greater than the overall system uptime on systems with multiple cores.
	r := args.Access.Run(ctx, resource, "cat /proc/uptime")
	if r.ExitCode != 0 {
		return nil, errors.Reason("uptime: fail. exit:%d, %s", r.ExitCode, r.Stderr).Err()
	}
	log.Debug(ctx, "Read value: %q.", r.Stdout)
	p, err := regexp.Compile("([\\d.]{6,})")
	if err != nil {
		return nil, errors.Annotate(err, "uptime").Err()
	}
	parts := p.FindStringSubmatch(r.Stdout)
	if len(parts) < 2 {
		return nil, errors.Reason("uptime: fail to read value from %s", r.Stdout).Err()
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
func IsSSHable(ctx context.Context, args *execs.RunArgs, resourceName string) error {
	if r := args.Access.Run(ctx, resourceName, "true"); r.ExitCode != 0 {
		return errors.Reason("is sshable: code %d, %s", r.ExitCode, r.Stderr).Err()
	}
	return nil
}

// WaitUntilSSHable waiting resource to be sshable.
func WaitUntilSSHable(ctx context.Context, args *execs.RunArgs, resourceName string, waitTime time.Duration) error {
	return retry.WithTimeout(ctx, sshAttemptInteval, waitTime, func() error {
		return IsSSHable(ctx, args, resourceName)
	}, "wait to ssh access")
}

// matchCrosSystemValueToExpectation reads value from crossystem and compared to expected value.
func matchCrosSystemValueToExpectation(ctx context.Context, args *execs.RunArgs, subcommand string, expectedValue string) error {
	r := args.Access.Run(ctx, args.ResourceName, "crossystem "+subcommand)
	if r.ExitCode != 0 {
		return errors.Reason("match crossystem value to expectation: fail read %s. fail with code: %d, %q", subcommand, r.ExitCode, r.Stderr).Err()
	}
	actualValue := strings.TrimSpace(r.Stdout)
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
	_, err := r(ctx, fmt.Sprintf(`test -e "%s"`, path))
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
	output, err := r(ctx, cmd)
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
