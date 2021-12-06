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
	// Glob used to find in-use flags files.
	// The shell expands "*", this argument must NOT be quoted when used in a shell command.
	// Examples: '/var/lib/servod/somebody_in_use'
	inUseFlagFileGlob = "/var/lib/servod/*_in_use"
	// Threshold we decide to ignore a in_use file lock. In minutes.
	inUseFlagFileExpirationMins = 90
)

// hasNoServoInUseExec fails if any servo is in-use now.
func hasNoServoInUseExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	// Recursively look for the in-use files which are modified less than or exactly X minutes ago.
	cmd := fmt.Sprintf("find %s -mmin -%d", inUseFlagFileGlob, inUseFlagFileExpirationMins)
	r := args.Access.Run(ctx, args.ResourceName, cmd)
	// Ignore exit code as if it fail to execute that mean no flag files.
	log.Debug(ctx, "Has no servo is-use: finished with code: %d, error: %s", r.ExitCode, r.Stderr)
	v := strings.TrimSpace(r.Stdout)
	if v == "" {
		log.Debug(ctx, "Does not have any servo in-use.")
		return nil
	}
	return errors.Reason("has no servo is in-use: found flags\n%s", v).Err()
}

func init() {
	execs.Register("cros_has_no_servo_in_use", hasNoServoInUseExec)
}
