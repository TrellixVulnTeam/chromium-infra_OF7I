// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package tasks

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestActions(t *testing.T) {
	t.Parallel()
	actionsCases := []struct {
		name string
		in   auditRun
		out  string
		err  bool
	}{
		{
			"default",
			auditRun{},
			"",
			true,
		},
		{
			"all-2",
			auditRun{
				runVerifyServoUSB:   true,
				runVerifyDUTStorage: true,
				runVerifyServoFw:    true,
			},
			"verify-dut-storage,verify-servo-usb-drive,verify-servo-fw",
			false,
		},
		{
			"runVerifyServoUSB",
			auditRun{
				runVerifyServoUSB: true,
			},
			"verify-servo-usb-drive",
			false,
		},
		{
			"runVerifyDUTStorage",
			auditRun{
				runVerifyDUTStorage: true,
			},
			"verify-dut-storage",
			false,
		},
		{
			"runVerifyServoFw",
			auditRun{
				runVerifyServoFw: true,
			},
			"verify-servo-fw",
			false,
		},
		{
			"runFlashServoKeyboardMap",
			auditRun{
				runFlashServoKeyboardMap: true,
			},
			"flash-servo-keyboard-map",
			false,
		},
	}
	for _, tt := range actionsCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.in.actions()
			if err != nil {
				if !tt.err {
					t.Errorf("%q got err %s\n", tt.name, err)
				}
			} else if diff := cmp.Diff(tt.out, got); diff != "" {
				t.Errorf("%q output mismatch (-want +got): %s\n", tt.name, diff)
			}
		})
	}
}
