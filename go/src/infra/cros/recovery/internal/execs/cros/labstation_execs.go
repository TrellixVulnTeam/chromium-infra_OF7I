// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	// Filter used to find in-use flags files.
	// Examples: '/var/lib/servod/somebody_in_use'
	inUseFlagFileFilter = "/var/lib/servod/*_in_use"
	// Threshold we decide to ignore a in_use file lock. In minutes.
	inUseFlagFileExpirationMins = 90
)

// hasServoInUseExec checks if any servo is in-use now.
func hasServoInUseExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	// Recursively look for the in-use files which are modified less than or exactly X minutes ago.
	cmd := fmt.Sprintf("find %s -mmin -%d", inUseFlagFileFilter, inUseFlagFileExpirationMins)
	r := args.Access.Run(ctx, args.ResourceName, cmd)
	// Ignore exit code as if it fail to execute that mean no flag files.
	log.Debug(ctx, "Has servo is-use: finished with code: %d, error: %s", r.ExitCode, r.Stderr)
	v := strings.TrimSpace(r.Stdout)
	if v == "" {
		log.Debug(ctx, "Does not have any servo in-use.")
		return nil
	}
	return errors.Reason("has servo is in-use: found flags\n%s", v).Err()
}

const (
	// Filter used to find reboot request flags files.
	// Examples: '/var/lib/servod/somebody_reboot'
	rebootFlagFileFilter = "/var/lib/servod/*_reboot"
)

// hasRebootRequestExec checks presence of reboot request flag on the host.
func hasRebootRequestExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, fmt.Sprintf("find %s", rebootFlagFileFilter))
	// Print finish result as we treat failure as no results.
	log.Debug(ctx, "Has reboot requests: finished with code: %d, error: %s", r.ExitCode, r.Stderr)
	rr := strings.TrimSpace(r.Stdout)
	if rr == "" {
		return errors.Reason("has reboot request: not request found").Err()
	}
	log.Info(ctx, "Found reboot requests:\n%s", rr)
	return nil
}

// removeRebootRequestsExec removes reboot flag file requests.
func removeRebootRequestsExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, fmt.Sprintf("rm %s", rebootFlagFileFilter))
	// Print finish result as we ignore any errors.
	log.Debug(ctx, "Has reboot requests: finished with code: %d, error: %s", r.ExitCode, r.Stderr)
	return nil
}

// cleanTmpOwnerRequestExec cleans tpm owner requests.
func cleanTmpOwnerRequestExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	r := args.Access.Run(ctx, args.ResourceName, "crossystem clear_tpm_owner_request=1")
	log.Debug(ctx, "Clean TMP before reboot: finished with code: %d, error: %s", r.ExitCode, r.Stderr)
	if r.ExitCode != 0 {
		return errors.Reason("clear tpm owner request: finished with code: %d, error: %s", r.ExitCode, r.Stderr).Err()
	}
	return nil
}

// validateUptime validate that host is up for more than 6 hours.
func validateUptime(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	dur, err := uptime(ctx, args.ResourceName, args)
	if err != nil {
		return errors.Annotate(err, "is up more 6 hours").Err()
	}
	h := dur.Hours()
	// TODO(otabek): Read expected hours limit from actionArgs and keep 6 as default.
	if h < 6 {
		return errors.Reason("is up more 6 hours: only %s is up", dur).Err()
	}
	return nil
}

const (
	// The flag-file indicates the host should not to be rebooted.
	noRebootFlagFile = "/tmp/no_reboot"
)

// hasNoRebootRequestExec  checks presence of no-reboot request flag on the host.
func hasNoRebootRequestExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	cmd := fmt.Sprintf("find %s", noRebootFlagFile)
	r := args.Access.Run(ctx, args.ResourceName, cmd)
	if r.ExitCode != 0 {
		return errors.Reason("has no-reboot request: finished with code: %d, error: %s", r.ExitCode, r.Stderr).Err()
	}
	log.Debug(ctx, "No-reboot request file found.")
	return nil
}

func init() {
	execs.Register("cros_has_servo_in_use", hasServoInUseExec)
	execs.Register("cros_remove_reboot_request", removeRebootRequestsExec)
	execs.Register("cros_has_reboot_request", hasRebootRequestExec)
	execs.Register("cros_clean_tmp_owner_request", cleanTmpOwnerRequestExec)
	execs.Register("cros_validate_uptime", validateUptime)
	execs.Register("cros_has_noreboot_request", hasNoRebootRequestExec)
}
