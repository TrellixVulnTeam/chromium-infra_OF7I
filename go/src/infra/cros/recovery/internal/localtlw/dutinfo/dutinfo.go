// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package dutinfo provides help function to work with DUT info.
package dutinfo

import (
	"fmt"
	"runtime/debug"
	"strings"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/tlw"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsdevice "infra/unifiedfleet/api/v1/models/chromeos/device"
	ufslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
)

// ConvertDut converts USF data to local representation of Dut instance.
func ConvertDut(data *ufspb.ChromeOSDeviceData) (dut *tlw.Dut, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Reason("convert dut: %v\n%s", r, debug.Stack()).Err()
		}
	}()
	if data.GetLabConfig().GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		return adaptUfsDutToTLWDut(data)
	} else if data.GetLabConfig().GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		return adaptUfsLabstationToTLWDut(data)
	}
	return nil, errors.Reason("convert dut: unexpected case!").Err()
}

// GenerateServodParams generates servod command based on device info.
// Expected output parameters for servod:
//  "BOARD=${VALUE}" - name of DUT board.
//  "MODEL=${VALUE}" - name of DUT model.
//  "PORT=${VALUE}" - port specified to run servod on servo-host.
//  "SERIAL=${VALUE}" - serial number of root servo.
//  "CONFIG=cr50.xml" - special parameter, for extra ability of CR50.
//  "REC_MODE=1" - start servod in recovery-mode, if root device found then servod will start event not all components detected.
func GenerateServodParams(data *ufspb.ChromeOSDeviceData, o *tlw.ServodOptions) (cmd []string, err error) {
	lc := data.GetLabConfig()
	name := lc.GetName()
	dut := lc.GetChromeosMachineLse().GetDeviceLse().GetDut()
	if dut == nil {
		return nil, errors.Reason("get servod params for %q: device is not DUT", name).Err()
	}
	var parts []string
	machine := data.GetMachine()
	if board := machine.GetChromeosMachine().GetBuildTarget(); board != "" {
		parts = append(parts, fmt.Sprintf("BOARD=%s", board))
		if model := machine.GetChromeosMachine().GetModel(); model != "" {
			parts = append(parts, fmt.Sprintf("MODEL=%s", model))
		}
	}
	servo := dut.GetPeripherals().GetServo()
	if servo == nil {
		return nil, errors.Reason("get servod params for %q: servo is not specified by device", name).Err()
	}
	parts = append(parts, fmt.Sprintf("PORT=%d", servo.GetServoPort()))

	if serial := servo.GetServoSerial(); serial != "" {
		parts = append(parts, fmt.Sprintf("SERIAL=%s", serial))
	}
	if setup := servo.GetServoSetup(); setup == ufslab.ServoSetupType_SERVO_SETUP_DUAL_V4 {
		parts = append(parts, "DUAL_V4=1")
	}
	if pools := dut.GetPools(); len(pools) > 0 {
		var hasCR50Pool bool
		for _, p := range pools {
			hasCR50Pool = hasCR50Pool || strings.Contains(p, "faft-cr50")
		}
		if hasCR50Pool {
			parts = append(parts, "CONFIG=cr50.xml")
		}
	}
	if o != nil && o.RecoveryMode {
		parts = append(parts, "REC_MODE=1")
	}
	return parts, nil
}

func adaptUfsDutToTLWDut(data *ufspb.ChromeOSDeviceData) (*tlw.Dut, error) {
	lc := data.GetLabConfig()
	p := lc.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
	ds := data.GetDutState()
	dc := data.GetDeviceConfig()
	machine := data.GetMachine()
	name := lc.GetName()
	var battery *tlw.DutBattery
	supplyType := tlw.PowerSupplyTypeUnspecified
	if dc != nil {
		switch dc.GetPower() {
		case ufsdevice.Config_POWER_SUPPLY_BATTERY:
			supplyType = tlw.PowerSupplyTypeBattery
			battery = &tlw.DutBattery{
				State: convertHardwareState(ds.GetBatteryState()),
			}
		case ufsdevice.Config_POWER_SUPPLY_AC_ONLY:
			supplyType = tlw.PowerSupplyTypeACOnly
		}
	}
	setup := tlw.DUTSetupTypeDefault
	if strings.Contains(name, "jetstream") {
		setup = tlw.DUTSetupTypeJetstream
	}
	d := &tlw.Dut{
		Name:            name,
		Board:           machine.GetChromeosMachine().GetBuildTarget(),
		Model:           machine.GetChromeosMachine().GetModel(),
		Hwid:            machine.GetChromeosMachine().GetHwid(),
		SerialNumber:    machine.GetSerialNumber(),
		SetupType:       setup,
		PowerSupplyType: supplyType,
		Storage:         createDUTStorage(dc, ds),
		Battery:         battery,
		ServoHost:       createServoHost(p, ds),
		RPMOutlet:       createRPMOutlet(p.GetRpm(), ds),
	}
	return d, nil
}

func adaptUfsLabstationToTLWDut(data *ufspb.ChromeOSDeviceData) (*tlw.Dut, error) {
	lc := data.GetLabConfig()
	l := lc.GetChromeosMachineLse().GetDeviceLse().GetLabstation()
	ds := data.GetDutState()
	dc := data.GetDeviceConfig()
	machine := data.GetMachine()
	name := lc.GetName()
	d := &tlw.Dut{
		Name:            name,
		Board:           machine.GetChromeosMachine().GetBuildTarget(),
		Model:           machine.GetChromeosMachine().GetModel(),
		Hwid:            machine.GetChromeosMachine().GetHwid(),
		SerialNumber:    machine.GetSerialNumber(),
		SetupType:       tlw.DUTSetupTypeLabstation,
		PowerSupplyType: tlw.PowerSupplyTypeACOnly,
		Storage:         createDUTStorage(dc, ds),
		RPMOutlet:       createRPMOutlet(l.GetRpm(), ds),
	}
	return d, nil
}

func createRPMOutlet(rpm *ufslab.OSRPM, ds *ufslab.DutState) *tlw.RPMOutlet {
	if rpm == nil || rpm.GetPowerunitName() == "" || rpm.GetPowerunitOutlet() == "" {
		return nil
	}
	return &tlw.RPMOutlet{
		Name:  fmt.Sprintf("%s|%s", rpm.GetPowerunitName(), rpm.GetPowerunitOutlet()),
		State: convertRPMState(ds.GetRpmState()),
	}
}

func createServoHost(p *ufslab.Peripherals, ds *ufslab.DutState) *tlw.ServoHost {
	if p.GetServo().GetServoHostname() == "" {
		return nil
	}
	return &tlw.ServoHost{
		Name:        p.GetServo().GetServoHostname(),
		UsbkeyState: convertHardwareState(ds.GetServoUsbState()),
		ServodPort:  int(p.GetServo().GetServoPort()),
		Servo: &tlw.Servo{
			State:           convertServoState(ds.GetServo()),
			SerialNumber:    p.GetServo().GetServoSerial(),
			FirmwareChannel: convertFirmwareChannel(p.GetServo().GetServoFwChannel()),
			Type:            p.GetServo().GetServoType(),
		},
		SmartUsbhubPresent: p.GetSmartUsbhub(),
	}
}

func createDUTStorage(dc *ufsdevice.Config, ds *ufslab.DutState) *tlw.DutStorage {
	return &tlw.DutStorage{
		Type:  convertStorageType(dc.GetStorage()),
		State: convertHardwareState(ds.GetStorageState()),
	}
}
