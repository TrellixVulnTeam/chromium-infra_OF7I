// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

// kernelBootPriorityChangedExec checks if kernel priority changed.
func kernelBootPriorityChangedExec(ctx context.Context, info *execs.ExecInfo) error {
	yes, err := IsKernelPriorityChanged(ctx, info.NewRunner(info.RunArgs.DUT.Name))
	if err != nil {
		return errors.Annotate(err, "kernel boot priority changed").Err()
	}
	if !yes {
		return errors.Reason("kernel boot priority changed: priority not changed").Err()
	}
	log.Debugf(ctx, "Kernel boot priority changed. Expecting reboot.")
	return nil
}

// kernelBootPriorityPersistExec checks if kernel priority has not changed.
func kernelBootPriorityPersistExec(ctx context.Context, info *execs.ExecInfo) error {
	yes, err := IsKernelPriorityChanged(ctx, info.NewRunner(info.RunArgs.DUT.Name))
	if err != nil {
		return errors.Annotate(err, "kernel boot priority persist").Err()
	}
	if yes {
		return errors.Reason("kernel boot priority persist: priority changed").Err()
	}
	log.Debugf(ctx, "Kernel boot priority persist. Reboot is not expected.")
	return nil
}

func init() {
	execs.Register("cros_kernel_priority_has_changed", kernelBootPriorityChangedExec)
	execs.Register("cros_kernel_priority_has_not_changed", kernelBootPriorityPersistExec)
}
