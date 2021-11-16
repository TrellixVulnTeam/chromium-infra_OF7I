// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package lex64

import (
	"encoding/hex"
	"sort"
	"strings"
	"testing"
	"testing/quick"

	"github.com/google/go-cmp/cmp"
)

// TestAlphabet checks that the alphabet is in lexicographic order.
func TestAlphabet(t *testing.T) {
	t.Parallel()

	chars := strings.Split(alphabet, "")
	sort.Strings(chars)
	sortedAlphabet := strings.Join(chars, "")

	if diff := cmp.Diff(sortedAlphabet, alphabet); diff != "" {
		t.Errorf("unexpected diff (-want +got): %s", diff)
	}
}

// TestEncodeAndDecode tests that encoding and decoding work and roundtrip as expected.
func TestEncodeAndDecode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "F1F2F3",
			in:   "\xF1\xF2\xF3",
			out:  "wUAn",
		},
		{
			name: "empty string",
			in:   "",
			out:  "",
		},
		{
			name: "single char",
			in:   "\x00",
			out:  "00--",
		},
		{
			name: "random string",
			in:   "\x67\x9a\x5c\x48\xbe\x97\x27\x75\xdf\x6a",
			out:  "OtdRHAuM9rMUPV--",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := []byte(tt.in)
			actual, err := Encode(in, true)
			if err != nil {
				t.Errorf("unexpected error for subtest %q: %s", tt.name, err)
			}
			expected := tt.out

			if diff := cmp.Diff(expected, actual); diff != "" {
				t.Errorf("unexpected diff for subtest %q (-want +got): %s", tt.name, diff)
			}

			roundTrip, err := Decode(actual, true)
			if err != nil {
				t.Errorf("unexpected error for subtest %q: %s", tt.name, err)
			}
			if diff := cmp.Diff(
				hex.EncodeToString(in),
				hex.EncodeToString(roundTrip),
			); diff != "" {
				t.Errorf("unexpected diff for subtest %q (-want +got): %s", tt.name, diff)
			}
		})
	}
}

// A ComparisonTestCase is a collection of two strings with a descriptive name.
// These two strings will be compared as strings, and then also compared once encoded.
type comparisonTestCase struct {
	name string
	a    string
	b    string
}

// TestRoundTripEquality tests that encoding and decoding with padding round-trips.
func TestRoundTripEquality(t *testing.T) {
	t.Parallel()

	roundTrip := func(a []byte) bool {
		encoded, err := Encode(a, true)
		if err != nil {
			panic(err.Error())
		}
		decoded, err := Decode(encoded, true)
		if err != nil {
			panic(err.Error())
		}
		return cmpBytes(a, decoded) == 0
	}

	if err := quick.Check(roundTrip, nil); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

// TestPreservationOfComparisonOrder tests that encoding with padding preserves comparison order.
func TestPreservationOfComparisonOrder(t *testing.T) {
	t.Parallel()

	expected := func(a []byte, b []byte) int {
		return cmpBytes(a, b)
	}

	actual := func(a []byte, b []byte) int {
		a1, err := Encode(a, true)
		if err != nil {
			panic(err.Error())
		}
		b1, err := Encode(b, true)
		if err != nil {
			panic(err.Error())
		}
		return strings.Compare(a1, b1)
	}
	if err := quick.CheckEqual(expected, actual, nil); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

// TestWeakPreservationOfComparisonOrder tests that encoding without padding never *reverses* a comparison.
func TestWeakPreservationOfComparisonOrder(t *testing.T) {
	t.Parallel()

	tester := func(a []byte, b []byte) bool {
		a1, err := Encode(a, false)
		if err != nil {
			panic(err.Error())
		}
		b1, err := Encode(b, false)
		if err != nil {
			panic(err.Error())
		}
		// output will be -1 if and only if a comparison was reversed.
		output := cmpBytes(a, b) * strings.Compare(a1, b1)
		if output == -1 {
			return false
		}
		return true
	}
	if err := quick.Check(tester, nil); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

// TestInputTruncation tests that using the padding-less encoder is equivalent to using padding and then removing the padding chars.
func TestInputTruncation(t *testing.T) {
	t.Parallel()

	expected := func(b []byte) string {
		out, err := Encode(b, true)
		if err != nil {
			panic(err.Error())
		}
		return strings.Trim(out, "-")
	}
	actual := func(b []byte) string {
		out, err := Encode(b, false)
		if err != nil {
			panic(err.Error())
		}
		return out
	}
	if err := quick.CheckEqual(expected, actual, nil); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

// CmpString compares two sequences of bytes and returns +1, 0, or -1.
func cmpBytes(a []byte, b []byte) int {
	return strings.Compare(string(a), string(b))
}
