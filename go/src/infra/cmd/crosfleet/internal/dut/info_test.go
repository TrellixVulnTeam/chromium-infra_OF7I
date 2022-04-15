// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"fmt"
	dutinfopb "infra/cmd/crosfleet/internal/proto"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufslabpb "infra/unifiedfleet/api/v1/models/chromeos/lab"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var testDutInfoAsBashVariablesData = []struct {
	info         *dutinfopb.DUTInfo
	wantBashVars string
}{
	{ // All variables found
		&dutinfopb.DUTInfo{
			Hostname: "sample-dut-hostname",
			Machine: &ufspb.Machine{Device: &ufspb.Machine_ChromeosMachine{
				ChromeosMachine: &ufspb.ChromeOSMachine{
					BuildTarget: "sample-board",
					Model:       "sample-model",
				},
			}},
			LabSetup: &ufspb.MachineLSE{Lse: &ufspb.MachineLSE_ChromeosMachineLse{
				ChromeosMachineLse: &ufspb.ChromeOSMachineLSE{ChromeosLse: &ufspb.ChromeOSMachineLSE_DeviceLse{
					DeviceLse: &ufspb.ChromeOSDeviceLSE{Device: &ufspb.ChromeOSDeviceLSE_Dut{
						Dut: &ufslabpb.DeviceUnderTest{Peripherals: &ufslabpb.Peripherals{
							Servo: &ufslabpb.Servo{
								ServoHostname: "sample-servo-hostname",
								ServoPort:     12345,
								ServoSerial:   "sample-serial",
							},
						}},
					}},
				}},
			}},
		},
		`DUT_HOSTNAME=sample-dut-hostname.cros.corp.google.com
MODEL=sample-model
BOARD=sample-board
SERVO_HOSTNAME=sample-servo-hostname.cros.corp.google.com
SERVO_PORT=12345
SERVO_SERIAL=sample-serial`,
	},
	{ // One variable found
		&dutinfopb.DUTInfo{
			Hostname: "sample-dut-hostname",
		},
		"DUT_HOSTNAME=sample-dut-hostname.cros.corp.google.com",
	},
	{ // No variables found
		&dutinfopb.DUTInfo{},
		"",
	},
}

func TestDutInfoAsBashVariables(t *testing.T) {
	t.Parallel()
	for _, tt := range testDutInfoAsBashVariablesData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantBashVars), func(t *testing.T) {
			t.Parallel()
			gotBashVars := dutInfoAsBashVariables(tt.info)
			if diff := cmp.Diff(tt.wantBashVars, gotBashVars); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
		})
	}
}
