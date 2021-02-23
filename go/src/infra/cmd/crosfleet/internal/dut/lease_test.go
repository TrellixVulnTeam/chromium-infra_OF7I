// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"flag"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"infra/cmd/crosfleet/internal/common"
	"testing"
)

var testValidateData = []struct {
	leaseFlags
	wantValidationErrString string
}{
	{ // All flags raise errors
		leaseFlags{
			durationMins: 0,
			reason:       "this desc is barely too long!!!",
			host:         "",
			model:        "",
		},
		`missing either model or host
duration should be greater than 0
reason cannot exceed 30 characters`,
	},
	{ // Some flags raise errors
		leaseFlags{
			durationMins: 1441,
			reason:       "this desc is just short enough",
			host:         "sample-host",
			model:        "sample-model",
		},
		`model and host cannot both be specified
duration cannot exceed 24 hours (1440 minutes)`,
	},
	{ // No flags raise errors
		leaseFlags{
			durationMins: 1440,
			reason:       "this desc is just short enough",
			host:         "",
			model:        "sample-model",
		},
		"",
	},
}

func TestValidate(t *testing.T) {
	t.Parallel()
	for _, tt := range testValidateData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantValidationErrString), func(t *testing.T) {
			t.Parallel()
			gotValidationErr := tt.leaseFlags.validate(&flag.FlagSet{})
			gotValidationErrString := common.ErrToString(gotValidationErr)
			if tt.wantValidationErrString != gotValidationErrString {
				t.Errorf("unexpected error: wanted %s, got %s", tt.wantValidationErrString, gotValidationErrString)
			}
		})
	}
}

var testBotDimsAndBuildTagsData = []struct {
	leaseFlags
	wantDims, wantTags map[string]string
}{
	{ // Model-based lease with added dims
		leaseFlags{
			model:     "sample-model",
			host:      "sample-host",
			reason:    "sample reason",
			addedDims: map[string]string{"added-key": "added-val"},
		},
		map[string]string{
			"added-key":   "added-val",
			"dut_state":   "ready",
			"label-model": "sample-model",
			"label-pool":  "DUT_POOL_QUOTA",
		},
		map[string]string{
			"added-key":      "added-val",
			"crosfleet-tool": "lease",
			"lease-by":       "model",
			"lease-reason":   "sample reason",
			"model":          "sample-model",
			"qs-account":     "leases",
		},
	},
	{ // Host-based lease without added dims
		leaseFlags{
			model:     "",
			host:      "sample-host",
			reason:    "sample reason",
			addedDims: nil,
		},
		map[string]string{"id": "sample-host"},
		map[string]string{
			"crosfleet-tool": "lease",
			"id":             "sample-host",
			"lease-by":       "host",
			"lease-reason":   "sample reason",
			"qs-account":     "leases",
		},
	},
}

func TestBotDimsAndBuildTagsData(t *testing.T) {
	t.Parallel()
	for _, tt := range testBotDimsAndBuildTagsData {
		tt := tt
		t.Run(fmt.Sprintf("(%s, %s)", tt.wantDims, tt.wantTags), func(t *testing.T) {
			gotDims, gotTags := tt.leaseFlags.botDimsAndBuildTags()
			if dimDiff := cmp.Diff(tt.wantDims, gotDims); dimDiff != "" {
				t.Errorf("unexpected bot dimension diff (%s)", dimDiff)
			}
			if tagDiff := cmp.Diff(tt.wantTags, gotTags); tagDiff != "" {
				t.Errorf("unexpected build tag diff (%s)", tagDiff)
			}
		})
	}
}
