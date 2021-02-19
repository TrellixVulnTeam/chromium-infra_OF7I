// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package common

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestToKeyvalSlice(t *testing.T) {
	wantSlice := []string{"foo:bar", "baz:lol"}
	gotSlice := ToKeyvalSlice(map[string]string{
		"foo": "bar",
		"baz": "lol",
	})
	if diff := cmp.Diff(wantSlice, gotSlice); diff != "" {
		t.Errorf("unexpected diff (%s)", diff)
	}
}
