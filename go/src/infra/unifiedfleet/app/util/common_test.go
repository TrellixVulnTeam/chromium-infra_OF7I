// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"strings"
	"testing"
)

func TestValidLabs(t *testing.T) {
	for _, lab := range validLabs {
		s := strings.Split(lab, "-")
		if len(s) != 2 || s[1] != "lab" {
			t.Errorf("%s in validLabs needs to be in a format of XXX-lab: %#v", lab, s)
		}
	}
}

func TestValidLabFilter(t *testing.T) {
	toTest := []string{"atl", "acs", "browser", "cros"}
	for _, test := range toTest {
		if !IsValidFilter(test) {
			t.Errorf("%s is a valid lab", test)
		}
	}
}
