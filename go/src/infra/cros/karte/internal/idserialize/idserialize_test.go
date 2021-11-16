// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package idserialize

import (
	"encoding/hex"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
)

// TestVersionlessBytes tests that lowering the version-free portion of an IDInfo works.
func TestVersionlessBytes(t *testing.T) {
	t.Parallel()
	input := &IDInfo{
		Version:        "zzzz",
		CoarseTime:     0xF1F2F3F4F5F6F7F8,
		FineTime:       0xF1F2F3F4,
		Disambiguation: 0xF1F2F3F4,
	}
	expected := hex.EncodeToString([]byte("\xF1\xF2\xF3\xF4\xF5\xF6\xF7\xF8\xF1\xF2\xF3\xF4\xF1\xF2\xF3\xF4"))
	bytes, err := input.VersionlessBytes()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	actual := hex.EncodeToString(bytes)
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("unexpected diff (-want +got): %s", diff)
	}
}

// TestEncodedResultContainsVersion tests that the encoded result begins with a version prefix.
func TestEncodedResultContainsVersion(t *testing.T) {
	t.Parallel()
	input := &IDInfo{
		Version:        "zzzz",
		CoarseTime:     2,
		FineTime:       3,
		Disambiguation: 4,
	}
	str, err := input.Encoded()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if strings.HasPrefix(str, "zzzz") {
		// Do nothing. Output string has correct prefix.
	} else {
		t.Errorf("str %q (hex) unexpectedly lacks prefix", hex.EncodeToString([]byte(str)))
	}
}

// TestEncodedReturnsValidUTF8 tests that the encoding strategy returns valid UTF8.
func TestEncodedReturnsValidUTF8(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   IDInfo
	}{
		{
			name: "empty",
			in: IDInfo{
				Version: "zzzz",
			},
		},
		{
			name: "ones",
			in: IDInfo{
				Version:        "zzzz",
				CoarseTime:     1,
				FineTime:       1,
				Disambiguation: 1,
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bytes, _ := tt.in.VersionlessBytes()
			str, err := tt.in.Encoded()
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			if utf8.ValidString(str) {
				// Do nothing. Test successful.
			} else {
				t.Errorf("idinfo does not serialize to a utf-8 string: %q --> %q", hex.EncodeToString(bytes), hex.EncodeToString([]byte(str)))
			}
		})
	}
}
