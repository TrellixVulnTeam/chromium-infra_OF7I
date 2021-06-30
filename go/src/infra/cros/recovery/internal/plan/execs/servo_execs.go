// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"log"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/retry"
	"infra/cros/recovery/tlw"
)

// NOTE: That is just fake execs for local testing during developing.
// TODO(otabek@): Replace with real execs.

func pingServoHostActionExec(ctx context.Context, args *RunArgs) error {
	return retry.LimitCount(ctx, 3, time.Second, func() error {
		return args.Access.Ping(ctx, args.DUT.ServoHost.Name, 2)
	}, "ping servo-host")
}

func sshServoHostActionExec(ctx context.Context, args *RunArgs) error {
	return retry.LimitCount(ctx, 3, time.Second, func() error {
		if r := args.Access.Run(ctx, args.DUT.ServoHost.Name, "true"); r.ExitCode != 0 {
			return errors.Reason("check ssh access, code: %d, %s", r.ExitCode, r.Stderr).Err()
		}
		return nil
	}, "check ssh access")
}

func servodInitActionExec(ctx context.Context, args *RunArgs) error {
	req := &tlw.InitServodRequest{
		Resource: args.DUT.Name,
		Options:  defaultServodOptions,
	}
	if err := args.Access.InitServod(ctx, req); err != nil {
		return errors.Annotate(err, "init servod").Err()
	}
	return nil
}

func servodStopActionExec(ctx context.Context, args *RunArgs) error {
	if err := args.Access.StopServod(ctx, args.DUT.Name); err != nil {
		return errors.Annotate(err, "stop servod").Err()
	}
	return nil
}

func servodRestartActionExec(ctx context.Context, args *RunArgs) error {
	if err := servodStopActionExec(ctx, args); err != nil {
		log.Printf("Servod restart: fail stop servod. Error: %s", err)
	}
	if err := servodInitActionExec(ctx, args); err != nil {
		return errors.Annotate(err, "restart servod").Err()
	}
	return nil
}

func init() {
	execMap["servo_host_ping"] = pingServoHostActionExec
	execMap["servo_host_ssh"] = sshServoHostActionExec
	execMap["servo_host_servod_init"] = servodInitActionExec
	execMap["servo_host_servod_stop"] = servodStopActionExec
	execMap["servo_host_servod_restart"] = servodRestartActionExec
}
