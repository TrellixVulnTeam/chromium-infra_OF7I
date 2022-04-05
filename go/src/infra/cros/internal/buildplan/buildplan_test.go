// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package buildplan

import (
	"testing"

	"infra/cros/internal/gerrit"

	cros_pb "go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	testplans_pb "go.chromium.org/chromiumos/infra/proto/go/testplans"
	bbproto "go.chromium.org/luci/buildbucket/proto"
)

func makeBuilderConfig(name string, idType cros_pb.BuilderConfig_Id_Type, rwMode cros_pb.BuilderConfig_General_RunWhen_Mode, rwPatterns []string) *cros_pb.BuilderConfig {
	b := &cros_pb.BuilderConfig{
		Id: &cros_pb.BuilderConfig_Id{
			Name: name,
			Type: idType,
		},
		General: &cros_pb.BuilderConfig_General{
			RunWhen: &cros_pb.BuilderConfig_General_RunWhen{
				Mode:         rwMode,
				FilePatterns: rwPatterns,
			},
		},
		Artifacts: &cros_pb.BuilderConfig_Artifacts{},
	}
	return b
}

func TestCheckBuilders_imageBuilderFiltering(t *testing.T) {
	changes := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2, Project: "chromiumos/public/example"}}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/main",
			Project: "chromiumos/public/example",
			Files: []string{
				"chromite-maybe/someotherdir/ignore_me.txt",
			},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{
		"chromiumos/public/example": {"refs/heads/main": "src/pub/ex"},
	}

	buildIrrelevanceCfg := &testplans_pb.BuildIrrelevanceCfg{
		IrrelevantFilePatterns: []*testplans_pb.FilePattern{
			{Pattern: "**/ignore_me.txt"},
		},
	}

	slimBuildCfg := &testplans_pb.SlimBuildCfg{}

	testReqsCfg := &testplans_pb.TargetTestRequirementsCfg{}

	builderConfigs := &cros_pb.BuilderConfigs{}

	b := []*cros_pb.BuilderConfig{
		makeBuilderConfig("my_image_builder", cros_pb.BuilderConfig_Id_TYPE_UNSPECIFIED, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
		makeBuilderConfig("chromite-not_an_image_builder", cros_pb.BuilderConfig_Id_TYPE_UNSPECIFIED, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
	}

	checkBuildersInput := &CheckBuildersInput{
		Builders:              b,
		Changes:               changes,
		ChangeRevs:            chRevData,
		RepoToBranchToSrcRoot: repoToBranchToSrcRoot,
		BuildIrrelevanceCfg:   buildIrrelevanceCfg,
		SlimBuildCfg:          slimBuildCfg,
		TestReqsCfg:           testReqsCfg,
		BuilderConfigs:        builderConfigs,
	}

	res, err := checkBuildersInput.CheckBuilders()
	if err != nil {
		t.Error(err)
	}
	if len(res.BuildsToRun) != 1 {
		t.Errorf("Expected BuildsToRun to have 1 element. Instead, %v", res.BuildsToRun)
	}
	if res.BuildsToRun[0].GetName() != "chromite-not_an_image_builder" {
		t.Errorf("Expected res.BuildsToRun[0].GetName() == \"chromite-not_an_image_builder\". Instead, %v", res.BuildsToRun[0].GetName())
	}
	if len(res.SkipForRunWhenRules) != 0 {
		t.Errorf("Expected SkipForRunWhenRules to be empty. Instead, %v", res.SkipForRunWhenRules)
	}
	if len(res.SkipForGlobalBuildIrrelevance) != 1 {
		t.Errorf("Expected SkipForGlobalBuildIrrelevance to have 1 element. Instead, %v", res.SkipForGlobalBuildIrrelevance)
	}
	if res.SkipForGlobalBuildIrrelevance[0].GetName() != "my_image_builder" {
		t.Errorf("Expected SkipForGlobalBuildIrrelevance[0].GetName() == \"my_image_builder\", instead %v", res.SkipForGlobalBuildIrrelevance[0].GetName())
	}
}

func TestCheckBuilders_hasManifestXMLChange(t *testing.T) {
	changes := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2, Project: "chromiumos/public/example"}}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/main",
			Project: "chromiumos/manifest-internal",
			Files: []string{
				"readme.md",
				"full.xml",
			},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{
		"chromiumos/manifest-internal": {"refs/heads/main": "manifest-internal"},
	}

	buildIrrelevanceCfg := &testplans_pb.BuildIrrelevanceCfg{}
	slimBuildCfg := &testplans_pb.SlimBuildCfg{}
	testReqsCfg := &testplans_pb.TargetTestRequirementsCfg{}
	builderConfigs := &cros_pb.BuilderConfigs{}

	b := []*cros_pb.BuilderConfig{
		makeBuilderConfig("my_image_builder", cros_pb.BuilderConfig_Id_TYPE_UNSPECIFIED, cros_pb.BuilderConfig_General_RunWhen_ONLY_RUN_ON_FILE_MATCH, []string{"some/path"}),
		makeBuilderConfig("chromite-not_an_image_builder", cros_pb.BuilderConfig_Id_TYPE_UNSPECIFIED, cros_pb.BuilderConfig_General_RunWhen_ONLY_RUN_ON_FILE_MATCH, []string{"other/path"}),
	}

	checkBuildersInput := &CheckBuildersInput{
		Builders:              b,
		Changes:               changes,
		ChangeRevs:            chRevData,
		RepoToBranchToSrcRoot: repoToBranchToSrcRoot,
		BuildIrrelevanceCfg:   buildIrrelevanceCfg,
		SlimBuildCfg:          slimBuildCfg,
		TestReqsCfg:           testReqsCfg,
		BuilderConfigs:        builderConfigs,
	}

	res, err := checkBuildersInput.CheckBuilders()
	if err != nil {
		t.Error(err)
	}
	if len(res.BuildsToRun) != 2 {
		t.Errorf("Expected BuildsToRun to have 2 elements. Instead, %v", res.BuildsToRun)
	}
	if len(res.SkipForRunWhenRules) != 0 {
		t.Errorf("Expected SkipForRunWhenRules to be empty. Instead, %v", res.SkipForRunWhenRules)
	}
	if len(res.SkipForGlobalBuildIrrelevance) != 0 {
		t.Errorf("Expected SkipForGlobalBuildIrrelevance to be empty. Instead, %v", res.SkipForGlobalBuildIrrelevance)
	}
}

func TestCheckBuilders_noGerritChanges(t *testing.T) {
	// When there are no GerritChanges, we run all of the builders as full builds.

	changes := []*bbproto.GerritChange{}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{})
	repoToBranchToSrcRoot := map[string]map[string]string{}

	buildIrrelevanceCfg := &testplans_pb.BuildIrrelevanceCfg{
		IrrelevantFilePatterns: []*testplans_pb.FilePattern{
			{Pattern: "**/ignore_me.txt"},
		},
	}

	slimBuildCfg := &testplans_pb.SlimBuildCfg{}

	testReqsCfg := &testplans_pb.TargetTestRequirementsCfg{}

	builderConfigs := &cros_pb.BuilderConfigs{}

	b := []*cros_pb.BuilderConfig{
		makeBuilderConfig("my_image_builder", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
		makeBuilderConfig("not_an_image_builder", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
		makeBuilderConfig("only_run_on_match", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ONLY_RUN_ON_FILE_MATCH, []string{"**/match_me.txt"}),
		makeBuilderConfig("no_run_on_match", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_NO_RUN_ON_FILE_MATCH, []string{"not/a/real/dir"}),
	}

	checkBuildersInput := &CheckBuildersInput{
		Builders:              b,
		Changes:               changes,
		ChangeRevs:            chRevData,
		RepoToBranchToSrcRoot: repoToBranchToSrcRoot,
		BuildIrrelevanceCfg:   buildIrrelevanceCfg,
		SlimBuildCfg:          slimBuildCfg,
		TestReqsCfg:           testReqsCfg,
		BuilderConfigs:        builderConfigs,
	}

	res, err := checkBuildersInput.CheckBuilders()
	if err != nil {
		t.Error(err)
	}
	if len(res.BuildsToRun) != 4 {
		t.Errorf("Expected BuildsToRun to have 4 elements. Instead, %v", res.BuildsToRun)
	}
	expectedBuilderNames := []string{"my_image_builder", "not_an_image_builder", "only_run_on_match", "no_run_on_match"}
	actualBuilderNames := make([]string, 0)
	for _, b := range res.BuildsToRun {
		actualBuilderNames = append(actualBuilderNames, b.GetName())
	}
	if len(sliceDiff(expectedBuilderNames, actualBuilderNames)) != 0 {
		t.Errorf("Expected res.BuildsToRun to contain builder names %v. Instead, %v", expectedBuilderNames, actualBuilderNames)
	}
	if len(res.SkipForRunWhenRules) != 0 {
		t.Errorf("Expected SkipForRunWhenRules to be empty. Instead, %v", res.SkipForRunWhenRules)
	}
	if len(res.SkipForGlobalBuildIrrelevance) != 0 {
		t.Errorf("Expected SkipForGlobalBuildIrrelevance to be empty. Instead, %v", res.SkipForGlobalBuildIrrelevance)
	}
}

func TestCheckBuilders_withGerritChangesNoAffectedFiles(t *testing.T) {
	// When there are GerritChanges, but no affected files, we run no builds.

	changes := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2, Project: "chromiumos/public/example"}}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/main",
			Project: "chromiumos/public/example",
			Files: []string{
				"chromite-maybe/someotherdir/ignore_me.txt",
			},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{
		"chromiumos/public/example": {"refs/heads/other": "src/pub/ex"},
	}

	buildIrrelevanceCfg := &testplans_pb.BuildIrrelevanceCfg{}
	slimBuildCfg := &testplans_pb.SlimBuildCfg{}
	testReqsCfg := &testplans_pb.TargetTestRequirementsCfg{}
	builderConfigs := &cros_pb.BuilderConfigs{}

	b := []*cros_pb.BuilderConfig{
		makeBuilderConfig("my_image_builder", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
		makeBuilderConfig("not_an_image_builder", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
		makeBuilderConfig("only_run_on_match", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ONLY_RUN_ON_FILE_MATCH, []string{"**/match_me.txt"}),
		makeBuilderConfig("no_run_on_match", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_NO_RUN_ON_FILE_MATCH, []string{"not/a/real/dir"}),
	}

	checkBuildersInput := &CheckBuildersInput{
		Builders:              b,
		Changes:               changes,
		ChangeRevs:            chRevData,
		RepoToBranchToSrcRoot: repoToBranchToSrcRoot,
		BuildIrrelevanceCfg:   buildIrrelevanceCfg,
		SlimBuildCfg:          slimBuildCfg,
		TestReqsCfg:           testReqsCfg,
		BuilderConfigs:        builderConfigs,
	}

	res, err := checkBuildersInput.CheckBuilders()
	if err != nil {
		t.Error(err)
	}
	if len(res.BuildsToRun) != 0 {
		t.Errorf("Expected BuildsToRun to be empty. Instead, %v", res.BuildsToRun)
	}
	if len(res.SkipForRunWhenRules) != 0 {
		t.Errorf("Expected SkipForRunWhenRules to be empty. Instead, %v", res.SkipForRunWhenRules)
	}
	if len(res.SkipForGlobalBuildIrrelevance) != 4 {
		t.Errorf("Expected SkipForGlobalBuildIrrelevance to have 4 elements. Instead, %v", res.SkipForGlobalBuildIrrelevance)
	}
	expectedBuilderNames := []string{"my_image_builder", "not_an_image_builder", "only_run_on_match", "no_run_on_match"}
	actualBuilderNames := make([]string, 0)
	for _, b := range res.SkipForGlobalBuildIrrelevance {
		actualBuilderNames = append(actualBuilderNames, b.GetName())
	}
	if len(sliceDiff(expectedBuilderNames, actualBuilderNames)) != 0 {
		t.Errorf("Expected res.SkipForGlobalBuildIrrelevance to contain builder names %v. Instead, %v", expectedBuilderNames, actualBuilderNames)
	}
}

func TestCheckBuilders_onlyRunOnFileMatch(t *testing.T) {
	changes := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2, Project: "chromiumos/public/example"}}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/main",
			Project: "chromiumos/public/example",
			Files: []string{
				"chromite-maybe/someotherdir/match_me.txt",
			},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{
		"chromiumos/public/example": {"refs/heads/main": "src/pub/ex"},
	}

	buildIrrelevanceCfg := &testplans_pb.BuildIrrelevanceCfg{}
	slimBuildCfg := &testplans_pb.SlimBuildCfg{}
	testReqsCfg := &testplans_pb.TargetTestRequirementsCfg{}
	builderConfigs := &cros_pb.BuilderConfigs{}

	b := []*cros_pb.BuilderConfig{
		makeBuilderConfig("board_to_run", cros_pb.BuilderConfig_Id_TYPE_UNSPECIFIED, cros_pb.BuilderConfig_General_RunWhen_ONLY_RUN_ON_FILE_MATCH, []string{"**/match_me.txt"}),
		makeBuilderConfig("board_to_skip", cros_pb.BuilderConfig_Id_TYPE_UNSPECIFIED, cros_pb.BuilderConfig_General_RunWhen_ONLY_RUN_ON_FILE_MATCH, []string{"not/a/real/dir"}),
	}

	checkBuildersInput := &CheckBuildersInput{
		Builders:              b,
		Changes:               changes,
		ChangeRevs:            chRevData,
		RepoToBranchToSrcRoot: repoToBranchToSrcRoot,
		BuildIrrelevanceCfg:   buildIrrelevanceCfg,
		SlimBuildCfg:          slimBuildCfg,
		TestReqsCfg:           testReqsCfg,
		BuilderConfigs:        builderConfigs,
	}

	res, err := checkBuildersInput.CheckBuilders()
	if err != nil {
		t.Error(err)
	}
	if len(res.BuildsToRun) != 1 {
		t.Errorf("Expected BuildsToRun to have 1 element. Instead, %v", res.BuildsToRun)
	}
	if res.BuildsToRun[0].GetName() != "board_to_run" {
		t.Errorf("Expected res.BuildsToRun[0].GetName() == \"board_to_run\". Instead, %v", res.BuildsToRun[0].GetName())
	}
	if len(res.SkipForGlobalBuildIrrelevance) != 0 {
		t.Errorf("Expected SkipForGlobalBuildIrrelevance to be empty. Instead, %v", res.SkipForGlobalBuildIrrelevance)
	}
	if len(res.SkipForRunWhenRules) != 1 {
		t.Errorf("Expected SkipForRunWhenRules to have 1 element. Instead, %v", res.SkipForRunWhenRules)
	}
	if res.SkipForRunWhenRules[0].GetName() != "board_to_skip" {
		t.Errorf("Expected SkipForRunWhenRules[0].GetName() == \"board_to_skip\", instead %v", res.SkipForRunWhenRules[0].GetName())
	}
}

func TestCheckBuilders_slimBuildersEligiblePaths(t *testing.T) {
	changes := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 124, Patchset: 1, Project: "chromiumos/third_party/kernel"},
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 124,
				Revision:  1,
			},
			Branch:  "refs/heads/main",
			Project: "chromiumos/third_party/kernel",
			Files: []string{
				"v4.14/someotherdir/example.txt",
			},
		},
	})

	repoToBranchToSrcRoot := map[string]map[string]string{
		"chromiumos/third_party/kernel": {"refs/heads/main": "src/third_party/kernel"},
	}
	buildIrrelevanceCfg := &testplans_pb.BuildIrrelevanceCfg{}

	slimBuildCfg := &testplans_pb.SlimBuildCfg{
		SlimEligibleFilePatterns: []*testplans_pb.FilePattern{
			{Pattern: "src/third_party/kernel/**"},
		},
	}

	testReqsCfg := &testplans_pb.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans_pb.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans_pb.TargetCriteria{
					BuilderName: "testable-builder-cq",
					TargetType: &testplans_pb.TargetCriteria_BuildTarget{
						BuildTarget: "testable-builder",
					},
				},
			},
		},
	}

	builderConfigs := &cros_pb.BuilderConfigs{
		BuilderConfigs: []*cros_pb.BuilderConfig{
			makeBuilderConfig("not-cq-builder", cros_pb.BuilderConfig_Id_TYPE_UNSPECIFIED, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
			makeBuilderConfig("testable-builder-cq", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
			makeBuilderConfig("testable-builder-slim-cq", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
			makeBuilderConfig("non-testable-builder-no-slim-variant-cq", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
			makeBuilderConfig("non-testable-builder-with-slim-variant-cq", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
			makeBuilderConfig("non-testable-builder-with-slim-variant-slim-cq", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
		},
	}

	b := []*cros_pb.BuilderConfig{
		makeBuilderConfig("not-cq-builder", cros_pb.BuilderConfig_Id_TYPE_UNSPECIFIED, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
		makeBuilderConfig("testable-builder-cq", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
		makeBuilderConfig("non-testable-builder-no-slim-variant-cq", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
		makeBuilderConfig("non-testable-builder-with-slim-variant-cq", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
	}

	checkBuildersInput := &CheckBuildersInput{
		Builders:              b,
		Changes:               changes,
		ChangeRevs:            chRevData,
		RepoToBranchToSrcRoot: repoToBranchToSrcRoot,
		BuildIrrelevanceCfg:   buildIrrelevanceCfg,
		SlimBuildCfg:          slimBuildCfg,
		TestReqsCfg:           testReqsCfg,
		BuilderConfigs:        builderConfigs,
	}

	res, err := checkBuildersInput.CheckBuilders()
	if err != nil {
		t.Error(err)
	}
	if len(res.BuildsToRun) != 4 {
		t.Errorf("Expected BuildsToRun to have 3 elements. Instead, %v", res.BuildsToRun)
	}
	expectedBuilderNames := []string{"not-cq-builder", "testable-builder-cq", "non-testable-builder-no-slim-variant-cq", "non-testable-builder-with-slim-variant-slim-cq"}
	actualBuilderNames := make([]string, 0)
	for _, b := range res.BuildsToRun {
		actualBuilderNames = append(actualBuilderNames, b.GetName())
	}
	if len(sliceDiff(expectedBuilderNames, actualBuilderNames)) != 0 {
		t.Errorf("Expected res.BuildsToRun to contain builder names %v. Instead, %v", expectedBuilderNames, actualBuilderNames)
	}
	if len(res.SkipForGlobalBuildIrrelevance) != 0 {
		t.Errorf("Expected SkipForGlobalBuildIrrelevance to be empty. Instead, %v", res.SkipForGlobalBuildIrrelevance)
	}
	if len(res.SkipForRunWhenRules) != 0 {
		t.Errorf("Expected SkipForRunWhenRules to be empty. Instead, %v", res.SkipForRunWhenRules)
	}
}

func TestCheckBuilders_slimBuildersIneligiblePaths(t *testing.T) {
	changes := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2, Project: "chromiumos/platform2"},
		{Host: "test-review.googlesource.com", Change: 124, Patchset: 1, Project: "chromiumos/ineligible/project"},
	}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/main",
			Project: "chromiumos/platform2",
			Files: []string{
				"somedir/example.txt",
			},
		}, {
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 124,
				Revision:  1,
			},
			Branch:  "refs/heads/main",
			Project: "chromiumos/ineligible/project",
			Files: []string{
				"someotherdir/example.txt",
			},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{
		"chromiumos/platform2":          {"refs/heads/main": "src/platform2"},
		"chromiumos/ineligible/project": {"refs/heads/main": "src/pub/ex"},
	}
	buildIrrelevanceCfg := &testplans_pb.BuildIrrelevanceCfg{}

	slimBuildCfg := &testplans_pb.SlimBuildCfg{
		SlimEligibleFilePatterns: []*testplans_pb.FilePattern{
			{Pattern: "src/third_party/kernel/**"},
		},
	}

	testReqsCfg := &testplans_pb.TargetTestRequirementsCfg{
		PerTargetTestRequirements: []*testplans_pb.PerTargetTestRequirements{
			{
				TargetCriteria: &testplans_pb.TargetCriteria{
					BuilderName: "testable-builder-cq",
					TargetType: &testplans_pb.TargetCriteria_BuildTarget{
						BuildTarget: "testable-builder",
					},
				},
			},
		},
	}

	builderConfigs := &cros_pb.BuilderConfigs{}

	b := []*cros_pb.BuilderConfig{
		makeBuilderConfig("not-cq-builder", cros_pb.BuilderConfig_Id_TYPE_UNSPECIFIED, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
		makeBuilderConfig("testable-builder-cq", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
		makeBuilderConfig("non-testable-builder-cq", cros_pb.BuilderConfig_Id_CQ, cros_pb.BuilderConfig_General_RunWhen_ALWAYS_RUN, []string{}),
	}

	checkBuildersInput := &CheckBuildersInput{
		Builders:              b,
		Changes:               changes,
		ChangeRevs:            chRevData,
		RepoToBranchToSrcRoot: repoToBranchToSrcRoot,
		BuildIrrelevanceCfg:   buildIrrelevanceCfg,
		SlimBuildCfg:          slimBuildCfg,
		TestReqsCfg:           testReqsCfg,
		BuilderConfigs:        builderConfigs,
	}

	res, err := checkBuildersInput.CheckBuilders()
	if err != nil {
		t.Error(err)
	}
	if len(res.BuildsToRun) != 3 {
		t.Errorf("Expected BuildsToRun to have 3 element. Instead, %v", res.BuildsToRun)
	}
	expectedBuilderNames := []string{"not-cq-builder", "testable-builder-cq", "non-testable-builder-cq"}
	actualBuilderNames := make([]string, 0)
	for _, b := range res.BuildsToRun {
		actualBuilderNames = append(actualBuilderNames, b.GetName())
	}
	if len(sliceDiff(expectedBuilderNames, actualBuilderNames)) != 0 {
		t.Errorf("Expected res.BuildsToRun to contain builder names %v. Instead, %v", expectedBuilderNames, actualBuilderNames)
	}
	if len(res.SkipForGlobalBuildIrrelevance) != 0 {
		t.Errorf("Expected SkipForGlobalBuildIrrelevance to be empty. Instead, %v", res.SkipForGlobalBuildIrrelevance)
	}
	if len(res.SkipForRunWhenRules) != 0 {
		t.Errorf("Expected SkipForRunWhenRules to be empty. Instead, %v", res.SkipForRunWhenRules)
	}
}

func TestCheckBuilders_NoRunOnFileMatch(t *testing.T) {
	changes := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2, Project: "chromiumos/public/example"}}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/main",
			Project: "chromiumos/public/example",
			Files: []string{
				"chromite-maybe/somedir/match_me_1.txt",
				"chromite-maybe/someotherdir/match_me_2.txt",
			},
		},
	})
	repoToBranchToSrcRoot := map[string]map[string]string{
		"chromiumos/public/example": {"refs/heads/main": "src/pub/ex"},
	}

	buildIrrelevanceCfg := &testplans_pb.BuildIrrelevanceCfg{}

	slimBuildCfg := &testplans_pb.SlimBuildCfg{}

	testReqsCfg := &testplans_pb.TargetTestRequirementsCfg{}

	builderConfigs := &cros_pb.BuilderConfigs{}

	b := []*cros_pb.BuilderConfig{
		makeBuilderConfig("board_to_skip", cros_pb.BuilderConfig_Id_TYPE_UNSPECIFIED, cros_pb.BuilderConfig_General_RunWhen_NO_RUN_ON_FILE_MATCH, []string{"**/match_me_1.txt", "**/match_me_2.txt"}),
		makeBuilderConfig("board_to_run", cros_pb.BuilderConfig_Id_TYPE_UNSPECIFIED, cros_pb.BuilderConfig_General_RunWhen_NO_RUN_ON_FILE_MATCH, []string{"not/a/real/dir"}),
	}

	checkBuildersInput := &CheckBuildersInput{
		Builders:              b,
		Changes:               changes,
		ChangeRevs:            chRevData,
		RepoToBranchToSrcRoot: repoToBranchToSrcRoot,
		BuildIrrelevanceCfg:   buildIrrelevanceCfg,
		SlimBuildCfg:          slimBuildCfg,
		TestReqsCfg:           testReqsCfg,
		BuilderConfigs:        builderConfigs,
	}

	res, err := checkBuildersInput.CheckBuilders()
	if err != nil {
		t.Error(err)
	}
	if len(res.BuildsToRun) != 1 {
		t.Errorf("Expected BuildsToRun to have 1 element. Instead, %v", res.BuildsToRun)
	}
	if res.BuildsToRun[0].GetName() != "board_to_run" {
		t.Errorf("Expected res.BuildsToRun[0].GetName() == \"board_to_run\". Instead, %v", res.BuildsToRun[0].GetName())
	}
	if len(res.SkipForGlobalBuildIrrelevance) != 0 {
		t.Errorf("Expected SkipForGlobalBuildIrrelevance to be empty. Instead, %v", res.SkipForGlobalBuildIrrelevance)
	}
	if len(res.SkipForRunWhenRules) != 1 {
		t.Errorf("Expected SkipForRunWhenRules to have 1 element. Instead, %v", res.SkipForRunWhenRules)
	}
	if res.SkipForRunWhenRules[0].GetName() != "board_to_skip" {
		t.Errorf("Expected SkipForRunWhenRules[0].GetName() == \"board_to_skip\", instead %v", res.SkipForRunWhenRules[0].GetName())
	}
}
