// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"flag"
	"fmt"
	"infra/cmd/crosfleet/internal/common"
	"testing"
)

var testValidateArgsData = []struct {
	testCommonFlags
	args                    []string
	wantValidationErrString string
}{
	{ // All errors raised
		testCommonFlags{
			board:    "",
			pool:     "",
			image:    "",
			priority: 256,
		},
		[]string{},
		`missing board flag
missing pool flag
missing image flag
priority flag should be in [50, 255]
missing suite-name arg`,
	},
	{ // One error raised
		testCommonFlags{
			board:    "",
			pool:     "sample-pool",
			image:    "sample-image",
			priority: 255,
		},
		[]string{"sample-suite-name"},
		"missing board flag",
	},
	{ // No errors raised
		testCommonFlags{
			board:    "sample-board",
			pool:     "sample-pool",
			image:    "sample-image",
			priority: 255,
		},
		[]string{"sample-suite-name"},
		"",
	},
}

func TestValidateArgs(t *testing.T) {
	t.Parallel()
	for _, tt := range testValidateArgsData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantValidationErrString), func(t *testing.T) {
			t.Parallel()
			var flagSet flag.FlagSet
			err := flagSet.Parse(tt.args)
			if err != nil {
				t.Fatalf("unexpected error parsing command line args %v for test: %v", tt.args, err)
			}
			gotValidationErr := tt.testCommonFlags.validateArgs(&flagSet, "suite-name")
			gotValidationErrString := common.ErrToString(gotValidationErr)
			if gotValidationErrString != tt.wantValidationErrString {
				t.Errorf("unexpected error: wanted '%s', got '%s'", tt.wantValidationErrString, gotValidationErrString)
			}
		})
	}
}
