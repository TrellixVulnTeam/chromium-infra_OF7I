// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmdlib

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var testFixSuspiciousHostnameData = []struct {
	in  string
	out string
}{
	{
		"crossk-chromeos6-row1-rack13-host1",
		"chromeos6-row1-rack13-host1",
	},
	{
		"chromeos6-row1-rack13-host1.cros",
		"chromeos6-row1-rack13-host1",
	},
	{
		"chromeos6-row1-rack13-host1",
		"chromeos6-row1-rack13-host1",
	},
	{
		"",
		"",
	},
}

func TestFixSuspiciousHostname(t *testing.T) {
	t.Parallel()
	for _, tt := range testFixSuspiciousHostnameData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.in), func(t *testing.T) {
			t.Parallel()
			actual := FixSuspiciousHostname(tt.in)
			diff := cmp.Diff(tt.out, actual)
			if diff != "" {
				msg := fmt.Sprintf("unexpected diff (%s)", diff)
				t.Errorf(msg)
			}
		})
	}
}
