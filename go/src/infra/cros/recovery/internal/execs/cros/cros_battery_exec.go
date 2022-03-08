// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"
	"time"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/execs/cros/power"
	"infra/cros/recovery/internal/retry"
)

// isBatteryCanChangeToExpectedLevelExec verifies that battery status.
//
// If DUT battery still in the factory mode then DUT required re-work.
func isBatteryCanChangeToExpectedLevelExec(ctx context.Context, info *execs.ExecInfo) error {
	d := info.RunArgs.DUT
	log := info.NewLogger()
	run := info.NewRunner(d.Name)
	argsMap := info.GetActionArgs(ctx)
	log.Info("Started to verify battery status")
	pi, err := power.ReadPowerInfo(ctx, run)
	if err != nil {
		return errors.Annotate(err, "battery can charge to expected level").Err()
	}
	if ok, err := pi.IsBatterySupportedState(); err != nil {
		return errors.Annotate(err, "battery can charge to expected level").Err()
	} else if !ok {
		log.Info("Unexpected battery status. Please verify that DUT prepared for deployment.")
		return errors.Reason("battery can charge to expected level: Unexpected battery status.").Err()
	}
	batteryExpectedLevel := argsMap.AsFloat64(ctx, "battery_expected_level", float64(minimumBatteryLevel))
	batteryChargingPerRetry := argsMap.AsFloat64(ctx, "battery_charge_per_retry", 4.0)
	var lastChargedLevel float64
	// help function to check that battery level reached expected level.
	reachedExpectedLevel := func() error {
		pi, err := power.ReadPowerInfo(ctx, run)
		if err != nil {
			return errors.Annotate(err, "reached expected level").Err()
		}
		bl, err := pi.BatteryLevel()
		if err != nil {
			return errors.Annotate(err, "reached expected level").Err()
		}
		log.Debug("Battery level: %f%%", bl)
		if bl >= batteryExpectedLevel {
			log.Debug("Battery reached expected level %f%%", batteryExpectedLevel)
			return nil
		}
		if lastChargedLevel > 0 && bl-lastChargedLevel < batteryChargingPerRetry {
			// Breaking the loop as battery is not charging.
			log.Debug("Battery is not charged or discharging. Please verify that DUT connected to power and charging.")
			log.Debug("Possible that the DUT is not ready for deployment in lab.")
			return errors.Reason("reached expected level: charged %f%% when expected %f%%", bl-lastChargedLevel, batteryChargingPerRetry).Tag(retry.LoopBreakTag()).Err()
		}
		lastChargedLevel = bl
		return errors.Reason("reached expected level: the %v%% lower expected %v%% level", bl, batteryExpectedLevel).Err()
	}
	// Verify battery level to avoid cases when DUT in factory mode which can
	// block battery from charging. Retry check will take 8 attempts by
	// 15 minutes to allow battery to reach required level.
	retryCount := argsMap.AsInt(ctx, "charge_retry_count", 8)
	retryInnterval := argsMap.AsDuration(ctx, "charge_retry_interval", 900, time.Second)
	if err := retry.LimitCount(ctx, retryCount, retryInnterval, reachedExpectedLevel, "check bettry level"); err != nil {
		return errors.Annotate(err, "battery can charge to expected level").Err()
	}
	log.Info("Battery status verification passed!")
	return nil
}

func init() {
	execs.Register("cros_battery_changable_to_expected_level", isBatteryCanChangeToExpectedLevelExec)
}
