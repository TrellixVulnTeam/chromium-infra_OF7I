// Copyright 2021 The Chromium OS Authors. All rights reserved. Use of
// this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"testing"
)

var mainServoDeviceTestCases = []struct {
	servoType  string
	mainDevice string
}{
	{
		"servo_v3",
		"servo_v3",
	},
	{
		"servo_v4",
		"servo_v4",
	},
	{
		"servo_v4_with_ccd_cr50",
		"ccd_cr50",
	},
	{
		"servo_v4_with_servo_micro_and_ccd_cr50",
		"servo_micro",
	},
}

func TestMainServoDeviceHelper(t *testing.T) {
	for _, testCase := range mainServoDeviceTestCases {
		r, err := mainServoDeviceHelper(testCase.servoType)
		if err != nil {
			t.Errorf("test main servo device helper: for servo type %q, error %q", testCase.servoType, err)
		}
		if r != testCase.mainDevice {
			t.Errorf("test main servo device helper: for servo type %q, expected main device %q, but got %q", testCase.servoType, testCase.mainDevice, r)
		}
	}
}
