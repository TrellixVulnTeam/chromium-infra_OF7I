// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/internal/planpb"
	"infra/cros/recovery/logger"
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

func TestRunDUTPlan(t *testing.T) {
	t.Parallel()
	Convey("bad cases", t, func() {
		ctx := context.Background()
		dut := &tlw.Dut{
			Name: "test_dut",
			ServoHost: &tlw.ServoHost{
				Name: "servo_host",
			},
		}
		args := &RunArgs{
			Logger: logger.NewLogger(),
		}
		execArgs := &execs.RunArgs{
			DUT:    dut,
			Logger: args.Logger,
		}
		config := &planpb.Configuration{}
		Convey("fail when no plans in config", func() {
			config.Plans = map[string]*planpb.Plan{
				"something": nil,
			}
			config.PlanNames = []string{"my_plan"}
			err := runDUTPlans(ctx, dut, config, args)
			if err == nil {
				t.Errorf("Expected fail but passed")
			} else {
				So(err.Error(), ShouldContainSubstring, "run dut \"test_dut\" plans:")
				So(err.Error(), ShouldContainSubstring, "not found in configuration")
			}
		})
		Convey("fail when one plan fail of plans fail", func() {
			config.Plans = map[string]*planpb.Plan{
				PlanServoRepair: {
					CriticalActions: []string{"sample_fail"},
					Actions: map[string]*planpb.Action{
						"sample_fail": {
							ExecName: "sample_fail",
						},
					},
				},
				PlanCrOSRepair: {
					CriticalActions: []string{"sample_pass"},
					Actions: map[string]*planpb.Action{
						"sample_pass": {
							ExecName: "sample_pass",
						},
					},
				},
			}
			config.PlanNames = []string{PlanServoRepair, PlanCrOSRepair}
			err := runDUTPlans(ctx, dut, config, args)
			if err == nil {
				t.Errorf("Expected fail but passed")
			} else {
				So(err.Error(), ShouldContainSubstring, "run plan \"servo_repair\" for \"servo_host\":")
				So(err.Error(), ShouldContainSubstring, "failed")
			}
		})
		Convey("fail when bad action in the plan", func() {
			plan := &planpb.Plan{
				CriticalActions: []string{"sample_fail"},
				Actions: map[string]*planpb.Action{
					"sample_fail": {
						ExecName: "sample_fail",
					},
				},
			}
			err := runDUTPlanPerResource(ctx, "test_dut", PlanCrOSRepair, plan, execArgs)
			if err == nil {
				t.Errorf("Expected fail but passed")
			} else {
				So(err.Error(), ShouldContainSubstring, "run plan \"cros_repair\" for \"test_dut\":")
				So(err.Error(), ShouldContainSubstring, ": failed")
			}
		})
	})
	Convey("Happy path", t, func() {
		ctx := context.Background()
		dut := &tlw.Dut{
			Name: "test_dut",
			ServoHost: &tlw.ServoHost{
				Name: "servo_host",
			},
		}
		args := &RunArgs{
			Logger: logger.NewLogger(),
		}
		execArgs := &execs.RunArgs{
			DUT: dut,
		}
		Convey("Run good plan", func() {
			plan := &planpb.Plan{
				CriticalActions: []string{"sample_pass"},
				Actions: map[string]*planpb.Action{
					"sample_pass": {
						ExecName: "sample_pass",
					},
				},
			}
			if err := runDUTPlanPerResource(ctx, "DUT3", PlanCrOSRepair, plan, execArgs); err != nil {
				t.Errorf("Expected pass but failed: %s", err)
			}
		})
		Convey("Run all good plans", func() {
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
					PlanServoRepair: {
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
		Convey("Run all plans even one allow to fail", func() {
			config := &planpb.Configuration{
				Plans: map[string]*planpb.Plan{
					PlanCrOSRepair: {
						CriticalActions: []string{"sample_fail"},
						Actions: map[string]*planpb.Action{
							"sample_fail": {
								ExecName: "sample_fail",
							},
						},
						AllowFail: true,
					},
					PlanServoRepair: {
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
		Convey("Do not fail even if closing plan failed", func() {
			config := &planpb.Configuration{
				Plans: map[string]*planpb.Plan{
					PlanCrOSRepair: {
						CriticalActions: []string{},
					},
					PlanServoRepair: {
						CriticalActions: []string{},
					},
					PlanClosing: {
						CriticalActions: []string{"sample_fail"},
						Actions: map[string]*planpb.Action{
							"sample_fail": {
								ExecName: "sample_fail",
							},
						},
					},
				},
			}
			if err := runDUTPlans(ctx, dut, config, args); err != nil {
				t.Errorf("Expected pass but failed: %s", err)
			}
		})
	})
}

func TestLocalproxyFlag(t *testing.T) {
	if useProxyForSSHAccess {
		t.Errorf("please keep useProxyForSSHAccess as false")
	}
}
