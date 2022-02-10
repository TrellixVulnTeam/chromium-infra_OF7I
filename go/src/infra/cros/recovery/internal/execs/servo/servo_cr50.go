// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"math"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/log"

	"go.chromium.org/luci/common/errors"
)

const (
	// The minimum SBU voltage required to successfully detect
	// usb-device.
	sbuThreshold = 2500.0

	// This represents the servod control 'SBU1 voltage level'.
	servoDutSbu1Cmd = "servo_dut_sbu1_mv"

	// This represents the servod control 'SBU2 voltage level'.
	servoDutSbu2Cmd = "servo_dut_sbu2_mv"
)

// AverageSbuValue determines the average SBU voltage for the servod
// control specified in the parameter.
func AverageSbuValue(ctx context.Context, args *execs.RunArgs, sbuControl string, checkCount int) (float64, error) {
	var sbuVal float64
	if checkCount <= 0 {
		return 0.0, errors.Reason("average sbu value: incorrect checkCount %d, it needs to be greater than 0.", checkCount).Err()
	}
	for i := 0; i < checkCount; i++ {
		if val, err := servodGetDouble(ctx, args, sbuControl); err != nil {
			return 0.0, errors.Annotate(err, "average sbu value").Err()
		} else {
			sbuVal += val
		}
	}
	if sbuVal <= 0 {
		return 0.0, errors.Reason("average sbu value: incorrect sbuVal %f, it needs to be greater than 0.", sbuVal).Err()
	}
	return sbuVal / float64(checkCount), nil
}

// MaximalAvgSbuValue determines the larger of the average SBU
// voltages for the controls 'servo_dut_sbu1_mv' and
// 'servo_dut_sbu2_mv'.
func MaximalAvgSbuValue(ctx context.Context, args *execs.RunArgs, checkCount int) (float64, error) {
	if err := args.NewServod().Has(ctx, servoDutSbu1Cmd); err != nil {
		log.Debug(ctx, "Maximal Average Sbu Value: control %q is not supported, returning -1", servoDutSbu1Cmd)
		return -1, errors.Annotate(err, "maximal avg sbu value").Err()
	}
	s1, err := AverageSbuValue(ctx, args, servoDutSbu1Cmd, checkCount)
	if err != nil {
		return 0.0, errors.Annotate(err, "maximal average sbu value").Err()
	}
	s2, err := AverageSbuValue(ctx, args, servoDutSbu2Cmd, checkCount)
	if err != nil {
		return 0.0, errors.Annotate(err, "maximal average sbu value").Err()
	}
	maxVal := math.Max(s1, s2)
	log.Debug(ctx, "Maximal Average Sbu Value: the max SBU voltage value is :%f", maxVal)
	return maxVal, nil
}
