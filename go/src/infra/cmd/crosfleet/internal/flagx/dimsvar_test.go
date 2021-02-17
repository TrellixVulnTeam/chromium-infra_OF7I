// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package flagx

import (
	"fmt"
	"infra/cmd/crosfleet/internal/common"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var testSplitKeyValData = []struct {
	in  string
	key string
	val string
	err string
}{
	{"", "", "", `string "" is a malformed key-value pair`},
	{"k:v=v", "", "", `string "k:v=v" is a malformed key-value pair`},
	{"a=", "a", "", ""},
	{"a:", "a", "", ""},
	{"k=v", "k", "v", ""},
	{"k:v", "k", "v", ""},
}

func TestSplitKeyVal(t *testing.T) {
	t.Parallel()
	for _, tt := range testSplitKeyValData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.in), func(t *testing.T) {
			t.Parallel()
			want := []string{tt.key, tt.val, tt.err}
			key, val, e := splitKeyVal(tt.in)
			got := []string{key, val, common.ErrToString(e)}
			diff := cmp.Diff(want, got)
			if diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
		})
	}
}

var testDimsVarData = []struct {
	startingDims  map[string]string
	keyvals       string
	wantDims      map[string]string
	wantErrString string
}{
	{nil, "", map[string]string{}, ""},
	{map[string]string{}, "", map[string]string{}, ""},
	{
		nil,
		"a:b",
		map[string]string{"a": "b"},
		"",
	},
	{
		nil,
		"a:b,c:d",
		map[string]string{"a": "b", "c": "d"},
		"",
	},
	{
		map[string]string{"a": "b"},
		"c:d",
		map[string]string{"a": "b", "c": "d"},
		"",
	},
	{
		map[string]string{"a": "b"},
		"c:d,e:f",
		map[string]string{"a": "b", "c": "d", "e": "f"},
		"",
	},
	{
		map[string]string{"a": "b"},
		"a:c",
		map[string]string{"a": "b"},
		`key "a" is already specified`,
	},
	{
		map[string]string{"a": "b"},
		"c:d,a:e",
		map[string]string{"a": "b", "c": "d"},
		`key "a" is already specified`,
	},
	{
		nil,
		"invalidKeyval",
		map[string]string{},
		`string "invalidKeyval" is a malformed key-value pair`,
	},
}

func TestDimsVar(t *testing.T) {
	t.Parallel()
	for _, tt := range testDimsVarData {
		tt := tt
		t.Run(fmt.Sprintf("(add %s to %v)", tt.keyvals, tt.startingDims), func(t *testing.T) {
			t.Parallel()
			m := tt.startingDims
			gotErr := KeyVals(&m).Set(tt.keyvals)
			diff := cmp.Diff(m, tt.wantDims)
			if diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
			gotErrString := common.ErrToString(gotErr)
			if gotErrString != tt.wantErrString {
				t.Errorf("unexpected error: wanted '%s', got '%s'", tt.wantErrString, gotErrString)
			}
		})
	}
}
