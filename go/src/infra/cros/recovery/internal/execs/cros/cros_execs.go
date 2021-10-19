// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

const (
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
	// PROVISION_FAILED - A flag file to indicate provision failures.  The
	// file is created at the start of any AU procedure (see
	// `ChromiumOSProvisioner._prepare_host()`).  The file's location in
	// stateful means that on successful update it will be removed.  Thus, if
	// this file exists, it indicates that we've tried and failed in a
	// previous attempt to update.
	provisionFailed = "/var/tmp/provision_failed"
	// The percentage of the battery that is considered to be not enough.
	minimumBatteryLevel = 80
	verify_gsc_cmd      = "gsctool -a -f"
)

// pingExec verifies the DUT is pingable.
func pingExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	return WaitUntilPingable(ctx, args, args.ResourceName, NormalBootingTime, 2)
}

// sshExec verifies ssh access to the DUT.
func sshExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	return WaitUntilSSHable(ctx, args, args.ResourceName, NormalBootingTime)
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

// isOnStableVersionExec matches device OS version to stable CrOS version.
func isOnStableVersionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	expected := args.DUT.StableVersion.CrosImage
	log.Debug(ctx, "Expected version: %s", expected)
	fromDevice, err := releaseBuildPath(ctx, args.ResourceName, args)
	if err != nil {
		return errors.Annotate(err, "match os version").Err()
	}
	log.Debug(ctx, "Version on device: %s", fromDevice)
	if fromDevice != expected {
		return errors.Reason("match os version: mismatch, expected %q, found %q", expected, fromDevice).Err()
	}
	return nil
}

// notOnStableVersionExec verifies devices OS is not matches stable CrOS version.
func notOnStableVersionExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	expected := args.DUT.StableVersion.CrosImage
	log.Debug(ctx, "Expected version: %s", expected)
	fromDevice, err := releaseBuildPath(ctx, args.ResourceName, args)
	if err != nil {
		return errors.Annotate(err, "match os version").Err()
	}
	log.Debug(ctx, "Version on device: %s", fromDevice)
	if fromDevice == expected {
		return errors.Reason("match os version: matched, expected %q, found %q", expected, fromDevice).Err()
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

// isFileSystemWritable confirms the stateful file systems are writable.
//
// The standard linux response to certain unexpected file system errors
// (including hardware errors in block devices) is to change the file
// system status to read-only.  This checks that that hasn't happened.
// The test covers the two file systems that need to be writable for
// critical operations like AU:
//     * The (unencrypted) stateful system which includes /mnt/stateful_partition.
//     * The encrypted stateful partition, which includes /var.
// The test doesn't check various bind mounts; those are expected to
// fail the same way as their underlying main mounts.  Whether the
// Linux kernel can guarantee that is untested...
func isFileSystemWritableExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	// N.B. Order matters here:  Encrypted stateful is loop-mounted from a file in unencrypted stateful,
	// so we don't test for errors in encrypted stateful if unencrypted fails.
	testDirs := []string{"/mnt/stateful_partition", "/var/tmp"}
	for _, testDir := range testDirs {
		filename := filepath.Join(testDir, "writable_my_test_file")
		command := fmt.Sprintf("touch %s && rm %s", filename, filename)
		r := args.Access.Run(ctx, args.ResourceName, command)
		if r.ExitCode != 0 {
			log.Debug(ctx, "Cannot create a file in %s! \n Probably the FS is read-only", testDir)
			return errors.Reason("file system writtable: failed with code: %d and %q", r.ExitCode, r.Stderr).Err()
		}
	}
	return nil
}

// hasPythonInterpreterExec confirm the presence of a working Python interpreter.
func hasPythonInterpreterExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, `python -c "import json"`)
	switch {
	case r.ExitCode == 0:
		// Python detected and import is working. do nothing
		return nil
	case r.ExitCode == 127:
		if strings.TrimSpace(args.Access.Run(ctx, args.ResourceName, "which python").Stdout) == "" {
			return errors.Reason("has python interpreter: python is missing; may be caused by powerwash").Err()
		}
		fallthrough
	default:
		return errors.Reason("has python interpreter: interpreter is broken").Err()
	}
}

// hasCriticalKernelErrorExec confirms we have seen critical file system kernel errors
func hasCriticalKernelErrorExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	// grep for stateful FS errors of the type "EXT4-fs error (device sda1):"
	command := `dmesg | grep -E "EXT4-fs error \(device $(cut -d ' ' -f 5,9 /proc/$$/mountinfo | grep -e '^/mnt/stateful_partition ' | cut -d ' ' -f 2 | cut -d '/' -f 3)\):"`
	r := args.Access.Run(ctx, args.ResourceName, command)
	if strings.TrimSpace(r.Stdout) != "" {
		sample := strings.Split(r.Stdout, `\n`)[0]
		// Log the first file system error.
		log.Error(ctx, "first file system error: %q", sample)
		return errors.Reason("has critical kernel error: saw file system error: %s", sample).Err()
	}
	// Check for other critical FS errors.
	command = `dmesg | grep "This should not happen!!  Data will be lost"`
	r = args.Access.Run(ctx, args.ResourceName, command)
	if strings.TrimSpace(r.Stdout) != "" {
		return errors.Reason("has critical kernel error: saw file system error: Data will be lost").Err()
	}
	log.Debug(ctx, "Could not determine stateful mount.")
	return nil
}

// isLastProvisionSuccessfulExec confirms that the DUT successfully finished its last provision job.
//
// At the start of any update (e.g. for a Provision job), the code creates a marker file named `PROVISION_FAILED`.
// The file is located in a part of the stateful partition that will be removed if an update finishes successfully.
// Thus, the presence of the file indicates that a prior update failed.
// The verifier tests for the existence of the marker file and fails if it still exists.
func isLastProvisionSuccessfulExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, fmt.Sprintf("test -f %s", provisionFailed))
	if r.ExitCode == 0 {
		return errors.Reason("last provision successful: last provision on this DUT failed").Err()
	}
	return nil
}

// isACPowerConnectedExec confirms whether the DUT is connected through AC power.
func isACPowerConnectedExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	p, err := ReadPowerInfo(ctx, args)
	if err != nil {
		return errors.Annotate(err, "ac power connected").Err()
	}
	powerByAC, err := p.IsACOnline()
	if err != nil {
		return errors.Annotate(err, "ac power connected").Err()
	}
	if !powerByAC {
		return errors.Reason("ac power connected: is not plugged in").Err()
	}
	return nil
}

// isBatteryExpectedExec confirms whether the DUT is expected to have battery according to inventory.
func isBatteryExpectedExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	if args.DUT.PowerSupplyType != tlw.PowerSupplyTypeBattery {
		return errors.Reason("is battery expected: battery is not expected").Err()
	}
	return nil
}

// isBatteryPresentExec confirms that the DUT has battery.
func isBatteryPresentExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	p, err := ReadPowerInfo(ctx, args)
	if err != nil {
		return errors.Annotate(err, "battery present").Err()
	}
	hasBattery, err := p.HasBattery()
	if err != nil {
		return errors.Annotate(err, "battery present").Err()
	}
	if !hasBattery {
		return errors.Reason("battery present: battery is not presented.").Err()
	}
	return nil
}

// isBatteryLevelGreaterThanMinimumExec confirms the battery has enough charge.
func isBatteryLevelGreaterThanMinimumExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	p, err := ReadPowerInfo(ctx, args)
	if err != nil {
		return errors.Annotate(err, "battery has enough charge").Err()
	}
	currentLevel, err := p.BatteryLevel()
	if err != nil {
		return errors.Annotate(err, "battery has enough charge").Err()
	}
	if currentLevel < minimumBatteryLevel {
		return errors.Reason("battery has enough charge: battery's current level is less than the minimum level: %d", minimumBatteryLevel).Err()
	}
	return nil
}

// isBatteryChargingExec confirms the battery is charging.
func isBatteryChargingExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	p, err := ReadPowerInfo(ctx, args)
	if err != nil {
		return errors.Annotate(err, "battery charging").Err()
	}
	isDischarging, err := p.IsBatteryDischarging()
	if err != nil {
		return errors.Annotate(err, "battery charging").Err()
	}
	if isDischarging {
		return errors.Reason("battery charging: battery is in discharging state").Err()
	}
	return nil
}

// isBatteryChargableOrGoodLevelExec confirms the battery either has enough charge or is charging.
func isBatteryChargableOrGoodLevelExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	batteryLevelError, batteryChargingError := isBatteryLevelGreaterThanMinimumExec(ctx, args, actionArgs), isBatteryChargingExec(ctx, args, actionArgs)
	if batteryLevelError != nil && batteryChargingError != nil {
		multiBatteryError := errors.NewMultiError(batteryLevelError, batteryChargingError)
		// Log both of batteryLevelError and batteryChargingError
		log.Error(ctx, multiBatteryError[0].Error()+" and "+multiBatteryError[1].Error())
		return errors.Annotate(multiBatteryError, "battery chargable or good level: battery does not have enough charge and in discharging state").Err()
	}
	if batteryLevelError != nil {
		log.Error(ctx, batteryLevelError.Error())
	}
	if batteryChargingError != nil {
		log.Error(ctx, batteryChargingError.Error())
	}
	return nil
}

// isNotVirtualMachineExec confirms that the given DUT is not a virtual device.
func isNotVirtualMachineExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, `cat /proc/cpuinfo | grep "model name"`)
	if r.ExitCode != 0 {
		return errors.Reason("not virtual machine: failed with code: %d, %q", r.ExitCode, r.Stderr).Err()
	}
	output := strings.TrimSpace(r.Stdout)
	if output == "" {
		return errors.Reason("not virtual machine: no cpu information found").Err()
	}
	if strings.Contains(strings.ToLower(output), "qemu") {
		return errors.Reason("not virtual machine: qemu is virtual machine").Err()
	}
	return nil
}

// waitForSystemExec waits for system-service to be running.
//
// Sometimes, update_engine will take a while to update firmware, so we
// should give this some time to finish. See crbug.com/765686#c38 for details.
func waitForSystemExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	serviceName := "system-services"
	// Check the status of an upstart init script
	cmd := fmt.Sprintf("status %s", serviceName)
	r := args.NewRunner(args.ResourceName)
	output, err := r(ctx, cmd)
	if err != nil {
		return errors.Annotate(err, "wait for system").Err()
	}
	if !strings.Contains(output, "start/running") {
		return errors.Reason("wait for system: service %s not running", serviceName).Err()
	}
	return nil
}

// isGscToolPresentExec confirms that GSC tool is functional.
//
// If board/model expected to have GSC tool but it does not have it then need
// to re-image the host to recover it.
// If host-info has label 'cr50' then we expect to have GSC tool on the host.
func isGscToolPresentExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.NewRunner(args.ResourceName)
	_, err := r(ctx, verify_gsc_cmd)
	if err != nil {
		return errors.Annotate(err, "gsc tool present: gsc tool issue detected").Err()
	}
	log.Debug(ctx, "GSC tool is functional")
	return nil
}

func init() {
	execs.Register("cros_ping", pingExec)
	execs.Register("cros_ssh", sshExec)
	execs.Register("cros_reboot", rebootExec)
	execs.Register("cros_is_on_stable_version", isOnStableVersionExec)
	execs.Register("cros_not_on_stable_version", notOnStableVersionExec)
	execs.Register("cros_is_default_boot_from_disk", isDefaultBootFromDiskExec)
	execs.Register("cros_is_not_in_dev_mode", isNotInDevModeExec)
	execs.Register("cros_run_shell_command", runShellCommandExec)
	execs.Register("cros_is_file_system_writable", isFileSystemWritableExec)
	execs.Register("cros_has_python_interpreter_working", hasPythonInterpreterExec)
	execs.Register("cros_has_critical_kernel_error", hasCriticalKernelErrorExec)
	execs.Register("cros_is_last_provision_successful", isLastProvisionSuccessfulExec)
	execs.Register("cros_is_ac_power_connected", isACPowerConnectedExec)
	execs.Register("cros_is_battery_expected", isBatteryExpectedExec)
	execs.Register("cros_is_battery_present", isBatteryPresentExec)
	execs.Register("cros_is_battery_level_greater_than_minimum", isBatteryLevelGreaterThanMinimumExec)
	execs.Register("cros_is_battery_charging", isBatteryChargingExec)
	execs.Register("cros_is_battery_chargable_or_good_level", isBatteryChargableOrGoodLevelExec)
	execs.Register("cros_is_not_virtual_machine", isNotVirtualMachineExec)
	execs.Register("cros_wait_for_system", waitForSystemExec)
	execs.Register("cros_is_gsc_tool_present", isGscToolPresentExec)
}
