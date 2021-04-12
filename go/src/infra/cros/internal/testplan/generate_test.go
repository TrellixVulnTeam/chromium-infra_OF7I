package testplan

import (
	"infra/cros/internal/cmd"
	"infra/cros/internal/gerrit"
	"infra/cros/internal/git"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	buildpb "go.chromium.org/chromiumos/config/go/build/api"
)

func TestGenerate(t *testing.T) {
	changeRevs := []*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "chromium-review.googlesource.com",
				ChangeNum: 123,
			},
			Project: "chromium/testprojectA",
			Ref:     "refs/changes/23/123/5",
			Files:   []string{"a/b/test1.txt", "a/b/test2.txt"},
		},
	}
	git.CommandRunnerImpl = &cmd.FakeCommandRunnerMulti{
		CommandRunners: []cmd.FakeCommandRunner{
			{
				ExpectedCmd: []string{"git", "clone", "https://chromium.googlesource.com/chromium/testprojectA", "testdata"},
			},
			{
				ExpectedCmd: []string{"git", "fetch", "https://chromium.googlesource.com/chromium/testprojectA", "refs/changes/23/123/5"},
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
			buildSummary("project1", "4.14", "chipsetA"),
			buildSummary("project2", "4.14", "chipsetB"),
			buildSummary("project3", "5.4", "chipsetA"),
		},
	}

	outputs, err := Generate(changeRevs, buildSummaryList)

	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	expectedOutputs := []*Output{
		{
			Name:         "kernel-4.14",
			BuildTargets: []string{"project1", "project2"},
		},
		{
			Name:         "kernel-5.4",
			BuildTargets: []string{"project3"},
		},
	}

	if diff := cmp.Diff(
		expectedOutputs,
		outputs,
		cmpopts.SortSlices(func(i, j *Output) bool {
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
