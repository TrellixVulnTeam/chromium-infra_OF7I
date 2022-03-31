// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package recovery

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"

	"infra/cros/recovery/config"
	"infra/cros/recovery/internal/execs"
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
		tlw.DUTSetupTypeUnspecified,
		tasknames.TaskName(""),
		nil,
	},
	{
		"default recovery",
		tlw.DUTSetupTypeUnspecified,
		tasknames.Recovery,
		nil,
	},
	{
		"default deploy",
		tlw.DUTSetupTypeUnspecified,
		tasknames.Deploy,
		nil,
	},
	{
		"default custom",
		tlw.DUTSetupTypeUnspecified,
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
	{
		"android no task",
		tlw.DUTSetupTypeAndroid,
		tasknames.TaskName(""),
		nil,
	},
	{
		"android recovery",
		tlw.DUTSetupTypeAndroid,
		tasknames.Recovery,
		[]string{"android"},
	},
	{
		"android deploy",
		tlw.DUTSetupTypeAndroid,
		tasknames.Deploy,
		[]string{"android"},
	},
	{
		"android custom",
		tlw.DUTSetupTypeAndroid,
		tasknames.Custom,
		nil,
	},
	{
		"android no task",
		tlw.DUTSetupTypeAndroid,
		tasknames.TaskName(""),
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
		c := &config.Configuration{}
		Convey("fail when no plans in config", func() {
			c.Plans = map[string]*config.Plan{
				"something": nil,
			}
			c.PlanNames = []string{"my_plan"}
			err := runDUTPlans(ctx, dut, c, args)
			if err == nil {
				t.Errorf("Expected fail but passed")
			} else {
				So(err.Error(), ShouldContainSubstring, "run dut \"test_dut\" plans:")
				So(err.Error(), ShouldContainSubstring, "not found in configuration")
			}
		})
		Convey("fail when one plan fail of plans fail", func() {
			c.Plans = map[string]*config.Plan{
				config.PlanServo: {
					CriticalActions: []string{"sample_fail"},
					Actions: map[string]*config.Action{
						"sample_fail": {
							ExecName: "sample_fail",
						},
					},
				},
				config.PlanCrOS: {
					CriticalActions: []string{"sample_pass"},
					Actions: map[string]*config.Action{
						"sample_pass": {
							ExecName: "sample_pass",
						},
					},
				},
			}
			c.PlanNames = []string{config.PlanServo, config.PlanCrOS}
			err := runDUTPlans(ctx, dut, c, args)
			if err == nil {
				t.Errorf("Expected fail but passed")
			} else {
				So(err.Error(), ShouldContainSubstring, "run plan \"servo\" for \"servo_host\":")
				So(err.Error(), ShouldContainSubstring, "failed")
			}
		})
		Convey("fail when bad action in the plan", func() {
			plan := &config.Plan{
				CriticalActions: []string{"sample_fail"},
				Actions: map[string]*config.Action{
					"sample_fail": {
						ExecName: "sample_fail",
					},
				},
			}
			err := runDUTPlanPerResource(ctx, "test_dut", config.PlanCrOS, plan, execArgs)
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
			plan := &config.Plan{
				CriticalActions: []string{"sample_pass"},
				Actions: map[string]*config.Action{
					"sample_pass": {
						ExecName: "sample_pass",
					},
				},
			}
			if err := runDUTPlanPerResource(ctx, "DUT3", config.PlanCrOS, plan, execArgs); err != nil {
				t.Errorf("Expected pass but failed: %s", err)
			}
		})
		Convey("Run all good plans", func() {
			c := &config.Configuration{
				Plans: map[string]*config.Plan{
					config.PlanCrOS: {
						CriticalActions: []string{"sample_pass"},
						Actions: map[string]*config.Action{
							"sample_pass": {
								ExecName: "sample_pass",
							},
						},
					},
					config.PlanServo: {
						CriticalActions: []string{"sample_pass"},
						Actions: map[string]*config.Action{
							"sample_pass": {
								ExecName: "sample_pass",
							},
						},
					},
				},
			}
			if err := runDUTPlans(ctx, dut, c, args); err != nil {
				t.Errorf("Expected pass but failed: %s", err)
			}
		})
		Convey("Run all plans even one allow to fail", func() {
			c := &config.Configuration{
				Plans: map[string]*config.Plan{
					config.PlanCrOS: {
						CriticalActions: []string{"sample_fail"},
						Actions: map[string]*config.Action{
							"sample_fail": {
								ExecName: "sample_fail",
							},
						},
						AllowFail: true,
					},
					config.PlanServo: {
						CriticalActions: []string{"sample_pass"},
						Actions: map[string]*config.Action{
							"sample_pass": {
								ExecName: "sample_pass",
							},
						},
					},
				},
			}
			if err := runDUTPlans(ctx, dut, c, args); err != nil {
				t.Errorf("Expected pass but failed: %s", err)
			}
		})
		Convey("Do not fail even if closing plan failed", func() {
			c := &config.Configuration{
				Plans: map[string]*config.Plan{
					config.PlanCrOS: {
						CriticalActions: []string{},
					},
					config.PlanServo: {
						CriticalActions: []string{},
					},
					config.PlanClosing: {
						CriticalActions: []string{"sample_fail"},
						Actions: map[string]*config.Action{
							"sample_fail": {
								ExecName: "sample_fail",
							},
						},
					},
				},
			}
			if err := runDUTPlans(ctx, dut, c, args); err != nil {
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
