package testplan

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	buildpb "go.chromium.org/chromiumos/config/go/build/api"
	testpb "go.chromium.org/chromiumos/config/go/test/api"
	"go.chromium.org/chromiumos/config/go/test/plan"
	"infra/cros/internal/cmd"
	"infra/cros/internal/gerrit"
	"infra/cros/internal/git"
)

func TestGenerate(t *testing.T) {
	ctx := context.Background()
	changeRevs := []*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "chromium-review.googlesource.com",
				ChangeNum: 123,
			},
			Project: "chromium/testprojectA",
			Ref:     "refs/changes/23/123/5",
			Files: []string{
				"go/src/infra/cros/internal/testplan/testdata/a/b/test1.txt",
				"go/src/infra/cros/internal/testplan/testdata/a/b/test2.txt",
			},
		},
	}
	git.CommandRunnerImpl = &cmd.FakeCommandRunnerMulti{
		CommandRunners: []cmd.FakeCommandRunner{
			{
				ExpectedCmd: []string{
					"git", "clone",
					"https://chromium.googlesource.com/chromium/testprojectA", "testdata",
					"--depth", "1", "--no-tags",
				},
			},
			{
				ExpectedCmd: []string{"git", "fetch",
					"https://chromium.googlesource.com/chromium/testprojectA", "refs/changes/23/123/5",
					"--depth", "1", "--no-tags",
				},
			},
			{
				ExpectedCmd: []string{"git", "checkout", "FETCH_HEAD"},
			},
		},
	}

	// Set workdirFn so the CommandRunners can know where commands are run,
	// and the DIR_METADATA in testdata is read. Don't cleanup the testdata.
	workdirFn = func(_, _ string) (string, error) { return "./testdata", nil }
	workdirCleanupFn = func(_ string) error { return nil }

	buildSummaryList := &buildpb.SystemImage_BuildSummaryList{
		Values: []*buildpb.SystemImage_BuildSummary{
			buildSummary("project1", "4.14", "chipsetA", "P"),
			buildSummary("project2", "4.14", "chipsetB", "R"),
			buildSummary("project3", "5.4", "chipsetA", ""),
		},
	}

	dutAttributeList := &testpb.DutAttributeList{
		DutAttributes: []*testpb.DutAttribute{
			{
				Id:        &testpb.DutAttribute_Id{Value: "fingerprint_location"},
				FieldPath: "design_list.configs.hardware_features.fingerprint.location",
			},
			{
				Id:        &testpb.DutAttribute_Id{Value: "system_build_target"},
				FieldPath: "software_configs.system_build_target.portage_build_target.overlay_name",
			},
		},
	}

	var sourceTestPlans []*plan.SourceTestPlan

	rules, err := Generate(ctx, changeRevs, sourceTestPlans, buildSummaryList, dutAttributeList)

	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	expectedRules := []*testpb.CoverageRule{
		{
			Name: "kernel:4.14",
			DutCriteria: []*testpb.DutCriterion{
				{
					AttributeId: &testpb.DutAttribute_Id{
						Value: "system_build_target",
					},
					Values: []string{"project1", "project2"},
				},
			},
			TestSuites: []*testpb.TestSuite{
				{
					TestCaseTagCriteria: &testpb.TestSuite_TestCaseTagCriteria{
						TagExcludes: []string{"flaky"},
					},
				},
			},
		},
		{
			Name: "kernel:5.4",
			DutCriteria: []*testpb.DutCriterion{
				{
					AttributeId: &testpb.DutAttribute_Id{
						Value: "system_build_target",
					},
					Values: []string{"project3"},
				},
			},
			TestSuites: []*testpb.TestSuite{
				{
					TestCaseTagCriteria: &testpb.TestSuite_TestCaseTagCriteria{
						TagExcludes: []string{"flaky"},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(
		expectedRules,
		rules,
		cmpopts.SortSlices(func(i, j *testpb.CoverageRule) bool {
			return i.Name < j.Name
		}),
		cmpopts.SortSlices(func(i, j string) bool {
			return i < j
		}),
		cmpopts.EquateEmpty(),
	); diff != "" {
		t.Errorf("generate returned unexpected diff (-want +got):\n%s", diff)
	}
}

func TestGenerateWithPlans(t *testing.T) {
	ctx := context.Background()

	buildSummaryList := &buildpb.SystemImage_BuildSummaryList{
		Values: []*buildpb.SystemImage_BuildSummary{
			buildSummary("project1", "4.14", "chipsetA", "P"),
			buildSummary("project2", "4.14", "chipsetB", "R"),
			buildSummary("project3", "5.4", "chipsetA", ""),
		},
	}

	dutAttributeList := &testpb.DutAttributeList{
		DutAttributes: []*testpb.DutAttribute{
			{
				Id:        &testpb.DutAttribute_Id{Value: "fingerprint_location"},
				FieldPath: "design_list.configs.hardware_features.fingerprint.location",
			},
			{
				Id:        &testpb.DutAttribute_Id{Value: "system_build_target"},
				FieldPath: "software_configs.system_build_target.portage_build_target.overlay_name",
			},
		},
	}

	sourceTestPlans := []*plan.SourceTestPlan{
		{
			EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
				plan.SourceTestPlan_HARDWARE,
			},
			Requirements: &plan.SourceTestPlan_Requirements{
				KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
			},
			TestTagExcludes: []string{"flaky"},
		},
	}

	var changeRevs []*gerrit.ChangeRev

	rules, err := Generate(ctx, changeRevs, sourceTestPlans, buildSummaryList, dutAttributeList)

	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	expectedRules := []*testpb.CoverageRule{
		{
			Name: "kernel:4.14",
			DutCriteria: []*testpb.DutCriterion{
				{
					AttributeId: &testpb.DutAttribute_Id{
						Value: "system_build_target",
					},
					Values: []string{"project1", "project2"},
				},
			},
			TestSuites: []*testpb.TestSuite{
				{
					TestCaseTagCriteria: &testpb.TestSuite_TestCaseTagCriteria{
						TagExcludes: []string{"flaky"},
					},
				},
			},
		},
		{
			Name: "kernel:5.4",
			DutCriteria: []*testpb.DutCriterion{
				{
					AttributeId: &testpb.DutAttribute_Id{
						Value: "system_build_target",
					},
					Values: []string{"project3"},
				},
			},
			TestSuites: []*testpb.TestSuite{
				{
					TestCaseTagCriteria: &testpb.TestSuite_TestCaseTagCriteria{
						TagExcludes: []string{"flaky"},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(
		expectedRules,
		rules,
		cmpopts.SortSlices(func(i, j *testpb.CoverageRule) bool {
			return i.Name < j.Name
		}),
		cmpopts.SortSlices(func(i, j string) bool {
			return i < j
		}),
		cmpopts.EquateEmpty(),
	); diff != "" {
		t.Errorf("generate returned unexpected diff (-want +got):\n%s", diff)
	}
}

func TestGenerateErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		sourceTestPlans []*plan.SourceTestPlan
		changeRevs      []*gerrit.ChangeRev
	}{
		{
			name: "changeRevs and plans non-empty",
			sourceTestPlans: []*plan.SourceTestPlan{
				{
					EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
						plan.SourceTestPlan_HARDWARE,
					},
					Requirements: &plan.SourceTestPlan_Requirements{
						KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
					},
					TestTagExcludes: []string{"flaky"},
				},
			},
			changeRevs: []*gerrit.ChangeRev{
				{
					ChangeRevKey: gerrit.ChangeRevKey{
						Host:      "chromium-review.googlesource.com",
						ChangeNum: 123,
					},
					Project: "chromium/testprojectA",
					Ref:     "refs/changes/23/123/5",
					Files: []string{
						"go/src/infra/cros/internal/testplan/testdata/a/b/test1.txt",
						"go/src/infra/cros/internal/testplan/testdata/a/b/test2.txt",
					},
				},
			},
		},
		{
			name: "plans has paths set",
			sourceTestPlans: []*plan.SourceTestPlan{
				{
					EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
						plan.SourceTestPlan_HARDWARE,
					},
					Requirements: &plan.SourceTestPlan_Requirements{
						KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
					},
					TestTagExcludes: []string{"flaky"},
					PathRegexps:     []string{"a/b/c"},
				},
			},
			changeRevs: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Generate(ctx, test.changeRevs, test.sourceTestPlans, buildSummaryList, dutAttributeList); err == nil {
				t.Error("Expected error from Generate")
			}
		})
	}
}
