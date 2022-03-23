// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"context"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/execs/cros/power"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"
)

// isACPowerConnectedExec confirms whether the DUT is connected through AC power.
func isACPowerConnectedExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	p, err := power.ReadPowerInfo(ctx, r)
	if err != nil {
		return errors.Annotate(err, "ac power connected").Err()
	}
	powerByAC, err := p.IsACOnline()
	if err != nil {
		return errors.Annotate(err, "ac power connected").Err()
	}
	if !powerByAC {
		return errors.Reason("ac power connected: is not plugged in").Err()
	}
	return nil
}

// isBatteryExpectedExec confirms whether the DUT is expected to have battery according to inventory.
func isBatteryExpectedExec(ctx context.Context, info *execs.ExecInfo) error {
	if info.RunArgs.DUT.PowerSupplyType != tlw.PowerSupplyTypeBattery {
		return errors.Reason("is battery expected: battery is not expected").Err()
	}
	return nil
}

// isBatteryPresentExec confirms that the DUT has battery.
func isBatteryPresentExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	p, err := power.ReadPowerInfo(ctx, r)
	if err != nil {
		return errors.Annotate(err, "battery present").Err()
	}
	hasBattery, err := p.HasBattery()
	if err != nil {
		return errors.Annotate(err, "battery present").Err()
	}
	if !hasBattery {
		return errors.Reason("battery present: battery is not presented.").Err()
	}
	return nil
}

// isBatteryLevelGreaterThanMinimumExec confirms the battery has enough charge.
func isBatteryLevelGreaterThanMinimumExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	p, err := power.ReadPowerInfo(ctx, r)
	if err != nil {
		return errors.Annotate(err, "battery has enough charge").Err()
	}
	currentLevel, err := p.BatteryLevel()
	if err != nil {
		return errors.Annotate(err, "battery has enough charge").Err()
	}
	if currentLevel < MinimumBatteryLevel {
		return errors.Reason("battery has enough charge: battery's current level is less than the minimum level: %d", MinimumBatteryLevel).Err()
	}
	return nil
}

// isBatteryChargingExec confirms the battery is charging.
func isBatteryChargingExec(ctx context.Context, info *execs.ExecInfo) error {
	r := info.DefaultRunner()
	p, err := power.ReadPowerInfo(ctx, r)
	if err != nil {
		return errors.Annotate(err, "battery charging").Err()
	}
	isDischarging, err := p.IsBatteryDischarging()
	if err != nil {
		return errors.Annotate(err, "battery charging").Err()
	}
	if isDischarging {
		return errors.Reason("battery charging: battery is in discharging state").Err()
	}
	return nil
}

// isBatteryChargableOrGoodLevelExec confirms the battery either has enough charge or is charging.
func isBatteryChargableOrGoodLevelExec(ctx context.Context, info *execs.ExecInfo) error {
	batteryLevelError, batteryChargingError := isBatteryLevelGreaterThanMinimumExec(ctx, info), isBatteryChargingExec(ctx, info)
	if batteryLevelError != nil && batteryChargingError != nil {
		multiBatteryError := errors.NewMultiError(batteryLevelError, batteryChargingError)
		// Log both of batteryLevelError and batteryChargingError
		log.Errorf(ctx, multiBatteryError[0].Error()+" and "+multiBatteryError[1].Error())
		return errors.Annotate(multiBatteryError, "battery chargable or good level: battery does not have enough charge and in discharging state").Err()
	}
	if batteryLevelError != nil {
		log.Errorf(ctx, batteryLevelError.Error())
	}
	if batteryChargingError != nil {
		log.Errorf(ctx, batteryChargingError.Error())
	}
	return nil
}

func init() {
	execs.Register("cros_is_ac_power_connected", isACPowerConnectedExec)
	execs.Register("cros_is_battery_expected", isBatteryExpectedExec)
	execs.Register("cros_is_battery_present", isBatteryPresentExec)
	execs.Register("cros_is_battery_level_greater_than_minimum", isBatteryLevelGreaterThanMinimumExec)
	execs.Register("cros_is_battery_charging", isBatteryChargingExec)
	execs.Register("cros_is_battery_chargable_or_good_level", isBatteryChargableOrGoodLevelExec)
}
