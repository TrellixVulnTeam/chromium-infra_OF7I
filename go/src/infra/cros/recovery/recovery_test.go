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
	"infra/cros/recovery/tasknames"
	"infra/cros/recovery/tlw"
)

// Test cases for TestDUTPlans
var dutPlansCases = []struct {
	name         string
	setupType    tlw.DUTSetupType
	taskName     tasknames.TaskName
	expPlanNames []string
}{
	{
		"default no task",
		tlw.DUTSetupTypeDefault,
		tasknames.TaskName(""),
		nil,
	},
	{
		"default recovery",
		tlw.DUTSetupTypeDefault,
		tasknames.Recovery,
		nil,
	},
	{
		"default deploy",
		tlw.DUTSetupTypeDefault,
		tasknames.Deploy,
		nil,
	},
	{
		"default custom",
		tlw.DUTSetupTypeDefault,
		tasknames.Custom,
		nil,
	},
	{
		"cros no task",
		tlw.DUTSetupTypeCros,
		tasknames.TaskName(""),
		nil,
	},
	{
		"cros recovery",
		tlw.DUTSetupTypeCros,
		tasknames.Recovery,
		[]string{"servo", "cros", "chameleon", "bluetooth_peer", "wifi_router", "close"},
	},
	{
		"cros deploy",
		tlw.DUTSetupTypeCros,
		tasknames.Deploy,
		[]string{"servo", "cros", "chameleon", "bluetooth_peer", "wifi_router", "close"},
	},
	{
		"cros custom",
		tlw.DUTSetupTypeCros,
		tasknames.Custom,
		nil,
	},
	{
		"labstation no task",
		tlw.DUTSetupTypeCros,
		tasknames.TaskName(""),
		nil,
	},
	{
		"labstation recovery",
		tlw.DUTSetupTypeLabstation,
		tasknames.Recovery,
		[]string{"cros"},
	},
	{
		"labstation deploy",
		tlw.DUTSetupTypeLabstation,
		tasknames.Deploy,
		[]string{"cros"},
	},
	{
		"labstation custom",
		tlw.DUTSetupTypeLabstation,
		tasknames.Custom,
		nil,
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
			got, err := loadConfiguration(ctx, dut, args)
			if len(cs.expPlanNames) == 0 {
				if err == nil {
					t.Errorf("%q -> expected to finish with error but passed", cs.name)
				}
				if len(got.GetPlanNames()) != 0 {
					t.Errorf("%q -> want: %v\n got: %v", cs.name, cs.expPlanNames, got.GetPlanNames())
				}
			} else {
				if !cmp.Equal(got.GetPlanNames(), cs.expPlanNames) {
					t.Errorf("%q ->want: %v\n got: %v", cs.name, cs.expPlanNames, got.GetPlanNames())
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
			got, err := ParsedDefaultConfiguration(ctx, c.taskName, c.setupType)
			if len(cs.expPlanNames) == 0 {
				if err == nil {
					t.Errorf("%q -> expected to finish with error but passed", cs.name)
				}
				if len(got.GetPlanNames()) != 0 {
					t.Errorf("%q -> want: %v\n got: %v", cs.name, cs.expPlanNames, got.GetPlanNames())
				}
			} else {
				if !cmp.Equal(got.GetPlanNames(), cs.expPlanNames) {
					t.Errorf("%q ->want: %v\n got: %v", cs.name, cs.expPlanNames, got.GetPlanNames())
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

// TestVerify is a smoke test for the verify method.
// TODO(gregorynisbet): Add fake TLW client so we can test a successful path.
func TestVerify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   *RunArgs
		good bool
	}{
		{
			"nil",
			nil,
			false,
		},
		{
			"empty",
			&RunArgs{},
			false,
		},
		{
			"missing tlw client",
			&RunArgs{
				UnitName: "a",
				LogRoot:  "b",
			},
			false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expected := tt.good
			e := tt.in.verify()
			actual := (e == nil)

			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("unexpected diff (-want +got): %s", diff)
			}
		})
	}
}
