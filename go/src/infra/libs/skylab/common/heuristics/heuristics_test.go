// Copyright 2022 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristics

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestLooksLikeCrosskBotName tests identification of bots.
func TestLooksLikeCrosskBotName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		out  bool
	}{
		{
			name: "empty string",
			in:   "",
			out:  false,
		},
		{
			name: "has prefix",
			in:   "crossk-a",
			out:  true,
		},
		{
			name: "no prefix",
			in:   "a",
			out:  false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expected := tt.out
			actual := LooksLikeCrosskBotName(tt.in)
			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("unexpected diff (-want +got): %s", diff)
			}
		})
	}
}

// TestLooksLikeValidPool tests whether strings are correctly identified as being valid pools.
func TestLooksLikeValidPool(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		out  bool
	}{
		{
			name: "empty string",
			in:   "",
			out:  false,
		},
		{
			name: "has [",
			in:   "a[",
			out:  false,
		},
		{
			name: "valid identifier",
			in:   "valid_identifier4",
			out:  true,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expected := tt.out
			actual := LooksLikeValidPool(tt.in)
			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("unexpected diff (-want +got): %s", diff)
			}
		})
	}
}
