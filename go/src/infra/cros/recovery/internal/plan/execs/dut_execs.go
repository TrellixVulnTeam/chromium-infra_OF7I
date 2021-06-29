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

// NOTE: That is just fake execs for local testing during developing.
// TODO(otabek@): Replace with real execs.

func pingDUTActionExec(ctx context.Context, args *RunArgs) error {
	return retry.LimitCount(ctx, 3, time.Second, func() error {
		return args.Access.Ping(ctx, args.DUT.Name, 2)
	}, "ping dut")
}

func sshDUTActionExec(ctx context.Context, args *RunArgs) error {
	return retry.LimitCount(ctx, 3, time.Second, func() error {
		if r := args.Access.Run(ctx, args.DUT.Name, "true"); r.ExitCode != 0 {
			return errors.Reason("check ssh access, code: %d, %s", r.ExitCode, r.Stderr).Err()
		}
		return nil
	}, "check ssh access")
}

func init() {
	execMap["dut_ping"] = pingDUTActionExec
	execMap["dut_ssh"] = sshDUTActionExec
}
