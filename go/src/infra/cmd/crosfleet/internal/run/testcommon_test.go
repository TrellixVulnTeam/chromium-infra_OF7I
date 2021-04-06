// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"flag"
	"fmt"
	"testing"
	"time"

	"infra/cmd/crosfleet/internal/common"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
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
			if err := flagSet.Parse(tt.args); err != nil {
				t.Fatalf("unexpected error parsing command line args %v for test: %v", tt.args, err)
			}
			gotValidationErr := tt.testCommonFlags.validateArgs(&flagSet, "suite-name")
			gotValidationErrString := common.ErrToString(gotValidationErr)
			if tt.wantValidationErrString != gotValidationErrString {
				t.Errorf("unexpected error: wanted '%s', got '%s'", tt.wantValidationErrString, gotValidationErrString)
			}
		})
	}
}

var testBuildTagsData = []struct {
	testCommonFlags
	wantTags map[string]string
}{
	{ // Missing all values
		testCommonFlags{
			board:     "",
			model:     "",
			pool:      "",
			image:     "",
			qsAccount: "",
			priority:  0,
			addedTags: nil,
		},
		map[string]string{
			"crosfleet-tool": "suite",
			"label-suite":    "sample-suite",
		},
	},
	{ // Missing some values
		testCommonFlags{
			board:     "sample-board",
			model:     "",
			pool:      "sample-pool",
			image:     "sample-image",
			qsAccount: "",
			priority:  99,
			addedTags: map[string]string{},
		},
		map[string]string{
			"crosfleet-tool": "suite",
			"label-suite":    "sample-suite",
			"label-board":    "sample-board",
			"label-pool":     "sample-pool",
			"label-image":    "sample-image",
			"label-priority": "99",
		},
	},
	{ // Includes all values
		testCommonFlags{
			board:     "sample-board",
			model:     "sample-model",
			pool:      "sample-pool",
			image:     "sample-image",
			qsAccount: "sample-qs-account",
			priority:  99,
			addedTags: map[string]string{
				"foo": "bar",
			},
		},
		map[string]string{
			"foo":                 "bar",
			"crosfleet-tool":      "suite",
			"label-suite":         "sample-suite",
			"label-board":         "sample-board",
			"label-model":         "sample-model",
			"label-pool":          "sample-pool",
			"label-image":         "sample-image",
			"label-quota-account": "sample-qs-account",
		},
	},
}

func TestBuildTags(t *testing.T) {
	t.Parallel()
	for _, tt := range testBuildTagsData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantTags), func(t *testing.T) {
			t.Parallel()
			gotTags := tt.testCommonFlags.buildTags("suite", "sample-suite")
			if diff := cmp.Diff(tt.wantTags, gotTags); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
		})
	}
}

var testSoftwareDependenciesData = []struct {
	testCommonFlags
	wantDeps      []*test_platform.Request_Params_SoftwareDependency
	wantErrString string
}{
	{ // Invalid label
		testCommonFlags{
			image:           "",
			provisionLabels: map[string]string{"foo-invalid": "bar"},
		},
		nil,
		"invalid provisionable label foo-invalid",
	},
	{ // No labels or image
		testCommonFlags{
			image:           "",
			provisionLabels: nil,
		},
		nil,
		"",
	},
	{ // Image, Lacros path, and one label
		testCommonFlags{
			image:      "sample-image",
			lacrosPath: "sample-lacros-path",
			provisionLabels: map[string]string{
				"fwrw-version": "foo-rw",
			},
		},
		[]*test_platform.Request_Params_SoftwareDependency{
			{Dep: &test_platform.Request_Params_SoftwareDependency_RwFirmwareBuild{RwFirmwareBuild: "foo-rw"}},
			{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "sample-image"}},
			{Dep: &test_platform.Request_Params_SoftwareDependency_LacrosGcsPath{LacrosGcsPath: "sample-lacros-path"}},
		},
		"",
	},
}

func TestSoftwareDependencies(t *testing.T) {
	t.Parallel()
	for _, tt := range testSoftwareDependenciesData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantDeps), func(t *testing.T) {
			t.Parallel()
			gotDeps, gotErr := tt.testCommonFlags.softwareDependencies()
			if diff := cmp.Diff(tt.wantDeps, gotDeps, common.CmpOpts); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
			gotErrString := common.ErrToString(gotErr)
			if tt.wantErrString != gotErrString {
				t.Errorf("unexpected error: wanted '%s', got '%s'", tt.wantErrString, gotErrString)
			}
		})
	}
}

var testSchedulingParamsData = []struct {
	testCommonFlags
	wantParams *test_platform.Request_Params_Scheduling
}{
	{ // Unmanaged pool, no quota account
		testCommonFlags{
			pool:     "sample-unmanaged-pool",
			priority: 100,
		},
		&test_platform.Request_Params_Scheduling{
			Pool:     &test_platform.Request_Params_Scheduling_UnmanagedPool{UnmanagedPool: "sample-unmanaged-pool"},
			Priority: 100,
		},
	},
	{ // Quota account and managed pool
		testCommonFlags{
			pool:      "MANAGED_POOL_SUITES",
			qsAccount: "sample-qs-account",
			priority:  100,
		},
		&test_platform.Request_Params_Scheduling{
			Pool:      &test_platform.Request_Params_Scheduling_ManagedPool_{ManagedPool: 3},
			QsAccount: "sample-qs-account",
		},
	},
	{ // No quota account and managed pool name entered in nonstandard format
		testCommonFlags{
			pool:     "dut-pool-suites",
			priority: 100,
		},
		&test_platform.Request_Params_Scheduling{
			Pool:     &test_platform.Request_Params_Scheduling_ManagedPool_{ManagedPool: 3},
			Priority: 100,
		},
	},
}

func TestSchedulingParams(t *testing.T) {
	t.Parallel()
	for _, tt := range testSchedulingParamsData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantParams), func(t *testing.T) {
			t.Parallel()
			gotParams := tt.testCommonFlags.schedulingParams()
			if diff := cmp.Diff(tt.wantParams, gotParams, common.CmpOpts); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
		})
	}
}

var testRetryParamsData = []struct {
	maxRetries int
	wantParams *test_platform.Request_Params_Retry
}{
	{ // With retries
		2,
		&test_platform.Request_Params_Retry{Max: 2, Allow: true},
	},
	{ // No retries
		0,
		&test_platform.Request_Params_Retry{Max: 0, Allow: false},
	},
}

func TestRetryParams(t *testing.T) {
	t.Parallel()
	for _, tt := range testRetryParamsData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantParams), func(t *testing.T) {
			t.Parallel()
			fs := testCommonFlags{maxRetries: tt.maxRetries}
			gotParams := fs.retryParams()
			if diff := cmp.Diff(tt.wantParams, gotParams, common.CmpOpts); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
		})
	}
}

func stringOfLength(length int) string {
	letters := make([]rune, length)
	for i := 0; i < length; i++ {
		letters[i] = 'a'
	}
	return string(letters)
}

var testTestOrSuiteNamesLabelData = []struct {
	names     []string
	wantLabel string
}{
	{
		[]string{"foo", "bar"},
		"[foo bar]",
	},
	{
		[]string{"foo"},
		"foo",
	},
	{
		[]string{stringOfLength(301)},
		stringOfLength(300),
	},
}

func TestTestOrSuiteNamesLabel(t *testing.T) {
	t.Parallel()
	for _, tt := range testTestOrSuiteNamesLabelData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantLabel), func(t *testing.T) {
			t.Parallel()
			gotLabel := testOrSuiteNamesLabel(tt.names)
			if tt.wantLabel != gotLabel {
				t.Errorf("unexpected error: wanted '%s', got '%s'", tt.wantLabel, gotLabel)
			}
		})
	}
}

func TestTestPlatformRequest(t *testing.T) {
	t.Parallel()
	cliFlags := &testCommonFlags{
		board:           "sample-board",
		model:           "sample-model",
		image:           "sample-image",
		pool:            "MANAGED_POOL_SUITES",
		qsAccount:       "sample-qs-account",
		priority:        100,
		maxRetries:      0,
		timeoutMins:     30,
		provisionLabels: map[string]string{"cros-version": "foo-cros"},
		addedDims:       map[string]string{"foo-dim": "bar-dim"},
		keyvals:         map[string]string{"foo-key": "foo-val"},
	}
	buildTags := map[string]string{"foo-tag": "bar-tag"}
	wantRequest := &test_platform.Request{
		TestPlan: &test_platform.Request_TestPlan{},
		Params: &test_platform.Request_Params{
			Scheduling: &test_platform.Request_Params_Scheduling{
				Pool:      &test_platform.Request_Params_Scheduling_ManagedPool_{ManagedPool: 3},
				QsAccount: "sample-qs-account",
			},
			FreeformAttributes: &test_platform.Request_Params_FreeformAttributes{
				SwarmingDimensions: []string{"foo-dim:bar-dim"},
			},
			HardwareAttributes: &test_platform.Request_Params_HardwareAttributes{
				Model: "sample-model",
			},
			SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
				BuildTarget: &chromiumos.BuildTarget{Name: "sample-board"},
			},
			SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
				{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "foo-cros"}},
				{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "sample-image"}},
			},
			Decorations: &test_platform.Request_Params_Decorations{
				AutotestKeyvals: map[string]string{"foo-key": "foo-val"},
				Tags:            []string{"foo-tag:bar-tag"},
			},
			Retry: &test_platform.Request_Params_Retry{
				Max:   0,
				Allow: false,
			},
			Metadata: &test_platform.Request_Params_Metadata{
				TestMetadataUrl:        "gs://chromeos-image-archive/sample-image",
				DebugSymbolsArchiveUrl: "gs://chromeos-image-archive/sample-image",
			},
			Time: &test_platform.Request_Params_Time{
				MaximumDuration: ptypes.DurationProto(
					time.Duration(1800000000000)),
			},
		},
	}
	runLauncher := ctpRunLauncher{
		testPlan:  &test_platform.Request_TestPlan{},
		buildTags: buildTags,
		cliFlags:  cliFlags,
	}
	gotRequest, err := runLauncher.testPlatformRequest()
	if err != nil {
		t.Fatalf("unexpected error constructing Test Platform request: %v", err)
	}
	if diff := cmp.Diff(wantRequest, gotRequest, common.CmpOpts); diff != "" {
		t.Errorf("unexpected diff (%s)", diff)
	}
}
