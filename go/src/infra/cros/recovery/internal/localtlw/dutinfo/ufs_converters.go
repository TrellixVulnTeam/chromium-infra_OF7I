// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dutinfo

import (
	"infra/cros/recovery/tlw"
	ufsdevice "infra/unifiedfleet/api/v1/models/chromeos/device"
	ufslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
)

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

var servoStates = map[ufslab.PeripheralState]tlw.ServoState{
	ufslab.PeripheralState_WORKING:               tlw.ServoStateWorking,
	ufslab.PeripheralState_MISSING_CONFIG:        tlw.ServoStateMissingConfig,
	ufslab.PeripheralState_WRONG_CONFIG:          tlw.ServoStateWrongConfig,
	ufslab.PeripheralState_NOT_CONNECTED:         tlw.ServoStateNotConnected,
	ufslab.PeripheralState_NO_SSH:                tlw.ServoStateNoSSH,
	ufslab.PeripheralState_BROKEN:                tlw.ServoStateBroken,
	ufslab.PeripheralState_NEED_REPLACEMENT:      tlw.ServoStateNeedReplacement,
	ufslab.PeripheralState_CR50_CONSOLE_MISSING:  tlw.ServoStateCr50ConsoleMissing,
	ufslab.PeripheralState_CCD_TESTLAB_ISSUE:     tlw.ServoStateCCDTestlabIssue,
	ufslab.PeripheralState_SERVOD_ISSUE:          tlw.ServoStateServodIssue,
	ufslab.PeripheralState_LID_OPEN_FAILED:       tlw.ServoStateLidOpenIssue,
	ufslab.PeripheralState_BAD_RIBBON_CABLE:      tlw.ServoStateBadRibbonCable,
	ufslab.PeripheralState_EC_BROKEN:             tlw.ServoStateECBroken,
	ufslab.PeripheralState_DUT_NOT_CONNECTED:     tlw.ServoStateDUTNotConnected,
	ufslab.PeripheralState_TOPOLOGY_ISSUE:        tlw.ServoStateTopologyIssue,
	ufslab.PeripheralState_SBU_LOW_VOLTAGE:       tlw.ServoStateSBULowVoltage,
	ufslab.PeripheralState_CR50_NOT_ENUMERATED:   tlw.ServoStateCr50NotEnumerated,
	ufslab.PeripheralState_SERVO_SERIAL_MISMATCH: tlw.ServoStateServoSerialMismatch,
	ufslab.PeripheralState_SERVOD_PROXY_ISSUE:    tlw.ServoStateServodProxyIssue,
	ufslab.PeripheralState_SERVO_HOST_ISSUE:      tlw.ServoStateServoHostIssue,
	ufslab.PeripheralState_SERVO_UPDATER_ISSUE:   tlw.ServoStateServoUpdaterIssue,
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

var rpmStates = map[ufslab.PeripheralState]tlw.RPMState{
	ufslab.PeripheralState_WORKING:        tlw.RPMStateWorking,
	ufslab.PeripheralState_MISSING_CONFIG: tlw.RPMStateMissingConfig,
	ufslab.PeripheralState_WRONG_CONFIG:   tlw.RPMStateWrongConfig,
}

func convertRPMState(s ufslab.PeripheralState) tlw.RPMState {
	if ns, ok := rpmStates[s]; ok {
		return ns
	}
	return tlw.RPMStateUnspecified
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
