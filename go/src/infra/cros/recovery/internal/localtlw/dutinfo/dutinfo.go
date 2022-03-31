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
	// TODO(otabek@): Add logic to read and update state file on the drones. (ProvisionedInfo)
	if data.GetLabConfig().GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		return adaptUfsDutToTLWDut(data)
	} else if data.GetLabConfig().GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		return adaptUfsLabstationToTLWDut(data)
	}
	return nil, errors.Reason("convert dut: unexpected case!").Err()
}

// ConvertAttachedDeviceToTlw converts USF data to local representation of Dut instance.
func ConvertAttachedDeviceToTlw(data *ufsAPI.AttachedDeviceData) (dut *tlw.Dut, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Reason("convert dut: %v\n%s", r, debug.Stack()).Err()
		}
	}()
	machine := data.GetMachine()
	machineLSE := data.GetLabConfig()
	if machine == nil || machineLSE == nil {
		return nil, errors.Reason("convert attached device to tlw: unexpected case!").Err()
	}
	// Determine type of device.
	setup := tlw.DUTSetupTypeUnspecified
	switch machine.GetAttachedDevice().GetDeviceType() {
	case ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_ANDROID_PHONE,
		ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_ANDROID_TABLET:
		setup = tlw.DUTSetupTypeAndroid
	case ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_APPLE_PHONE,
		ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_APPLE_TABLET:
		setup = tlw.DUTSetupTypeIOS
	default:
		setup = tlw.DUTSetupTypeUnspecified
	}
	return &tlw.Dut{
		Id:              machine.GetName(),
		Name:            machineLSE.GetHostname(),
		Board:           machine.GetAttachedDevice().GetBuildTarget(),
		Model:           machine.GetAttachedDevice().GetModel(),
		SerialNumber:    machine.GetSerialNumber(),
		SetupType:       setup,
		State:           dutstate.ConvertFromUFSState(machineLSE.GetResourceState()),
		ExtraAttributes: map[string][]string{
			// Attached device does not have pools. Need read from Scheduling unit.
			// tlw.ExtraAttributePools: pools,
		},
		ProvisionedInfo: &tlw.DUTProvisionedInfo{},
	}, nil
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

func adaptUfsDutToTLWDut(data *ufspb.ChromeOSDeviceData) (*tlw.Dut, error) {
	lc := data.GetLabConfig()
	dut := lc.GetChromeosMachineLse().GetDeviceLse().GetDut()
	p := dut.GetPeripherals()
	ds := data.GetDutState()
	dc := data.GetDeviceConfig()
	machine := data.GetMachine()
	make := data.GetManufacturingConfig()
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
	setup := tlw.DUTSetupTypeCros
	if strings.Contains(name, "jetstream") {
		setup = tlw.DUTSetupTypeJetstream
	}

	d := &tlw.Dut{
		Id:                  machine.GetName(),
		Name:                name,
		Board:               machine.GetChromeosMachine().GetBuildTarget(),
		Model:               machine.GetChromeosMachine().GetModel(),
		Hwid:                machine.GetChromeosMachine().GetHwid(),
		Phase:               make.GetDevicePhase().String()[len("PHASE_"):],
		SerialNumber:        machine.GetSerialNumber(),
		SetupType:           setup,
		State:               dutstate.ConvertFromUFSState(lc.GetResourceState()),
		PowerSupplyType:     supplyType,
		Storage:             createDUTStorage(dc, ds),
		Wifi:                createDUTWifi(make, ds),
		WifiRouterHosts:     createWifiRouterHosts(p.GetWifi()),
		PeripheralWifiState: convertPeripheralWifiState(ds.GetWifiPeripheralState()),
		Bluetooth:           createDUTBluetooth(ds, dc),
		BluetoothPeerHosts:  createBluetoothPeerHosts(p),
		Battery:             battery,
		ServoHost:           createServoHost(p, ds),
		ChameleonHost:       createChameleonHost(name, ds),
		RPMOutlet:           createRPMOutlet(p.GetRpm(), ds),
		Cr50Phase:           convertCr50Phase(ds.GetCr50Phase()),
		Cr50KeyEnv:          convertCr50KeyEnv(ds.GetCr50KeyEnv()),
		Audio: &tlw.DUTAudio{
			LoopbackState: convertAudioLoopbackState(ds.GetAudioLoopbackDongle()),
		},
		DeviceSku: machine.GetChromeosMachine().GetSku(),
		ExtraAttributes: map[string][]string{
			tlw.ExtraAttributePools: dut.GetPools(),
		},
		ProvisionedInfo: &tlw.DUTProvisionedInfo{},
	}
	if audio := p.GetAudio(); audio != nil {
		d.Audio.InBox = audio.AudioBox
		d.Audio.StaticCable = audio.AudioCable
	}
	if p.GetServo().GetServoSetup() == ufslab.ServoSetupType_SERVO_SETUP_DUAL_V4 {
		d.ExtraAttributes[tlw.ExtraAttributeServoSetup] = []string{tlw.ExtraAttributeServoSetupDual}
	}
	return d, nil
}

// createBluetoothPeerHosts use the UFS states for Bluetooth peer devices to create
// the equivalent tlw slice.
func createBluetoothPeerHosts(peripherals *ufslab.Peripherals) []*tlw.BluetoothPeerHost {
	var bluetoothPeerHosts []*tlw.BluetoothPeerHost
	for _, btp := range peripherals.GetBluetoothPeers() {
		var (
			hostname string
			state    tlw.BluetoothPeerState
		)
		switch d := btp.GetDevice().(type) {
		case *ufslab.BluetoothPeer_RaspberryPi:
			hostname = d.RaspberryPi.GetHostname()
			state = convertBluetoothPeerState(d.RaspberryPi.GetState())
		default:
			// We never want this to fail. It does create a risk
			// for silent errors however. Introduction of new device
			// types is very infrequent and also a very conscious
			// event, which helps counterweight that risk.
			continue
		}
		bluetoothPeerHosts = append(bluetoothPeerHosts, &tlw.BluetoothPeerHost{
			Name:  hostname,
			State: state,
		})
	}

	return bluetoothPeerHosts
}

func adaptUfsLabstationToTLWDut(data *ufspb.ChromeOSDeviceData) (*tlw.Dut, error) {
	lc := data.GetLabConfig()
	l := lc.GetChromeosMachineLse().GetDeviceLse().GetLabstation()
	ds := data.GetDutState()
	dc := data.GetDeviceConfig()
	machine := data.GetMachine()
	make := data.GetManufacturingConfig()
	name := lc.GetName()
	d := &tlw.Dut{
		Id:              machine.GetName(),
		Name:            name,
		Board:           machine.GetChromeosMachine().GetBuildTarget(),
		Model:           machine.GetChromeosMachine().GetModel(),
		Hwid:            machine.GetChromeosMachine().GetHwid(),
		Phase:           make.GetDevicePhase().String()[len("PHASE_"):],
		SerialNumber:    machine.GetSerialNumber(),
		SetupType:       tlw.DUTSetupTypeLabstation,
		PowerSupplyType: tlw.PowerSupplyTypeACOnly,
		Storage:         createDUTStorage(dc, ds),
		RPMOutlet:       createRPMOutlet(l.GetRpm(), ds),
		Cr50Phase:       convertCr50Phase(ds.GetCr50Phase()),
		Cr50KeyEnv:      convertCr50KeyEnv(ds.GetCr50KeyEnv()),
		DeviceSku:       machine.GetChromeosMachine().GetSku(),
		ExtraAttributes: map[string][]string{
			tlw.ExtraAttributePools: l.GetPools(),
		},
		ProvisionedInfo: &tlw.DUTProvisionedInfo{},
	}
	return d, nil
}

func createRPMOutlet(rpm *ufslab.OSRPM, ds *ufslab.DutState) *tlw.RPMOutlet {
	if rpm == nil || rpm.GetPowerunitName() == "" || rpm.GetPowerunitOutlet() == "" {
		return &tlw.RPMOutlet{
			State: convertRPMState(ds.GetRpmState()),
		}
	}
	return &tlw.RPMOutlet{
		Hostname: rpm.GetPowerunitName(),
		Outlet:   rpm.GetPowerunitOutlet(),
		State:    convertRPMState(ds.GetRpmState()),
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
		ContainerName:      p.GetServo().GetDockerContainerName(),
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

// createWifiRouterHosts convert ufslab.Wifi.WifiRouters to []*tlw.WifiRouterHost
// It include router hostname, model, board, state, rpm information which will be used to verification and recovery
func createWifiRouterHosts(wifi *ufslab.Wifi) []*tlw.WifiRouterHost {
	var routers []*tlw.WifiRouterHost
	for _, ufsRouter := range wifi.GetWifiRouters() {
		tlwRpm := tlw.RPMOutlet{
			// TODO update when http://b/216315183 is done.
			//set to unknown till rpm is updated to enable peripherals.
			//currently,rpm only supports on dut. router rpm state is not defined in proto yet and no api for rpmoutlet for non dut
			State: convertRPMState(ufslab.PeripheralState_UNKNOWN),
		}
		if rpm := ufsRouter.GetRpm(); rpm != nil && rpm.GetPowerunitName() != "" && rpm.GetPowerunitOutlet() != "" {
			tlwRpm.Hostname = rpm.GetPowerunitName()
			tlwRpm.Outlet = rpm.GetPowerunitOutlet()
		}
		routers = append(routers, &tlw.WifiRouterHost{
			Name:      ufsRouter.GetHostname(),
			State:     convertWifiRouterState(ufsRouter.GetState()),
			Model:     ufsRouter.GetModel(),
			Board:     ufsRouter.GetBuildTarget(),
			RPMOutlet: &tlwRpm,
		})
	}
	return routers
}

func createDUTBluetooth(ds *ufslab.DutState, dc *ufsdevice.Config) *tlw.DUTBluetooth {
	return &tlw.DUTBluetooth{
		Expected: configHasFeature(dc, ufsdevice.Config_HARDWARE_FEATURE_BLUETOOTH),
		State:    convertHardwareState(ds.GetBluetoothState()),
	}
}

func configHasFeature(dc *ufsdevice.Config, hf ufsdevice.Config_HardwareFeature) bool {
	for _, f := range dc.GetHardwareFeatures() {
		if f == hf {
			return true
		}
	}
	return false
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
	// TODO: update logic if required by b/184391605
	dutMeta.DeviceSku = dut.DeviceSku
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
	state.Chameleon = ufslab.PeripheralState_UNKNOWN
	state.WorkingBluetoothBtpeer = 0

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
			if ls == rpm.GetState() {
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
		for us, rs := range chameleonStates {
			if ch.State == rs {
				state.Chameleon = us
			}
		}
	}
	for _, btph := range dut.BluetoothPeerHosts {
		if btph.State == tlw.BluetoothPeerStateWorking {
			state.WorkingBluetoothBtpeer += 1
		}
	}
	if dut.Audio != nil && dut.Audio.GetLoopbackState() == tlw.DUTAudio_LOOPBACK_WORKING {
		state.AudioLoopbackDongle = ufslab.PeripheralState_WORKING
	} else {
		state.AudioLoopbackDongle = ufslab.PeripheralState_UNKNOWN
	}
	return state
}
