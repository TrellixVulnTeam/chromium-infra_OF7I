// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package commands

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestToCommand(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		input  *CommandWithFlags
		output []string
	}{
		{
			"full example",
			&CommandWithFlags{
				Commands: []string{"a", "b", "c"},
				Flags: map[string]*FlagEntry{
					"d": {"e"},
					"f": nil,
					"g": {"h"},
				},
				PositionalArgs: []string{"i", "j", "k"},
			},
			[]string{"a", "b", "c", "-d", "e", "-f", "-g", "h", "i", "j", "k"},
		},
		{
			"single positional param",
			&CommandWithFlags{
				Commands: []string{"a"},
				Flags: map[string]*FlagEntry{
					"b": nil,
				},
			},
			[]string{"a", "-b"},
		},
	}

	for _, tt := range cases {
		name := tt.name
		expected := tt.output
		actual := tt.input.ToCommand()
		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("subtest %s: unexpected diff: %s", name, diff)
		}
	}
}

func TestApplyFlagFilter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name            string
		input           *CommandWithFlags
		defaultDecision bool
		overrideMap     map[string]bool
		output          []string
	}{
		{
			"full example",
			&CommandWithFlags{
				Commands: []string{"a", "b", "c"},
				Flags: map[string]*FlagEntry{
					"d": {"e"},
					"f": nil,
					"g": {"h"},
				},
				PositionalArgs: []string{"i", "j", "k"},
			},
			false,
			map[string]bool{
				"f": true,
			},
			[]string{"a", "b", "c", "-f", "i", "j", "k"},
		},
		{
			"exclude all flags",
			&CommandWithFlags{
				Commands: []string{"a", "b", "c"},
				Flags: map[string]*FlagEntry{
					"d": {"e"},
					"f": nil,
					"g": {"h"},
				},
				PositionalArgs: []string{"i", "j", "k"},
			},
			false,
			nil,
			[]string{"a", "b", "c", "i", "j", "k"},
		},
		{
			"include all flags",
			&CommandWithFlags{
				Commands: []string{"a", "b", "c"},
				Flags: map[string]*FlagEntry{
					"d": {"e"},
					"f": nil,
					"g": {"h"},
				},
				PositionalArgs: []string{"i", "j", "k"},
			},
			true,
			nil,
			[]string{"a", "b", "c", "-d", "e", "-f", "-g", "h", "i", "j", "k"},
		},
		{
			"exclude one flag",
			&CommandWithFlags{
				Commands: []string{"a"},
				Flags: map[string]*FlagEntry{
					"d": {"e"},
					"f": nil,
					"g": {"h"},
				},
				PositionalArgs: []string{"i", "j", "k"},
			},
			true,
			map[string]bool{
				"d": true,
				"g": false,
			},
			[]string{"a", "-d", "e", "-f", "i", "j", "k"},
		},
	}
	for _, tt := range cases {
		name := tt.name
		expected := tt.output
		actual := tt.input.ApplyFlagFilter(
			tt.defaultDecision,
			tt.overrideMap,
		).ToCommand()
		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("subtest %s: unexpected diff: %s", name, diff)
		}
	}
}
