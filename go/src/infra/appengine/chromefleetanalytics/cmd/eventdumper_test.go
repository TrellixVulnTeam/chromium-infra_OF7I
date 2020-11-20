// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

const rawEventLine = "GOOD\t----\tverify.PASS\ttimestamp=1605634967\tlocaltime=Nov 17 17:42:47"
const indentedRawEventLine = "\t\tGOOD\t----\tverify.PASS\ttimestamp=1605634967\tlocaltime=Nov 17 17:42:47"
const endGoodLine = "END GOOD\t----\treset\ttimestamp=1597278536\tlocaltime=Aug 12 17:28:56\tchromeos1-row1-rack10-host2 reset successfully"

func TestExtractTimestamp(t *testing.T) {
	t.Parallel()

	data := []struct {
		name     string
		in       string
		out      int64
		errIsNil bool
	}{
		{
			"empty",
			"",
			0,
			false,
		},
		{
			"timestamp=4",
			"timestamp=4",
			4,
			true,
		},
		{
			"tab",
			"a\ttimestamp=100",
			100,
			true,
		},
	}

	for _, tt := range data {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			t.Parallel()
			i, err := extractTimestamp(tt.in)
			isGood := err == nil

			if diff := cmp.Diff(tt.out, i); diff != "" {
				t.Errorf("unexpected diff: %s", diff)
			}

			if diff := cmp.Diff(tt.errIsNil, isGood); diff != "" {
				t.Errorf("err is %v unexpected diff: %s", err, diff)
			}
		})
	}
}

func TestEventToString(t *testing.T) {
	t.Parallel()
	e := &event{
		timestamp: 400,
		level:     4,
		status:    "y",
		name:      "e",
		isEnd:     false,
	}
	expected := `400 4 "y" "e" false`
	s := e.toString()
	if diff := cmp.Diff(expected, s); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}
}

func TestNormalizeIndent(t *testing.T) {
	t.Parallel()
	data := []struct {
		name   string
		in     []string
		outIdx int
		out    []string
	}{
		{
			"empty",
			[]string{},
			0,
			nil,
		},
		{
			"nil",
			nil,
			0,
			nil,
		},
		{
			"singleton",
			[]string{"a"},
			0,
			[]string{"a"},
		},
		{
			"singleton",
			[]string{"", "a"},
			1,
			[]string{"a"},
		},
	}

	for _, tt := range data {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			t.Parallel()
			outIdx, out := normalizeIndent(tt.in)
			if diff := cmp.Diff(tt.outIdx, outIdx); diff != "" {
				t.Errorf("unexpected diff: %s", diff)
			}
			if diff := cmp.Diff(tt.out, out); diff != "" {
				t.Errorf("unexpected diff: %s", diff)
			}
		})
	}
}

func TestParseEvent(t *testing.T) {
	t.Parallel()
	data := []struct {
		name     string
		in       string
		event    *event
		errIsNil bool
	}{
		{
			"empty",
			"",
			nil,
			false,
		},
		{
			"eventLine",
			rawEventLine,
			&event{
				1605634967,
				0,
				"GOOD",
				"verify.PASS",
				false,
			},
			true,
		},
		{
			"indentedRawEventLine",
			indentedRawEventLine,
			&event{
				1605634967,
				2,
				"GOOD",
				"verify.PASS",
				false,
			},
			true,
		},
		{
			"endGoodLine",
			endGoodLine,
			&event{
				1597278536,
				0,
				"END GOOD",
				"reset",
				true,
			},
			true,
		},
	}

	for _, tt := range data {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			// t.Parallel()
			event, e := parseEvent(tt.in)
			errIsNil := e == nil
			if diff := cmp.Diff(tt.event, event); diff != "" {
				t.Errorf("unexpected diff: %s", diff)
			}
			if diff := cmp.Diff(tt.errIsNil, errIsNil); diff != "" {
				t.Errorf("err is %v. unexpected diff: %s", e, diff)
			}
		})
	}
}
