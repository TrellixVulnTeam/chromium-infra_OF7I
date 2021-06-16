// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dutinfo

import (
	"infra/cros/recovery/tlw"
	ufsdevice "infra/unifiedfleet/api/v1/models/chromeos/device"
	ufslab "infra/unifiedfleet/api/v1/models/chromeos/lab"
)

func convertHardwareState(s ufslab.HardwareState) tlw.HardwareState {
	switch s {
	case ufslab.HardwareState_HARDWARE_NORMAL:
		return tlw.HardwareStateNormal
	case ufslab.HardwareState_HARDWARE_ACCEPTABLE:
		return tlw.HardwareStateAcceptable
	case ufslab.HardwareState_HARDWARE_NEED_REPLACEMENT:
		return tlw.HardwareStateNeedReplacement
	case ufslab.HardwareState_HARDWARE_NOT_DETECTED:
		return tlw.HardwareStateNotDetected
	default:
		return tlw.HardwareStateUnknown
	}
}

func convertFirmwareChannel(s ufslab.ServoFwChannel) tlw.ServoFirmwareChannel {
	switch s {
	case ufslab.ServoFwChannel_SERVO_FW_ALPHA:
		return tlw.ServoFirmwareChannelAlpha
	case ufslab.ServoFwChannel_SERVO_FW_DEV:
		return tlw.ServoFirmwareChannelDev
	case ufslab.ServoFwChannel_SERVO_FW_PREV:
		return tlw.ServoFirmwareChannelPrev
	default:
		return tlw.ServoFirmwareChannelStable
	}
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

func convertServoState(s ufslab.PeripheralState) tlw.ServoState {
	switch s {
	case ufslab.PeripheralState_WORKING:
		return tlw.ServoStateWorking
	case ufslab.PeripheralState_MISSING_CONFIG:
		return tlw.ServoStateMissingConfig
	case ufslab.PeripheralState_WRONG_CONFIG:
		return tlw.ServoStateWrongConfig
	case ufslab.PeripheralState_NOT_CONNECTED:
		return tlw.ServoStateNotConnected
	case ufslab.PeripheralState_NO_SSH:
		return tlw.ServoStateNoSSH
	case ufslab.PeripheralState_BROKEN:
		return tlw.ServoStateBroken
	case ufslab.PeripheralState_NEED_REPLACEMENT:
		return tlw.ServoStateNeedReplacement
	case ufslab.PeripheralState_CR50_CONSOLE_MISSING:
		return tlw.ServoStateCr50ConsoleMissing
	case ufslab.PeripheralState_CCD_TESTLAB_ISSUE:
		return tlw.ServoStateCCDTestlabIssue
	case ufslab.PeripheralState_SERVOD_ISSUE:
		return tlw.ServoStateServodIssue
	case ufslab.PeripheralState_LID_OPEN_FAILED:
		return tlw.ServoStateLidOpenIssue
	case ufslab.PeripheralState_BAD_RIBBON_CABLE:
		return tlw.ServoStateBadRibbonCable
	case ufslab.PeripheralState_EC_BROKEN:
		return tlw.ServoStateECBroken
	case ufslab.PeripheralState_DUT_NOT_CONNECTED:
		return tlw.ServoStateDUTNotConnected
	case ufslab.PeripheralState_TOPOLOGY_ISSUE:
		return tlw.ServoStateTopologyIssue
	case ufslab.PeripheralState_SBU_LOW_VOLTAGE:
		return tlw.ServoStateSBULowVoltage
	case ufslab.PeripheralState_CR50_NOT_ENUMERATED:
		return tlw.ServoStateCr50NotEnumerated
	case ufslab.PeripheralState_SERVO_SERIAL_MISMATCH:
		return tlw.ServoStateServoSerialMismatch
	case ufslab.PeripheralState_SERVOD_PROXY_ISSUE:
		return tlw.ServoStateServodProxyIssue
	case ufslab.PeripheralState_SERVO_HOST_ISSUE:
		return tlw.ServoStateServoHostIssue
	case ufslab.PeripheralState_SERVO_UPDATER_ISSUE:
		return tlw.ServoStateServoUpdaterIssue
	default:
		return tlw.ServoStateUnspecified
	}
}

func convertRPMState(s ufslab.PeripheralState) tlw.RPMState {
	switch s {
	case ufslab.PeripheralState_WORKING:
		return tlw.RPMStateWorking
	case ufslab.PeripheralState_MISSING_CONFIG:
		return tlw.RPMStateMissingConfig
	case ufslab.PeripheralState_WRONG_CONFIG:
		return tlw.RPMStateWrongConfig
	default:
		return tlw.RPMStateUnspecified
	}
}
