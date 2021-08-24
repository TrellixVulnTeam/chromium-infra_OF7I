// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package filterexp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestParse tests parsing filter expressions into anded-together lists of expressions.
func TestParse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		input  string
		output []Expression
		ok     bool
	}{
		{
			name:   "empty should be no-op",
			input:  "",
			output: nil,
			ok:     true,
		},
		{
			name:  "single comparison",
			input: "a > b",
			output: []Expression{NewApplication(
				"_>_",
				NewIdentifier("a"),
				NewIdentifier("b"),
			)},
			ok: false,
		},
		{
			name:  "multiple comparison",
			input: "a > b && c < d",
			output: []Expression{
				NewApplication(
					"_>_",
					NewIdentifier("a"),
					NewIdentifier("b"),
				),
				NewApplication(
					"_<_",
					NewIdentifier("c"),
					NewIdentifier("d"),
				),
			},
			ok: true,
		},
		{
			name:   "parse error",
			input:  "a >",
			output: nil,
			ok:     false,
		},
		{
			name:  `valid (a && b) && c comparison`,
			input: `(a == "A" && b == "B") && c == "C"`,
			output: []Expression{
				NewApplication(
					"_==_",
					NewIdentifier("a"),
					NewConstant("A"),
				),
				NewApplication(
					"_==_",
					NewIdentifier("b"),
					NewConstant("B"),
				),
				NewApplication(
					"_==_",
					NewIdentifier("c"),
					NewConstant("C"),
				),
			},
			ok: true,
		},
		{
			name:  `valid (a && b) && (c && (d && e)) comparison`,
			input: `(a == "A" && b == "B") && (c == "C" && (d == "D") && (e == "E"))`,
			output: []Expression{
				NewApplication(
					"_==_",
					NewIdentifier("a"),
					NewConstant("A"),
				),
				NewApplication(
					"_==_",
					NewIdentifier("b"),
					NewConstant("B"),
				),
				NewApplication(
					"_==_",
					NewIdentifier("c"),
					NewConstant("C"),
				),
				NewApplication(
					"_==_",
					NewIdentifier("d"),
					NewConstant("D"),
				),
				NewApplication(
					"_==_",
					NewIdentifier("e"),
					NewConstant("E"),
				),
			},
			ok: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expected := tt.output
			actual, err := Parse(tt.input)
			if tt.ok {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error to not be nil")
				}
			}
			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("unexpected diff: %s", diff)
			}
		})
	}
}
