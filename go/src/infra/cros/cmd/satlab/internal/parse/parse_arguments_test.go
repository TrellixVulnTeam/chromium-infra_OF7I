// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package parse

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestClassifyFlag tests whether unary, equality, and non-flags are classified correctly.
func TestClassifyFlag(t *testing.T) {
	t.Parallel()
	args := []string{"a", "-b", "c", "-d=e"}
	if cl := classifyFlag(args, 0, nil); cl != nonFlag {
		t.Errorf("wrong classification for non-flag: %d", cl)
	}
	if cl := classifyFlag(args, 1, nil); cl != unaryFlag {
		t.Errorf("wrong classification for unary flag: %d", cl)
	}
	if cl := classifyFlag(args, 2, nil); cl != nonFlag {
		t.Errorf("wrong classification for non-flag: %d", cl)
	}
}

// TestParseArguments tests arguments provided to the satlab utility.
func TestParseArguments(t *testing.T) {
	t.Parallel()
	expected := &ArgumentParseResult{
		PositionalArgs: []string{"get", "dut", "host1"},
		NullaryFlags: map[string]bool{
			"json": true,
		},
		Flags: map[string]string{},
	}
	r, err := parseArguments(
		[]string{"get", "dut", "-json", "host1"},
		map[string]bool{
			"json": true,
		},
	)
	if err != nil {
		t.Errorf("error parsing argument: %s", err)
	}
	if diff := cmp.Diff(expected, r); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}
}
