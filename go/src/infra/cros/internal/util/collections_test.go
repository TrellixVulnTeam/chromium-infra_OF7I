// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package util

import (
	"testing"
)

func TestUnorderedEqual(t *testing.T) {
	a := []string{"a", "b", "c", "a"}
	b := []string{"b", "c", "a", "a"}
	c := []string{"a", "b", "b", "c"}
	if !UnorderedEqual(a, b) {
		t.Fatalf("UnorderedEqual: got false, expected true")
	}
	if UnorderedEqual(a, c) {
		t.Fatalf("UnorderedEqual: got true, expected false")
	}
}

func TestUnorderedContains(t *testing.T) {
	a := []string{"a", "b", "c", "a"}
	b := []string{"b", "c"}
	c := []string{"b", "d"}

	if !UnorderedContains(a, b) {
		t.Fatalf("UnorderedEqual: got false, expected true")
	}
	if UnorderedContains(a, c) {
		t.Fatalf("UnorderedEqual: got true, expected false")
	}
}
