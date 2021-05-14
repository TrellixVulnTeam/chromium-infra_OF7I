// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package tlw provides an abstract representation of the TLW API which is used by recovery
package tlw

import (
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
	Ping(resourceName string) (bool, error)
	// Execute command on the device related to resource name.
	Run(resourceName, command string) *RunResult
	// Execute command on servo related to resource name.
	// Commands will be run against servod on servo-host.
	CallServod(resourceName, command string) *tls.CallServoResponse
	// Copy file to destination device from local.
	CopyFileTo(req *CopyRequest) error
	// Copy file from destination device to local.
	CopyFileFrom(req *CopyRequest) error
	// Copy directory to destination device from local, recursively.
	CopyDirectoryTo(req *CopyRequest) error
	// Copy directory from destination device to local, recursively.
	CopyDirectoryFrom(req *CopyRequest) error
	// Manage power supply for requested.
	SetPowerSupply(req *SetPowerSupplyRequest) *SetPowerSupplyResponse
	// Provide list of resources names related to target unit.
	// All test and task scheduling against the target unit which can link to 1 or more resources.
	ListResourcesForUnit(unitName string) ([]string, error)
	// Get DUT info per requested resource name from inventory.
	GetDut(resourceName string) *tls.Dut
	// Update DUT info into inventory.
	UpdateDut(dut *tls.Dut) error
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

// CopyRequest represent data to perform copy data from/to resource.
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

// PowerSupplyStatus represent response status from attempt to changes state of power supplier.
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
