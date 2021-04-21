// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package assert contains common assert methods.
package assert

import (
	"strings"
	"testing"
)

// Assert checks that the given bool is true.
func Assert(t *testing.T, b bool) {
	t.Helper()
	if !b {
		t.Fatal("assert failed: bool is false")
	}
}

// NilError checks that the given error is nil.
func NilError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("assert failed: non-nil error %v", err)
	}
}

// ErrorContains checks that the given string exists in an error.
func ErrorContains(t *testing.T, err error, s string) {
	t.Helper()

	if err == nil {
		t.Fatalf("assert failed: nil error")
	}

	if !strings.Contains(err.Error(), s) {
		t.Fatalf("assert failed: %v does not contain \"%s\"", err, s)
	}
}

// BoolsEqual checks that the two bools are equal.
func BoolsEqual(t *testing.T, a, b bool) {
	t.Helper()
	if a != b {
		t.Fatalf("assert failed: %v != %v", a, b)
	}
}

// IntsEqual checks that the two ints are equal.
func IntsEqual(t *testing.T, a, b int) {
	t.Helper()
	if a != b {
		t.Fatalf("assert failed: %d != %d", a, b)
	}
}

// StringsEqual checks that the two strings are equal.
func StringsEqual(t *testing.T, a, b string) {
	t.Helper()
	if a != b {
		t.Fatalf("assert failed: \"%s\" != \"%s\"", a, b)
	}
}
