// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestToKeyvalSlice(t *testing.T) {
	wantSlice := []string{"foo:bar"}
	gotSlice := ToKeyvalSlice(map[string]string{
		"foo": "bar",
	})
	if diff := cmp.Diff(wantSlice, gotSlice); diff != "" {
		t.Errorf("unexpected diff (%s)", diff)
	}
}
