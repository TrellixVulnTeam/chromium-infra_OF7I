// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"
)

const (
	// This represents the number of times we will attempt to collect
	// SBU voltage to calculate the average value.
	sbuVoltageTotalCheckCount = 10
)

// servoCR50LowSBUExec verifies whether SBU voltage is below a
// threshold (2500 mv) blocking enumeration of CR50 component.
//
// This verifier is conditioned on whether the value of servod control
// 'dut_sbu_voltage_float_fault' is on or not.
func servoCR50LowSBUExec(ctx context.Context, info *execs.ExecInfo) error {
	sbuValue, err := MaximalAvgSbuValue(ctx, info.NewServod(), sbuVoltageTotalCheckCount)
	if err != nil {
		return errors.Annotate(err, "servo CR50 low sbu exec").Err()
	}
	log.Debugf(ctx, "Servo CR50 Low Sbu Exec: avg SBU value is %f", sbuValue)
	if sbuValue <= sbuThreshold {
		return errors.Reason("servo CR50 low sbu exec: CR50 not detected due to low SBU voltage").Err()
	}
	return nil
}

// servoCR50EnumeratedExec verifies whether CR50 cannot be enumerated
// despite the voltage being higher than a threshold (2500 mV). This
// can happen when CR50 is in deep sleep.
//
// Please use condition to verify that 'dut_sbu_voltage_float_fault'
// has the value 'on'.
func servoCR50EnumeratedExec(ctx context.Context, info *execs.ExecInfo) error {
	sbuValue, err := MaximalAvgSbuValue(ctx, info.NewServod(), sbuVoltageTotalCheckCount)
	if err != nil {
		return errors.Annotate(err, "servo CR50 enumerated exec").Err()
	}
	log.Debugf(ctx, "Servo CR50 Enumerated Exec: avg SBU value is %f", sbuValue)
	if sbuValue > sbuThreshold {
		return errors.Reason("servo CR50 enumerated exec: CR50 SBU voltage is greater than the threshold").Err()
	}
	return nil
}

func init() {
	execs.Register("servo_cr50_low_sbu", servoCR50LowSBUExec)
	execs.Register("servo_cr50_enumerated", servoCR50EnumeratedExec)
}
