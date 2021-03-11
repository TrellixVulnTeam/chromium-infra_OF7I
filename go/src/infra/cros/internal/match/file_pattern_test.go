// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package match

import (
	"testing"

	"go.chromium.org/chromiumos/infra/proto/go/testplans"
)

func TestFilePatternMatches(t *testing.T) {
	tests := []struct {
		name        string
		filePattern *testplans.FilePattern
		sourcePath  string
		want        bool
	}{
		{
			"matches",
			&testplans.FilePattern{
				Pattern: "a/b/**",
			},
			"a/b/c/d.txt",
			true,
		},
		{
			"doesn't match",
			&testplans.FilePattern{
				Pattern: "a/b/**",
			},
			"a/e/c/d.txt",
			false,
		},
		{
			"matches exclude",
			&testplans.FilePattern{
				Pattern:         "a/b/**",
				ExcludePatterns: []string{"a/b/c/**", "a/b/c/otherfile.json"},
			},
			"a/b/c/d.txt",
			false,
		},
		{
			"doesn't match exclude",
			&testplans.FilePattern{
				Pattern:         "a/b/**",
				ExcludePatterns: []string{"a/b/c/**", "a/b/c/otherfile.json"},
			},
			"a/b/d.txt",
			true,
		},
		{
			"empty pattern",
			&testplans.FilePattern{},
			"a/b.txt",
			false,
		},
		{
			"empty pattern matches exclude",
			&testplans.FilePattern{ExcludePatterns: []string{"a/**"}},
			"a/b.txt",
			false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := FilePatternMatches(test.filePattern, test.sourcePath)
			if err != nil {
				t.Fatalf("FilePatternMatches(%q, %s) failed: %s", test.filePattern, test.sourcePath, err)
			}

			if got != test.want {
				t.Errorf("FilePatternMatches(%q, %s) = %t, want %t", test.filePattern, test.sourcePath, got, test.want)
			}
		})
	}
}
