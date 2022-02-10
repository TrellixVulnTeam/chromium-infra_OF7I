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
	expectedErr        error
}{
	{
		"success: set servo state to SBU_LOW_VOLTAGE",
		[]string{
			"state:SBU_LOW_VOLTAGE",
		},
		tlw.ServoStateSBULowVoltage,
		nil,
	},
	{
		"fail: missing state info found",
		[]string{
			"test:SBU_LOW_VOLTAGE",
		},
		tlw.ServoStateUnspecified,
		errors.Reason("set servo state: missing servo state information in the argument").Err(),
	},
	{
		"fail: state info is empty",
		[]string{
			"state:",
		},
		tlw.ServoStateUnspecified,
		errors.Reason("set servo state: the servo state string is empty").Err(),
	},
	{
		"fail: state info in wrong format",
		[]string{
			"state:sbu_LOW_VOLTAGE",
		},
		tlw.ServoStateUnspecified,
		errors.Reason("set servo state: the servo state string is in wrong format").Err(),
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
				DUT: &tlw.Dut{
					ServoHost: &tlw.ServoHost{
						Servo: &tlw.Servo{
							State: tlw.ServoStateUnspecified,
						},
					},
				},
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
			actualServoState := info.RunArgs.DUT.ServoHost.Servo.State
			if actualServoState != tt.expectedServoState {
				t.Errorf("Expected servo state %q, but got %q", tt.expectedServoState, actualServoState)
			}
		})
	}
}
