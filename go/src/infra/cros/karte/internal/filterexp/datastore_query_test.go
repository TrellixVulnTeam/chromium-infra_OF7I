// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package filterexp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestValidateComparison tests that ValidateComparison is capable of extracting
// fields and values and successfully rejects ill-formed comparisons.
func TestValidateComparison(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		input  Expression
		output *comparisonParseResult
		ok     bool
	}{
		{
			name:   "nil",
			input:  nil,
			output: nil,
			ok:     false,
		},
		{
			name:   "constant",
			input:  NewConstant(4),
			output: nil,
			ok:     false,
		},
		{
			name: "valid comparison",
			input: NewApplication(
				"_==_",
				NewIdentifier("a"),
				NewConstant("b"),
			),
			output: &comparisonParseResult{
				comparator: "_==_",
				field:      "a",
				value:      "b",
			},
			ok: true,
		},
		{
			name: "reversed comparison",
			input: NewApplication(
				"_==_",
				NewConstant(4),
				NewIdentifier("a"),
			),
			output: nil,
			ok:     false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			expected := tt.output
			actual, err := validateComparison(tt.input)
			if tt.ok {
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error to not be nil")
				}
			}
			if diff := cmp.Diff(expected, actual, cmpopts...); diff != "" {
				t.Errorf("unexpected diff: %s", diff)
			}
		})
	}
}
