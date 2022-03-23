// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package battery

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"time"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/execs/cros/power"
	"infra/cros/recovery/internal/log"
	"infra/cros/recovery/tlw"

	"go.chromium.org/luci/common/errors"
)

// batteryInfo struct holds info regarding the battery of the DUT.
type batteryInfo struct {
	// path of the battery file.
	DeviceDirPath string
	// full capacity of the battery now.
	FullChargeCapacity float64
	// designed full capcaity of the battery.
	FullChargeCapacityDesigned float64
	// battery hardware state determined by the battery info.
	ChargeCycleCount float64
}

const (
	// name of the file that contains the ChargeFull information regarding DUT's battery.
	fullChargeFileName = "charge_full"
	// name of the file that contains the ChargeFullDesign information regarding DUT's battery.
	fullChargeCapacityDesignedFileName = "charge_full_design"
	// name of the file that contains the cycle count information regarding DUT's battery.
	chargeCycleCountFileName = "cycle_count"
)

// ReadBatteryInfo reads battery related information from the powerSupplyInfo struct in the power package
// and assign value to each of the field for the batteryInfo struct.
// returns an error when it cannot read related battery information.
func ReadBatteryInfo(ctx context.Context, r execs.Runner) (*batteryInfo, error) {
	powerSupplyInfo, err := power.ReadPowerInfo(ctx, r)
	if err != nil {
		return nil, errors.Annotate(err, "read battery info").Err()
	}
	b := &batteryInfo{}
	if b.DeviceDirPath, err = powerSupplyInfo.ReadBatteryPath(); err != nil {
		return nil, errors.Annotate(err, "read battery info").Err()
	}
	if b.DeviceDirPath == "" {
		log.Infof(ctx, "Battery path is not present")
		return nil, errors.Reason("read battery info: battery file path not present").Err()
	}
	log.Debugf(ctx, "Battery path: %s", b.DeviceDirPath)
	if b.FullChargeCapacity, err = b.readFile(ctx, r, fullChargeFileName); err != nil {
		return nil, errors.Annotate(err, "read battery info").Err()
	}
	if b.FullChargeCapacityDesigned, err = b.readFile(ctx, r, fullChargeCapacityDesignedFileName); err != nil {
		return nil, errors.Annotate(err, "read battery info").Err()
	}
	if b.ChargeCycleCount, err = b.readFile(ctx, r, chargeCycleCountFileName); err != nil {
		log.Errorf(ctx, err.Error())
	}
	log.Debugf(ctx, "Battery cycle_count %v", b.ChargeCycleCount)
	return b, nil
}

// readFile read battery related value from the given file name.
func (b *batteryInfo) readFile(ctx context.Context, r execs.Runner, fileName string) (float64, error) {
	pathToRead := path.Join(b.DeviceDirPath, fileName)
	cmd := fmt.Sprintf("cat %s", pathToRead)
	output, err := r(ctx, time.Minute, cmd)
	if err != nil {
		return -1, errors.Annotate(err, "read file: %s", fileName).Err()
	}
	outputValue, err := strconv.ParseFloat(output, 64)
	if err != nil {
		return -1, errors.Annotate(err, "read file: %s", fileName).Err()
	}
	return outputValue, nil
}

const (
	// Battery's minimum level to set the HardwareState to be Normal.
	auditCapacityNormalLevel = 70
	// Battery's minimum level to set the HardwareState to be Acceptable.
	auditCapacityAcceptableLevel = 40
)

// DetermineHardwareStatus determines the battery hardwareState
// based on the charging capacity of the DUT's battery.
//
// The logic for determining hardware state based on:
//   if capacity >= 70% then NORMAL
//   if capacity >= 40% then ACCEPTABLE
//   if capacity  < 40% then NEED_REPLACEMENT
func DetermineHardwareStatus(ctx context.Context, fullChargeCapacity float64, fullChargeCapacityDesigned float64) tlw.HardwareState {
	if fullChargeCapacity == 0 {
		log.Debugf(ctx, "charge_full is 0. Skip update battery_state!")
		return tlw.HardwareStateUnspecified
	}
	if fullChargeCapacityDesigned == 0 {
		log.Debugf(ctx, "charge_full_design is 0. Skip update battery_state!")
		return tlw.HardwareStateUnspecified
	}
	capacity := (100 * fullChargeCapacity) / fullChargeCapacityDesigned
	log.Infof(ctx, "battery capacity: %.2f%%", capacity)
	switch {
	case capacity >= auditCapacityNormalLevel:
		return tlw.HardwareStateNormal
	case capacity >= auditCapacityAcceptableLevel:
		return tlw.HardwareStateAcceptable
	default:
		return tlw.HardwareStateNeedReplacement
	}
}
