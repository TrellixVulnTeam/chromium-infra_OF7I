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
func pingExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	return retry.WithTimeout(ctx, 20*time.Second, normalBootingTime, func() error {
		return args.Access.Ping(ctx, args.ResourceName, 2)
	}, "cros dut ping")
}

// sshExec verifies ssh access to the DUT.
func sshExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	return retry.WithTimeout(ctx, 20*time.Second, normalBootingTime, func() error {
		if r := args.Access.Run(ctx, args.ResourceName, "true"); r.ExitCode != 0 {
			return errors.Reason("cros dut ssh access, code: %d, %s", r.ExitCode, r.Stderr).Err()
		}
		return nil
	}, "cros dut ssh access")
}

// rebootExec reboots the cros DUT.
func rebootExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	log.Debug(ctx, "Run: %s", rebootCommand)
	r := args.Access.Run(ctx, args.ResourceName, rebootCommand)
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
func matchStableOSVersionToDeviceExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	expected := args.DUT.StableVersion.CrosImage
	log.Debug(ctx, "Expected version: %s", expected)
	fromDevice, err := releaseBuildPath(ctx, args.ResourceName, args)
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

// isDefaultBootFromDiskExec confirms the resource is set to boot from disk by default.
func isDefaultBootFromDiskExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, "crossystem dev_default_boot")
	if r.ExitCode != 0 {
		return errors.Reason("default boot from disk: failed with code: %d and %q", r.ExitCode, r.Stderr).Err()
	}
	defaultBoot := strings.TrimSpace(r.Stdout)
	if defaultBoot != "disk" {
		return errors.Reason("default boot from disk: failed, expected: disk, but got: %q", defaultBoot).Err()
	}
	return nil
}

// isNotInDevModeExec confirms that the host is not in dev mode.
func isNotInDevModeExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, "crossystem devsw_boot")
	if r.ExitCode != 0 {
		return errors.Reason("not in dev mode: failed with code: %d, %q", r.ExitCode, r.Stderr).Err()
	}
	devModeResult := strings.TrimSpace(r.Stdout)
	if devModeResult != "0" {
		return errors.Reason("not in dev mode: failed").Err()
	}
	return nil
}

// matchSerialNumberToInvExec matches serial number from the resource to value in the Inventory.
func matchSerialNumberToInvExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, "vpd -g serial_number")
	if r.ExitCode != 0 {
		return errors.Reason("match serial number to inventory: failed with code: %d and %q", r.ExitCode, r.Stderr).Err()
	}
	expectedSerialNumber := args.DUT.SerialNumber
	actualSerialNumber := strings.TrimSpace(r.Stdout)
	if actualSerialNumber != expectedSerialNumber {
		return errors.Reason("match serial number to inventory: failed, expected: %q, but got %q", expectedSerialNumber, expectedSerialNumber).Err()
	}
	return nil
}

// hasKernelBootPriorityChangeExec checks if kernel priority changed and waiting for reboot.
func hasKernelBootPriorityChangeExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	yes, err := IsKernelPriorityChanged(ctx, args.DUT.Name, args.Access)
	if err != nil {
		return errors.Annotate(err, "has kernel boot priority changed").Err()
	}
	if !yes {
		return errors.Reason("has kernel boot priority changed: priority not changed").Err()
	}
	return nil
}

// runShellCommandExec runs a given action exec arguments in shell.
func runShellCommandExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if len(actionArgs) != 0 {
		log.Debug(ctx, "Run shell command: arguments %s.", actionArgs)
		cmd := strings.Join(actionArgs, " ")
		r := args.Access.Run(ctx, args.ResourceName, cmd)
		if r.ExitCode != 0 {
			return errors.Reason("run shell command: failed with code: %d and %q", r.ExitCode, r.Stderr).Err()
		}
	} else {
		log.Debug(ctx, "Run shell command: no arguments passed.")
	}
	return nil
}

func init() {
	execs.Register("cros_ping", pingExec)
	execs.Register("cros_ssh", sshExec)
	execs.Register("cros_reboot", rebootExec)
	execs.Register("cros_match_stable_os_version_to_device", matchStableOSVersionToDeviceExec)
	execs.Register("cros_is_default_boot_from_disk", isDefaultBootFromDiskExec)
	execs.Register("cros_is_not_in_dev_mode", isNotInDevModeExec)
	execs.Register("cros_match_serial_number_inventory", matchSerialNumberToInvExec)
	execs.Register("cros_has_kernel_priority_change", hasKernelBootPriorityChangeExec)
	execs.Register("cros_run_shell_command", runShellCommandExec)
}
