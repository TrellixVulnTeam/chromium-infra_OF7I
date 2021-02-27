// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"fmt"
	"testing"
)

var testCorrectedHostnameData = []struct {
	startingHostname, wantCorrectedHostname string
}{
	{
		"crossk-foo-hostname.cros",
		"foo-hostname",
	},
	{
		"crossk-bar-hostname",
		"bar-hostname",
	},
	{
		"baz-hostname.cros",
		"baz-hostname",
	},
	{
		"lol-hostname",
		"lol-hostname",
	},
}

func TestCorrectedHostname(t *testing.T) {
	t.Parallel()
	for _, tt := range testCorrectedHostnameData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.startingHostname), func(t *testing.T) {
			t.Parallel()
			gotCorrectedHostname := correctedHostname(tt.startingHostname)
			if tt.wantCorrectedHostname != gotCorrectedHostname {
				t.Errorf("unexpected error: wanted '%s', got '%s'", tt.wantCorrectedHostname, gotCorrectedHostname)
			}
		})
	}
}
