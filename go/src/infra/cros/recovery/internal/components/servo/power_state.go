// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
)

// PowerStateValue specifies value to set for power_state.
type PowerStateValue string

const (
	powerStateControl = "power_state"
	// Power on the device.
	PowerStateValueON PowerStateValue = "on"
	// Power off the device.
	PowerStateValueOFF PowerStateValue = "off"
	// Cold reset the device.
	PowerStateValueReset PowerStateValue = "reset"
	// Request boot device in recovery mode.
	PowerStateValueRecoveryMode PowerStateValue = "rec"
)

// SetPowerState change the state of power_state control on servod.
//
// Control implemented as call of the function by this reason it does not have getter.
func SetPowerState(ctx context.Context, servod components.Servod, val PowerStateValue) error {
	if err := servod.Set(ctx, powerStateControl, string(val)); err != nil {
		return errors.Annotate(err, "power state %q", val).Err()
	}
	return nil
}
