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

	"infra/cros/dutstate"
	"infra/cros/recovery/tlw"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsdevice "infra/unifiedfleet/api/v1/models/chromeos/device"
	ufslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsmake "infra/unifiedfleet/api/v1/models/chromeos/manufacturing"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
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

// CreateUpdateDutRequest creates request instance to update UFS.
func CreateUpdateDutRequest(dutID string, dut *tlw.Dut) (req *ufsAPI.UpdateDutStateRequest, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Reason("update dut specs: %v\n%s", r, debug.Stack()).Err()
		}
	}()
	return &ufsAPI.UpdateDutStateRequest{
		DutState: getUFSDutComponentStateFromSpecs(dutID, dut),
		DutMeta:  getUFSDutMetaFromSpecs(dutID, dut),
		LabMeta:  getUFSLabMetaFromSpecs(dutID, dut),
	}, nil
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

// GetRpmInfo provides hostname and outlet of the device.
func GetRpmInfo(data *ufspb.ChromeOSDeviceData) (hostname, outlet string) {
	dle := data.GetLabConfig().GetChromeosMachineLse().GetDeviceLse()
	var rpm *ufslab.OSRPM
	if dut := dle.GetDut(); dut != nil {
		rpm = dle.GetDut().GetPeripherals().GetRpm()
	} else if l := dle.GetLabstation(); l != nil {
		rpm = l.GetRpm()
	}
	if rpm != nil {
		return rpm.GetPowerunitName(), rpm.GetPowerunitOutlet()
	}
	return "", ""
}

func adaptUfsDutToTLWDut(data *ufspb.ChromeOSDeviceData) (*tlw.Dut, error) {
	lc := data.GetLabConfig()
	p := lc.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals()
	ds := data.GetDutState()
	dc := data.GetDeviceConfig()
	machine := data.GetMachine()
	name := lc.GetName()
	var battery *tlw.DUTBattery
	supplyType := tlw.PowerSupplyTypeUnspecified
	if dc != nil {
		switch dc.GetPower() {
		case ufsdevice.Config_POWER_SUPPLY_BATTERY:
			supplyType = tlw.PowerSupplyTypeBattery
			battery = &tlw.DUTBattery{
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
	// Generate list of peers we can have with states.
	// This line of change to avoid creating extra field to drive count and
	// then remove when peripherals will start to be supported.
	// TODO(otabek@): replace when peripherals when supported.
	var bluetoothPeerHosts []*tlw.BluetoothPeerHost
	goodPeerCount := ds.GetWorkingBluetoothBtpeer()
	for i := 1; i <= 4; i++ {
		state := tlw.BluetoothPeerStateUnspecified
		if i <= int(goodPeerCount) {
			// As e do not have data which peer is good we set state for
			// the first peers. Later when peripherals will be supported by UFS
			// we can reeive proper information.
			state = tlw.BluetoothPeerStateWorking
		}
		bluetoothPeerHosts = append(bluetoothPeerHosts, &tlw.BluetoothPeerHost{
			Name:  fmt.Sprintf("%s-btpeer%d", name, i),
			State: state,
		})
	}

	d := &tlw.Dut{
		Name:               name,
		Board:              machine.GetChromeosMachine().GetBuildTarget(),
		Model:              machine.GetChromeosMachine().GetModel(),
		Hwid:               machine.GetChromeosMachine().GetHwid(),
		SerialNumber:       machine.GetSerialNumber(),
		SetupType:          setup,
		State:              dutstate.ConvertFromUFSState(lc.GetResourceState()),
		PowerSupplyType:    supplyType,
		Storage:            createDUTStorage(dc, ds),
		Wifi:               createDUTWifi(data.GetManufacturingConfig(), ds),
		Bluetooth:          createDUTBluetooth(ds),
		BluetoothPeerHosts: bluetoothPeerHosts,
		Battery:            battery,
		ServoHost:          createServoHost(p, ds),
		ChameleonHost:      createChameleonHost(name, ds),
		RPMOutlet:          createRPMOutlet(p.GetRpm(), ds),
		Cr50Phase:          convertCr50Phase(ds.GetCr50Phase()),
		Cr50KeyEnv:         convertCr50KeyEnv(ds.GetCr50KeyEnv()),
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
		Cr50Phase:       convertCr50Phase(ds.GetCr50Phase()),
		Cr50KeyEnv:      convertCr50KeyEnv(ds.GetCr50KeyEnv()),
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
		ServoTopology:      convertServoTopologyFromUFS(p.GetServo().GetServoTopology()),
	}
}

func createChameleonHost(dutName string, ds *ufslab.DutState) *tlw.ChameleonHost {
	return &tlw.ChameleonHost{
		Name:  fmt.Sprintf("%s-chameleon", dutName),
		State: convertChameleonState(ds.GetChameleon()),
	}
}

func createDUTStorage(dc *ufsdevice.Config, ds *ufslab.DutState) *tlw.DUTStorage {
	return &tlw.DUTStorage{
		Type:  convertStorageType(dc.GetStorage()),
		State: convertHardwareState(ds.GetStorageState()),
	}
}

func createDUTWifi(make *ufsmake.ManufacturingConfig, ds *ufslab.DutState) *tlw.DUTWifi {
	return &tlw.DUTWifi{
		State:    convertHardwareState(ds.GetWifiState()),
		ChipName: make.GetWifiChip(),
	}
}

func createDUTBluetooth(ds *ufslab.DutState) *tlw.DUTBluetooth {
	return &tlw.DUTBluetooth{
		State: convertHardwareState(ds.GetBluetoothState()),
	}
}

func getUFSDutMetaFromSpecs(dutID string, dut *tlw.Dut) *ufspb.DutMeta {
	dutMeta := &ufspb.DutMeta{
		ChromeosDeviceId: dutID,
		Hostname:         dut.Name,
	}
	if dut.SerialNumber != "" {
		dutMeta.SerialNumber = dut.SerialNumber
	}
	if dut.Hwid != "" {
		dutMeta.HwID = dut.Hwid
	}
	// blocked by b/184391605
	// dutMeta.DeviceSku = specs.GetLabels().GetSku()
	return dutMeta
}

func getUFSLabMetaFromSpecs(dutID string, dut *tlw.Dut) (labconfig *ufspb.LabMeta) {
	labMeta := &ufspb.LabMeta{
		ChromeosDeviceId: dutID,
		Hostname:         dut.Name,
	}
	if sh := dut.ServoHost; sh != nil {
		labMeta.ServoType = sh.Servo.Type
		labMeta.SmartUsbhub = sh.SmartUsbhubPresent
		labMeta.ServoTopology = convertServoTopologyToUFS(sh.ServoTopology)
	}
	return labMeta
}

// getUFSDutComponentStateFromSpecs collects all states for DUT and peripherals.
func getUFSDutComponentStateFromSpecs(dutID string, dut *tlw.Dut) *ufslab.DutState {
	state := &ufslab.DutState{
		Id:       &ufslab.ChromeOSDeviceID{Value: dutID},
		Hostname: dut.Name,
	}
	// Set all default state first and update later.
	// If component missing the this will reset the state.
	state.Servo = ufslab.PeripheralState_MISSING_CONFIG
	state.ServoUsbState = ufslab.HardwareState_HARDWARE_UNKNOWN
	state.RpmState = ufslab.PeripheralState_MISSING_CONFIG
	state.StorageState = ufslab.HardwareState_HARDWARE_UNKNOWN
	state.BatteryState = ufslab.HardwareState_HARDWARE_UNKNOWN
	state.WifiState = ufslab.HardwareState_HARDWARE_UNKNOWN
	state.BluetoothState = ufslab.HardwareState_HARDWARE_UNKNOWN

	// Update states for present components.
	if sh := dut.ServoHost; sh != nil {
		for us, ls := range servoStates {
			if ls == sh.Servo.State {
				state.Servo = us
			}
		}
		state.ServoUsbState = convertHardwareStateToUFS(sh.UsbkeyState)
	}
	if rpm := dut.RPMOutlet; rpm != nil {
		for us, ls := range rpmStates {
			if ls == rpm.State {
				state.RpmState = us
			}
		}
	}
	for us, ls := range cr50Phases {
		if ls == dut.Cr50Phase {
			state.Cr50Phase = us
		}
	}
	for us, ls := range cr50KeyEnvs {
		if ls == dut.Cr50KeyEnv {
			state.Cr50KeyEnv = us
		}
	}
	if s := dut.Storage; s != nil {
		state.StorageState = convertHardwareStateToUFS(s.State)
	}
	if b := dut.Battery; b != nil {
		state.BatteryState = convertHardwareStateToUFS(b.State)
	}
	if w := dut.Wifi; w != nil {
		state.WifiState = convertHardwareStateToUFS(w.State)
	}
	if b := dut.Bluetooth; b != nil {
		state.BluetoothState = convertHardwareStateToUFS(b.State)
	}
	if ch := dut.ChameleonHost; ch != nil {
		if ch.State == tlw.ChameleonStateWorking {
			state.Chameleon = ufslab.PeripheralState_WORKING
		}
	}
	state.WorkingBluetoothBtpeer = 0
	for _, btph := range dut.BluetoothPeerHosts {
		if btph.State == tlw.BluetoothPeerStateWorking {
			state.WorkingBluetoothBtpeer += 1
		}
	}

	// TODO(otabek@): get more data and convert.
	// if p.GetAudioLoopbackDongle() {
	// 	state.AudioLoopbackDongle = ufslab.PeripheralState_WORKING
	// }
	return state
}
