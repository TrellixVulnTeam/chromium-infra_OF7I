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
	}{
		{
			"all",
			auditRun{},
			"verify-dut-storage,verify-servo-usb-drive",
		},
		{
			"all-2",
			auditRun{
				runVerifyServoUSB:   true,
				runVerifyDUTStorage: true,
			},
			"verify-dut-storage,verify-servo-usb-drive",
		},
		{
			"runVerifyServoUSB",
			auditRun{
				runVerifyServoUSB: true,
			},
			"verify-servo-usb-drive",
		},
		{
			"runVerifyDUTStorage",
			auditRun{
				runVerifyDUTStorage: true,
			},
			"verify-dut-storage",
		},
	}
	for _, tt := range actionsCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.in.actions()
			if err != nil {
				t.Errorf("%q got err %s\n", tt.name, err)
			}
			if diff := cmp.Diff(tt.out, got); diff != "" {
				t.Errorf("%q output mismatch (-want +got): %s\n", tt.name, diff)
			}
		})
	}
}
