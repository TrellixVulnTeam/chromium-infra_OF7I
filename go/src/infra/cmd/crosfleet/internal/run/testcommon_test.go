// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package run

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"infra/cmd/crosfleet/internal/common"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"google.golang.org/grpc"

	models "infra/unifiedfleet/api/v1/models"
	ufsapi "infra/unifiedfleet/api/v1/rpc"
)

var testValidateArgsData = []struct {
	testCommonFlags
	args                    []string
	wantValidationErrString string
}{
	{ // All errors raised
		testCommonFlags{
			board:    "",
			models:   []string{"model1", "model2"},
			repeats:  7,
			pool:     "",
			image:    "sample-image",
			release:  "sample-release",
			priority: 256,
		},
		[]string{},
		`missing board flag
missing pool flag
cannot specify both image and release branch
priority flag should be in [50, 255]
total number of CTP runs launched (# models specified * repeats) cannot exceed 12
missing suite arg`,
	},
	{ // One error raised
		testCommonFlags{
			board:    "sample-board",
			models:   []string{},
			repeats:  13,
			pool:     "sample-pool",
			image:    "sample-image",
			priority: 255,
		},
		[]string{"sample-suite-name"},
		"total number of CTP runs launched (# models specified * repeats) cannot exceed 12",
	},
	{ // One error raised
		testCommonFlags{
			board:           "sample-board",
			models:          []string{},
			repeats:         1,
			pool:            "sample-pool",
			image:           "sample-image",
			secondaryBoards: []string{"coral", "eve"},
			priority:        255,
		},
		[]string{"sample-suite-name"},
		"number of requested secondary-boards: 2 does not match with number of requested secondary-images: 0",
	},
	{ // One error raised
		testCommonFlags{
			board:           "sample-board",
			models:          []string{},
			repeats:         1,
			pool:            "sample-pool",
			image:           "sample-image",
			secondaryModels: []string{"babytiger", "eve"},
			priority:        255,
		},
		[]string{"sample-suite-name"},
		"number of requested secondary-boards: 0 does not match with number of requested secondary-models: 2",
	},
	{ // One error raised
		testCommonFlags{
			board:                "sample-board",
			models:               []string{},
			repeats:              1,
			pool:                 "sample-pool",
			image:                "sample-image",
			secondaryLacrosPaths: []string{"foo", "bar"},
			priority:             255,
		},
		[]string{"sample-suite-name"},
		"number of requested secondary-boards: 0 does not match with number of requested secondary-lacros-paths: 2",
	},
	{ // No errors raised
		testCommonFlags{
			board:    "sample-board",
			models:   []string{"model1", "model2"},
			repeats:  6,
			pool:     "sample-pool",
			release:  "sample-release",
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
			gotValidationErr := tt.testCommonFlags.validateArgs(&flagSet, "suite")
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
			models:    []string{""},
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
			models:    []string{""},
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
			models:    []string{"sample-model"},
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
			gotTags := tt.testCommonFlags.buildTagsForModel("suite", tt.testCommonFlags.models[0], "sample-suite")
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
			gotLabel := testOrSuiteNamesTag(tt.names)
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
		testPlan: &test_platform.Request_TestPlan{},
		cliFlags: cliFlags,
	}
	gotRequest, err := runLauncher.testPlatformRequest("sample-model", buildTags)
	if err != nil {
		t.Fatalf("unexpected error constructing Test Platform request: %v", err)
	}
	if diff := cmp.Diff(wantRequest, gotRequest, common.CmpOpts); diff != "" {
		t.Errorf("unexpected diff (%s)", diff)
	}
}

var testSecondaryDevicesData = []struct {
	testCommonFlags
	wantDeps []*test_platform.Request_Params_SecondaryDevice
}{
	{ // Test board only request.
		testCommonFlags{
			secondaryBoards: []string{"board1", "board2"},
			secondaryImages: []string{"board1-release/10000.0.0", "board2-release/9999.0.0"},
		},
		[]*test_platform.Request_Params_SecondaryDevice{
			{
				SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
					BuildTarget: &chromiumos.BuildTarget{Name: "board1"},
				},
				SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
					{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "board1-release/10000.0.0"}},
				},
			},
			{
				SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
					BuildTarget: &chromiumos.BuildTarget{Name: "board2"},
				},
				SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
					{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "board2-release/9999.0.0"}},
				},
			},
		},
	},
	{ // Test with model request.
		testCommonFlags{
			secondaryBoards: []string{"board1", "board2"},
			secondaryModels: []string{"model1", "model2"},
			secondaryImages: []string{"board1-release/10000.0.0", "board2-release/9999.0.0"},
		},
		[]*test_platform.Request_Params_SecondaryDevice{
			{
				SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
					BuildTarget: &chromiumos.BuildTarget{Name: "board1"},
				},
				HardwareAttributes: &test_platform.Request_Params_HardwareAttributes{Model: "model1"},
				SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
					{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "board1-release/10000.0.0"}},
				},
			},
			{
				SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
					BuildTarget: &chromiumos.BuildTarget{Name: "board2"},
				},
				HardwareAttributes: &test_platform.Request_Params_HardwareAttributes{Model: "model2"},
				SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
					{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "board2-release/9999.0.0"}},
				},
			},
		},
	},
	{ // Test with lacros provision request.
		testCommonFlags{
			secondaryBoards:      []string{"board1", "board2"},
			secondaryImages:      []string{"board1-release/10000.0.0", "board2-release/9999.0.0"},
			secondaryLacrosPaths: []string{"path1", "path2"},
		},
		[]*test_platform.Request_Params_SecondaryDevice{
			{
				SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
					BuildTarget: &chromiumos.BuildTarget{Name: "board1"},
				},
				SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
					{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "board1-release/10000.0.0"}},
					{Dep: &test_platform.Request_Params_SoftwareDependency_LacrosGcsPath{LacrosGcsPath: "path1"}},
				},
			},
			{
				SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
					BuildTarget: &chromiumos.BuildTarget{Name: "board2"},
				},
				SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
					{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "board2-release/9999.0.0"}},
					{Dep: &test_platform.Request_Params_SoftwareDependency_LacrosGcsPath{LacrosGcsPath: "path2"}},
				},
			},
		},
	},
	{ // Test skip image on a secondary DUT.
		testCommonFlags{
			secondaryBoards: []string{"board1", "board2"},
			secondaryImages: []string{"board1-release/10000.0.0", "skip"},
		},
		[]*test_platform.Request_Params_SecondaryDevice{
			{
				SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
					BuildTarget: &chromiumos.BuildTarget{Name: "board1"},
				},
				SoftwareDependencies: []*test_platform.Request_Params_SoftwareDependency{
					{Dep: &test_platform.Request_Params_SoftwareDependency_ChromeosBuild{ChromeosBuild: "board1-release/10000.0.0"}},
				},
			},
			{
				SoftwareAttributes: &test_platform.Request_Params_SoftwareAttributes{
					BuildTarget: &chromiumos.BuildTarget{Name: "board2"},
				},
			},
		},
	},
}

func TestSecondaryDevices(t *testing.T) {
	t.Parallel()
	for _, tt := range testSecondaryDevicesData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantDeps), func(t *testing.T) {
			t.Parallel()
			gotDeps := tt.testCommonFlags.secondaryDevices()
			if diff := cmp.Diff(tt.wantDeps, gotDeps, common.CmpOpts); diff != "" {
				t.Errorf("unexpected diff (%s)", diff)
			}
		})
	}
}

const (
	INVALID_BOARD = "Invalid Board"
	INVALID_MODEL = "Invalid Model"
	INVALID_IMAGE = "Invalid Image"
	INVALID_TEST  = "Invalid Test"
)

var testValidatePublicChromiumOnChromeOsData = []struct {
	testCommonFlags
	wantValidationErrString string
	testNames               []string
	validTests              []string
	validModels             []string
	testCmdName             string
	status                  bool
	ufsError                string
	allowPublicUserAcct     bool
}{
	{ // Invalid Board
		testCommonFlags{
			board:    "eve",
			models:   []string{"eve", "kevin"},
			repeats:  7,
			pool:     "",
			image:    "sample-image",
			release:  "sample-release",
			priority: 256,
		},
		INVALID_BOARD,
		[]string{"tast.lacros"},
		nil,
		nil,
		"",
		false,
		INVALID_BOARD,
		true,
	},
	{ // Invalid Model
		testCommonFlags{
			board:    "eve",
			models:   []string{"eve"},
			repeats:  7,
			pool:     "",
			image:    "sample-image",
			release:  "sample-release",
			priority: 256,
		},
		INVALID_MODEL,
		[]string{"tast.lacros"},
		nil,
		nil,
		"",
		false,
		INVALID_MODEL,
		true,
	},
	{ // Invalid Image
		testCommonFlags{
			board:    "eve",
			models:   []string{"eve", "kevin"},
			repeats:  7,
			pool:     "",
			image:    "sample-image",
			release:  "sample-release",
			priority: 256,
		},
		INVALID_IMAGE,
		[]string{"tast.lacros"},
		nil,
		nil,
		"",
		false,
		INVALID_IMAGE,
		true,
	},
	{ // Invalid Test
		testCommonFlags{
			board:    "eve",
			models:   []string{"eve"},
			repeats:  7,
			pool:     "",
			image:    "sample-image",
			release:  "sample-release",
			priority: 256,
		},
		INVALID_TEST,
		[]string{"tast.lacros"},
		nil,
		nil,
		"",
		false,
		INVALID_TEST,
		true,
	},
	{ // One valid Test
		testCommonFlags{
			board:    "eve",
			models:   []string{"eve"},
			repeats:  7,
			pool:     "",
			image:    "sample-image",
			release:  "sample-release",
			priority: 256,
		},
		INVALID_TEST,
		[]string{"tast.lacros", "tast.lacros2"},
		[]string{"tast.lacros2"},
		[]string{"eve"},
		"",
		true,
		INVALID_TEST,
		true,
	},
	{ // One valid Model
		testCommonFlags{
			board:    "eve",
			models:   []string{"eve", "kevin"},
			repeats:  7,
			pool:     "",
			image:    "sample-image",
			release:  "sample-release",
			priority: 256,
		},
		INVALID_MODEL,
		[]string{"tast.lacros"},
		[]string{"tast.lacros"},
		[]string{"kevin"},
		"",
		true,
		INVALID_MODEL,
		true,
	},
	{ // No Models Specified
		testCommonFlags{
			board:    "eve",
			models:   []string{},
			repeats:  7,
			pool:     "",
			image:    "sample-image",
			release:  "sample-release",
			priority: 256,
		},
		`model is required for public users`,
		[]string{"tast.lacros"},
		[]string{"tast.lacros"},
		nil,
		"",
		true,
		`model is required for public users`,
		true,
	},
	{ // No Models Specified
		testCommonFlags{
			board:    "eve",
			models:   []string{},
			repeats:  7,
			pool:     "",
			image:    "sample-image",
			release:  "sample-release",
			priority: 256,
		},
		`model is required for public users`,
		[]string{"tast.lacros"},
		[]string{"tast.lacros"},
		nil,
		"",
		true,
		`model is required for public users`,
		true,
	},
}

func TestValidatePublicChromiumTest(t *testing.T) {
	t.Parallel()
	for _, tt := range testValidatePublicChromiumOnChromeOsData {
		tt := tt
		t.Run(fmt.Sprintf("(%s)", tt.wantValidationErrString), func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			ctx = context.WithValue(ctx, "status", tt.ufsError)

			results, err := tt.testCommonFlags.verifyFleetTestsPolicy(ctx, &fakeUfsClient{}, tt.testCmdName, tt.testNames, true)

			if err != nil {
				if !strings.Contains(common.ErrToString(err), tt.wantValidationErrString) {
					t.Errorf("unexpected error: wanted '%s', got '%s'", tt.wantValidationErrString, common.ErrToString(err))
				}
				return
			}
			gotValidationErrString := ""
			if results.testValidationErrors != nil && len(results.testValidationErrors) != 0 {
				gotValidationErrString = results.testValidationErrors[0]
			}
			if !strings.Contains(gotValidationErrString, tt.wantValidationErrString) {
				t.Errorf("unexpected error: wanted '%s', got '%s'", tt.wantValidationErrString, gotValidationErrString)
			}
			if results.anyValidTests != tt.status {
				t.Errorf("unexpected error: wanted valid tests : %v", tt.status)
			}
			if diff := cmp.Diff(results.validTests, tt.validTests, common.CmpOpts); diff != "" {
				t.Errorf("unexpected tests (%s)", diff)
			}
			if diff := cmp.Diff(results.validModels, tt.validModels, common.CmpOpts); diff != "" {
				t.Errorf("unexpected models (%s)", diff)
			}
		})
	}
}

// FakeGetPoolsClient mimics a UFS client and records what it was asked to look up.
type fakeUfsClient struct {
	requestCount int
}

func (f *fakeUfsClient) GetMachineLSE(ctx context.Context, req *ufsapi.GetMachineLSERequest, opts ...grpc.CallOption) (*models.MachineLSE, error) {
	return nil, nil
}

func (f *fakeUfsClient) GetMachine(ctx context.Context, req *ufsapi.GetMachineRequest, opts ...grpc.CallOption) (*models.Machine, error) {
	return nil, nil
}

// CheckFleetTestsPolicy returns a dummy response.
func (f *fakeUfsClient) CheckFleetTestsPolicy(ctx context.Context, in *ufsapi.CheckFleetTestsPolicyRequest, opts ...grpc.CallOption) (*ufsapi.CheckFleetTestsPolicyResponse, error) {
	var status ufsapi.TestStatus_Code
	response := &ufsapi.CheckFleetTestsPolicyResponse{}
	f.requestCount++
	if f.requestCount == 1 {
		response.TestStatus = &ufsapi.TestStatus{
			Code: status,
		}
		return response, nil
	}
	msg := fmt.Sprint(ctx.Value("status"))
	if ctx.Value("status") != nil && f.requestCount == 2 {
		if msg == INVALID_BOARD {
			status = ufsapi.TestStatus_NOT_A_PUBLIC_BOARD
		} else if msg == INVALID_MODEL {
			status = ufsapi.TestStatus_NOT_A_PUBLIC_MODEL
		} else if msg == INVALID_IMAGE {
			status = ufsapi.TestStatus_NOT_A_PUBLIC_IMAGE
		} else if msg == INVALID_TEST {
			status = ufsapi.TestStatus_NOT_A_PUBLIC_TEST
		}
		response.TestStatus = &ufsapi.TestStatus{
			Code:    status,
			Message: msg,
		}
		return response, nil
	}
	response.TestStatus = &ufsapi.TestStatus{
		Code: ufsapi.TestStatus_OK,
	}
	return response, nil
}
