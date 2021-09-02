// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	b64 "encoding/base64"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Test cases for TestDUTPlans
var dutPlansCases = []struct {
	name   string
	in     string
	isNull bool
}{
	{
		"no Data",
		"",
		true,
	},
	{
		"Some data",
		`{
			"Field":"something",
			"number': 765
		}`,
		false,
	},
	{
		"strange data",
		"!@#$%^&*()__)(*&^%$#retyuihjo{:>\"?{",
		false,
	},
}

// Testing dutPlans method
func TestGetConfiguration(t *testing.T) {
	t.Parallel()
	for _, c := range dutPlansCases {
		cs := c
		t.Run(cs.name, func(t *testing.T) {
			t.Parallel()
			sEnc := b64.StdEncoding.EncodeToString([]byte(cs.in))
			r, err := getConfiguration(sEnc)
			if err != nil {
				t.Errorf("Case %s: %s", cs.name, err)
			}
			if cs.isNull {
				if r != nil {
					t.Errorf("Case %s: expected nil", cs.name)
				}
			} else {
				got, err := io.ReadAll(r)
				if err != nil {
					t.Errorf("Case %s: %s", cs.name, err)
				}
				if !cmp.Equal(string(got), cs.in) {
					t.Errorf("got: %v\nwant: %v", string(got), cs.in)
				}
			}
		})
	}
}
