// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package tlw provides an abstract representation of the TLW API which is used by recovery
package tlw

import (
	"context"

	"go.chromium.org/chromiumos/config/go/api/test/tls"
)

// Access represent TLW level to access to the devices and inventory.
// Each device in the lab is representing as resource with name.
// For now the resource name matche to host-name but later can became different.
// Examples:
// 	Hostname: lab1-row1-rack1-device1, lab1-row1-rack1-ap1
// 	Resource Name: TestDevice256, CustomApV3.0
type Access interface {
	// Ping the device related to resource name.
	Ping(ctx context.Context, resourceName string) (bool, error)
	// Execute command on the device related to resource name.
	Run(ctx context.Context, resourceName, command string) *RunResult
	// Execute command on servo related to resource name.
	// Commands will be run against servod on servo-host.
	CallServod(ctx context.Context, resourceName, command string) *tls.CallServoResponse
	// Copy file to destination device from local.
	CopyFileTo(ctx context.Context, req *CopyRequest) error
	// Copy file from destination device to local.
	CopyFileFrom(ctx context.Context, req *CopyRequest) error
	// Copy directory to destination device from local, recursively.
	CopyDirectoryTo(ctx context.Context, req *CopyRequest) error
	// Copy directory from destination device to local, recursively.
	CopyDirectoryFrom(ctx context.Context, req *CopyRequest) error
	// Manage power supply for requested.
	SetPowerSupply(ctx context.Context, req *SetPowerSupplyRequest) *SetPowerSupplyResponse
	// Provide list of resources names related to target unit.
	// All test and task scheduling against the target unit which can link to 1 or more resources.
	ListResourcesForUnit(ctx context.Context, unitName string) ([]string, error)
	// Get DUT info per requested resource name from inventory.
	GetDut(ctx context.Context, resourceName string) (*Dut, error)
	// Update DUT info into inventory.
	UpdateDut(ctx context.Context, dut *Dut) error
}

// RunResult represents result of executed command.
type RunResult struct {
	// Command executed on the resource.
	Command string
	// Exit code return.
	// Eg: 0 - everything is good
	// 	   1 - executed stop with error code `1`
	//     15 - timeout of execution
	ExitCode int
	// Standard output
	Stdout string
	// Standard error output
	Stderr string
}

// CopyRequest represents data to perform copy data from/to resource.
type CopyRequest struct {
	// Resource name
	Resource string
	// Path to source file or directory.
	PathSource string
	// Path to destination file or directory.
	PathDestination string
}

// PowerSupplyState represents action expecting to perform on power supplier.
type PowerSupplyAction int

const (
	PowerSupplyActionUnknown PowerSupplyAction = iota
	// Switch state to ON.
	PowerSupplyActionOn
	// Switch state to OFF.
	PowerSupplyActionOff
	// Switch state to OFF and then ON with delay 5 seconds.
	PowerSupplyActionCycle
)

// SetPowerSupplyRequest represents data to perform state change for power supplier.
type SetPowerSupplyRequest struct {
	// Resource name
	Resource string
	// Expected state to switch on.
	State PowerSupplyAction
}

// PowerSupplyStatus represents response status from attempt to changes state of power supplier.
type PowerSupplyResponseStatus int

const (
	PowerSupplyResponseStatusUnknown PowerSupplyResponseStatus = iota
	PowerSupplyResponseStatusOK
	// RPM config is not present of incorrect.
	PowerSupplyResponseStatusNoConfig
	// Request data incorrect or in unexpected state.
	PowerSupplyResponseStatusBadRequest
	// Fail to switch to required state.
	PowerSupplyResponseStatusError
)

// SetPowerSupplyResponse represents data to perform state change for power supplier.
type SetPowerSupplyResponse struct {
	// New state.
	Status PowerSupplyResponseStatus
	// Error details
	Reason string
}

// DUTSetupType describes different DUT setups.
type DUTSetupType string

const (
	DUTSetupTypeDefault DUTSetupType = "DEFAULT"
	// Special setup of servo-host represented as labstation.
	DUTSetupTypeLabstation DUTSetupType = "LABSTATION"
	// Special setup for routers.
	DUTSetupTypeJetstream DUTSetupType = "JETSTREAM"
)

// PowerSupplyType describes different power supply types for a DUT.
type PowerSupplyType string

const (
	PowerSupplyTypeUnspecified PowerSupplyType = "UNSPECIFIED"
	// Primary power source of the devices is wall-power. Devices does not have a battery.
	PowerSupplyTypeACOnly PowerSupplyType = "AC_ONLY"
	// Primary power source of the devices is battery. Devices still connected to wall-power to charge it.
	PowerSupplyTypeBattery PowerSupplyType = "BATTERY"
)

// Dut holds info about setup used as testbed.
type Dut struct {
	// Name is the resource name for the DUT.
	Name string
	// Board name of the DUT.
	Board string
	// Model name of the DUT.
	Model string
	// Hardware identifier.
	Hwid string
	// Serial number of the DUT.
	SerialNumber string
	// SetupType describes the setup of the DUT, which affects how it is verified/repaired.
	SetupType DUTSetupType
	// PowerSupplyType describes the DUT's power supply type.
	PowerSupplyType PowerSupplyType
	// Physical parts of DUT.
	// Internal storage info.
	Storage *DutStorage
	// Battery info (if applicable).
	Battery *DutBattery
	// RPMOutlet of the DUT setup.
	RPMOutlet *RPMOutlet
}

// HardwareState describes the state of hardware components.
type HardwareState string

const (
	// Keep for all unknown state by default.
	HardwareStateUnknown HardwareState = "UNKNOWN"
	// Hardware is in good shape and pass all verifiers.
	HardwareStateNormal HardwareState = "NORMAL"
	// Hardware is still good but close to became bad.
	// Example: DUT storage when usage reached 98% usage limit.
	HardwareStateAcceptable HardwareState = "ACCEPTABLE"
	// Hardware is broken, bad or reached limit when it has to be replaced.
	HardwareStateNeedReplacement HardwareState = "NEED_REPLACEMENT"
	// Hardware expected to be present but not detected.
	HardwareStateNotDetected HardwareState = "NOT_DETECTED"
)

// StorageType describes which type or storage used on the DUT.
type StorageType string

const (
	StorageTypeUnspecified StorageType = "UNSPECIFIED"
	StorageTypeSSD         StorageType = "SSD"
	StorageTypeHDD         StorageType = "HDD"
	StorageTypeMMC         StorageType = "MMC"
	StorageTypeNVME        StorageType = "NVME"
	StorageTypeUFS         StorageType = "UFS"
)

// DutStorage holds info about internal storage of the DUT.
type DutStorage struct {
	// State of the component.
	State HardwareState
	// Type of storage used on device.
	Type StorageType
}

// DutBattery holds info about battery of the DUT.
type DutBattery struct {
	// State of the component.
	State HardwareState
}

// ServoHost holds info about host to manage servod services and verify connected servo devices.
// Example: labstation, servo-host container.
type ServoHost struct {
	// Name is the resource name.
	Name string
	// State of the USB-key connected to the servo.
	UsbkeyState HardwareState
	// Servo device specified for this DUT.
	Servo *Servo
	// Port user on the host to run servod daemon. Expected value between 9900 and 9999.
	ServodPort int
}

// ServoState describes the state of setup/communication issue related to servo functionality provided by servo.
type ServoState string

const (
	ServoStateUnspecified ServoState = "UNSPECIFIED"
	// Device and software on it is working as expected.
	ServoStateWorking ServoState = "WORKING"
	// Configuration for device is not provided.
	ServoStateMissingConfig ServoState = "MISSING_CONFIG"
	// Configuration contains incorrect information.
	ServoStateWrongConfig ServoState = "WRONG_CONFIG"
	// Device is not connected/plugged.
	ServoStateNotConnected ServoState = "NOT_CONNECTED"
	// Device is not reachable over ssh.
	ServoStateNoSSH ServoState = "NO_SSH"
	// Device is broken or not working as expected. the state used if no specified state for the issue.
	ServoStateBroken ServoState = "BROKEN"
	// Device cannot be repaired or required manual attention to fix/replace it.
	ServoStateNeedReplacement ServoState = "NEED_REPLACEMENT"
	// Cr50 console missing or unresponsive.
	ServoStateCr50ConsoleMissing ServoState = "CR50_CONSOLE_MISSING"
	// Servod daemon cannot start on servo-host because cr50 testlab not enabled.
	ServoStateCCDTestlabIssue ServoState = "CCD_TESTLAB_ISSUE"
	// Servod daemon cannot start on servo-host.
	ServoStateServodIssue ServoState = "SERVOD_ISSUE"
	// Device lid is not open.
	ServoStateLidOpenIssue ServoState = "LID_OPEN_FAILED"
	// The ribbon cable between servo and DUT is broken or not connected.
	ServoStateBadRibbonCable ServoState = "BAD_RIBBON_CABLE"
	// The EC on the DUT has issue.
	ServoStateECBroken ServoState = "EC_BROKEN"
	// Servo is not connected to the DUT.
	ServoStateDUTNotConnected ServoState = "DUT_NOT_CONNECTED"
	// Some component in servo-topology missed or not detected.
	ServoStateTopologyIssue ServoState = "TOPOLOGY_ISSUE"
	// SBU voltage issues effect CR50 detection.
	ServoStateSBULowVoltage ServoState = "SBU_LOW_VOLTAGE"
	// CR50 SBU voltage detected but device was not enumerated.
	ServoStateCr50NotEnumerated ServoState = "CR50_NOT_ENUMERATED"
	// Servo serial mismatch, when servo not detected and another serial detected on previous used port.
	ServoStateServoSerialMismatch ServoState = "SERVO_SERIAL_MISMATCH"
	// Issue to connect to servod by XMLRPC proxy.
	ServoStateServodProxyIssue ServoState = "SERVOD_PROXY_ISSUE"
	// Issue related to servo-host. Timeout to start servod or issue with detecting devices.
	ServoStateServoHostIssue ServoState = "SERVO_HOST_ISSUE"
	// Issue related to servo_updater on the servo-host.
	ServoStateServoUpdaterIssue ServoState = "SERVO_UPDATER_ISSUE"
)

// ServoFirmwareChannel describes the state of setup/communication issue related to servo functionality provided by servo.
type ServoFirmwareChannel string

const (
	// Default channel.
	// Servo firmware from Stable channel.
	ServoFirmwareChannelStable ServoFirmwareChannel = "stable"
	// The previous Servo firmware from Stable channel.
	ServoFirmwareChannelPrev ServoFirmwareChannel = "prev"
	// Servo firmware from Dev channel.
	ServoFirmwareChannelDev ServoFirmwareChannel = "dev"
	// Servo firmware from Alpha channel.
	ServoFirmwareChannelAlpha ServoFirmwareChannel = "alpha"
)

type Servo struct {
	// State of the servo.
	State ServoState
	// Serial number of the root servo device.
	SerialNumber string
	// Channel of firmware used on servo devices.
	FirmwareChannel ServoFirmwareChannel
	// Self representation of servo-setup by servod.
	// Example: servo_v4_with_servo_micro, servo_v4_with_ccd_cr50.
	Type string
}

// RPMState describes the state of RPM outlet.
type RPMState string

const (
	RPMStateUnspecified RPMState = "UNSPECIFIED"
	// Configuration for RPM outlet missed which block from execution the actions.
	RPMStateMissingConfig RPMState = "MISSING_CONFIG"
	// Configuration for RPM outlet provided but does not working which can be several reasons.
	RPMStateWrongConfig RPMState = "WRONG_CONFIG"
	// RPM outlet can successfully perform the actions.
	RPMStateWorking RPMState = "WORKING"
)

// RPMOutlet is wall-power source for DUT which allow us perfrom action to do OFF/ON/CYCLE on it.
type RPMOutlet struct {
	// Name is the resource name.
	Name string
	// State of the component.
	State RPMState
}
