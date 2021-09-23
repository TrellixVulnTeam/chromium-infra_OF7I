// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cros

import (
	"fmt"
	"reflect"
	"testing"
)

var testCases = []struct {
	testName string
	in       string
	out      map[string]map[string]string
}{
	{
		"Power & Battery",
		`
			Device: Line Power
				online:                  yes
				type:                    Mains
				voltage (V):             0
				current (A):             0
			Device: Battery
				state:                   Discharging
				percentage:              95.9276
				technology:              Li-ion
		`,
		map[string]map[string]string{
			"Line Power": {
				"online":      "yes",
				"type":        "Mains",
				"voltage (V)": "0",
				"current (A)": "0",
			},
			"Battery": {
				"state":      "Discharging",
				"percentage": "95.9276",
				"technology": "Li-ion",
			},
		},
	},
	{
		"Power Only",
		`
			Device: Line Power
				online:                  yes
				type:                    Mains
				voltage (V):             0
				current (A):             0
		`,
		map[string]map[string]string{
			"Line Power": {
				"online":      "yes",
				"type":        "Mains",
				"voltage (V)": "0",
				"current (A)": "0",
			},
		},
	},
	{
		"Battery Only",
		`
			Device: Battery
				state:                   Discharging
				percentage:              95.9276
				technology:              Li-ion
		`,
		map[string]map[string]string{
			"Battery": {
				"state":      "Discharging",
				"percentage": "95.9276",
				"technology": "Li-ion",
			},
		},
	},
	{
		"Empty Data",
		`
		`,
		map[string]map[string]string{},
	},
	{
		"Wrong Data when fields present out of device scope",
		`
			state:                   Discharging
			percentage:              95.9276
			technology:              Li-ion
		`,
		map[string]map[string]string{},
	},
}

func TestGetPowerSupplyInfoInMap(t *testing.T) {
	t.Parallel()
	for _, tt := range testCases {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			result := getPowerSupplyInfoInMap(tt.in)
			fmt.Println(result)
			if !reflect.DeepEqual(tt.out, result) {
				t.Errorf("\n got %q \n want %q", result, tt.out)
			}
		})
	}
}
