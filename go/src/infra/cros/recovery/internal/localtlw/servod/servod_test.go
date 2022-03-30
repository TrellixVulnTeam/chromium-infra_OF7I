// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servod

import (
	"testing"

	"infra/cros/recovery/tlw"
)

// Test cases for TestDUTPlans
var generateParamsTestCases = []struct {
	name string
	got  *tlw.ServodOptions
	exp  map[string]bool
}{
	{
		"empty",
		nil,
		nil,
	},
	{
		"only recovery",
		&tlw.ServodOptions{
			RecoveryMode: true,
		},
		map[string]bool{
			"REC_MODE=1": true,
		},
	},
	{
		"happy path full",
		&tlw.ServodOptions{
			DutBoard:      "my-Board",
			DutModel:      "my-model",
			ServodPort:    63,
			ServoSerial:   "serial-number",
			ServoDual:     true,
			UseCr50Config: true,
			RecoveryMode:  true,
		},
		map[string]bool{
			"REC_MODE=1":           true,
			"PORT=63":              true,
			"BOARD=my-Board":       true,
			"MODEL=my-model":       true,
			"SERIAL=serial-number": true,
			"DUAL_V4=1":            true,
			"CONFIG=cr50.xml":      true,
		},
	},
	{
		"happy path simple",
		&tlw.ServodOptions{
			DutBoard:    "my-super-Board",
			DutModel:    "my-model2",
			ServodPort:  85,
			ServoSerial: "serial-number3",
		},
		map[string]bool{
			"PORT=85":               true,
			"BOARD=my-super-Board":  true,
			"MODEL=my-model2":       true,
			"SERIAL=serial-number3": true,
		},
	},
}

// Verify that we generate the same set of data
func TestGenerateParams(t *testing.T) {
	t.Parallel()
	for _, c := range generateParamsTestCases {
		cs := c
		t.Run(cs.name, func(t *testing.T) {
			got := GenerateParams(cs.got)
			if len(got) != len(cs.exp) {
				t.Errorf("%q -> size of result is mot match, got: %v, exp: %v", cs.name, got, cs.exp)
			} else {
				vmap := make(map[string]bool)
				for _, v := range got {
					if !cs.exp[v] {
						t.Errorf("%q -> did not found %q in expected set %v", cs.name, v, cs.exp)
					}
					vmap[v] = true
				}
				for k := range cs.exp {
					if !vmap[k] {
						t.Errorf("%q -> expected %q but not received", cs.name, k)
					}
				}
			}
		})
	}
}
