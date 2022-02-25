// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package servo

import (
	"context"
	"strings"
	"testing"

	"go.chromium.org/luci/common/errors"

	"infra/cros/recovery/internal/execs"
	"infra/cros/recovery/tlw"
)

var setServoStateExecTestCases = []struct {
	testName           string
	actionArgs         []string
	expectedServoState tlw.ServoState
	servoHost          *tlw.ServoHost
	expectedErr        error
}{
	{
		"success: set servo state to SBU_LOW_VOLTAGE",
		[]string{
			"state:SBU_LOW_VOLTAGE",
		},
		tlw.ServoStateSBULowVoltage,
		&tlw.ServoHost{
			Servo: &tlw.Servo{
				State: tlw.ServoStateUnspecified,
			},
		},
		nil,
	},
	{
		"fail: missing state info found",
		[]string{
			"test:SBU_LOW_VOLTAGE",
		},
		tlw.ServoStateUnspecified,
		&tlw.ServoHost{
			Servo: &tlw.Servo{
				State: tlw.ServoStateUnspecified,
			},
		},
		errors.Reason("set servo state: state is not provided").Err(),
	},
	{
		"fail: state info is empty",
		[]string{
			"state:",
		},
		tlw.ServoStateUnspecified,
		&tlw.ServoHost{
			Servo: &tlw.Servo{
				State: tlw.ServoStateUnspecified,
			},
		},
		errors.Reason("set servo state: state is not provided").Err(),
	},
	{
		"fail: state info in wrong format",
		[]string{
			"state:sbu_LOW_VOLTAGE",
		},
		tlw.ServoStateSBULowVoltage,
		&tlw.ServoHost{
			Servo: &tlw.Servo{
				State: tlw.ServoStateUnspecified,
			},
		},
		nil,
	},
	{
		"fail: do not update if servo is not supported in structure",
		[]string{
			"state:sbu_LOW_VOLTAGE",
		},
		tlw.ServoStateSBULowVoltage,
		nil,
		errors.Reason("set servo state: servo is not supported").Err(),
	},
}

func TestSetServoStateExec(t *testing.T) {
	t.Parallel()
	for _, tt := range setServoStateExecTestCases {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			args := &execs.RunArgs{
				DUT: &tlw.Dut{ServoHost: tt.servoHost},
			}
			info := &execs.ExecInfo{
				RunArgs:    args,
				ActionArgs: tt.actionArgs,
			}
			actualErr := setServoStateExec(ctx, info)
			if actualErr != nil && tt.expectedErr != nil {
				if !strings.Contains(actualErr.Error(), tt.expectedErr.Error()) {
					t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
				}
			}
			if (actualErr == nil && tt.expectedErr != nil) || (actualErr != nil && tt.expectedErr == nil) {
				t.Errorf("Expected error %q, but got %q", tt.expectedErr, actualErr)
			}
			if tt.servoHost != nil {
				actualServoState := tt.servoHost.Servo.State
				if actualServoState != tt.expectedServoState {
					t.Errorf("Expected servo state %q, but got %q", tt.expectedServoState, actualServoState)
				}
			}
		})
	}
}
