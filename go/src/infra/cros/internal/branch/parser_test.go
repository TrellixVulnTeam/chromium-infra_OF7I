// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build linux

package branch

import (
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/cmd"
	"infra/cros/internal/git"
)

func TestExtractBuildNum(t *testing.T) {
	assert.IntsEqual(t, extractBuildNum("release-R90-13816.B"), 13816)
	assert.IntsEqual(t, extractBuildNum("stabilize-nocturne-10986.B"), 10986)
	assert.IntsEqual(t, extractBuildNum("stabilize-5116.113.B"), 5116)
	assert.IntsEqual(t, extractBuildNum("stabilize-ambassador-13597.79.B"), 13597)
	assert.IntsEqual(t, extractBuildNum("firmware-eve-campfire-9584.131.B"), 9584)
	assert.IntsEqual(t, extractBuildNum("factory-rammus-11289.B"), 11289)
	assert.IntsEqual(t, extractBuildNum("main"), -1)
	assert.IntsEqual(t, extractBuildNum("foo"), -1)
}

func TestReleaseBranches(t *testing.T) {
	branches := []string{
		"release-R89-13729.B",
		"stabilize-13851.B",
		"release-R90-13816.B",
		"stabilize-13856.B",
		"release-R91-13904.B",
		"stabilize-13895.B",
	}

	branches, err := releaseBranches(branches, 90)
	assert.NilError(t, err)
	assert.StringsEqual(t, branches[0], "release-R90-13816.B")
	assert.StringsEqual(t, branches[1], "release-R91-13904.B")
}

func TestNonReleaseBranches(t *testing.T) {
	branches := []string{
		"release-R89-13729.B",
		"stabilize-13851.B",
		"release-R90-13816.B",
		"stabilize-13856.B",
		"release-R91-13904.B",
		"stabilize-13895.B",
		"firmware-eve-campfire-9584.131.B",
		"factory-rammus-11289.B",
	}

	branches = nonReleaseBranches(branches, 13816)
	assert.StringArrsEqual(t, branches, []string{
		"stabilize-13851.B",
		"stabilize-13856.B",
		"stabilize-13895.B",
	})
}

var fakeGitData = `
	origin/release-R89-13729.B
	origin/release-R90-13816.B
	origin/release-R91-13904.B
	origin/stabilize-atlas-11022.B
	origin/stabilize-13851.B
	origin/stabilize-13856.B
	origin/stabilize-13895.B
	origin/firmware-eve-campfire-9584.131.B
	origin/factory-rammus-11289.B
`

func TestBranchesFromMilestone(t *testing.T) {
	fakeGitRepo := "test_data/manifest-internal"

	git.CommandRunnerImpl = &cmd.FakeCommandRunnerMulti{
		CommandRunners: []cmd.FakeCommandRunner{
			{
				ExpectedDir: fakeGitRepo,
				ExpectedCmd: []string{"git", "fetch", "--all"},
			},
			{
				ExpectedDir: fakeGitRepo,
				ExpectedCmd: []string{"git", "branch", "-r"},
				Stdout:      fakeGitData,
			},
		},
	}

	branches, err := BranchesFromMilestone("test_data", 90)
	assert.NilError(t, err)
	assert.StringArrsEqual(t, branches, []string{
		"release-R90-13816.B",
		"release-R91-13904.B",
		"stabilize-13851.B",
		"stabilize-13856.B",
		"stabilize-13895.B",
	})
}
