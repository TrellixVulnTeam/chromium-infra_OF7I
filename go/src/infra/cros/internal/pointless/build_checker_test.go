// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
package pointless

import (
	"testing"

	"infra/cros/internal/gerrit"

	"github.com/bmatcuk/doublestar"
	testplans_pb "go.chromium.org/chromiumos/infra/proto/go/testplans"
	bbproto "go.chromium.org/luci/buildbucket/proto"
)

func TestCheckBuilder_irrelevantToRelevantPaths(t *testing.T) {
	// In this test, there's a CL that is fully irrelevant to relevant paths, so the build is pointless.

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
			Files:   []string{"relevantfile", "irrelevantdir2"},
		},
	})
	relevantPaths := []*testplans_pb.PointlessBuildCheckRequest_Path{{Path: "src/dep/graph/path"}}
	repoToBranchToSrcRoot := map[string]map[string]string{
		"chromiumos/public/example": {"refs/heads/main": "src/pub/ex"},
	}
	cfg := testplans_pb.BuildIrrelevanceCfg{}

	affectedFiles, err := ExtractAffectedFiles(changes, chRevData, repoToBranchToSrcRoot)
	res, err := CheckBuilder(affectedFiles, relevantPaths, false, &cfg)
	if err != nil {
		t.Error(err)
	}
	if !res.BuildIsPointless.Value {
		t.Errorf("expected build_is_pointless, instead got result %v", res)
	}
	if res.PointlessBuildReason != testplans_pb.PointlessBuildCheckResponse_IRRELEVANT_TO_DEPS_GRAPH {
		t.Errorf("expected IRRELEVANT_TO_DEPS_GRAPH, instead got result %v", res)
	}
}

func TestCheckBuilder_relevantToForcedRelevantPaths(t *testing.T) {
	// In this test, there's a CL that is relevant to the forced relevant paths, so the build isn't pointless.
	proj := "chromiumos/chromite"
	main := "refs/heads/main"

	changes := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2, Project: proj}}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  main,
			Project: proj,
			Files:   []string{"api/service/thing.py", "README.md"},
		},
	})
	relevantPaths := []*testplans_pb.PointlessBuildCheckRequest_Path{{Path: "chromite/api/**"}}
	repoToBranchToSrcRoot := map[string]map[string]string{
		proj: {main: "chromite"},
	}

	cfg := testplans_pb.BuildIrrelevanceCfg{
		RelevantFilePatterns: []*testplans_pb.FilePattern{
			{Pattern: "chromite/api/**"},
		},
		// This irrelevant config which overlaps the relevant config ensures that we test the forced relevance first.
		IrrelevantFilePatterns: []*testplans_pb.FilePattern{
			{Pattern: "chromite/**"},
		},
	}

	affectedFiles, err := ExtractAffectedFiles(changes, chRevData, repoToBranchToSrcRoot)
	res, err := CheckBuilder(affectedFiles, relevantPaths, false, &cfg)
	if err != nil {
		t.Error(err)
	}
	if res.BuildIsPointless.Value {
		t.Errorf("expected build_is_pointless == false, instead got result %v", res)
	}
	if res.PointlessBuildReason != testplans_pb.PointlessBuildCheckResponse_RELEVANT_TO_KNOWN_NON_PORTAGE_DIRECTORIES {
		t.Errorf("expected RELEVANT_TO_KNOWN_NON_PORTAGE_DIRECTORIES, instead got result %v", res)
	}

	// Test that if we force ignore the non portage paths that we respect the option.
	res, err = CheckBuilder(affectedFiles, relevantPaths, true, &cfg)
	if err != nil {
		t.Error(err)
	}
	if !res.BuildIsPointless.Value {
		t.Errorf("expected build_is_pointless == true, instead got result %v", res)
	}
}

func TestCheckBuilder_relevantToRelevantPaths(t *testing.T) {
	// In this test, there are two CLs, with one of them being related to the relevant paths. The
	// build thus is necessary.

	changes := []*bbproto.GerritChange{
		{Host: "test-review.googlesource.com", Change: 123, Patchset: 2, Project: "chromiumos/public/example"},
		{Host: "test-internal-review.googlesource.com", Change: 234, Patchset: 3, Project: "chromiumos/internal/example"}}
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-review.googlesource.com",
				ChangeNum: 123,
				Revision:  2,
			},
			Branch:  "refs/heads/main",
			Project: "chromiumos/public/example",
			Files:   []string{"a/b/c"},
		},
		{
			ChangeRevKey: gerrit.ChangeRevKey{
				Host:      "test-internal-review.googlesource.com",
				ChangeNum: 234,
				Revision:  3,
			},
			Branch:  "refs/heads/main",
			Project: "chromiumos/internal/example",
			Files:   []string{"important_stuff/important_file"},
		},
	})
	relevantPaths := []*testplans_pb.PointlessBuildCheckRequest_Path{
		{Path: "src/internal/ex/important_stuff"},
	}
	repoToBranchToSrcRoot := map[string]map[string]string{
		"chromiumos/public/example":   {"refs/heads/main": "src/pub/ex"},
		"chromiumos/internal/example": {"refs/heads/main": "src/internal/ex"},
	}
	cfg := testplans_pb.BuildIrrelevanceCfg{}

	affectedFiles, err := ExtractAffectedFiles(changes, chRevData, repoToBranchToSrcRoot)
	res, err := CheckBuilder(affectedFiles, relevantPaths, false, &cfg)
	if err != nil {
		t.Error(err)
	}
	if res.BuildIsPointless.Value {
		t.Errorf("expected !build_is_pointless, instead got result %v", res)
	}
}

func TestCheckBuilder_buildIrrelevantPaths(t *testing.T) {
	// In this test, the only files touched are those that are explicitly listed as being not relevant
	// to Portage.

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
	relevantPaths := []*testplans_pb.PointlessBuildCheckRequest_Path{
		{Path: "src/pub/ex/chromite-maybe"},
	}
	repoToBranchToSrcRoot := map[string]map[string]string{
		"chromiumos/public/example": {"refs/heads/main": "src/pub/ex"},
	}

	cfg := testplans_pb.BuildIrrelevanceCfg{
		IrrelevantFilePatterns: []*testplans_pb.FilePattern{
			{Pattern: "**/ignore_me.txt"},
		},
	}

	affectedFiles, err := ExtractAffectedFiles(changes, chRevData, repoToBranchToSrcRoot)
	res, err := CheckBuilder(affectedFiles, relevantPaths, false, &cfg)
	if err != nil {
		t.Error(err)
	}
	if !res.BuildIsPointless.Value {
		t.Errorf("expected build_is_pointless, instead got result %v", res)
	}
	if res.PointlessBuildReason != testplans_pb.PointlessBuildCheckResponse_IRRELEVANT_TO_KNOWN_NON_PORTAGE_DIRECTORIES {
		t.Errorf("expected IRRELEVANT_TO_KNOWN_NON_PORTAGE_DIRECTORIES, instead got result %v", res)
	}
}

func TestCheckBuilder_noGerritChangesMeansPointlessBuild(t *testing.T) {
	var changes []*bbproto.GerritChange
	chRevData := gerrit.GetChangeRevsForTest([]*gerrit.ChangeRev{})
	relevantPaths := []*testplans_pb.PointlessBuildCheckRequest_Path{
		{Path: "src/pub/ex/chromite-maybe"},
	}
	repoToBranchToSrcRoot := map[string]map[string]string{
		"chromiumos/public/example": {"refs/heads/main": "src/pub/ex"},
	}
	cfg := testplans_pb.BuildIrrelevanceCfg{}

	affectedFiles, err := ExtractAffectedFiles(changes, chRevData, repoToBranchToSrcRoot)
	res, err := CheckBuilder(affectedFiles, relevantPaths, false, &cfg)
	if err != nil {
		t.Error(err)
	}
	if !res.BuildIsPointless.Value {
		t.Errorf("expected build_is_pointless, instead got result %v", res)
	}
}

func doesMatch(t *testing.T, pattern, name string) {
	m, err := doublestar.Match(pattern, name)
	if err != nil {
		t.Errorf("error trying to match pattern %s against name %s: %v", pattern, name, err)
	} else {
		if !m {
			t.Errorf("expected pattern %s to match against name %s, but it did not match", pattern, name)
		}
	}
}

func notMatch(t *testing.T, pattern, name string) {
	m, err := doublestar.Match(pattern, name)
	if err != nil {
		t.Errorf("error trying to match pattern %s against name %s: %v", pattern, name, err)
	} else {
		if m {
			t.Errorf("expected pattern %s not to match against name %s, but it did match", pattern, name)
		}
	}
}

func TestDoubleStar(t *testing.T) {
	// A test that demonstrates/verifies operation of the doublestar matching package.

	doesMatch(t, "**/OWNERS", "OWNERS")
	doesMatch(t, "**/OWNERS", "some/deep/subdir/OWNERS")
	notMatch(t, "**/OWNERS", "OWNERS/fds")

	doesMatch(t, "chromite/config/**", "chromite/config/config_dump.json")

	doesMatch(t, "**/*.md", "a/b/c/README.md")
}
