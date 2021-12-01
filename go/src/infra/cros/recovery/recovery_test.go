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
	name      string
	setupType tlw.DUTSetupType
	taskName  TaskName
	exp       []string
}{
	{
		"default no task",
		tlw.DUTSetupTypeDefault,
		TaskName(""),
		nil,
	},
	{
		"default recovery",
		tlw.DUTSetupTypeDefault,
		TaskNameRecovery,
		nil,
	},
	{
		"default deploy",
		tlw.DUTSetupTypeDefault,
		TaskNameDeploy,
		nil,
	},
	{
		"cros no task",
		tlw.DUTSetupTypeCros,
		TaskName(""),
		nil,
	},
	{
		"cros recovery",
		tlw.DUTSetupTypeCros,
		TaskNameRecovery,
		[]string{"servo", "cros", "chameleon", "bluetooth_peer", "close"},
	},
	{
		"cros deploy",
		tlw.DUTSetupTypeCros,
		TaskNameDeploy,
		[]string{"servo", "cros", "chameleon", "bluetooth_peer", "close"},
	},
	{
		"labstation no task",
		tlw.DUTSetupTypeCros,
		TaskName(""),
		nil,
	},
	{
		"labstation recovery",
		tlw.DUTSetupTypeLabstation,
		TaskNameRecovery,
		[]string{"cros"},
	},
	{
		"labstation deploy",
		tlw.DUTSetupTypeLabstation,
		TaskNameDeploy,
		[]string{"cros"},
	},
}

// TestLoadConfiguration tests default configuration used for recovery flow is loading right and parsibale without any issue.
//
// Goals:
//  1) Parsed without any issue
//  2) plan using only existing execs
//  3) configuration contain all required plans in order.
func TestLoadConfiguration(t *testing.T) {
	t.Parallel()
	for _, c := range dutPlansCases {
		cs := c
		t.Run(cs.name, func(t *testing.T) {
			ctx := context.Background()
			args := &RunArgs{}
			if c.taskName != "" {
				args.TaskName = c.taskName
			}
			dut := &tlw.Dut{SetupType: c.setupType}
			got, _ := loadConfiguration(ctx, dut, args)
			if len(cs.exp) == 0 {
				if len(got.GetPlanNames()) != 0 {
					t.Errorf("%q -> want: %v\n got: %v", cs.name, cs.exp, got.GetPlanNames())
				}

			} else {
				if !cmp.Equal(got.GetPlanNames(), cs.exp) {
					t.Errorf("%q ->want: %v\n got: %v", cs.name, cs.exp, got.GetPlanNames())
				}
			}
		})
	}
}

// TestParsedDefaultConfiguration tests default configurations are loading right and parsibale without any issue.
//
// Goals:
//  1) Parsed without any issue
//  2) plan using only existing execs
//  3) configuration contain all required plans in order.
func TestParsedDefaultConfiguration(t *testing.T) {
	t.Parallel()
	for _, c := range dutPlansCases {
		cs := c
		t.Run(cs.name, func(t *testing.T) {
			ctx := context.Background()
			got, _ := ParsedDefaultConfiguration(ctx, c.taskName, c.setupType)
			if len(cs.exp) == 0 {
				if len(got.GetPlanNames()) != 0 {
					t.Errorf("%q -> want: %v\n got: %v", cs.name, cs.exp, got.GetPlanNames())
				}

			} else {
				if !cmp.Equal(got.GetPlanNames(), cs.exp) {
					t.Errorf("%q ->want: %v\n got: %v", cs.name, cs.exp, got.GetPlanNames())
				}
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
				PlanServo: {
					CriticalActions: []string{"sample_fail"},
					Actions: map[string]*planpb.Action{
						"sample_fail": {
							ExecName: "sample_fail",
						},
					},
				},
				PlanCrOS: {
					CriticalActions: []string{"sample_pass"},
					Actions: map[string]*planpb.Action{
						"sample_pass": {
							ExecName: "sample_pass",
						},
					},
				},
			}
			config.PlanNames = []string{PlanServo, PlanCrOS}
			err := runDUTPlans(ctx, dut, config, args)
			if err == nil {
				t.Errorf("Expected fail but passed")
			} else {
				So(err.Error(), ShouldContainSubstring, "run plan \"servo\" for \"servo_host\":")
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
			err := runDUTPlanPerResource(ctx, "test_dut", PlanCrOS, plan, execArgs)
			if err == nil {
				t.Errorf("Expected fail but passed")
			} else {
				So(err.Error(), ShouldContainSubstring, "run plan \"cros\" for \"test_dut\":")
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
			if err := runDUTPlanPerResource(ctx, "DUT3", PlanCrOS, plan, execArgs); err != nil {
				t.Errorf("Expected pass but failed: %s", err)
			}
		})
		Convey("Run all good plans", func() {
			config := &planpb.Configuration{
				Plans: map[string]*planpb.Plan{
					PlanCrOS: {
						CriticalActions: []string{"sample_pass"},
						Actions: map[string]*planpb.Action{
							"sample_pass": {
								ExecName: "sample_pass",
							},
						},
					},
					PlanServo: {
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
					PlanCrOS: {
						CriticalActions: []string{"sample_fail"},
						Actions: map[string]*planpb.Action{
							"sample_fail": {
								ExecName: "sample_fail",
							},
						},
						AllowFail: true,
					},
					PlanServo: {
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
					PlanCrOS: {
						CriticalActions: []string{},
					},
					PlanServo: {
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
