// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package branch

import (
	"reflect"
	"regexp"
	"testing"

	"infra/cros/internal/assert"
	mv "infra/cros/internal/chromeosversion"
	"infra/cros/internal/repo"
)

var branchNameTestManifest = repo.Manifest{
	Projects: []repo.Project{
		// Basic project. Only one checkout, so we can just use the branch name.
		{Path: "bar/", Name: "chromiumos/bar"},
		// Project with multiple checkouts. Upstream/revision will be used as a suffix.
		{Path: "foo1/", Name: "foo", Revision: "100", Upstream: "refs/heads/factory-100"},
		{Path: "foo2/", Name: "foo", Revision: "101"},
		// Project with multiple checkouts that were created as part of a previous branching operation.
		// Will be properly named using the `original` parameter.
		{Path: "baz1/", Name: "baz", Upstream: "refs/heads/oldbranch-factory-100"},
		{Path: "baz2/", Name: "baz", Upstream: "refs/heads/oldbranch-factory-101"},
		{Path: "baz2/", Name: "baz", Upstream: "refs/heads/oldbranch-factory-101"},
		// Project with an upstream that is from a CrOS branch name.
		{Path: "baz2/", Name: "baz", Upstream: "refs/heads/release-R77-12371.B-myfactory/2.6"},

		// Project has two checkouts but one is ToT, so no suffix.
		{Path: "qux1/", Name: "qux"},
		{Path: "qux2/", Name: "qux", Annotations: []repo.Annotation{
			{Name: "branch-mode", Value: "tot"},
		}},

		// Cases covered by the mapping feature
		{Path: "src/third_party/coreboot", Name: "chromiumos/third_party/coreboot", Revision: "8dddd11bc804c01b905b87407e42a2d58d044384", Upstream: "refs/heads/firmware-puff-13324.B-chromeos-2016.05"},
		{Path: "src/third_party/coreboot", Name: "chromiumos/third_party/coreboot", Revision: "8dddd11bc804c01b905b87407e42a2d58d044385"},
	},
}

var canBranchTestManifest = repo.Manifest{
	Projects: []repo.Project{
		{Path: "foo/", Name: "foo",
			Annotations: []repo.Annotation{
				{Name: "branch-mode", Value: "create"},
			},
		},
		{Path: "bar/", Name: "bar",
			Annotations: []repo.Annotation{
				{Name: "branch-mode", Value: "pin"},
			},
		},
	},
}

var testBranchNames = []string{
	"firmware-test-1234.12.3.B",
	"firmware-test-1234.12.3.B",
	"firmware-7132.B",
	"release-R17-9876.B",
	"release-R18-4567.B",
	"stabilize-23478.221.B",
	"stabilize-go-12334.21.B",
	"stabilize-go-12336.B",
	"factory-a-6212.B",
	"factory-b-1234.13.B",
	"factory-c-1234.12.B",
	"release-R16-1234.B",
}

func TestBranchExist(t *testing.T) {

	// Slices of the test names
	passing := testBranchNames[2:9]
	majorCollision := testBranchNames[2:]
	duplicateName := testBranchNames[0:6]

	pattern := regexp.MustCompile(`.*-1234.12.3.B$`)

	// Passing branch
	exist, err := BranchExists(pattern, "1234.12", "firmware", passing)
	assert.BoolsEqual(t, exist, false)
	assert.NilError(t, err)

	// Major collision
	exist, err = BranchExists(pattern, "1234.12", "firmware", majorCollision)
	assert.BoolsEqual(t, exist, true)
	assert.ErrorContains(t, err, "Major version collision on branch")

	// Duplicate branch name
	exist, err = BranchExists(pattern, "1234.12", "firmware", duplicateName)
	assert.BoolsEqual(t, exist, true)
	assert.NilError(t, err)

}

func TestProjectBranchName(t *testing.T) {
	manifest := branchNameTestManifest
	c := Client{
		WorkingManifest: manifest,
	}
	assert.StringsEqual(t, c.projectBranchName("mybranch", manifest.Projects[0], ""), "mybranch")
	assert.StringsEqual(t, c.projectBranchName("mybranch", manifest.Projects[1], ""), "mybranch-factory-100")
	assert.StringsEqual(t, c.projectBranchName("mybranch", manifest.Projects[2], ""), "mybranch-101")
	assert.StringsEqual(t, c.projectBranchName("mybranch", manifest.Projects[6], ""), "mybranch-myfactory-2.6")
	assert.StringsEqual(t, c.projectBranchName("mybranch", manifest.Projects[7], ""), "mybranch")
}

func TestProjectBranchName_MappingFunctionality(t *testing.T) {
	manifest := branchNameTestManifest
	c := Client{
		WorkingManifest: manifest,
	}
	assert.StringsEqual(t, c.projectBranchName("coreboot", manifest.Projects[9], ""), "coreboot")
}

func TestProjectBranchName_withOriginal(t *testing.T) {
	manifest := branchNameTestManifest
	c := Client{
		WorkingManifest: manifest,
	}
	assert.StringsEqual(t, c.projectBranchName("mybranch", manifest.Projects[3], "oldbranch"), "mybranch-factory-100")
	assert.StringsEqual(t, c.projectBranchName("mybranch", manifest.Projects[4], "oldbranch"), "mybranch-factory-101")
	assert.StringsEqual(t, c.projectBranchName("mybranch", manifest.Projects[6], "oldbranch"), "mybranch-myfactory-2.6")
}

func TestCanBranchProject(t *testing.T) {
	manifest := canBranchTestManifest
	assert.Assert(t, canBranchProject(manifest, manifest.Projects[0]))
	assert.Assert(t, !canBranchProject(manifest, manifest.Projects[1]))
}

var branchesTestManifest = repo.Manifest{
	Projects: []repo.Project{
		// Basic project. Only one checkout, so we can just use the branch name.
		{Path: "bar/", Name: "chromiumos/bar", Revision: "100", RemoteName: "cros"},
		// Project with multiple checkouts. Upstream/revision will be used as a suffix.
		{Path: "foo1/", Name: "foo", Upstream: "refs/heads/factory-100",
			Annotations: []repo.Annotation{
				{Name: "branch-mode", Value: "create"},
			},
		},
		{Path: "foo2/", Name: "foo"},
	},
	Remotes: []repo.Remote{
		{Name: "cros"},
	},
}

func TestProjectBranches(t *testing.T) {
	manifest := branchesTestManifest
	c := Client{
		WorkingManifest: manifest,
	}
	expected := []ProjectBranch{
		{Project: manifest.Projects[0], BranchName: "mybranch"},
		{Project: manifest.Projects[1], BranchName: "mybranch-factory-100"},
	}

	branchNames := c.ProjectBranches("mybranch", "oldbranch")
	assert.Assert(t, reflect.DeepEqual(expected, branchNames))
}

func TestGetBranchesByPath(t *testing.T) {
	branches := []ProjectBranch{
		{Project: repo.Project{Path: "foo/"}, BranchName: "foo-branch"},
		{Project: repo.Project{Path: "bar/"}, BranchName: "bar-branch"},
	}
	branchMap := map[string]string{
		"foo/": "foo-branch",
		"bar/": "bar-branch",
	}
	assert.Assert(t, reflect.DeepEqual(getBranchesByPath(branches), branchMap))
}

func TestWhichVersionShouldBump_successPatch(t *testing.T) {
	vinfo := mv.VersionInfo{
		ChromeBranch:      0xfa,
		BuildNumber:       0xca,
		BranchBuildNumber: 0xde,
		PatchNumber:       0x00,
	}

	component, err := WhichVersionShouldBump(vinfo)
	assert.NilError(t, err)
	assert.StringsEqual(t, string(component), string(mv.Patch))
}

func TestWhichVersionShouldBump_successBranch(t *testing.T) {
	vinfo := mv.VersionInfo{
		ChromeBranch:      0xfe,
		BuildNumber:       0xed,
		BranchBuildNumber: 0x00,
		PatchNumber:       0x00,
	}

	component, err := WhichVersionShouldBump(vinfo)
	assert.NilError(t, err)
	assert.StringsEqual(t, string(component), string(mv.Branch))
}

func TestNewBranchName_Custom(t *testing.T) {
	assert.StringsEqual(t, NewBranchName(mv.VersionInfo{}, "custom-name", "", false, false, false, false), "custom-name")
}

var vinfo = mv.VersionInfo{
	ChromeBranch:      77,
	BuildNumber:       123,
	BranchBuildNumber: 1,
	PatchNumber:       0,
}

func TestNewBranchName_Release(t *testing.T) {
	assert.StringsEqual(t, NewBranchName(vinfo, "", "", true, false, false, false), "release-R77-123.1.B")
}

func TestNewBranchName_Factory(t *testing.T) {
	assert.StringsEqual(t, NewBranchName(vinfo, "", "foo", false, true, false, false), "factory-foo-123.1.B")
}

func TestNewBranchName_Firmware(t *testing.T) {
	assert.StringsEqual(t, NewBranchName(vinfo, "", "", false, false, true, false), "firmware-123.1.B")
}

func TestNewBranchName_Stabilize(t *testing.T) {
	assert.StringsEqual(t, NewBranchName(vinfo, "", "", false, false, false, true), "stabilize-123.1.B")
}
