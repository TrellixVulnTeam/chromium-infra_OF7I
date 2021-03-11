// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package generator

import (
	"testing"

	"infra/cros/internal/gerrit"

	_struct "github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/testplans"
	bbproto "go.chromium.org/luci/buildbucket/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

const (
	gsBucket     = "gs://chromeos-image-archive"
	gsPathPrefix = "gs/path/"
)

var (
	emptyGerritChanges []*bbproto.GerritChange
)

func simpleFilesByArtifactValue() *_struct.Value {
	return &_struct.Value{Kind: &_struct.Value_StructValue{StructValue: &_struct.Struct{
		Fields: map[string]*_struct.Value{
			"AUTOTEST_FILES": {Kind: &_struct.Value_ListValue{}},
		},
	}}}
}

func simpleFilesByArtifact() *_struct.Struct {
	return &_struct.Struct{Fields: simpleFilesByArtifactValue().GetStructValue().Fields}
}

func makeBuildbucketBuild(buildTarget string, builderName string, status bbproto.Status, critical bool) *bbproto.Build {
	var criticalVal bbproto.Trinary
	if critical {
		criticalVal = bbproto.Trinary_YES
	} else {
		criticalVal = bbproto.Trinary_NO
	}
	b := &bbproto.Build{
		Builder:  &bbproto.BuilderID{Builder: builderName},
		Critical: criticalVal,
		Input:    &bbproto.Build_Input{},
		Output: &bbproto.Build_Output{
			Properties: &_struct.Struct{
				Fields: map[string]*_struct.Value{
					"build_target": {
						Kind: &_struct.Value_StructValue{StructValue: &_struct.Struct{
							Fields: map[string]*_struct.Value{
								"name": {Kind: &_struct.Value_StringValue{StringValue: buildTarget}},
							},
						}},
					},
					"artifacts": {
						Kind: &_struct.Value_StructValue{StructValue: &_struct.Struct{
							Fields: map[string]*_struct.Value{
								"gs_bucket":         {Kind: &_struct.Value_StringValue{StringValue: gsBucket}},
								"gs_path":           {Kind: &_struct.Value_StringValue{StringValue: gsPathPrefix + buildTarget}},
								"files_by_artifact": simpleFilesByArtifactValue(),
							},
						}},
					},
				},
			},
		},
		Status: status,
	}
	return b
}

func TestCreateCombinedTestPlan_oneUnitSuccess(t *testing.T) {
	kevinHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common:          &testplans.TestSuiteCommon{},
			Suite:           "HW kevin",
			SkylabBoard:     "kev",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		},
	}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				HwTestCfg: kevinHWTestCfg,
			},
		},
	}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern:     &testplans.FilePattern{Pattern: "hw/tests/not/needed/here/**"},
				SubtractiveRule: &testplans.SubtractiveRule{DisableHwTests: true},
			},
		}}
	bbBuilds := []*bbproto.Build{
		makeBuildbucketBuild("kevin", "kevin-cq", bbproto.Status_SUCCESS, true),
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/repo/name": {"refs/heads/master": "src/to/file"}}

	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, emptyGerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{
		HwTestUnits: []*testplans.HwTestUnit{
			{
				Common: &testplans.TestUnitCommon{
					BuildPayload: &testplans.BuildPayload{
						ArtifactsGsBucket: gsBucket,
						ArtifactsGsPath:   gsPathPrefix + "kevin",
						FilesByArtifact:   simpleFilesByArtifact(),
					},
					BuilderName: "kevin-cq",
					BuildTarget: &chromiumos.BuildTarget{Name: "kevin"}},
				HwTestCfg: kevinHWTestCfg,
			},
		},
	}
	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}

func TestCreateCombinedTestPlan_manyUnitSuccess(t *testing.T) {
	kevinDebugVMTestCfg := &testplans.VmTestCfg{VmTest: []*testplans.VmTestCfg_VmTest{
		{TestSuite: "VM kevin debug kernel", Common: &testplans.TestSuiteCommon{Critical: &wrappers.BoolValue{Value: true}}},
	}}
	kevinHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common:      &testplans.TestSuiteCommon{},
			Suite:       "HW kevin",
			SkylabBoard: "kev",
		},
	}}
	kevinTastVMTestCfg := &testplans.TastVmTestCfg{TastVmTest: []*testplans.TastVmTestCfg_TastVmTest{
		{
			Common:    &testplans.TestSuiteCommon{},
			SuiteName: "Tast kevin"},
	}}
	kevinVMTestCfg := &testplans.VmTestCfg{VmTest: []*testplans.VmTestCfg_VmTest{
		{
			Common:    &testplans.TestSuiteCommon{},
			TestSuite: "VM kevin"},
	}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-debug-kernel-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				VmTestCfg: kevinDebugVMTestCfg},
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				HwTestCfg:           kevinHWTestCfg,
				DirectTastVmTestCfg: kevinTastVMTestCfg,
				VmTestCfg:           kevinVMTestCfg},
		},
	}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern:     &testplans.FilePattern{Pattern: "hw/tests/not/needed/here/**"},
				SubtractiveRule: &testplans.SubtractiveRule{DisableHwTests: true},
			},
		}}
	bbBuilds := []*bbproto.Build{
		makeBuildbucketBuild("kevin", "kevin-cq", bbproto.Status_SUCCESS, true),
		makeBuildbucketBuild("kevin", "kevin-debug-kernel-cq", bbproto.Status_SUCCESS, true),
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/master",
			Project: "chromiumos/repo/name",
			Files:   []string{"a/b/c"},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/repo/name": {"refs/heads/master": "src/to/file"}}
	gerritChanges := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2}}
	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, gerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{
		HwTestUnits: []*testplans.HwTestUnit{
			{
				Common: &testplans.TestUnitCommon{
					BuildPayload: &testplans.BuildPayload{
						ArtifactsGsBucket: gsBucket,
						ArtifactsGsPath:   gsPathPrefix + "kevin",
						FilesByArtifact:   simpleFilesByArtifact(),
					},
					BuilderName: "kevin-cq",
					BuildTarget: &chromiumos.BuildTarget{Name: "kevin"}},
				HwTestCfg: kevinHWTestCfg},
		},
		DirectTastVmTestUnits: []*testplans.TastVmTestUnit{
			{
				Common: &testplans.TestUnitCommon{
					BuildPayload: &testplans.BuildPayload{
						ArtifactsGsBucket: gsBucket,
						ArtifactsGsPath:   gsPathPrefix + "kevin",
						FilesByArtifact:   simpleFilesByArtifact(),
					},
					BuilderName: "kevin-cq",
					BuildTarget: &chromiumos.BuildTarget{Name: "kevin"}},
				TastVmTestCfg: kevinTastVMTestCfg},
		},
		VmTestUnits: []*testplans.VmTestUnit{
			{
				Common: &testplans.TestUnitCommon{
					BuildPayload: &testplans.BuildPayload{
						ArtifactsGsBucket: gsBucket,
						ArtifactsGsPath:   gsPathPrefix + "kevin",
						FilesByArtifact:   simpleFilesByArtifact(),
					},
					BuilderName: "kevin-debug-kernel-cq",
					BuildTarget: &chromiumos.BuildTarget{Name: "kevin"}},
				VmTestCfg: kevinDebugVMTestCfg},
			{
				Common: &testplans.TestUnitCommon{
					BuildPayload: &testplans.BuildPayload{
						ArtifactsGsBucket: gsBucket,
						ArtifactsGsPath:   gsPathPrefix + "kevin",
						FilesByArtifact:   simpleFilesByArtifact(),
					},
					BuilderName: "kevin-cq",
					BuildTarget: &chromiumos.BuildTarget{Name: "kevin"}},
				VmTestCfg: kevinVMTestCfg},
		}}
	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}

func TestCreateCombinedTestPlan_successDespiteOneFailedBuilder(t *testing.T) {
	// In this test, the kevin builder failed, so the output test plan will not contain a test unit
	// for kevin.

	reefHwTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common:          &testplans.TestSuiteCommon{},
			SkylabBoard:     "some reef",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST},
	}}
	kevinVMTestCfg := &testplans.VmTestCfg{VmTest: []*testplans.VmTestCfg_VmTest{
		{
			Common:    &testplans.TestSuiteCommon{},
			TestSuite: "VM kevin"},
	}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "reef-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "reef"}},
				HwTestCfg: reefHwTestCfg},
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				VmTestCfg: kevinVMTestCfg},
		},
	}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern:     &testplans.FilePattern{Pattern: "hw/tests/not/needed/here/**"},
				SubtractiveRule: &testplans.SubtractiveRule{DisableHwTests: true},
			},
		}}
	bbBuilds := []*bbproto.Build{
		makeBuildbucketBuild("kevin", "kevin-cq", bbproto.Status_FAILURE, true),
		makeBuildbucketBuild("reef", "reef-cq", bbproto.Status_SUCCESS, true),
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/master",
			Project: "chromiumos/repo/name",
			Files:   []string{"a/b/c"},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/repo/name": {"refs/heads/master": "src/to/file"}}
	gerritChanges := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2},
	}

	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, gerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{
		HwTestUnits: []*testplans.HwTestUnit{
			{
				Common: &testplans.TestUnitCommon{
					BuildPayload: &testplans.BuildPayload{
						ArtifactsGsBucket: gsBucket,
						ArtifactsGsPath:   gsPathPrefix + "reef",
						FilesByArtifact:   simpleFilesByArtifact(),
					},
					BuilderName: "reef-cq",
					BuildTarget: &chromiumos.BuildTarget{Name: "reef"}},
				HwTestCfg: reefHwTestCfg},
		}}

	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}

func TestCreateCombinedTestPlan_skipsUnnecessaryHardwareTest(t *testing.T) {
	kevinHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common:          &testplans.TestSuiteCommon{},
			Suite:           "HW kevin",
			SkylabBoard:     "kev",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		},
	}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				HwTestCfg: kevinHWTestCfg},
		},
	}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern:     &testplans.FilePattern{Pattern: "no/hw/tests/here/some/**"},
				SubtractiveRule: &testplans.SubtractiveRule{DisableHwTests: true},
			},
		}}
	bbBuilds := []*bbproto.Build{
		makeBuildbucketBuild("kevin", "kevin-cq", bbproto.Status_SUCCESS, true),
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/master",
			Project: "chromiumos/test/repo/name",
			Files:   []string{"some/file"},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/test/repo/name": {"refs/heads/master": "no/hw/tests/here"}}
	gerritChanges := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2},
	}

	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, gerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{}

	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}

func TestCreateCombinedTestPlan_doesOnlyTest(t *testing.T) {
	kevinHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common: &testplans.TestSuiteCommon{
				DisplayName:     "kev-cq.bvt-some-suite",
				TestSuiteGroups: []*testplans.TestSuiteCommon_TestSuiteGroup{{TestSuiteGroup: "my-test-testGroup"}},
			},
			Suite:           "HW kevin",
			SkylabBoard:     "kev",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		},
	}}
	bobHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common: &testplans.TestSuiteCommon{
				DisplayName:     "bob-cq.bvt-some-suite",
				TestSuiteGroups: []*testplans.TestSuiteCommon_TestSuiteGroup{{TestSuiteGroup: "some-other-test-testGroup"}},
			},
			Suite:           "HW bob",
			SkylabBoard:     "bob board",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		},
	}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				HwTestCfg: kevinHWTestCfg},
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "bob-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "bob"}},
				HwTestCfg: bobHWTestCfg},
		},
	}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern: &testplans.FilePattern{Pattern: "no/hw/tests/here/some/**"},
				SubtractiveRule: &testplans.SubtractiveRule{
					OnlyKeepAllSuitesInGroups: &testplans.TestGroups{Name: []string{"my-test-testGroup"}},
				},
			},
		}}
	bbBuilds := []*bbproto.Build{
		makeBuildbucketBuild("kevin", "kevin-cq", bbproto.Status_SUCCESS, true),
		makeBuildbucketBuild("bob", "bob-cq", bbproto.Status_SUCCESS, true),
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/master",
			Project: "chromiumos/test/repo/name",
			Files:   []string{"some/file"},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/test/repo/name": {"refs/heads/master": "no/hw/tests/here"}}
	gerritChanges := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2},
	}

	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, gerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{
		HwTestUnits: []*testplans.HwTestUnit{
			{
				Common: &testplans.TestUnitCommon{
					BuildPayload: &testplans.BuildPayload{
						ArtifactsGsBucket: gsBucket,
						ArtifactsGsPath:   gsPathPrefix + "kevin",
						FilesByArtifact:   simpleFilesByArtifact(),
					},
					BuilderName: "kevin-cq",
					BuildTarget: &chromiumos.BuildTarget{Name: "kevin"}},
				HwTestCfg: kevinHWTestCfg},
		},
	}

	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}

func TestCreateCombinedTestPlan_doesOnlyOneofTest(t *testing.T) {
	boardPriorityList := &testplans.BoardPriorityList{
		BoardPriorities: []*testplans.BoardPriority{
			{SkylabBoard: "kev", Priority: -1},
			{SkylabBoard: "bob", Priority: 1},
		},
	}
	kevinHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common: &testplans.TestSuiteCommon{
				DisplayName:     "kev-cq.bvt-some-suite",
				TestSuiteGroups: []*testplans.TestSuiteCommon_TestSuiteGroup{{TestSuiteGroup: "my-test-testGroup"}},
			},
			Suite:           "HW kevin",
			SkylabBoard:     "kev",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		},
	}}
	bobHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common: &testplans.TestSuiteCommon{
				DisplayName:     "bob-cq.bvt-some-suite",
				TestSuiteGroups: []*testplans.TestSuiteCommon_TestSuiteGroup{{TestSuiteGroup: "my-test-testGroup"}},
			},
			Suite:           "HW bob",
			SkylabBoard:     "bob board",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		},
	}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				HwTestCfg: kevinHWTestCfg},
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "bob-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "bob"}},
				HwTestCfg: bobHWTestCfg},
		},
	}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern: &testplans.FilePattern{Pattern: "no/hw/tests/here/some/**"},
				SubtractiveRule: &testplans.SubtractiveRule{
					OnlyKeepOneSuiteFromEachGroup: &testplans.TestGroups{Name: []string{"my-test-testGroup", "irrelevant-test-testGroup"}},
				},
			},
		}}
	bbBuilds := []*bbproto.Build{
		makeBuildbucketBuild("kevin", "kevin-cq", bbproto.Status_SUCCESS, true),
		makeBuildbucketBuild("bob", "bob-cq", bbproto.Status_SUCCESS, true),
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/master",
			Project: "chromiumos/test/repo/name",
			Files:   []string{"some/file"},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/test/repo/name": {"refs/heads/master": "no/hw/tests/here"}}
	gerritChanges := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2},
	}

	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, boardPriorityList, bbBuilds, gerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{
		HwTestUnits: []*testplans.HwTestUnit{
			{
				Common: &testplans.TestUnitCommon{
					BuildPayload: &testplans.BuildPayload{
						ArtifactsGsBucket: gsBucket,
						ArtifactsGsPath:   gsPathPrefix + "kevin",
						FilesByArtifact:   simpleFilesByArtifact(),
					},
					BuilderName: "kevin-cq",
					BuildTarget: &chromiumos.BuildTarget{Name: "kevin"}},
				HwTestCfg: kevinHWTestCfg},
		},
	}

	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}

func TestCreateCombinedTestPlan_doesAddOneofTest(t *testing.T) {
	boardPriorityList := &testplans.BoardPriorityList{
		BoardPriorities: []*testplans.BoardPriority{
			{SkylabBoard: "kev", Priority: -1},
			{SkylabBoard: "bob", Priority: 1},
		},
	}
	kevinHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common: &testplans.TestSuiteCommon{
				DisplayName:      "kev-cq.bvt-some-suite",
				TestSuiteGroups:  []*testplans.TestSuiteCommon_TestSuiteGroup{{TestSuiteGroup: "my-test-testGroup"}},
				DisableByDefault: true,
			},
			Suite:           "HW kevin",
			SkylabBoard:     "kev",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		},
	}}
	bobHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common: &testplans.TestSuiteCommon{
				DisplayName:      "bob-cq.bvt-some-suite",
				TestSuiteGroups:  []*testplans.TestSuiteCommon_TestSuiteGroup{{TestSuiteGroup: "my-test-testGroup"}},
				DisableByDefault: true,
			},
			Suite:           "HW bob",
			SkylabBoard:     "bob board",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		},
	}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				HwTestCfg: kevinHWTestCfg},
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "bob-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "bob"}},
				HwTestCfg: bobHWTestCfg},
		},
	}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern: &testplans.FilePattern{Pattern: "no/hw/tests/here/some/**"},
				AdditiveRule: &testplans.AdditiveRule{
					AddOneSuiteFromEachGroup: &testplans.TestGroups{Name: []string{"my-test-testGroup", "irrelevant-test-testGroup"}},
				},
			},
		}}
	bbBuilds := []*bbproto.Build{
		makeBuildbucketBuild("kevin", "kevin-cq", bbproto.Status_SUCCESS, true),
		makeBuildbucketBuild("bob", "bob-cq", bbproto.Status_SUCCESS, true),
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/master",
			Project: "chromiumos/test/repo/name",
			Files:   []string{"some/file"},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/test/repo/name": {"refs/heads/master": "no/hw/tests/here"}}
	gerritChanges := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2},
	}

	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, boardPriorityList, bbBuilds, gerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{
		HwTestUnits: []*testplans.HwTestUnit{
			{
				Common: &testplans.TestUnitCommon{
					BuildPayload: &testplans.BuildPayload{
						ArtifactsGsBucket: gsBucket,
						ArtifactsGsPath:   gsPathPrefix + "kevin",
						FilesByArtifact:   simpleFilesByArtifact(),
					},
					BuilderName: "kevin-cq",
					BuildTarget: &chromiumos.BuildTarget{Name: "kevin"}},
				HwTestCfg: kevinHWTestCfg},
		},
	}

	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}

func TestCreateCombinedTestPlan_doesAlsoTest(t *testing.T) {
	kevinHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common: &testplans.TestSuiteCommon{
				DisplayName:      "kevin-cq.bvt-some-suite",
				TestSuiteGroups:  []*testplans.TestSuiteCommon_TestSuiteGroup{{TestSuiteGroup: "my-test-testGroup"}},
				DisableByDefault: true,
			},
			Suite:           "HW kevin",
			SkylabBoard:     "kev",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		},
	}}
	bobHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common: &testplans.TestSuiteCommon{
				DisplayName:      "bob-cq.bvt-some-suite",
				TestSuiteGroups:  []*testplans.TestSuiteCommon_TestSuiteGroup{{TestSuiteGroup: "some-other-test-testGroup"}},
				DisableByDefault: true,
			},
			Suite:           "HW bob",
			SkylabBoard:     "bob board",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
			Pool:            "my little pool",
		},
	}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				HwTestCfg: kevinHWTestCfg},
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "bob-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "bob"}},
				HwTestCfg: bobHWTestCfg},
		},
	}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern: &testplans.FilePattern{Pattern: "no/hw/tests/here/some/**"},
				AdditiveRule: &testplans.AdditiveRule{
					AddAllSuitesInGroups: &testplans.TestGroups{Name: []string{"my-test-testGroup"}},
				},
			},
		}}
	bbBuilds := []*bbproto.Build{
		makeBuildbucketBuild("kevin", "kevin-cq", bbproto.Status_SUCCESS, true),
		makeBuildbucketBuild("bob", "bob-cq", bbproto.Status_SUCCESS, true),
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/master",
			Project: "chromiumos/test/repo/name",
			Files:   []string{"some/file"},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/test/repo/name": {"refs/heads/master": "no/hw/tests/here"}}
	gerritChanges := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2},
	}

	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, gerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{
		HwTestUnits: []*testplans.HwTestUnit{
			{
				Common: &testplans.TestUnitCommon{
					BuildPayload: &testplans.BuildPayload{
						ArtifactsGsBucket: gsBucket,
						ArtifactsGsPath:   gsPathPrefix + "kevin",
						FilesByArtifact:   simpleFilesByArtifact(),
					},
					BuilderName: "kevin-cq",
					BuildTarget: &chromiumos.BuildTarget{Name: "kevin"}},
				HwTestCfg: kevinHWTestCfg,
			}}}

	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}

func TestCreateCombinedTestPlan_fileExcludePattern(t *testing.T) {
	kevinHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common: &testplans.TestSuiteCommon{
				DisplayName:      "kev-cq.bvt-some-suite",
				TestSuiteGroups:  []*testplans.TestSuiteCommon_TestSuiteGroup{{TestSuiteGroup: "my-test-testGroup"}},
				DisableByDefault: true,
			},
			Suite:           "HW kevin",
			SkylabBoard:     "kev",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		},
	}}
	bobHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common: &testplans.TestSuiteCommon{
				DisplayName:      "bob-cq.bvt-some-suite",
				TestSuiteGroups:  []*testplans.TestSuiteCommon_TestSuiteGroup{{TestSuiteGroup: "my-test-testGroup"}},
				DisableByDefault: true,
			},
			Suite:           "HW bob",
			SkylabBoard:     "bob board",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		},
	}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				HwTestCfg: kevinHWTestCfg},
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "bob-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "bob"}},
				HwTestCfg: bobHWTestCfg},
		},
	}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern: &testplans.FilePattern{
					Pattern: "run/my-test-testGroup/here/some/**",
					ExcludePatterns: []string{
						"run/my-test-testGroup/here/some/**/except/here/**",
						"run/my-test-testGroup/here/some/**/or/here/**",
					},
				},
				AdditiveRule: &testplans.AdditiveRule{
					AddOneSuiteFromEachGroup: &testplans.TestGroups{Name: []string{"my-test-testGroup", "irrelevant-test-testGroup"}},
				},
			},
		}}
	bbBuilds := []*bbproto.Build{
		makeBuildbucketBuild("kevin", "kevin-cq", bbproto.Status_SUCCESS, true),
		makeBuildbucketBuild("bob", "bob-cq", bbproto.Status_SUCCESS, true),
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/main",
			Project: "chromiumos/test/repo/name",
			Files:   []string{"some/file/except/here/a.txt", "some/file/or/here/b.txt"},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/test/repo/name": {"refs/heads/main": "run/my-test-testGroup/here"}}
	gerritChanges := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2},
	}

	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, gerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{}

	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}

func TestCreateCombinedTestPlan_inputMissingTargetType(t *testing.T) {
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			// This is missing a TargetType.
			{TargetCriteria: &testplans.TargetCriteria{}},
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"},
				}}}}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{}
	bbBuilds := []*bbproto.Build{}
	if _, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, emptyGerritChanges, &gerrit.ChangeRevData{}, map[string]map[string]string{}); err == nil {
		t.Errorf("Expected an error to be returned")
	}
}

func TestCreateCombinedTestPlan_skipsPointlessBuild(t *testing.T) {
	kevinHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common:          &testplans.TestSuiteCommon{},
			Suite:           "HW kevin",
			SkylabBoard:     "kev",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		}}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				HwTestCfg: kevinHWTestCfg},
		}}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern:     &testplans.FilePattern{Pattern: "hw/tests/not/needed/here/**"},
				SubtractiveRule: &testplans.SubtractiveRule{DisableHwTests: true},
			},
		}}
	bbBuild := makeBuildbucketBuild("kevin", "kevin-cq", bbproto.Status_SUCCESS, true)
	bbBuild.Output.Properties.Fields["pointless_build"] = &_struct.Value{Kind: &_struct.Value_BoolValue{BoolValue: true}}
	bbBuilds := []*bbproto.Build{bbBuild}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/master",
			Project: "chromiumos/repo/name",
			Files:   []string{"a/b/c"},
		}})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/repo/name": {"refs/heads/master": "src/to/file"}}
	gerritChanges := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2},
	}

	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, gerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{}

	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}

func TestCreateTestPlan_succeedsOnNoBuildTarget(t *testing.T) {
	testReqs := &testplans.TargetTestRequirementsCfg{}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{}
	bbBuilds := []*bbproto.Build{
		// build target is empty.
		makeBuildbucketBuild("", "kevin-cq", bbproto.Status_FAILURE, true),
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{})
	repoToBranchToSrcRoot := map[string]map[string]string{}

	_, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, emptyGerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Errorf("expected no error, but got %v", err)
	}
}

func TestCreateCombinedTestPlan_doesNotSkipNonCritical(t *testing.T) {
	// In this test, the build is not critical, but we test it anyway!

	reefHwTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common:          &testplans.TestSuiteCommon{},
			SkylabBoard:     "my reef",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		}}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "reef-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "reef"}},
				HwTestCfg: reefHwTestCfg,
			}}}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern:     &testplans.FilePattern{Pattern: "hw/tests/not/needed/here/**"},
				SubtractiveRule: &testplans.SubtractiveRule{DisableHwTests: true},
			},
		}}
	bbBuilds := []*bbproto.Build{
		makeBuildbucketBuild("reef", "reef-cq", bbproto.Status_SUCCESS, false),
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/master",
			Project: "chromiumos/repo/name",
			Files:   []string{"a/b/c"},
		}})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/repo/name": {"refs/heads/master": "src/to/file"}}
	gerritChanges := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2},
	}

	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, gerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{
		HwTestUnits: []*testplans.HwTestUnit{
			{
				Common: &testplans.TestUnitCommon{
					BuildPayload: &testplans.BuildPayload{
						ArtifactsGsBucket: gsBucket,
						ArtifactsGsPath:   gsPathPrefix + "reef",
						FilesByArtifact:   simpleFilesByArtifact(),
					},
					BuilderName: "reef-cq",
					BuildTarget: &chromiumos.BuildTarget{Name: "reef"}},
				HwTestCfg: reefHwTestCfg,
			}}}

	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}

func TestCreateCombinedTestPlan_ignoresNonArtifactBuild(t *testing.T) {
	kevinHWTestCfg := &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
		{
			Common:          &testplans.TestSuiteCommon{},
			Suite:           "HW kevin",
			SkylabBoard:     "kev",
			HwTestSuiteType: testplans.HwTestCfg_AUTOTEST,
		}}}
	testReqs := &testplans.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans.TargetCriteria{
					BuilderName: "kevin-cq",
					TargetType:  &testplans.TargetCriteria_BuildTarget{BuildTarget: "kevin"}},
				HwTestCfg: kevinHWTestCfg,
			}}}
	sourceTreeTestCfg := &testplans.SourceTreeTestCfg{
		SourceTestRules: []*testplans.SourceTestRules{
			{
				FilePattern:     &testplans.FilePattern{Pattern: "hw/tests/not/needed/here/**"},
				SubtractiveRule: &testplans.SubtractiveRule{DisableHwTests: true},
			},
		}}
	build := makeBuildbucketBuild("kevin", "kevin-cq", bbproto.Status_SUCCESS, true)

	// Remove the AUTOTEST_FILES files_by_artifact key, thus making this whole
	// build unusable for testing.
	delete(
		build.GetOutput().GetProperties().GetFields()["artifacts"].GetStructValue().GetFields()["files_by_artifact"].GetStructValue().GetFields(),
		"AUTOTEST_FILES")
	bbBuilds := []*bbproto.Build{build}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{})
	repoToBranchToSrcRoot := map[string]map[string]string{"chromiumos/repo/name": {"refs/heads/master": "src/to/file"}}

	actualTestPlan, err := CreateTestPlan(testReqs, sourceTreeTestCfg, &testplans.BoardPriorityList{}, bbBuilds, emptyGerritChanges, chRevData, repoToBranchToSrcRoot)
	if err != nil {
		t.Error(err)
	}

	expectedTestPlan := &testplans.GenerateTestPlanResponse{}
	if diff := cmp.Diff(expectedTestPlan, actualTestPlan, protocmp.Transform()); diff != "" {
		t.Errorf("CreateCombinedTestPlan bad result (-want/+got)\n%s", diff)
	}
}
