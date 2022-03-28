// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package attached_device implements conversion of attached device labels to
// Autotest labels.
package attached_device

import (
	"strings"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

// Convert converts attached device labels to Autotest labels.
func Convert(attachedDeviceData *ufsapi.AttachedDeviceData) []string {
	var labels []string
	machine := attachedDeviceData.GetMachine()
	machineLSE := attachedDeviceData.GetLabConfig()
	// Associated hostname.
	if hostname := machineLSE.GetAttachedDeviceLse().GetAssociatedHostname(); hostname != "" {
		labels = append(labels, "associated_hostname:"+strings.ToLower(hostname))
	}
	// Attached device DUT name.
	if name := machineLSE.GetHostname(); name != "" {
		labels = append(labels, "name:"+strings.ToLower(name))
	}
	// Attached device serial number.
	if serialNumber := machine.GetSerialNumber(); serialNumber != "" {
		labels = append(labels, "serial_number:"+serialNumber)
	}
	// Attached device model codename.
	if model := machine.GetAttachedDevice().GetModel(); model != "" {
		labels = append(labels, "model:"+strings.ToLower(model))
	}
	// Attached device board name
	if board := machine.GetAttachedDevice().GetBuildTarget(); board != "" {
		labels = append(labels, "board:"+strings.ToLower(board))
	}

	// Attached device os type.
	switch machine.GetAttachedDevice().GetDeviceType() {
	case ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_ANDROID_PHONE,
		ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_ANDROID_TABLET:
		labels = append(labels, "os:android")
	case ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_APPLE_PHONE,
		ufspb.AttachedDeviceType_ATTACHED_DEVICE_TYPE_APPLE_TABLET:
		labels = append(labels, "os:ios")
	default:
		labels = append(labels, "os:unknown")
	}
	return labels
}
