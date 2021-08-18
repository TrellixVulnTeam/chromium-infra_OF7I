// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/retry"
)

const (
	// Time to wait a rebooting ChromeOS, in seconds.
	normalBootingTime = 150
	// Default reboot command for ChromeOS devices.
	// Each command set sleep 1 second to wait for reaction of the command from left part.
	rebootCommand = "(echo begin 1; sync; echo end 1 \"$?\")& sleep 1;" +
		"(echo begin 2; reboot; echo end 2 \"$?\")& sleep 1;" +
		// Force reboot is not calling shutdown.
		"(echo begin 3; reboot -f; echo end 3 \"$?\")& sleep 1;" +
		// Force reboot without sync.
		"(echo begin 4; reboot -nf; echo end 4 \"$?\")& sleep 1;" +
		// telinit 6 sets run level for process initialized, which is equivalent to reboot.
		"(echo begin 5; telinit 6; echo end 5 \"$?\")"
)

// pingExec verifies the DUT is pingable.
func pingExec(ctx context.Context, args *execs.RunArgs) error {
	return retry.WithTimeout(ctx, 20*time.Second, normalBootingTime, func() error {
		return args.Access.Ping(ctx, args.DUT.Name, 2)
	}, "cros dut ping")
}

// sshExec verifies ssh access to the DUT.
func sshExec(ctx context.Context, args *execs.RunArgs) error {
	return retry.WithTimeout(ctx, 20*time.Second, normalBootingTime, func() error {
		if r := args.Access.Run(ctx, args.DUT.Name, "true"); r.ExitCode != 0 {
			return errors.Reason("cros dut ssh access, code: %d, %s", r.ExitCode, r.Stderr).Err()
		}
		return nil
	}, "cros dut ssh access")
}

// rebootExec reboots the cros DUT.
func rebootExec(ctx context.Context, args *execs.RunArgs) error {
	log.Debug(ctx, "Run: %s", rebootCommand)
	r := args.Access.Run(ctx, args.DUT.Name, rebootCommand)
	if r.ExitCode == -2 {
		// Client closed connected as rebooting.
		log.Debug(ctx, "Client exit as device rebooted: %s", r.Stderr)
	} else if r.ExitCode != 0 {
		return errors.Reason("cros reboot: failed, code: %d, %s", r.ExitCode, r.Stderr).Err()
	}
	log.Debug(ctx, "Stdout: %s", r.Stdout)
	return nil
}

// matchStableOSVersionToDeviceExec matches stable CrOS version to the device OS.
func matchStableOSVersionToDeviceExec(ctx context.Context, args *execs.RunArgs) error {
	expected := args.DUT.StableVersion.CrosImage
	log.Debug(ctx, "Expected version: %s", expected)
	fromDevice, err := releaseBuildPath(ctx, args.DUT.Name, args)
	if err != nil {
		return errors.Annotate(err, "match os version").Err()
	}
	fromDevice = strings.TrimSuffix(fromDevice, "\n")
	log.Debug(ctx, "Version on device: %s", fromDevice)
	if fromDevice != expected {
		return errors.Reason("match os version: mismatch, expected %q, found %q", expected, fromDevice).Err()
	}
	return nil
}

func init() {
	execs.Register("cros_ping", pingExec)
	execs.Register("cros_ssh", sshExec)
	execs.Register("cros_reboot", rebootExec)
	execs.Register("cros_match_stable_os_version_to_device", matchStableOSVersionToDeviceExec)
}
