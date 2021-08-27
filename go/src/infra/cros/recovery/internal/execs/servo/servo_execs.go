// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// NOTE: That is just fake execs for local testing during developing.
// TODO(otabek@): Replace with real execs.

func servodInitActionExec(ctx context.Context, args *execs.RunArgs) error {
	req := &tlw.InitServodRequest{
		Resource: args.DUT.Name,
		Options:  defaultServodOptions,
	}
	if err := args.Access.InitServod(ctx, req); err != nil {
		return errors.Annotate(err, "init servod").Err()
	}
	return nil
}

func servodStopActionExec(ctx context.Context, args *execs.RunArgs) error {
	if err := args.Access.StopServod(ctx, args.DUT.Name); err != nil {
		return errors.Annotate(err, "stop servod").Err()
	}
	return nil
}

func servodRestartActionExec(ctx context.Context, args *execs.RunArgs) error {
	if err := servodStopActionExec(ctx, args); err != nil {
		log.Debug(ctx, "Servod restart: fail stop servod. Error: %s", err)
	}
	if err := servodInitActionExec(ctx, args); err != nil {
		return errors.Annotate(err, "restart servod").Err()
	}
	return nil
}

func init() {
	execs.Register("servo_host_servod_init", servodInitActionExec)
	execs.Register("servo_host_servod_stop", servodStopActionExec)
	execs.Register("servo_host_servod_restart", servodRestartActionExec)
}
