// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package execs

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Test cases for TestDUTPlans
var parseActionArgsCases = []struct {
	name     string
	args     []string
	splitter string
	expected ParsedArgs
}{
	{
		"empty",
		nil,
		DefaultSplitter,
		nil,
	},
	{
		"empty 2",
		[]string{" ", "", "    "},
		DefaultSplitter,
		nil,
	},
	{
		"simple args",
		[]string{"my", "1", "&&&", "---9433"},
		DefaultSplitter,
		map[string]string{
			"my": "", "1": "", "&&&": "", "---9433": "",
		},
	},
	{
		"pair values args",
		[]string{"my", "&&&:1234", "---9433: my value "},
		DefaultSplitter,
		map[string]string{
			"my":      "",
			"&&&":     "1234",
			"---9433": "my value",
		},
	},
	{
		"complicated cases",
		[]string{
			"my",
			"&&&:1234",
			"key: val:split "},
		DefaultSplitter,
		map[string]string{
			"my":  "",
			"&&&": "1234",
			"key": "val:split",
		},
	},
}

func TestParseActionArgs(t *testing.T) {
	t.Parallel()
	for _, c := range parseActionArgsCases {
		cs := c
		t.Run(cs.name, func(t *testing.T) {
			ctx := context.Background()
			got := ParseActionArgs(ctx, cs.args, cs.splitter)
			if len(cs.expected) == 0 && len(got) == len(cs.expected) {
				// Everything is good.
			} else {
				if !cmp.Equal(got, cs.expected) {
					t.Errorf("%q ->want: %v\n got: %v", cs.name, cs.expected, got)
				}
			}
		})
	}
}
