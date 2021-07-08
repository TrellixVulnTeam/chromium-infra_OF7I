package testplan_test

import (
	"context"
	"infra/cros/internal/cmd"
	"infra/cros/internal/gerrit"
	"infra/cros/internal/git"
	"infra/cros/internal/testplan"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.chromium.org/chromiumos/config/go/test/plan"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestFindRelevantPlans(t *testing.T) {
	ctx := context.Background()
	// Two changes from testprojectA, one from testprojectB.
	changeRevs := []*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "chromium-review.googlesource.com",
				ChangeNum: 123,
			},
			Project: "chromium/testprojectA",
			Ref:     "refs/changes/23/123/5",
			Files:   []string{"go/src/infra/cros/internal/testplan/testdata/DIR_METADATA"},
		},
	}

	// The newest change for each project should be checked out.
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
				ExpectedCmd: []string{
					"git", "fetch",
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
	workdirFn := func() (string, func() error, error) {
		cleanup := func() error { return nil }
		return "./testdata", cleanup, nil
	}

	relevantPlans, err := testplan.FindRelevantPlans(
		ctx, changeRevs, workdirFn,
	)
	if err != nil {
		t.Fatalf("testplan.FindRelevantPlans(%q) failed: %s", changeRevs, err)
	}

	expectedPlan := &plan.SourceTestPlan{
		EnabledTestEnvironments: []plan.SourceTestPlan_TestEnvironment{
			plan.SourceTestPlan_HARDWARE,
		},
		Requirements: &plan.SourceTestPlan_Requirements{
			KernelVersions: &plan.SourceTestPlan_Requirements_KernelVersions{},
		},
		TestTagExcludes: []string{"flaky"},
	}

	if len(relevantPlans) != 1 {
		t.Fatalf("testplan.FindRelevantPlans(%q) returned %d plans, expected 1", changeRevs, len(relevantPlans))
	}

	if diff := cmp.Diff(expectedPlan, relevantPlans[0], protocmp.Transform()); diff != "" {
		t.Errorf(
			"testplan.FindRelevantPlans(%q) returned unexpected diff on plan (-want +got):\n%s",
			changeRevs, diff,
		)
	}
}
