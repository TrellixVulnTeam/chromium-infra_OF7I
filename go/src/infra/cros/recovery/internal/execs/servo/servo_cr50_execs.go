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
func servoCR50LowSBUExec(ctx context.Context, args *execs.RunArgs, actionArgs []string) error {
	sbuValue, err := MaximalAvgSbuValue(ctx, args, sbuVoltageTotalCheckCount)
	if err != nil {
		return errors.Reason("servo CR50 low sbu exec: could not compute the average SBU voltage value.").Err()
	}
	log.Debug(ctx, "Servo CR50 Low Sbu Exec: avg SBU value is %f", sbuValue)
	if sbuValue <= sbuThreshold {
		return errors.Reason("servo CR50 low sbu exec: CR50 not detected due to low SBU voltage").Err()
	}
	return nil
}

func init() {
	execs.Register("servo_cr50_low_sbu", servoCR50LowSBUExec)
}
