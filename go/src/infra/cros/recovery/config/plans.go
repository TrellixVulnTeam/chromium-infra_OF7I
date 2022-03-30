// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

// List of known plans.
const (
	PlanAndroid       = "android"
	PlanCrOS          = "cros"
	PlanServo         = "servo"
	PlanChameleon     = "chameleon"
	PlanBluetoothPeer = "bluetooth_peer"
	PlanWifiRouter    = "wifi_router"
	// That is final plan which will run always if present in configuration.
	// The goal is execution final step to clean up stages if something left
	// over in the devices.
	PlanClosing = "close"
)
