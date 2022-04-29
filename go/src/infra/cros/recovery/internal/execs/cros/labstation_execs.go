// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	// Default minimum labstation uptime.
	minLabstationUptime = 6 * time.Hour
)

// cleanTmpOwnerRequestExec cleans tpm owner requests.
func cleanTmpOwnerRequestExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	_, err := run(ctx, time.Minute, "crossystem clear_tpm_owner_request=1")
	return errors.Annotate(err, "clear tpm owner request").Err()
}

// validateUptime validate that host is up for more than 6 hours.
func validateUptime(ctx context.Context, info *execs.ExecInfo) error {
	maxUptime := minLabstationUptime
	for _, arg := range info.ActionArgs {
		if strings.HasPrefix(arg, "min_duration:") {
			d, err := time.ParseDuration(strings.Split(arg, ":")[1])
			if err != nil {
				return errors.Annotate(err, "validate uptime: parse action args").Err()
			}
			maxUptime = d
		}
	}
	dur, err := uptime(ctx, info.DefaultRunner())
	if err != nil {
		return errors.Annotate(err, "validate uptime").Err()
	}
	if *dur < maxUptime {
		return errors.Reason("validate uptime: only %s is up", dur).Err()
	}
	return nil
}

const (
	// The flag-file indicates the host should not to be rebooted.
	noRebootFlagFile = "/tmp/no_reboot"
)

// allowedRebootExec checks if DUT is allowed to reboot.
// If system has /tmp/no_reboot file then reboot is not allowed.
func allowedRebootExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	cmd := fmt.Sprintf("test %s", noRebootFlagFile)
	_, err := run(ctx, time.Minute, cmd)
	if err != nil {
		return errors.Annotate(err, "has no-reboot request").Err()
	}
	log.Debugf(ctx, "No-reboot request file found.")
	return nil
}

// filesystemIoNotBlockedExec check if the labstation's filesystem IO is blocked.
func filesystemIoNotBlockedExec(ctx context.Context, info *execs.ExecInfo) error {
	run := info.DefaultRunner()
	cmd := "ps axl | awk '$10 ~ /D/'"
	output, err := run(ctx, info.ActionTimeout, cmd)
	if err != nil {
		return errors.Annotate(err, "filesystem is not blocked").Err()
	}
	// Good labstation may occasionally have an process in uninterruptible
	// sleep state transiently, so we look for these who have 2+ processes
	// stuck in such a state.
	if len(output) > 1 {
		return errors.Reason("filesystem is not blocked: more than one processes in uninterruptible sleep state, I/O is likely blocked.").Err()
	}
	return nil
}

func init() {
	execs.Register("cros_clean_tmp_owner_request", cleanTmpOwnerRequestExec)
	execs.Register("cros_validate_uptime", validateUptime)
	execs.Register("cros_allowed_reboot", allowedRebootExec)
	execs.Register("cros_filesystem_io_not_blocked", filesystemIoNotBlockedExec)
}
