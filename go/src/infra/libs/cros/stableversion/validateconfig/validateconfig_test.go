// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package validateconfig

import (
	"fmt"
	"testing"
)

func errorWithDefault(e error, def string) string {
	if e == nil {
		return def
	}
	return e.Error()
}

var testInspectBufferData = []struct {
	name string
	in   string
	out  string
}{
	{
		"len zero string",
		"",
		FileLenZero,
	},
	{
		"not UTF-8",
		"\xee\xee\xee\xff",
		FileNotUTF8,
	},
	{
		"invalid JSON",
		"aaaa",
		FileNotJSON,
	},
	{
		"doesn't fit schema",
		"[2, 3, 4]",
		FileNotStableVersionProto,
	},
	{
		"well-formed but empty",
		"{}",
		FileSeemsLegit,
	},
}

func TestInspectBuffer(t *testing.T) {
	t.Parallel()
	for _, tt := range testInspectBufferData {
		t.Run(tt.name, func(t *testing.T) {
			if res := errorWithDefault(InspectBuffer([]byte(tt.in)), FileSeemsLegit); res != tt.out {
				msg := fmt.Sprintf("name (%s): got: (%q), want: (%q)", tt.name, res, tt.out)
				t.Errorf(msg)
			}
		})
	}
}

var testIsValidJSONData = []struct {
	in  string
	out bool
}{
	{
		"{}",
		true,
	},
	{
		"{",
		false,
	},
}

func TestIsValidJSON(t *testing.T) {
	t.Parallel()
	for _, tt := range testIsValidJSONData {
		t.Run(tt.in, func(t *testing.T) {
			if res := isValidJSON([]byte(tt.in)); res != tt.out {
				msg := fmt.Sprintf("item (%s): got: (%v), want: (%v)", tt.in, res, tt.out)
				t.Errorf(msg)
			}
		})
	}
}
