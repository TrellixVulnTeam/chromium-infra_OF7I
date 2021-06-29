// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"infra/cros/recovery/tlw"
)

// Test cases for TestDUTPlans
var dutPlansCases = []struct {
	name string
	dut  *tlw.Dut
	exp  []string
}{
	{
		"default no servo",
		&tlw.Dut{
			SetupType: tlw.DUTSetupTypeDefault,
		},
		[]string{PlanRepairDUT},
	},
	{
		"default with servo",
		&tlw.Dut{
			SetupType: tlw.DUTSetupTypeDefault,
			ServoHost: &tlw.ServoHost{},
		},
		[]string{PlanRepairServo, PlanRepairDUT},
	},
	{
		"labstation",
		&tlw.Dut{
			SetupType: tlw.DUTSetupTypeLabstation,
		},
		[]string{PlanRepairLabstation},
	},
	{
		"jetstream",
		&tlw.Dut{
			SetupType: tlw.DUTSetupTypeJetstream,
		},
		[]string{PlanRepairServo, PlanRepairJetstream},
	},
}

// Testing dutPlans method
func TestDUTPlans(t *testing.T) {
	for _, c := range dutPlansCases {
		cs := c
		t.Run(cs.name, func(t *testing.T) {
			t.Parallel()
			got := dutPlans(cs.dut)
			if !cmp.Equal(got, cs.exp) {
				t.Errorf("got: %v\nwant: %v", got, cs.exp)
			}
		})
	}
}
