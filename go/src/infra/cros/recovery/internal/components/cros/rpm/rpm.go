// Copyright 2021 The Chromium OS Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpm

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/components/cros"
	"infra/cros/recovery/internal/components/cros/power"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/internal/retry"
)

// ValidatePowerState will validate whether the value observed power
// state matches the expected power state.
func ValidatePowerState(ctx context.Context, run components.Runner, pinger components.Pinger, expectedOnline bool, timeOut, waitInterval time.Duration) error {
	return retry.WithTimeout(ctx, waitInterval, timeOut, func() error {
		if err := cros.IsSSHable(ctx, run); err != nil {
			log.Debugf(ctx, "Validate Power Source: host is not reachable over SSH.")
			return errors.Annotate(err, "validate power state").Err()
		}
		p, err := power.ReadPowerInfo(ctx, run)
		if err != nil {
			return errors.Annotate(err, "validate power state").Err()
		}
		isOnline, err := p.IsACOnline()
		if err != nil {
			return errors.Annotate(err, "validate power state").Err()
		}
		log.Debugf(ctx, "Validate Power Source: expected power state: %t, observed power state : %t.", expectedOnline, isOnline)
		if isOnline == expectedOnline {
			log.Debugf(ctx, "Validate Power Source: expected power state value matches the observed value.")
			return nil
		}
		log.Debugf(ctx, "Validate Power Source: expected power state value does not match the observed value.")
		return errors.Reason("validate power state: actual power online status %t does not match the expected value %t", isOnline, expectedOnline).Err()
	}, "validate power state")
}
