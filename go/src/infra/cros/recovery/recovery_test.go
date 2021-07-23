// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"

	"infra/cros/recovery/internal/planpb"
	"infra/cros/recovery/tlw"
)

// Test cases for TestDUTPlans
var dutPlansCases = []struct {
	name     string
	dut      *tlw.Dut
	exp      []string
	taskName TaskName
}{
	{
		"default no servo",
		&tlw.Dut{
			SetupType:          tlw.DUTSetupTypeDefault,
			ServoHost:          &tlw.ServoHost{},
			ChameleonHost:      &tlw.ChameleonHost{},
			BluetoothPeerHosts: []*tlw.BluetoothPeerHost{},
		},
		[]string{"cros_repair"},
		"",
	},
	{
		"default with servo",
		&tlw.Dut{
			SetupType:          tlw.DUTSetupTypeDefault,
			ServoHost:          &tlw.ServoHost{Name: "Servo"},
			ChameleonHost:      &tlw.ChameleonHost{},
			BluetoothPeerHosts: []*tlw.BluetoothPeerHost{},
		},
		[]string{"servo_repair", "cros_repair"},
		"",
	},
	{
		"default with servo and chameleon",
		&tlw.Dut{
			SetupType:          tlw.DUTSetupTypeDefault,
			ServoHost:          &tlw.ServoHost{Name: "Servo"},
			ChameleonHost:      &tlw.ChameleonHost{Name: "Chameleon"},
			BluetoothPeerHosts: []*tlw.BluetoothPeerHost{},
		},
		[]string{"servo_repair", "chameleon_repair", "cros_repair"},
		"",
	},
	{
		"default with servo,chameleon and bluetoothPeer",
		&tlw.Dut{
			SetupType:     tlw.DUTSetupTypeDefault,
			ServoHost:     &tlw.ServoHost{Name: "Servo"},
			ChameleonHost: &tlw.ChameleonHost{Name: "Chameleon"},
			BluetoothPeerHosts: []*tlw.BluetoothPeerHost{
				{
					Name: "bp",
				},
			},
		},
		[]string{"servo_repair", "chameleon_repair", "bluetooth_peer_repair", "cros_repair"},
		"",
	},
	{
		"labstation",
		&tlw.Dut{
			SetupType:          tlw.DUTSetupTypeLabstation,
			ServoHost:          &tlw.ServoHost{},
			ChameleonHost:      &tlw.ChameleonHost{},
			BluetoothPeerHosts: []*tlw.BluetoothPeerHost{},
		},
		[]string{"labstation_repair"},
		"",
	},
	{
		"deploy default no servo",
		&tlw.Dut{
			SetupType:          tlw.DUTSetupTypeDefault,
			ServoHost:          &tlw.ServoHost{},
			ChameleonHost:      &tlw.ChameleonHost{},
			BluetoothPeerHosts: []*tlw.BluetoothPeerHost{},
		},
		[]string{"cros_deploy"},
		TaskNameDeploy,
	},
	{
		"default with servo",
		&tlw.Dut{
			SetupType:     tlw.DUTSetupTypeDefault,
			ServoHost:     &tlw.ServoHost{Name: "Servo"},
			ChameleonHost: &tlw.ChameleonHost{Name: "Chameleon"},
			BluetoothPeerHosts: []*tlw.BluetoothPeerHost{
				{
					Name: "bp",
				},
			},
		},
		[]string{"servo_repair", "cros_deploy"},
		TaskNameDeploy,
	},
	{
		"deploy labstation",
		&tlw.Dut{
			SetupType:          tlw.DUTSetupTypeLabstation,
			ServoHost:          &tlw.ServoHost{},
			ChameleonHost:      &tlw.ChameleonHost{},
			BluetoothPeerHosts: []*tlw.BluetoothPeerHost{},
		},
		[]string{"labstation_deploy"},
		TaskNameDeploy,
	},
}

// Testing dutPlans method
func TestDUTPlans(t *testing.T) {
	t.Parallel()
	for _, c := range dutPlansCases {
		cs := c
		t.Run(cs.name, func(t *testing.T) {
			args := &RunArgs{}
			if c.taskName != "" {
				args.TaskName = c.taskName
			}
			got := dutPlans(cs.dut, args)
			if !cmp.Equal(got, cs.exp) {
				t.Errorf("got: %v\nwant: %v", got, cs.exp)
			}
		})
	}
}

func TestRunDUTPlans(t *testing.T) {
	t.Parallel()
	Convey("bad cases", t, func() {
		ctx := context.Background()
		dut := &tlw.Dut{
			Name: "test_dut",
		}
		args := &RunArgs{}
		config := &planpb.Configuration{}
		Convey("fail when no plans in config", func() {
			config.Plans = map[string]*planpb.Plan{
				"something": nil,
			}
			err := runDUTPlans(ctx, dut, config, args)
			if err == nil {
				t.Errorf("Expected fail but passed")
			}
			So(err.Error(), ShouldContainSubstring, "run dut \"test_dut\" plans:")
			So(err.Error(), ShouldContainSubstring, "not found in configuration")
			// So("jk", ShouldContainSubstring, "j")
		})
		Convey("fail when bad action in the plan", func() {
			config.Plans = map[string]*planpb.Plan{
				PlanCrOSRepair: {
					CriticalActions: []string{"sample_fail"},
					Actions: map[string]*planpb.Action{
						"sample_fail": {
							ExecName: "sample_fail",
						},
					},
				},
			}
			err := runDUTPlans(ctx, dut, config, args)
			if err == nil {
				t.Errorf("Expected fail but passed")
			}
			So(err.Error(), ShouldContainSubstring, "run plan \"cros_repair\" for \"test_dut\":")
			So(err.Error(), ShouldContainSubstring, ": failed")
		})
	})
	Convey("Happy path", t, func() {
		ctx := context.Background()
		dut := &tlw.Dut{
			Name: "test_dut",
		}
		args := &RunArgs{}
		config := &planpb.Configuration{
			Plans: map[string]*planpb.Plan{
				PlanCrOSRepair: {
					CriticalActions: []string{"sample_pass"},
					Actions: map[string]*planpb.Action{
						"sample_pass": {
							ExecName: "sample_pass",
						},
					},
				},
			},
		}
		if err := runDUTPlans(ctx, dut, config, args); err != nil {
			t.Errorf("Expected pass but failed: %s", err)
		}
	})
}
