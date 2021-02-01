// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHostInfoStorePath(t *testing.T) {
	t.Parallel()

	data := []struct {
		name       string
		controlDir string
		hostname   string
		out        string
	}{
		{
			"good hostname",
			"/tr",
			"FAKE-HOSTNAME",
			"/tr/host_info_store/FAKE-HOSTNAME.store",
		},
		{
			"no control dir",
			"",
			"FAKE-HOSTNAME",
			"",
		},
		{
			"no hostname",
			"/tr",
			"",
			"",
		},
		{
			"no hostname or control dir",
			"",
			"",
			"",
		},
	}

	for _, subtest := range data {
		t.Run(subtest.name, func(t *testing.T) {
			wanted := subtest.out
			got, _ := hostInfoStorePath(subtest.controlDir, subtest.hostname)
			if diff := cmp.Diff(wanted, got); diff != "" {
				t.Errorf("wanted: (%s) got: (%s)\n(%s)", wanted, got, diff)
			}
		})
	}
}
