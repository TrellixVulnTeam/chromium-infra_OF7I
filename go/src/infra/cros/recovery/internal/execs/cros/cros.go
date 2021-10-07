// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"regexp"
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
