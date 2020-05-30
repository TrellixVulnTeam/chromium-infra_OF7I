// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dutprep

import (
	"reflect"
	"testing"
)

func TestSortActions(t *testing.T) {
	var cases = []struct {
		in   []Action
		want []Action
	}{
		{
			in:   []Action{StageUSB, InstallTestImage, InstallFirmware, RunPreDeployVerification, VerifyRecoveryMode, SetupLabstation, UpdateLabel},
			want: []Action{StageUSB, InstallTestImage, InstallFirmware, RunPreDeployVerification, VerifyRecoveryMode, SetupLabstation, UpdateLabel},
		},
		{
			in:   []Action{InstallTestImage, SetupLabstation, InstallFirmware, UpdateLabel, VerifyRecoveryMode, RunPreDeployVerification, StageUSB},
			want: []Action{StageUSB, InstallTestImage, InstallFirmware, RunPreDeployVerification, VerifyRecoveryMode, SetupLabstation, UpdateLabel},
		},
		{
			in:   []Action{InstallTestImage, StageUSB},
			want: []Action{StageUSB, InstallTestImage},
		},
	}

	for _, c := range cases {
		got := SortActions(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Incorrect SortActions(%v): got %v, want %v", c.in, got, c.want)
		}
	}
}
