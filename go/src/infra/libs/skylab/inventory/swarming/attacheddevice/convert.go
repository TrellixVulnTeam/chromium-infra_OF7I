// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// Package attacheddevice implements conversion of attached device labels to
// Swarming dimensions.
package attacheddevice

import (
	"infra/libs/skylab/inventory/swarming"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

// Convert converts attached device labels to Swarming dimensions.
func Convert(attachedDeviceData *ufsapi.AttachedDeviceData) swarming.Dimensions {
	dims := make(swarming.Dimensions)
	machine := attachedDeviceData.GetMachine()
	machineLSE := attachedDeviceData.GetLabConfig()
	// Android DUT id and name.
	if name := machineLSE.GetHostname(); name != "" {
		dims["dut_id"] = []string{name}
		dims["dut_name"] = []string{name}
	}
	// Associated hostname.
	if hostname := machineLSE.GetAttachedDeviceLse().GetAssociatedHostname(); hostname != "" {
		dims["label-associated_hostname"] = []string{hostname}
	}
	// Android DUT model codename.
	if model := machine.GetAttachedDevice().GetModel(); model != "" {
		dims["label-model"] = []string{model}
	}
	// Board name
	if board := machine.GetAttachedDevice().GetBuildTarget(); board != "" {
		dims["label-board"] = []string{board}
	}
	// Android DUT serial number.
	if serialNumber := machine.GetSerialNumber(); serialNumber != "" {
		dims["serial_number"] = []string{serialNumber}
	}
	return dims
}
