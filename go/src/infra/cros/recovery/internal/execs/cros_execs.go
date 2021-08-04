// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/retry"
)

const (
	// defaultAttemptCount tells default count of retries.
	defaultAttemptCount = 3
)

// pingCrosDUTActionExec performs ping action to the ChromeOS DUT.
func pingCrosDUTActionExec(ctx context.Context, args *RunArgs) error {
	return retry.LimitCount(ctx, defaultAttemptCount, time.Second, func() error {
		return args.Access.Ping(ctx, args.DUT.Name, 2)
	}, "cros dut ping")
}

// sshCrosDUTActionExec performs ssh present verification for ChromeOS DUT.
func sshCrosDUTActionExec(ctx context.Context, args *RunArgs) error {
	return retry.LimitCount(ctx, defaultAttemptCount, time.Second, func() error {
		if r := args.Access.Run(ctx, args.DUT.Name, "true"); r.ExitCode != 0 {
			return errors.Reason("cros dut ssh access, code: %d, %s", r.ExitCode, r.Stderr).Err()
		}
		return nil
	}, "cros dut ssh access")
}

func init() {
	execMap["cros_ping"] = pingCrosDUTActionExec
	execMap["cros_ssh"] = sshCrosDUTActionExec
}
