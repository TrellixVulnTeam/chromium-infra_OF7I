// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package tlw provides an abstract representation of the TLW API which is used by recovery
package tlw

import (
	"context"

	"go.chromium.org/chromiumos/config/go/api/test/xmlrpc"

	"infra/cros/dutstate"
)

// Access represent TLW level to access to the devices and inventory.
// Each device in the lab is representing as resource with name.
// For now the resource name matche to host-name but later can became different.
// Examples:
// 	Hostname: lab1-row1-rack1-device1, lab1-row1-rack1-ap1
// 	Resource Name: TestDevice256, CustomApV3.0
type Access interface {
	// Ping performs ping by resource name.
	Ping(ctx context.Context, resourceName string, count int) error
	// Run executes command on device by SSH related to resource name.
	Run(ctx context.Context, resourceName, command string) *RunResult
	// InitServod initiates servod daemon on servo-host.
	InitServod(ctx context.Context, req *InitServodRequest) error
	// StopServod stops servod daemon on servo-host.
	StopServod(ctx context.Context, resourceName string) error
	// CallServod executes a command on servod related to resource name.
	// Commands will be run against servod on servo-host.
	CallServod(ctx context.Context, req *CallServodRequest) *CallServodResponse
	// CopyFileTo copies file to destination device from local.
	CopyFileTo(ctx context.Context, req *CopyRequest) error
	// CopyFileFrom copies file from remote device to local.
	CopyFileFrom(ctx context.Context, req *CopyRequest) error
	// CopyDirectoryTo copies directory to destination device from local, recursively.
	CopyDirectoryTo(ctx context.Context, req *CopyRequest) error
	// CopyDirectoryFrom copies directory from destination device to local, recursively.
	CopyDirectoryFrom(ctx context.Context, req *CopyRequest) error
	// SetPowerSupply manages power supply for requested.
	SetPowerSupply(ctx context.Context, req *SetPowerSupplyRequest) *SetPowerSupplyResponse
	// ListResourcesForUnit provides list of resources names related to target unit.
	// All test and task scheduling against the target unit which can link to 1 or more resources.
	ListResourcesForUnit(ctx context.Context, unitName string) ([]string, error)
	// GetDut provides DUT info per requested resource name from inventory.
	GetDut(ctx context.Context, resourceName string) (*Dut, error)
	// UpdateDut updates DUT info into inventory.
	UpdateDut(ctx context.Context, dut *Dut) error
	// GetImageUrl provides URL to the image requested to load.
	// URL will use to download image to USB-drive and provisioning.
	GetImageUrl(ctx context.Context, resourceName, imageName string) (string, error)
	// Close closes all used resources.
	Close() error
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
type PowerSupplyAction string

const (
	PowerSupplyActionUnspecified PowerSupplyAction = "UNSPECIFIED"
	// Switch state to ON.
	PowerSupplyActionOn PowerSupplyAction = "ON"
	// Switch state to OFF.
	PowerSupplyActionOff PowerSupplyAction = "OFF"
	// Switch state to OFF and then ON with delay 5 seconds.
	PowerSupplyActionCycle PowerSupplyAction = "CYCLE"
)

// SetPowerSupplyRequest represents data to perform state change for power supplier.
type SetPowerSupplyRequest struct {
	// Resource name
	Resource string
	// Expected state to switch on.
	State PowerSupplyAction
}

// PowerSupplyStatus represents response status from attempt to changes state of power supplier.
type PowerSupplyResponseStatus string

const (
	PowerSupplyResponseStatusUnspecified PowerSupplyResponseStatus = "UNSPECIFIED"
	PowerSupplyResponseStatusOK          PowerSupplyResponseStatus = "OK"
	// RPM config is not present of incorrect.
	PowerSupplyResponseStatusNoConfig PowerSupplyResponseStatus = "NO_CONFIG"
	// Request data incorrect or in unexpected state.
	PowerSupplyResponseStatusBadRequest PowerSupplyResponseStatus = "BAD REQUEST"
	// Fail to switch to required state.
	PowerSupplyResponseStatusError PowerSupplyResponseStatus = "ERROR"
)

// SetPowerSupplyResponse represents data result from performing state change for power supplier.
type SetPowerSupplyResponse struct {
	// New state.
	Status PowerSupplyResponseStatus
	// Error details
	Reason string
}

// ServodMethod represents types of methods supporting by servod daemon.
// Examples:
//   get: to read data need to pass method:`get`, command:`lid_open`.
//   set: to update state need to pass method:`set`, command:`lid_open`, value:`no`.
type ServodMethod string

const (
	// Reading data by servod daemon.
	// Example: ec_board, lid_open.
	ServodMethodGet ServodMethod = "get"
	// Set methods used to set values or call methods with providing paramenter.
	// Example: power_state:reset, lid_open:no.
	ServodMethodSet ServodMethod = "set"
	// Verify if control is known and present in servod daemon.
	// Example: ec_board, lid_open.
	ServodMethodDoc ServodMethod = "doc"
)

// ServodOption represents options to start servod.
type ServodOptions struct {
	// Use recovery mode when start servod
	RecoveryMode bool
}

// InitServodRequest represents data to initiate servod daemon on servo-host.
type InitServodRequest struct {
	Resource string
	Options  *ServodOptions
}

// CallServodRequest represents data to run command on servod.
type CallServodRequest struct {
	Resource string
	Method   ServodMethod
	Args     []*xmlrpc.Value
	Options  *ServodOptions
}

// CallServodResponse represents result data from running command on servod.
type CallServodResponse struct {
	Value *xmlrpc.Value
	Fault bool
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

// Cr50Phase describes different phases of CR50 firmware used on DUT.
type Cr50Phase string

const (
	Cr50PhaseUnspecified Cr50Phase = "UNSPECIFIED"
	Cr50PhasePVT         Cr50Phase = "CR50_PHASE_PREPVT"
	Cr50PhasePREPVT      Cr50Phase = "CR50_PHASE_PVT"
)

// Cr50KeyEnv describes key env for cr50 RW version.
type Cr50KeyEnv string

const (
	Cr50KeyEnvUnspecified Cr50KeyEnv = "UNSPECIFIED"
	Cr50KeyEnvProd        Cr50KeyEnv = "CR50_KEYENV_PROD"
	Cr50KeyEnvDev         Cr50KeyEnv = "CR50_KEYENV_DEV"
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
	// Cr50 firmware phase used on the DUT.
	Cr50Phase Cr50Phase
	// Key env for RW Cr50 firmware version.
	Cr50KeyEnv Cr50KeyEnv
	// State of the DUT.
	State dutstate.State

	// Physical parts of DUT.
	// Internal storage info.
	Storage *DUTStorage
	// Battery info.
	Battery *DUTBattery
	// Wifi info.
	Wifi *DUTWifi
	// Bluetooth info.
	Bluetooth *DUTBluetooth

	// Peripheral devices.
	// ServoHost of the DUT setup.
	ServoHost *ServoHost
	// Chameleon device of the DUT setup.
	ChameleonHost *ChameleonHost
	// BluetoothPeer info of DUT setup.
	BluetoothPeerHosts []*BluetoothPeerHost
	// RPMOutlet of the DUT setup.
	RPMOutlet *RPMOutlet

	// StableVersion of the DUT.
	StableVersion *StableVersion
}

// StableVersion holds info about stable versions used for recovery devices.
type StableVersion struct {
	// ChromeOS stable image in standard GS path format.
	// Example: board-release/R90-13816.47.0
	CrosImage string
	// ChromeOS firmware version in a format that align with omaha manifest data,
	// and crossystem or chromeos-firmwareupdate output.
	// Example: Google_Board.13434.261.0
	CrosFirmwareVersion string
	// ChromeOS firmware image in standard GS path format.
	// Example: board-firmware/R87-13434.261.0
	CrosFirmwareImage string
}

// HardwareState describes the state of hardware components.
type HardwareState string

const (
	// Keep for all unknown state by default.
	HardwareStateUnspecified HardwareState = "UNSPECIFIED"
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

// DUTStorage holds info about internal storage of the DUT.
type DUTStorage struct {
	// State of the component.
	State HardwareState
	// Type of storage used on device.
	Type StorageType
}

// DUTWifi holds info about internal wifi of the DUT.
type DUTWifi struct {
	// State of the component.
	State HardwareState
	// Name of wifi chip used on the device.
	ChipName string
}

// DUTBluetooth holds info about internal bluetooth of the DUT.
type DUTBluetooth struct {
	// State of the component.
	State HardwareState
}

// DUTBattery holds info about battery of the DUT.
type DUTBattery struct {
	// State of the component.
	State HardwareState
}

// ChameleonState describes the state of chameleon device.
type ChameleonState string

const (
	ChameleonStateUnspecified ChameleonState = "UNSPECIFIED"
	// Device and software on it is working as expected.
	ChameleonStateWorking ChameleonState = "WORKING"
	// Device is broken or not working as expected.
	ChameleonStateBroken ChameleonState = "BROKEN"
)

// ChameleonHost holds info about chameleon device.
type ChameleonHost struct {
	// Name is the resource name.
	Name string
	// State of the device.
	State ChameleonState
}

// BluetoothPeerState describes the state of bluetooth peer device.
type BluetoothPeerState string

const (
	BluetoothPeerStateUnspecified BluetoothPeerState = "UNSPECIFIED"
	// Device and software on it is working as expected.
	BluetoothPeerStateWorking BluetoothPeerState = "WORKING"
	// Device is broken or not working as expected.
	BluetoothPeerStateBroken BluetoothPeerState = "BROKEN"
)

// BluetoothPeer peer devices
type BluetoothPeerHost struct {
	// Name is the resource name.
	Name string
	// State of the device.
	State BluetoothPeerState
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

// RPMOutlet is wall-power source for DUT which allow us perform action to do OFF/ON/CYCLE on it.
type RPMOutlet struct {
	// Name is the resource name.
	Name string
	// State of the component.
	State RPMState
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
	// Smart USB-hub is present on setup.
	SmartUsbhubPresent bool
	// Servo Topology of servo devices.
	ServoTopology *ServoTopology
}

// ServoTopology describes servo devices used to provide servo functionality.
type ServoTopology struct {
	Root     *ServoTopologyItem
	Children []*ServoTopologyItem
}

// ServoTopologyItem describes details of one servo device.
type ServoTopologyItem struct {
	// type provides the type of servo device. Keeping as String to avoid issue with introduce new type.
	Type string
	// sysfs_product provides the product name of the device recorded in File System.
	SysfsProduct string
	// serial provides the serial number of the device.
	Serial string
	// usb_hub_port provides the port connection to the device.
	// e.g. '1-6.2.2' where
	//   '1-6'  - port on the labstation
	//   '2'    - port on smart-hub connected to the labstation
	//   '2'    - port on servo hub (part of servo_v4 or servo_v4.1) connected to the smart-hub
	// The same path will look '1-6.2' if connected servo_v4 directly to the labstation.
	UsbHubPort string
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

// Servo holds info about servo functionality.
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
