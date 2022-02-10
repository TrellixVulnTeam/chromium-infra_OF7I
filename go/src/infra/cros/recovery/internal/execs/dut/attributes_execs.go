// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// notInPoolExec verifies that DUT is not used in special pools.
// List of pools which should not be in device pool provided by actionArgs.
func notInPoolExec(ctx context.Context, info *execs.ExecInfo) error {
	if len(info.ActionArgs) == 0 {
		log.Debug(ctx, "Not in pool: no action arguments provided.")
		return nil
	}
	pools := info.RunArgs.DUT.ExtraAttributes["pool"]
	if len(pools) == 0 {
		log.Debug(ctx, "Not in pools: device does not have any pools.")
		return nil
	}
	poolMap := make(map[string]bool)
	for _, pool := range pools {
		poolMap[pool] = true
	}
	for _, arg := range info.ActionArgs {
		if poolMap[arg] {
			return errors.Reason("not in pool: dut is in pool %q", arg).Err()
		}
	}
	log.Debug(ctx, "Not in pools: no intersection found.")
	return nil
}

func init() {
	execs.Register("dut_not_in_pool", notInPoolExec)
}
