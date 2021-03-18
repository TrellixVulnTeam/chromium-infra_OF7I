// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build linux

package test

import (
	"encoding/xml"
	// "fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"infra/cros/internal/assert"
	mv "infra/cros/internal/chromeosversion"
	"infra/cros/internal/git"
	"infra/cros/internal/repo"
	rh "infra/cros/internal/repoharness"
)

var testManifest repo.Manifest = repo.Manifest{
	Projects: []repo.Project{
		// Single-checkout project.
		{
			Name: "chromiumos/version",
			Path: "chromiumos/version",
		},
		// Explicitly pinned project.
		{
			Name: "explicit-pinned",
			Path: "explicit-pinned",
			Annotations: []repo.Annotation{
				{Name: "branch-mode", Value: "pin"},
			},
			Revision: "refs/heads/explicit-pinned",
		},
		// Implicitly pinned project.
		{
			Name:     "external/implicit-pinned",
			Path:     "src/third_party/implicit-pinned",
			Revision: "refs/heads/implicit-pinned",
		},
		// Multi-checkout project.
		{
			Name:     "chromiumos/multicheckout",
			Path:     "src/third_party/multicheckout-a",
			Revision: "refs/heads/multicheckout-a",
		},
		{
			Name:     "chromiumos/multicheckout",
			Path:     "src/third_party/multicheckout-b",
			Revision: "refs/heads/multicheckout-b",
		},
		// ToT project.
		{
			Name: "tot",
			Path: "tot",
			Annotations: []repo.Annotation{
				{Name: "branch-mode", Value: "tot"},
			},
		},
	},
	Remotes: []repo.Remote{
		{Name: "cros"},
	},
	Default: repo.Default{
		RemoteName: "cros",
		Revision:   "refs/heads/main",
	},
}

func testInitialize(t *testing.T, config *CrosRepoHarnessConfig) {
	harness := &CrosRepoHarness{}
	defer harness.Teardown()
	err := harness.Initialize(config)
	assert.NilError(t, err)
}

func TestInitializeSimple(t *testing.T) {
	testInitialize(t, &DefaultCrosHarnessConfig)
}

func TestInitializeAllProjectTypes(t *testing.T) {
	config := &CrosRepoHarnessConfig{
		Manifest:       testManifest,
		VersionProject: "chromiumos/version",
	}
	testInitialize(t, config)
}

func TestInitialize_badVersionProject(t *testing.T) {
	config := &CrosRepoHarnessConfig{
		Manifest:       testManifest,
		VersionProject: "bogus",
	}
	harness := &CrosRepoHarness{}
	defer harness.Teardown()
	err := harness.Initialize(config)
	assert.ErrorContains(t, err, "does not exist")
}

func TestSetVersion(t *testing.T) {
	config := DefaultCrosHarnessConfig
	r := &CrosRepoHarness{}
	defer r.Teardown()
	err := r.Initialize(&config)
	assert.NilError(t, err)

	versionFileName := "version.sh"
	version := mv.VersionInfo{
		ChromeBranch:      1,
		BuildNumber:       2,
		BranchBuildNumber: 3,
		PatchNumber:       4,
		VersionFile:       versionFileName,
	}
	assert.NilError(t, r.SetVersion("", version))

	// Test that version file was written correctly.
	harnessRoot := r.Harness.HarnessRoot()
	tmpRepo, err := ioutil.TempDir(harnessRoot, "test_harness_test")
	assert.NilError(t, err)
	versionProject := rh.GetRemoteProject(*r.versionProject)
	versionProjectPath := filepath.Join(harnessRoot, versionProject.RemoteName, versionProject.ProjectName)

	assert.NilError(t, git.Clone(versionProjectPath, tmpRepo))
	// Explicitly checkout main to avoid COIL issues with bots.
	assert.NilError(t, git.Checkout(tmpRepo, "main"))
	contents, err := ioutil.ReadFile(filepath.Join(tmpRepo, versionFileName))
	assert.NilError(t, err)
	vinfo, err := mv.ParseVersionInfo(contents)
	assert.NilError(t, err)
	assert.Assert(t, mv.VersionsEqual(vinfo, version))
}

func TestTakeSnapshot(t *testing.T) {
	config := DefaultCrosHarnessConfig
	r := &CrosRepoHarness{}
	defer r.Teardown()
	err := r.Initialize(&config)
	assert.NilError(t, err)

	assert.NilError(t, r.TakeSnapshot())

	// Check that snapshots exist.
	for _, remote := range config.Manifest.Remotes {
		snapshotPath, ok := r.recentRemoteSnapshots[remote.Name]
		assert.Assert(t, ok)
		_, err := os.Stat(snapshotPath)
		assert.NilError(t, err)
	}
}

func TestAssertCrosBranches_true(t *testing.T) {
	manifest := testManifest
	config := &CrosRepoHarnessConfig{
		Manifest:       manifest,
		VersionProject: "chromiumos/version",
	}
	r := &CrosRepoHarness{}
	defer r.Teardown()
	err := r.Initialize(config)
	assert.NilError(t, err)

	crosBranchName := "mybranch"
	// Set up CrOS branch.
	// Create appropriate refs for non-pinned/tot projects.
	// chromiumos/project
	assert.NilError(t, r.Harness.CreateRemoteRef(rh.GetRemoteProject(manifest.Projects[0]), crosBranchName, ""))
	// chromiumos/multicheckout-a
	assert.NilError(t, r.Harness.CreateRemoteRef(rh.GetRemoteProject(manifest.Projects[3]), crosBranchName+"-multicheckout-a", ""))
	// chromiumos/multicheckout-b
	assert.NilError(t, r.Harness.CreateRemoteRef(rh.GetRemoteProject(manifest.Projects[4]), crosBranchName+"-multicheckout-b", ""))

	assert.NilError(t, r.AssertCrosBranches([]string{crosBranchName}))

	// Also test AssertCrosBranchesMissing
	assert.NilError(t, r.AssertCrosBranchesMissing([]string{"bad"}))
	assert.ErrorContains(t, r.AssertCrosBranchesMissing([]string{crosBranchName}), "mismatch")
}

func TestAssertCrosBranches_false(t *testing.T) {
	manifest := testManifest
	config := &CrosRepoHarnessConfig{
		Manifest:       manifest,
		VersionProject: "chromiumos/version",
	}
	r := &CrosRepoHarness{}
	defer r.Teardown()
	err := r.Initialize(config)
	assert.NilError(t, err)

	crosBranchName := "mybranch"
	// Set up CrOS branch.
	// Create appropriate refs for non-pinned/tot projects.
	// chromiumos/project
	assert.NilError(t, r.Harness.CreateRemoteRef(rh.GetRemoteProject(manifest.Projects[0]), crosBranchName, ""))
	// chromiumos/multicheckout-a
	// Don't add suffix to branch name (this will result in an invalid CrOS branch).
	assert.NilError(t, r.Harness.CreateRemoteRef(rh.GetRemoteProject(manifest.Projects[3]), crosBranchName, ""))
	assert.ErrorContains(t, r.AssertCrosBranches([]string{crosBranchName}), "mismatch")
}

func TestAssertCrosBranchFromManifest_true(t *testing.T) {
	manifest := testManifest
	config := &CrosRepoHarnessConfig{
		Manifest:       manifest,
		VersionProject: "chromiumos/version",
	}
	r := &CrosRepoHarness{}
	defer r.Teardown()
	err := r.Initialize(config)
	assert.NilError(t, err)
	assert.NilError(t, r.TakeSnapshot())

	crosBranchName := "mybranch"
	// Set up CrOS branch. We create the new refs from the corresponding main refs so
	// that the new branch WILL descend from the manifest.
	// Create appropriate refs for non-pinned/tot projects.
	// chromiumos/project
	assert.NilError(t, r.Harness.CreateRemoteRef(rh.GetRemoteProject(manifest.Projects[0]), crosBranchName, "refs/heads/main"))
	// chromiumos/multicheckout-a
	assert.NilError(t, r.Harness.CreateRemoteRef(rh.GetRemoteProject(manifest.Projects[3]), crosBranchName+"-multicheckout-a", "refs/heads/multicheckout-a"))
	// chromiumos/multicheckout-b
	assert.NilError(t, r.Harness.CreateRemoteRef(rh.GetRemoteProject(manifest.Projects[4]), crosBranchName+"-multicheckout-b", "refs/heads/multicheckout-b"))

	assert.NilError(t, r.AssertCrosBranchFromManifest(manifest, crosBranchName, ""))
}

func TestAssertCrosBranchFromManifest_false(t *testing.T) {
	manifest := testManifest
	config := &CrosRepoHarnessConfig{
		Manifest:       manifest,
		VersionProject: "chromiumos/version",
	}
	r := &CrosRepoHarness{}
	defer r.Teardown()
	err := r.Initialize(config)
	assert.NilError(t, err)
	assert.NilError(t, r.TakeSnapshot())

	crosBranchName := "mybranch"

	// Set up CrOS branch. We create the new refs from the corresponding main refs so
	// that the new branch will NOT descend from the manifest.
	// Specifically, we create the multicheckout branches from refs/heads/main instead of
	// their set revisions.

	// chromiumos/project
	assert.NilError(t, r.Harness.CreateRemoteRef(rh.GetRemoteProject(manifest.Projects[0]), crosBranchName, "refs/heads/main"))
	// chromiumos/multicheckout-a
	assert.NilError(t, r.Harness.CreateRemoteRef(rh.GetRemoteProject(manifest.Projects[3]), crosBranchName+"-multicheckout-a", "refs/heads/main"))
	// chromiumos/multicheckout-b
	assert.NilError(t, r.Harness.CreateRemoteRef(rh.GetRemoteProject(manifest.Projects[4]), crosBranchName+"-multicheckout-b", "refs/heads/main"))

	assert.ErrorContains(t, r.AssertCrosBranchFromManifest(manifest, crosBranchName, ""), "does not descend")
}

func TestAssertCrosVersion(t *testing.T) {
	config := DefaultCrosHarnessConfig
	r := &CrosRepoHarness{}
	defer r.Teardown()
	err := r.Initialize(&config)
	assert.NilError(t, err)

	versionFileName := "version.sh"
	version := mv.VersionInfo{
		ChromeBranch:      1,
		BuildNumber:       2,
		BranchBuildNumber: 3,
		PatchNumber:       4,
		VersionFile:       versionFileName,
	}
	assert.NilError(t, r.SetVersion("", version))

	assert.NilError(t, r.AssertCrosVersion("main", version))
	// Wrong version.
	version.ChromeBranch = 5
	assert.ErrorContains(t, r.AssertCrosVersion("main", version), "version mismatch")
}

func TestAssertNoDefaultRevisions(t *testing.T) {
	manifest := repo.Manifest{
		Default: repo.Default{},
		Remotes: []repo.Remote{
			{Name: "remote"},
		},
	}
	assert.NilError(t, AssertNoDefaultRevisions(manifest))

	manifest = repo.Manifest{
		Default: repo.Default{
			Revision: "foo",
		},
		Remotes: []repo.Remote{
			{Name: "remote"},
		},
	}
	assert.ErrorContains(t, AssertNoDefaultRevisions(manifest), "<default>")

	manifest = repo.Manifest{
		Default: repo.Default{},
		Remotes: []repo.Remote{
			{Name: "remote", Revision: "foo"},
		},
	}
	assert.ErrorContains(t, AssertNoDefaultRevisions(manifest), "<remote>")
}

func TestAssertProjectRevisionsMatchBranch(t *testing.T) {
	config := CrosRepoHarnessConfig{
		Manifest:       testManifest,
		VersionProject: "chromiumos/version",
	}
	r := &CrosRepoHarness{}
	defer r.Teardown()
	err := r.Initialize(&config)
	assert.NilError(t, err)

	manifest := r.Harness.Manifest()
	// Deep copy projects so that we can change manifest without changing r.Harness.manifest
	manifest.Projects = append([]repo.Project(nil), manifest.Projects...)

	// To avoid all the work of actually branching, just switch the revisions on
	// pinned projects to be SHA-1's instead of 'refs/heads/main'.
	for _, project := range manifest.GetPinnedProjects() {
		repoPath := r.Harness.GetRemotePath(rh.GetRemoteProject(*project))
		mainSha, err := git.GetGitRepoRevision(repoPath, project.Revision)
		assert.NilError(t, err)
		project.Revision = mainSha
	}
	// Also, to pretend that main is a proper CrOS branch, we need to adjust
	// the multicheckout revisions.
	for _, project := range manifest.GetMultiCheckoutProjects() {
		project.Revision = git.NormalizeRef("main-" + git.StripRefs(project.Revision))
	}

	assert.NilError(t, r.AssertProjectRevisionsMatchBranch(manifest, "main", ""))
	assert.Assert(t, r.AssertProjectRevisionsMatchBranch(manifest, "foo", "") != nil)
}

func TestAssertManifestProjectRepaired(t *testing.T) {
	getLocalCheckoutFunc = getLocalCheckout
	cleanupLocalCheckoutFunc = cleanupLocalCheckout

	configManifest := testManifest
	configManifest.Projects = append(configManifest.Projects, DefaultManifestProject)
	config := CrosRepoHarnessConfig{
		Manifest:       configManifest,
		VersionProject: "chromiumos/version",
	}
	r := &CrosRepoHarness{}
	defer r.Teardown()
	err := r.Initialize(&config)
	assert.NilError(t, err)

	// Set up new branch. We have to actually do this because of pinned projects.
	newBranch := "newbranch"
	manifestProject := rh.GetRemoteProject(DefaultManifestProject)
	assert.NilError(t, r.Harness.CreateRemoteRef(manifestProject, newBranch, "main"))

	manifest := r.Harness.Manifest()
	// Deep copy projects so that we can change manifest without changing r.Harness.manifest
	manifest.Projects = append([]repo.Project(nil), manifest.Projects...)

	// Switch the revisions on pinned projects to be SHA-1's instead of 'refs/heads/main'.
	for _, project := range manifest.GetPinnedProjects() {
		pinnedProject := rh.GetRemoteProject(*project)
		repoPath := r.Harness.GetRemotePath(pinnedProject)
		assert.NilError(t, r.Harness.CreateRemoteRef(pinnedProject, newBranch, project.Revision))
		mainSha, err := git.GetGitRepoRevision(repoPath, newBranch)
		assert.NilError(t, err)
		project.Revision = mainSha
	}
	for _, project := range manifest.GetSingleCheckoutProjects() {
		project.Revision = git.NormalizeRef(newBranch)
	}
	for _, project := range manifest.GetMultiCheckoutProjects() {
		project.Revision = git.NormalizeRef(newBranch + "-" + git.StripRefs(project.Revision))
	}

	// Clear default revisions.
	for i := range manifest.Remotes {
		manifest.Remotes[i].Revision = ""
	}
	manifest.Default.Revision = ""

	// Write manifest to file.
	manifestData, err := xml.Marshal(manifest)
	assert.NilError(t, err)
	manifestFile := rh.File{
		Name:     "manifest.xml",
		Contents: []byte(manifestData),
	}
	_, err = r.Harness.AddFile(manifestProject, newBranch, manifestFile)
	assert.NilError(t, err)

	assert.NilError(t, r.AssertManifestProjectRepaired(manifestProject, newBranch, []string{"manifest.xml"}))
}

const foo = `
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="cros" fetch="bar" />
  <project path="src/repohooks" name="chromiumos/repohooks"
           groups="minilayout,firmware,buildtools,labtools,crosvm"
           revision="refs/heads/bar" />
  <repo-hooks in-project="chromiumos/repohooks" enabled-list="pre-upload" />

  <!--This comment should persist.-->
  <project name="chromiumos/manifest"
           path="manifest"
           revision="refs/heads/bar"
           upstream="bogus" />

  <new-element name="this should persist" />

  <project name="chromiumos/overlays/chromiumos-overlay"
           path="src/third_party/chromiumos-overlay"
           revision="refs/heads/bar"/>

  <project name="external/implicit-pinned"
           path="src/third_party/implicit-pinned"
           revision="refs/heads/implicit-pinned"/>

  <!--This comment should also persist.-->
  <project name="chromiumos/multicheckout"
           path="src/third_party/multicheckout-a"
           revision="refs/heads/multicheckout-a"/>

  <project name="chromiumos/multicheckout"
           path="src/third_party/multicheckout-b"
           revision="refs/heads/multicheckout-b"/>

</manifest>
`

const bar = `
<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="cros" fetch="bar" />
  <project path="src/repohooks" name="chromiumos/repohooks"
           groups="minilayout,firmware,buildtools,labtools,crosvm"
           revision="refs/heads/bar" />
  <repo-hooks in-project="chromiumos/repohooks" enabled-list="pre-upload" />

  <!--This comment should persist.-->
  <project name="chromiumos/manifest"
           path="manifest"
           revision="refs/heads/bar"/>

  <new-element name="this should persist" />

  <project name="external/implicit-pinned"
           path="src/third_party/implicit-pinned"
           revision="refs/heads/implicit-pinned"/>

  <!--This comment will be lost :( -->
  <project name="chromiumos/overlays/chromiumos-overlay"
           path="src/third_party/chromiumos-overlay"
           revision="refs/heads/bar"/>

  <!--This comment should also persist.-->
  <project name="chromiumos/multicheckout"
           path="src/third_party/multicheckout-a"
           revision="refs/heads/multicheckout-a"/>

  <project name="chromiumos/multicheckout"
           path="src/third_party/multicheckout-b"
           revision="refs/heads/multicheckout-b"/>

</manifest>
`

func TestAssertCommentsPersist(t *testing.T) {
	getLocalCheckoutFunc = func(r *CrosRepoHarness, project rh.RemoteProject, branch string) (string, error) {
		return "test_data/", nil
	}
	cleanupLocalCheckoutFunc = func(checkout string) {
		return
	}

	r := &CrosRepoHarness{}

	sourceManifestFiles := map[string]string{
		"foo.xml": foo,
	}
	assert.NilError(t, r.AssertCommentsPersist(rh.RemoteProject{}, "", sourceManifestFiles))

	sourceManifestFiles = map[string]string{
		"foo.xml": bar,
	}
	assert.ErrorContains(t, r.AssertCommentsPersist(rh.RemoteProject{}, "", sourceManifestFiles), "This comment will be lost")
}

func TestAssertMinimalManifestChanges(t *testing.T) {
	getLocalCheckoutFunc = func(r *CrosRepoHarness, project rh.RemoteProject, branch string) (string, error) {
		return "test_data/", nil
	}
	cleanupLocalCheckoutFunc = func(checkout string) {
		return
	}

	r := &CrosRepoHarness{}

	expectedManifestFiles := map[string]string{
		"foo.xml": foo,
	}
	assert.NilError(t, r.AssertMinimalManifestChanges(rh.RemoteProject{}, "", expectedManifestFiles))

	expectedManifestFiles = map[string]string{
		"foo.xml": bar,
	}
	assert.ErrorContains(t, r.AssertMinimalManifestChanges(rh.RemoteProject{}, "", expectedManifestFiles), "mismatch")
}
