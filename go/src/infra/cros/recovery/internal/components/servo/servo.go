// Copyright 2022 The Chromium OS Authors. All rights reserved.  Use
// of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/components"
	"infra/cros/recovery/internal/log"
)

// ServoSupportsBuiltInPDControl checks whether or not the attached
// servo device supports power-deliver related servod controls.
//
// This is applicable only with Servo V4 Type-C.
func ServoSupportsBuiltInPDControl(ctx context.Context, servod components.Servod) (bool, error) {
	if connectionType, err := GetString(ctx, servod, "root.dut_connection_type"); err != nil {
		return false, errors.Annotate(err, "servo supports built-in PD control").Err()
	} else if connectionType != "type-c" {
		log.Debugf(ctx, "Servo Supports Built-In PD Control: connection type %q does not match type-c", connectionType)
		return false, nil
	}
	// The minimum expected voltage on the charger port on servo V4.
	chargerPortMinVoltage := 4400.0
	if chgPortMv, err := GetDouble(ctx, servod, "ppchg5_mv"); err != nil {
		return false, errors.Annotate(err, "servo supports built in PD control").Err()
	} else if chgPortMv < chargerPortMinVoltage {
		log.Debugf(ctx, "Servo Supports Built in PD Control: charger not plugged into servo V4, charger port voltage %f is less than the thresohld %f", chgPortMv, chargerPortMinVoltage)
		return false, nil
	} else {
		log.Debugf(ctx, "Servo Supports Built in PD Control: Charger port voltage %f is at least equal to the thresohld %f", chgPortMv, chargerPortMinVoltage)
	}
	return true, nil
}
