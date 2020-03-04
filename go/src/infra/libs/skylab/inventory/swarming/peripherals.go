// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"fmt"
	"strconv"
	"strings"

	"infra/libs/skylab/inventory"

	"go.chromium.org/chromiumos/infra/proto/go/lab"
)

func init() {
	converters = append(converters, boolPeripheralsConverter)
	reverters = append(reverters, boolPeripheralsReverter)
	converters = append(converters, otherPeripheralsConverter)
	reverters = append(reverters, otherPeripheralsReverter)
}

func boolPeripheralsConverter(dims Dimensions, ls *inventory.SchedulableLabels) {
	p := ls.GetPeripherals()
	if p.GetAudioBoard() {
		dims["label-audio_board"] = []string{"True"}
	}
	if p.GetAudioBox() {
		dims["label-audio_box"] = []string{"True"}
	}
	if p.GetAudioLoopbackDongle() {
		dims["label-audio_loopback_dongle"] = []string{"True"}
	}
	if p.GetCamerabox() {
		dims["label-camerabox"] = []string{"True"}
	}
	if p.GetChameleon() {
		dims["label-chameleon"] = []string{"True"}
	}
	if p.GetConductive() {
		dims["label-conductive"] = []string{"True"}
	}
	if p.GetHuddly() {
		dims["label-huddly"] = []string{"True"}
	}
	if p.GetMimo() {
		dims["label-mimo"] = []string{"True"}
	}
	if p.GetServo() {
		dims["label-servo"] = []string{"True"}
	}
	if p.GetStylus() {
		dims["label-stylus"] = []string{"True"}
	}
	if p.GetWificell() {
		dims["label-wificell"] = []string{"True"}
	}
	if p.GetRouter_802_11Ax() {
		dims["label-router_802_11ax"] = []string{"True"}
	}
}

func boolPeripheralsReverter(ls *inventory.SchedulableLabels, d Dimensions) Dimensions {
	p := ls.Peripherals
	d = assignLastBoolValueAndDropKey(d, p.AudioBoard, "label-audio_board")
	d = assignLastBoolValueAndDropKey(d, p.AudioBox, "label-audio_box")
	d = assignLastBoolValueAndDropKey(d, p.AudioLoopbackDongle, "label-audio_loopback_dongle")
	d = assignLastBoolValueAndDropKey(d, p.Camerabox, "label-camerabox")
	d = assignLastBoolValueAndDropKey(d, p.Chameleon, "label-chameleon")
	d = assignLastBoolValueAndDropKey(d, p.Conductive, "label-conductive")
	d = assignLastBoolValueAndDropKey(d, p.Huddly, "label-huddly")
	d = assignLastBoolValueAndDropKey(d, p.Mimo, "label-mimo")
	d = assignLastBoolValueAndDropKey(d, p.Servo, "label-servo")
	d = assignLastBoolValueAndDropKey(d, p.Stylus, "label-stylus")
	d = assignLastBoolValueAndDropKey(d, p.Wificell, "label-wificell")
	d = assignLastBoolValueAndDropKey(d, p.Router_802_11Ax, "label-router_802_11ax")
	return d
}

func otherPeripheralsConverter(dims Dimensions, ls *inventory.SchedulableLabels) {
	p := ls.GetPeripherals()
	for _, v := range p.GetChameleonType() {
		appendDim(dims, "label-chameleon_type", v.String())
	}

	if invSState := p.GetServoState(); invSState != inventory.PeripheralState_UNKNOWN {
		if labSState, ok := lab.PeripheralState_name[int32(invSState)]; ok {
			dims["label-servo_state"] = []string{labSState}
		}
	}

	n := p.GetWorkingBluetoothBtpeer()
	btpeers := make([]string, n)
	for i := range btpeers {
		btpeers[i] = fmt.Sprint(i + 1)
	}
	// Empty dimensions may cause swarming page to fail to load: crbug.com/1056285
	if len(btpeers) > 0 {
		dims["label-working_bluetooth_btpeer"] = btpeers
	}
}

func otherPeripheralsReverter(ls *inventory.SchedulableLabels, d Dimensions) Dimensions {
	p := ls.Peripherals

	p.ChameleonType = make([]inventory.Peripherals_ChameleonType, len(d["label-chameleon_type"]))
	for i, v := range d["label-chameleon_type"] {
		if ct, ok := inventory.Peripherals_ChameleonType_value[v]; ok {
			p.ChameleonType[i] = inventory.Peripherals_ChameleonType(ct)
		}
	}
	delete(d, "label-chameleon_type")

	if labSStateName, ok := getLastStringValue(d, "label-servo_state"); ok {
		servoState := inventory.PeripheralState_UNKNOWN
		if ssIndex, ok := lab.PeripheralState_value[strings.ToUpper(labSStateName)]; ok {
			servoState = inventory.PeripheralState(ssIndex)
		}
		p.ServoState = &servoState
		delete(d, "label-servo_state")
	}

	btpeers := d["label-working_bluetooth_btpeer"]
	max := 0
	for _, v := range btpeers {
		if i, err := strconv.Atoi(v); err == nil && i > max {
			max = i
		}
	}
	*p.WorkingBluetoothBtpeer = int32(max)
	delete(d, "label-working_bluetooth_btpeer")

	return d
}
