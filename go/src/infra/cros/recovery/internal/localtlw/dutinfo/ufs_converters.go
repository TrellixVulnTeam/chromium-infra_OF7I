// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dutinfo

import (
	"infra/cros/recovery/tlw"
	ufsdevice "infra/unifiedfleet/api/v1/models/chromeos/device"
	ufslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
)

// TODO(otabek@): Use bidirectional maps when will be available.

var hardwareStates = map[ufslab.HardwareState]tlw.HardwareState{
	ufslab.HardwareState_HARDWARE_NORMAL:           tlw.HardwareStateNormal,
	ufslab.HardwareState_HARDWARE_ACCEPTABLE:       tlw.HardwareStateAcceptable,
	ufslab.HardwareState_HARDWARE_NEED_REPLACEMENT: tlw.HardwareStateNeedReplacement,
	ufslab.HardwareState_HARDWARE_NOT_DETECTED:     tlw.HardwareStateNotDetected,
}

func convertHardwareState(s ufslab.HardwareState) tlw.HardwareState {
	if ns, ok := hardwareStates[s]; ok {
		return ns
	}
	return tlw.HardwareStateUnspecified
}

func convertHardwareStateToUFS(s tlw.HardwareState) ufslab.HardwareState {
	for us, ls := range hardwareStates {
		if ls == s {
			return us
		}
	}
	return ufslab.HardwareState_HARDWARE_UNKNOWN
}

var firmwareChannels = map[ufslab.ServoFwChannel]tlw.ServoFirmwareChannel{
	ufslab.ServoFwChannel_SERVO_FW_STABLE: tlw.ServoFirmwareChannelStable,
	ufslab.ServoFwChannel_SERVO_FW_ALPHA:  tlw.ServoFirmwareChannelAlpha,
	ufslab.ServoFwChannel_SERVO_FW_DEV:    tlw.ServoFirmwareChannelDev,
	ufslab.ServoFwChannel_SERVO_FW_PREV:   tlw.ServoFirmwareChannelPrev,
}

func convertFirmwareChannel(s ufslab.ServoFwChannel) tlw.ServoFirmwareChannel {
	if ns, ok := firmwareChannels[s]; ok {
		return ns
	}
	return tlw.ServoFirmwareChannelStable
}

func convertStorageType(t ufsdevice.Config_Storage) tlw.StorageType {
	switch t {
	case ufsdevice.Config_STORAGE_SSD:
		return tlw.StorageTypeSSD
	case ufsdevice.Config_STORAGE_HDD:
		return tlw.StorageTypeHDD
	case ufsdevice.Config_STORAGE_MMC:
		return tlw.StorageTypeMMC
	case ufsdevice.Config_STORAGE_NVME:
		return tlw.StorageTypeNVME
	case ufsdevice.Config_STORAGE_UFS:
		return tlw.StorageTypeUFS
	default:
		return tlw.StorageTypeUnspecified
	}
}

func convertAudioLoopbackState(s ufslab.PeripheralState) tlw.DUTAudio_LoopbackState {
	if s == ufslab.PeripheralState_WORKING {
		return tlw.DUTAudio_LOOPBACK_WORKING
	}
	return tlw.DUTAudio_LOOPBACK_UNSPECIFIED
}

var servoStates = map[ufslab.PeripheralState]tlw.ServoState{
	ufslab.PeripheralState_WORKING:                       tlw.ServoStateWorking,
	ufslab.PeripheralState_MISSING_CONFIG:                tlw.ServoStateMissingConfig,
	ufslab.PeripheralState_WRONG_CONFIG:                  tlw.ServoStateWrongConfig,
	ufslab.PeripheralState_NOT_CONNECTED:                 tlw.ServoStateNotConnected,
	ufslab.PeripheralState_NO_SSH:                        tlw.ServoStateNoSSH,
	ufslab.PeripheralState_BROKEN:                        tlw.ServoStateBroken,
	ufslab.PeripheralState_NEED_REPLACEMENT:              tlw.ServoStateNeedReplacement,
	ufslab.PeripheralState_CR50_CONSOLE_MISSING:          tlw.ServoStateCr50ConsoleMissing,
	ufslab.PeripheralState_CCD_TESTLAB_ISSUE:             tlw.ServoStateCCDTestlabIssue,
	ufslab.PeripheralState_SERVOD_ISSUE:                  tlw.ServoStateServodIssue,
	ufslab.PeripheralState_LID_OPEN_FAILED:               tlw.ServoStateLidOpenIssue,
	ufslab.PeripheralState_BAD_RIBBON_CABLE:              tlw.ServoStateBadRibbonCable,
	ufslab.PeripheralState_EC_BROKEN:                     tlw.ServoStateECBroken,
	ufslab.PeripheralState_DUT_NOT_CONNECTED:             tlw.ServoStateDUTNotConnected,
	ufslab.PeripheralState_TOPOLOGY_ISSUE:                tlw.ServoStateTopologyIssue,
	ufslab.PeripheralState_SBU_LOW_VOLTAGE:               tlw.ServoStateSBULowVoltage,
	ufslab.PeripheralState_CR50_NOT_ENUMERATED:           tlw.ServoStateCr50NotEnumerated,
	ufslab.PeripheralState_SERVO_SERIAL_MISMATCH:         tlw.ServoStateServoSerialMismatch,
	ufslab.PeripheralState_SERVOD_PROXY_ISSUE:            tlw.ServoStateServodProxyIssue,
	ufslab.PeripheralState_SERVO_HOST_ISSUE:              tlw.ServoStateServoHostIssue,
	ufslab.PeripheralState_SERVO_UPDATER_ISSUE:           tlw.ServoStateServoUpdaterIssue,
	ufslab.PeripheralState_SERVOD_DUT_CONTROLLER_MISSING: tlw.ServoStateServodDutControllerMissing,
	ufslab.PeripheralState_COLD_RESET_PIN_ISSUE:          tlw.ServoStateColdResetPinIssue,
	ufslab.PeripheralState_WARM_RESET_PIN_ISSUE:          tlw.ServoStateWarmResetPinIssue,
	ufslab.PeripheralState_POWER_BUTTON_PIN_ISSUE:        tlw.ServoStatePowerButtonPinIssue,
}

func convertServoState(s ufslab.PeripheralState) tlw.ServoState {
	if ns, ok := servoStates[s]; ok {
		return ns
	}
	return tlw.ServoStateUnspecified
}

var chameleonStates = map[ufslab.PeripheralState]tlw.ChameleonState{
	ufslab.PeripheralState_WORKING: tlw.ChameleonStateWorking,
	ufslab.PeripheralState_BROKEN:  tlw.ChameleonStateBroken,
}

func convertChameleonState(s ufslab.PeripheralState) tlw.ChameleonState {
	if ns, ok := chameleonStates[s]; ok {
		return ns
	}
	return tlw.ChameleonStateUnspecified
}

var bluetoothPeerStates = map[ufslab.PeripheralState]tlw.BluetoothPeerState{
	ufslab.PeripheralState_WORKING: tlw.BluetoothPeerStateWorking,
	ufslab.PeripheralState_BROKEN:  tlw.BluetoothPeerStateBroken,
}

func convertBluetoothPeerState(s ufslab.PeripheralState) tlw.BluetoothPeerState {
	if ns, ok := bluetoothPeerStates[s]; ok {
		return ns
	}
	return tlw.BluetoothPeerStateUnspecified
}

func convertBluetoothPeerStateToUFS(s tlw.BluetoothPeerState) ufslab.PeripheralState {
	for ufsState, tlwState := range bluetoothPeerStates {
		if s == tlwState {
			return ufsState
		}
	}
	return ufslab.PeripheralState_UNKNOWN
}

// WifiRouterStates maps the router UFS state to TLW  state
// it is used to in convertWifiRouterState to convert ufs periperal state to tlw router state
var wifiRouterStates = map[ufslab.PeripheralState]tlw.WifiRouterHost_State{
	ufslab.PeripheralState_WORKING: tlw.WifiRouterHost_WORKING,
	ufslab.PeripheralState_BROKEN:  tlw.WifiRouterHost_BROKEN,
}

// converts WifiRouter UFS state to TLW state
func convertWifiRouterState(s ufslab.PeripheralState) tlw.WifiRouterHost_State {
	if ns, ok := wifiRouterStates[s]; ok {
		return ns
	}
	return tlw.WifiRouterHost_UNSPECIFIED
}

func convertWifiRouterStateToUFS(s tlw.WifiRouterHost_State) ufslab.PeripheralState {
	for us, ls := range wifiRouterStates {
		if ls == s {
			return us
		}
	}
	return ufslab.PeripheralState_UNKNOWN
}

// peripheralWifiStates maps the ufs peripheral state to tlw peripheral wifi state
var peripheralWifiStates = map[ufslab.PeripheralState]tlw.PeripheralWifiState{
	ufslab.PeripheralState_WORKING: tlw.PeripheralWifiStateWorking,
	ufslab.PeripheralState_BROKEN:  tlw.PeripheralWifiStateBroken,
}

// convert wifiRouterState UFS state to TLW peripheralWifiState
func convertPeripheralWifiState(s ufslab.PeripheralState) tlw.PeripheralWifiState {
	if ns, ok := peripheralWifiStates[s]; ok {
		return ns
	}
	return tlw.PeripheralWifiStateUnspecified
}

// convertPeripheralWifiState tlw state to UFS peripheral state
func convertPeripheralWifiStateToUFS(s tlw.PeripheralWifiState) ufslab.PeripheralState {
	for us, ls := range peripheralWifiStates {
		if ls == s {
			return us
		}
	}
	return ufslab.PeripheralState_UNKNOWN
}

var rpmStates = map[ufslab.PeripheralState]tlw.RPMOutlet_State{
	ufslab.PeripheralState_WORKING:        tlw.RPMOutlet_WORKING,
	ufslab.PeripheralState_MISSING_CONFIG: tlw.RPMOutlet_MISSING_CONFIG,
	ufslab.PeripheralState_WRONG_CONFIG:   tlw.RPMOutlet_WRONG_CONFIG,
}

func convertRPMState(s ufslab.PeripheralState) tlw.RPMOutlet_State {
	if ns, ok := rpmStates[s]; ok {
		return ns
	}
	return tlw.RPMOutlet_UNSPECIFIED
}

var cr50Phases = map[ufslab.DutState_CR50Phase]tlw.Cr50Phase{
	ufslab.DutState_CR50_PHASE_PREPVT: tlw.Cr50PhasePREPVT,
	ufslab.DutState_CR50_PHASE_PVT:    tlw.Cr50PhasePVT,
}

func convertCr50Phase(p ufslab.DutState_CR50Phase) tlw.Cr50Phase {
	if p, ok := cr50Phases[p]; ok {
		return p
	}
	return tlw.Cr50PhaseUnspecified
}

var cr50KeyEnvs = map[ufslab.DutState_CR50KeyEnv]tlw.Cr50KeyEnv{
	ufslab.DutState_CR50_KEYENV_PROD: tlw.Cr50KeyEnvProd,
	ufslab.DutState_CR50_KEYENV_DEV:  tlw.Cr50KeyEnvDev,
}

func convertCr50KeyEnv(p ufslab.DutState_CR50KeyEnv) tlw.Cr50KeyEnv {
	if p, ok := cr50KeyEnvs[p]; ok {
		return p
	}
	return tlw.Cr50KeyEnvUnspecified
}

func convertServoTopologyItemFromUFS(i *ufslab.ServoTopologyItem) *tlw.ServoTopologyItem {
	if i == nil {
		return nil
	}
	return &tlw.ServoTopologyItem{
		Type:         i.GetType(),
		SysfsProduct: i.GetSysfsProduct(),
		Serial:       i.GetSerial(),
		UsbHubPort:   i.GetUsbHubPort(),
	}
}

func convertServoTopologyFromUFS(st *ufslab.ServoTopology) *tlw.ServoTopology {
	var t *tlw.ServoTopology
	if st != nil {
		var children []*tlw.ServoTopologyItem
		for _, child := range st.GetChildren() {
			children = append(children, convertServoTopologyItemFromUFS(child))
		}
		t = &tlw.ServoTopology{
			Root:     convertServoTopologyItemFromUFS(st.Main),
			Children: children,
		}
	}
	return t
}

func convertServoTopologyItemToUFS(i *tlw.ServoTopologyItem) *ufslab.ServoTopologyItem {
	if i == nil {
		return nil
	}
	return &ufslab.ServoTopologyItem{
		Type:         i.Type,
		SysfsProduct: i.SysfsProduct,
		Serial:       i.Serial,
		UsbHubPort:   i.UsbHubPort,
	}
}

func convertServoTopologyToUFS(st *tlw.ServoTopology) *ufslab.ServoTopology {
	var t *ufslab.ServoTopology
	if st != nil {
		var children []*ufslab.ServoTopologyItem
		for _, child := range st.Children {
			children = append(children, convertServoTopologyItemToUFS(child))
		}
		t = &ufslab.ServoTopology{
			Main:     convertServoTopologyItemToUFS(st.Root),
			Children: children,
		}
	}
	return t
}
