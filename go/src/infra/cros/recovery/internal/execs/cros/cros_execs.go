// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components/cros"
	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/retry"
)

const (
	// TODO (vkjoshi@): Migrate the constants (such as
	// MinimumBatteryLevel) and helper functions from
	// internal/execs/cros package to internal/components/cros. Bug
	// b/223061820 captures this task.
	// The percentage of the battery that is considered to be not enough.
	MinimumBatteryLevel = 80
	verify_gsc_cmd      = "gsctool -a -f"
)

// pingExec verifies the DUT is pingable.
func pingExec(ctx context.Context, info *execs.ExecInfo) error {
	return WaitUntilPingable(ctx, info, info.RunArgs.ResourceName, info.ActionTimeout, 2)
}

// sshExec verifies ssh access to the current plan's device (named by the default resource name).
func sshExec(ctx context.Context, info *execs.ExecInfo) error {
	return WaitUntilSSHable(ctx, info.DefaultRunner(), info.ActionTimeout)
}

// sshDUTExec verifies ssh access to the DUT.
func sshDUTExec(ctx context.Context, info *execs.ExecInfo) error {
	return WaitUntilSSHable(ctx, info.NewRunner(info.RunArgs.DUT.Name), info.ActionTimeout)
}

// rebootExec reboots the cros DUT.
func rebootExec(ctx context.Context, info *execs.ExecInfo) error {
	if err := cros.Reboot(ctx, info.NewRunner(info.RunArgs.DUT.Name), info.ActionTimeout); err != nil {
		return errors.Annotate(err, "cros reboot").Err()
	}
	return nil
}

// isOnStableVersionExec matches device OS version to stable CrOS version.
func isOnStableVersionExec(ctx context.Context, info *execs.ExecInfo) error {
	sv, err := info.Versioner().Cros(ctx, info.RunArgs.DUT.Name)
	if err != nil {
		return errors.Annotate(err, "match os version").Err()
	}
	argsMap := info.GetActionArgs(ctx)
	expected := argsMap.AsString(ctx, "os_name", sv.OSImage)
	log.Debugf(ctx, "Expected version: %s", expected)
	fromDevice, err := releaseBuildPath(ctx, info.DefaultRunner())
	if err != nil {
		return errors.Annotate(err, "match os version").Err()
	}
	log.Debugf(ctx, "Version on device: %s", fromDevice)
	if fromDevice != expected {
		return errors.Reason("match os version: mismatch, expected %q, found %q", expected, fromDevice).Err()
	}
	return nil
}

// notOnStableVersionExec verifies devices OS is not matches stable CrOS version.
func notOnStableVersionExec(ctx context.Context, info *execs.ExecInfo) error {
	sv, err := info.Versioner().Cros(ctx, info.RunArgs.DUT.Name)
	if err != nil {
		return errors.Annotate(err, "match os version").Err()
	}
	expected := sv.OSImage
	log.Debugf(ctx, "Expected version: %s", expected)
	fromDevice, err := releaseBuildPath(ctx, info.DefaultRunner())
	if err != nil {
		return errors.Annotate(err, "match os version").Err()
	}
	log.Debugf(ctx, "Version on device: %s", fromDevice)
	if fromDevice == expected {
		return errors.Reason("match os version: matched, expected %q, found %q", expected, fromDevice).Err()
	}
	return nil
}

// isDefaultBootFromDiskExec confirms the resource is set to boot from disk by default.
func isDefaultBootFromDiskExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	err := cros.MatchCrossystemValueToExpectation(ctx, run, "dev_default_boot", "disk")
	return errors.Annotate(err, "default boot from disk").Err()
}

// isNotInDevModeExec confirms that the host is not in dev mode.
func isNotInDevModeExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	err := cros.MatchCrossystemValueToExpectation(ctx, run, "devsw_boot", "0")
	return errors.Annotate(err, "not in dev mode").Err()
}

// runShellCommandExec runs a given action exec arguments in shell.
func runShellCommandExec(ctx context.Context, info *execs.ExecInfo) error {
	// TODO: Convert to single line command and always use linux shell.
	actionArgs := info.ActionArgs
	if len(actionArgs) > 0 {
		log.Debugf(ctx, "Run shell command: arguments %s.", actionArgs)
		run := info.DefaultRunner()
		if out, err := run(ctx, info.ActionTimeout, actionArgs[0], actionArgs[1:]...); err != nil {
			return errors.Annotate(err, "run shell command").Err()
		} else {
			log.Debugf(ctx, "Run shell command: output: %s", out)
		}
	} else {
		log.Debugf(ctx, "Run shell command: no arguments passed.")
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
func isFileSystemWritableExec(ctx context.Context, info *execs.ExecInfo) error {
	// N.B. Order matters here:  Encrypted stateful is loop-mounted from a file in unencrypted stateful,
	// so we don't test for errors in encrypted stateful if unencrypted fails.
	testDirs := []string{"/mnt/stateful_partition", "/var/tmp"}
	for _, testDir := range testDirs {
		filename := filepath.Join(testDir, "writable_my_test_file")
		command := fmt.Sprintf("touch %s && rm %s", filename, filename)
		run := info.DefaultRunner()
		_, err := run(ctx, time.Minute, command)
		if err != nil {
			log.Debugf(ctx, "Cannot create a file in %s! \n Probably the FS is read-only", testDir)
			return errors.Annotate(err, "file system writtable").Err()
		}
	}
	return nil
}

// hasPythonInterpreterExec confirm the presence of a working Python interpreter.
func hasPythonInterpreterExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	_, err := run(ctx, time.Minute, `python -c "import json"`)
	switch {
	case err == nil:
		// Python detected and import is working. do nothing
		return nil
	case execs.SSHErrorCLINotFound.In(err):
		if pOut, pErr := run(ctx, time.Minute, "which python"); pErr != nil {
			return errors.Annotate(pErr, "has python interpreter: python is missing").Err()
		} else if pOut == "" {
			return errors.Reason("has python interpreter: python is missing; may be caused by powerwash").Err()
		}
		fallthrough
	default:
		return errors.Annotate(err, "has python interpreter: interpreter is broken").Err()
	}
}

// hasCriticalKernelErrorExec confirms we have seen critical file system kernel errors
func hasCriticalKernelErrorExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	// grep for stateful FS errors of the type "EXT4-fs error (device sda1):"
	command := `dmesg | grep -E "EXT4-fs error \(device $(cut -d ' ' -f 5,9 /proc/$$/mountinfo | grep -e '^/mnt/stateful_partition ' | cut -d ' ' -f 2 | cut -d '/' -f 3)\):"`
	out, _ := run(ctx, time.Minute, command)
	if out != "" {
		sample := strings.Split(out, `\n`)[0]
		// Log the first file system error.
		log.Errorf(ctx, "first file system error: %q", sample)
		return errors.Reason("has critical kernel error: saw file system error: %s", sample).Err()
	}
	// Check for other critical FS errors.
	command = `dmesg | grep "This should not happen!!  Data will be lost"`
	out, _ = run(ctx, time.Minute, command)
	if out != "" {
		return errors.Reason("has critical kernel error: saw file system error: Data will be lost").Err()
	}
	log.Debugf(ctx, "Could not determine stateful mount.")
	return nil
}

// isNotVirtualMachineExec confirms that the given DUT is not a virtual device.
func isNotVirtualMachineExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	out, err := run(ctx, time.Minute, `cat /proc/cpuinfo | grep "model name"`)
	if err != nil {
		return errors.Annotate(err, "not virtual machine").Err()
	}
	if out == "" {
		return errors.Reason("not virtual machine: no cpu information found").Err()
	}
	if strings.Contains(strings.ToLower(out), "qemu") {
		return errors.Reason("not virtual machine: qemu is virtual machine").Err()
	}
	return nil
}

// waitForSystemExec waits for system-service to be running.
//
// Sometimes, update_engine will take a while to update firmware, so we
// should give this some time to finish. See crbug.com/765686#c38 for details.
func waitForSystemExec(ctx context.Context, info *execs.ExecInfo) error {
	serviceName := "system-services"
	// Check the status of an upstart init script
	cmd := fmt.Sprintf("status %s", serviceName)
	r := info.DefaultRunner()
	output, err := r(ctx, time.Minute, cmd)
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
func isGscToolPresentExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	_, err := r(ctx, time.Minute, verify_gsc_cmd)
	if err != nil {
		return errors.Annotate(err, "gsc tool present: gsc tool issue detected").Err()
	}
	log.Debugf(ctx, "GSC tool is functional")
	return nil
}

// isToolPresentExec checks the presence of the tool on the DUT.
//
// For example, the tool "dfu-programmer" is checked by running the command:
// "hash dfu-programmer" on the DUT
// The actionArgs should be in the format of ["tools:dfu-programmer,python,..."]
func isToolPresentExec(ctx context.Context, info *execs.ExecInfo) error {
	toolMap := info.GetActionArgs(ctx)
	toolNames := toolMap.AsStringSlice(ctx, "tools", nil)
	if len(toolNames) == 0 {
		return errors.Reason("tool present: tools argument is empty or not provided").Err()
	}
	r := info.DefaultRunner()
	for _, toolName := range toolNames {
		toolName = strings.TrimSpace(toolName)
		if toolName == "" {
			return errors.Reason("tool present: tool name given in the tools argument is empty").Err()
		}
		if _, err := r(ctx, time.Minute, "hash", toolName); err != nil {
			return errors.Annotate(err, "tool present").Err()
		}
	}
	return nil
}

// crosSetGbbFlagsExec sets the GBB flags on the DUT.
func crosSetGbbFlagsExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.NewRunner(info.RunArgs.DUT.Name)
	if _, err := run(ctx, info.ActionTimeout, "/usr/share/vboot/bin/set_gbb_flags.sh 0"); err != nil {
		log.Debugf(ctx, "Cros Set Gbb Flags: %s", err)
		return errors.Annotate(err, "cros set gbb flags").Err()
	}
	return nil
}

// crosSwitchToSecureModeExec disables booting into dev-mode on the DUT.
func crosSwitchToSecureModeExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.NewRunner(info.RunArgs.DUT.Name)
	if _, err := run(ctx, info.ActionTimeout, "crossystem", "disable_dev_request=1"); err != nil {
		log.Debugf(ctx, "Cros Switch to Secure Mode %s", err)
		return errors.Annotate(err, "cros switch to secure mode").Err()
	}
	return nil
}

// enrollmentCleanupExec cleans up the enrollment state on the
// ChromeOS device.
func enrollmentCleanupExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	run := info.NewRunner(info.RunArgs.DUT.Name)
	// 1. Reset VPD enrollment state
	repairTimeout := argsMap.AsDuration(ctx, "repair_timeout", 120, time.Second)
	log.Debugf(ctx, "enrollment cleanup: using repair timeout :%s", repairTimeout)
	run(ctx, repairTimeout, "/usr/sbin/update_rw_vpd check_enrollment", "0")
	// 2. clear tpm owner state
	clearTpmOwnerTimeout := argsMap.AsDuration(ctx, "clear_tpm_owner_timeout", 60, time.Second)
	log.Debugf(ctx, "enrollment cleanup: using clear tpm owner timeout :%s", clearTpmOwnerTimeout)
	if _, err := run(ctx, clearTpmOwnerTimeout, "crossystem", "clear_tpm_owner_request=1"); err != nil {
		log.Debugf(ctx, "enrollment cleanup: unable to clear TPM.")
		return errors.Annotate(err, "enrollment cleanup").Err()
	}
	filesToRemove := []string{
		"/home/chronos/.oobe_completed",
		"/home/chronos/Local\\ State",
		"/var/cache/shill/default.profile",
	}
	dirsToRemove := []string{
		"/home/.shadow/*",
		filepath.Join("/var/cache/shill/default.profile", "*"),
		"/var/lib/whitelist/*", // nocheck
		"/var/cache/app_pack",
		"/var/lib/tpm",
	}
	// We do not care about any errors that might be returned by the
	// following two command executions.
	fileDeletionTimeout := argsMap.AsDuration(ctx, "file_deletion_timeout", 120, time.Second)
	run(ctx, fileDeletionTimeout, "sudo", "rm", "-rf", strings.Join(filesToRemove, " "), strings.Join(dirsToRemove, " "))
	run(ctx, fileDeletionTimeout, "sync")
	rebootTimeout := argsMap.AsDuration(ctx, "reboot_timeout", 10, time.Second)
	log.Debugf(ctx, "enrollment cleanup: using reboot timeout :%s", rebootTimeout)
	if err := SimpleReboot(ctx, run, rebootTimeout, info); err != nil {
		return errors.Annotate(err, "enrollment cleanup").Err()
	}
	// Finally, we will read the TPM status, and will check whether it
	// has been cleared or not.
	tpmTimeout := argsMap.AsDuration(ctx, "tpm_timeout", 150, time.Second)
	log.Debugf(ctx, "enrollment cleanup: using tpm timeout :%s", tpmTimeout)
	retry.WithTimeout(ctx, time.Second, tpmTimeout, func() error {
		tpmStatus := NewTpmStatus(ctx, run, repairTimeout)
		if tpmStatus.hasSuccess() {
			return nil
		}
		return errors.Reason("enrollment cleanup: failed to read TPM status.").Err()
	}, "wait to read tpm status")
	tpmStatus := NewTpmStatus(ctx, run, repairTimeout)
	isOwned, err := tpmStatus.isOwned()
	if err != nil {
		return errors.Reason("enrollment cleanup: failed to read TPM status.").Err()
	}
	if isOwned {
		return errors.Reason("enrollment cleanup: failed to clear TPM.").Err()
	}
	return nil
}

// updateCrossystemExec update the value of the command to the value passed in from the config file.
//
// the actionArgs should be in the format of ["command:....", "value:....", "check_after_update:true/false"]
func updateCrossystemExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	command := argsMap.AsString(ctx, "command", "")
	if command == "" {
		return errors.Reason("update crossystem: command cannot be empty").Err()
	}
	val := argsMap.AsString(ctx, "value", "")
	if val == "" {
		return errors.Reason("update crossystem: value cannot be empty").Err()
	}
	checkAfterUpdate := argsMap.AsBool(ctx, "check_after_update", false)
	run := info.NewRunner(info.RunArgs.DUT.Name)
	return errors.Annotate(cros.UpdateCrossystem(ctx, run, command, val, checkAfterUpdate), "update crossystem").Err()
}

func init() {
	execs.Register("cros_ping", pingExec)
	execs.Register("cros_ssh", sshExec)
	execs.Register("cros_ssh_dut", sshDUTExec)
	execs.Register("cros_reboot", rebootExec)
	execs.Register("cros_is_on_stable_version", isOnStableVersionExec)
	execs.Register("cros_not_on_stable_version", notOnStableVersionExec)
	execs.Register("cros_is_default_boot_from_disk", isDefaultBootFromDiskExec)
	execs.Register("cros_is_not_in_dev_mode", isNotInDevModeExec)
	execs.Register("cros_run_shell_command", runShellCommandExec)
	execs.Register("cros_is_file_system_writable", isFileSystemWritableExec)
	execs.Register("cros_has_python_interpreter_working", hasPythonInterpreterExec)
	execs.Register("cros_has_critical_kernel_error", hasCriticalKernelErrorExec)
	execs.Register("cros_is_not_virtual_machine", isNotVirtualMachineExec)
	execs.Register("cros_wait_for_system", waitForSystemExec)
	execs.Register("cros_is_gsc_tool_present", isGscToolPresentExec)
	execs.Register("cros_is_tool_present", isToolPresentExec)
	execs.Register("cros_set_gbb_flags", crosSetGbbFlagsExec)
	execs.Register("cros_switch_to_secure_mode", crosSwitchToSecureModeExec)
	execs.Register("cros_enrollment_cleanup", enrollmentCleanupExec)
	execs.Register("cros_update_crossystem", updateCrossystemExec)
}
