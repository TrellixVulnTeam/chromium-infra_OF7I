// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// resetEcExec resets EC from DUT side to wake CR50 up.
//
// @params: actionArgs should be in the format of:
// Ex: ["wait_timeout:x"]
func resetEcExec(ctx context.Context, info *execs.ExecInfo) error {
	argsMap := info.GetActionArgs(ctx)
	// Delay to wait for the ec reset command to be efftive. Default to be 30s.
	waitTimeout := argsMap.AsDuration(ctx, "wait_timeout", 30, time.Second)
	// TODO: (@otabek & @yunzhiyu) Figure out why, in PARIS,
	// we must add '&& exit' to make ssh proxy return immediately without
	// waiting for the command to finish execution.
	//
	// Command to reset EC from DUT side.
	const ecResetCmd = "ectool reboot_ec cold && exit"
	run := info.NewRunner(info.RunArgs.DUT.Name)
	if out, err := run(ctx, 30*time.Second, ecResetCmd); err != nil {
		// Client closed connected as rebooting.
		log.Debugf(ctx, "Client exit as device rebooted: %s", err)
		return errors.Annotate(err, "reset ec").Err()
	} else {
		log.Debugf(ctx, "Stdout: %s", out)
	}
	log.Debugf(ctx, "waiting for %d seconds to let ec reset be effective.", waitTimeout)
	time.Sleep(waitTimeout)
	return nil
}

func init() {
	execs.Register("cros_reset_ec", resetEcExec)
}
