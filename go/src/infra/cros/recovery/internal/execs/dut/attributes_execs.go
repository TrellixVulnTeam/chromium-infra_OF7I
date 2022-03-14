// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"context"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// notInPoolExec verifies that DUT is not used in special pools.
// List of pools should be listed as part of ActionArgs.
func notInPoolExec(ctx context.Context, info *execs.ExecInfo) error {
	if len(info.ActionArgs) == 0 {
		log.Debug(ctx, "Not in pool: no pools passed as arguments.")
		return nil
	}
	poolMap := getDUTPoolMap(ctx, info.RunArgs.DUT)
	for _, pool := range info.ActionArgs {
		pool = strings.TrimSpace(pool)
		if poolMap[pool] {
			return errors.Reason("not in pool: dut is in pool %q", pool).Err()
		}
		log.Debug(ctx, "Not in pools: %q pool is not matched.", pool)
	}
	log.Debug(ctx, "Not in pools: no intersection found.")
	return nil
}

// isInPoolExec verifies that DUT is used in special pools.
// List of pools should be listed as part of ActionArgs.
func isInPoolExec(ctx context.Context, info *execs.ExecInfo) error {
	if len(info.ActionArgs) == 0 {
		log.Debug(ctx, "Is in pool: no pools passed as arguments.")
		return nil
	}
	poolMap := getDUTPoolMap(ctx, info.RunArgs.DUT)
	for _, pool := range info.ActionArgs {
		pool = strings.TrimSpace(pool)
		if poolMap[pool] {
			log.Debug(ctx, "Is in pools: %q pool listed at the DUT.", pool)
			return nil
		}
		log.Debug(ctx, "Is in pools: %q pool is not matched.", pool)
	}
	return errors.Reason("is in pool: not match found").Err()
}

// getDUTPoolMap extract map of pools listed under DUT.
func getDUTPoolMap(ctx context.Context, d *tlw.Dut) map[string]bool {
	poolMap := make(map[string]bool)
	pools := d.ExtraAttributes["pool"]
	if len(pools) == 0 {
		log.Debug(ctx, "device does not have any pools.")
		return poolMap
	}
	for _, pool := range pools {
		poolMap[pool] = true
	}
	return poolMap
}

func init() {
	execs.Register("dut_not_in_pool", notInPoolExec)
	execs.Register("dut_is_in_pool", isInPoolExec)
}
