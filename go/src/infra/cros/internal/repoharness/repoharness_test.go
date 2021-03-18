// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build linux

package repoharness

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/cmd"
	"infra/cros/internal/git"
	"infra/cros/internal/repo"
	"infra/cros/internal/util"
)

var simpleHarnessConfig = Config{
	Manifest: repo.Manifest{
		Remotes: []repo.Remote{
			{Name: "cros"},
			{Name: "cros-internal"},
		},
		Default: repo.Default{
			RemoteName: "cros",
			Revision:   "refs/heads/main",
		},
		Projects: []repo.Project{
			{Path: "foo1/", Name: "foo", Revision: "refs/heads/foo1"},
			{Path: "foo2/", Name: "foo", Revision: "refs/heads/foo2"},
			{Path: "bar/", Name: "bar"},
			{Path: "baz/", Name: "baz", RemoteName: "cros-internal"},
		},
	},
}

// Maps project path to files.
var simpleFiles = map[string]([]File){
	"foo1/": []File{
		{Name: "README", Contents: []byte("foo1")},
		{Name: "code/DOCS", Contents: []byte("this is a document in a subdirectory")},
	},
	"foo2/": []File{
		{Name: "ALSO_README", Contents: []byte("foo2")},
	},
	"bar/": []File{
		{Name: "README", Contents: []byte("bar")},
	},
	"baz/": []File{
		{Name: "SECRET", Contents: []byte("internal only")},
	},
}
var multilevelProjectHarnessConfig = Config{
	Manifest: repo.Manifest{
		Remotes: []repo.Remote{
			{Name: "remote"},
		},
		Default: repo.Default{
			RemoteName: "remote",
			Revision:   "refs/heads/main",
		},
		Projects: []repo.Project{
			{Path: "src/foo/bar", Name: "foo/bar"},
		},
	},
}

func testGetRemoteProject(t *testing.T) {
	project := repo.Project{
		Name:       "foo",
		RemoteName: "bar",
		Path:       "baz",
	}
	remoteProject := RemoteProject{
		RemoteName:  "bar",
		ProjectName: "foo",
	}
	assert.Assert(t, reflect.DeepEqual(remoteProject, GetRemoteProject(project)))
}

func testInitialize(t *testing.T, config *Config) {
	harnessConfig := config
	harness := &RepoHarness{}
	defer harness.Teardown()
	err := harness.Initialize(harnessConfig)
	assert.NilError(t, err)

	// Check that snapshots/ dir was created.
	_, err = os.Stat(filepath.Join(harness.harnessRoot, "snapshots"))
	assert.NilError(t, err)

	// Check that all remotes were created.
	for _, remote := range harnessConfig.Manifest.Remotes {
		_, err := os.Stat(filepath.Join(harness.harnessRoot, remote.Name))
		assert.NilError(t, err)
	}

	// Check that appropraite projects were created and initialized.
	for _, project := range harnessConfig.Manifest.Projects {
		projectPath := filepath.Join(harness.harnessRoot, project.RemoteName, project.Name)
		_, err := os.Stat(projectPath)
		assert.NilError(t, err)
		branches, err := git.MatchBranchName(projectPath, regexp.MustCompile("main"))
		assert.NilError(t, err)
		assert.Assert(t, util.UnorderedContains(branches, []string{git.NormalizeRef("main")}))

		// If project has revision set, check that that branch was create too.
		if project.Revision != "" && project.Revision != "main" {
			revisionBranch := project.Revision
			branches, err := git.MatchBranchName(projectPath, regexp.MustCompile(revisionBranch))
			assert.NilError(t, err)
			assert.Assert(t, util.UnorderedContains(branches, []string{revisionBranch}))
		}
	}
}

func TestInitialize_simple(t *testing.T) {
	testInitialize(t, &simpleHarnessConfig)
}

// Test that a project with a multi-level name (e.g. foo/bar) is properly
// created in the appropriate remote.
func TestInitialize_multilevelProject(t *testing.T) {
	testInitialize(t, &multilevelProjectHarnessConfig)
}

func TestInitialize_badRevision(t *testing.T) {
	harnessConfig := Config{
		Manifest: repo.Manifest{
			Projects: []repo.Project{
				{Name: "foo",
					Revision:   "deadbeef",
					RemoteName: "cros"},
			},
			Remotes: []repo.Remote{
				{Name: "cros"},
			},
		},
	}
	harness := &RepoHarness{}
	defer harness.Teardown()
	assert.ErrorContains(t, harness.Initialize(&harnessConfig), "refs/heads")
}

func TestInitializeNoRemotes(t *testing.T) {
	config := &Config{
		Manifest: repo.Manifest{
			Projects: []repo.Project{
				{Name: "foo", Path: "foo/"},
			},
		},
	}
	harness := &RepoHarness{}
	defer harness.Teardown()
	err := harness.Initialize(config)
	assert.ErrorContains(t, err, "remotes")
}

func TestInitializeDefaultDefault(t *testing.T) {
	config := &Config{
		Manifest: repo.Manifest{
			Remotes: []repo.Remote{
				{Name: "cros"},
			},
		},
	}
	r := &RepoHarness{}
	defer r.Teardown()
	err := r.Initialize(config)
	assert.NilError(t, err)
	assert.StringsEqual(t, r.Manifest().Default.RemoteName, "cros")
	assert.StringsEqual(t, r.Manifest().Default.Revision, "refs/heads/main")
}

func TestCreateRemoteRef(t *testing.T) {
	root, err := ioutil.TempDir("", "create_remote_ref_test")
	defer os.RemoveAll(root)
	assert.NilError(t, err)

	harness := &RepoHarness{
		manifest: repo.Manifest{
			Remotes: []repo.Remote{
				{Name: "cros"},
			},
			Projects: []repo.Project{
				{Path: "foo/", Name: "foo", RemoteName: "cros"},
			},
		},
		harnessRoot: root,
	}
	// Set up remote.
	remotePath := filepath.Join(harness.harnessRoot, harness.manifest.Remotes[0].Name)
	assert.NilError(t, os.Mkdir(remotePath, dirPerms))
	// Set up remote project.
	project := harness.manifest.Projects[0]
	remoteProjectPath := filepath.Join(remotePath, project.Name)
	assert.NilError(t, os.Mkdir(remoteProjectPath, dirPerms))
	assert.NilError(t, git.Init(remoteProjectPath, false))

	// Make initial commit.
	_, err = git.RunGit(remoteProjectPath, []string{"commit", "-m", "init", "--allow-empty"})
	assert.NilError(t, err)
	output, err := git.RunGit(remoteProjectPath, []string{"rev-parse", "HEAD"})
	assert.NilError(t, err)
	commit := strings.TrimSpace(output.Stdout)
	assert.NilError(t, harness.CreateRemoteRef(GetRemoteProject(project), "ref1", commit))
	assert.NilError(t, harness.CreateRemoteRef(GetRemoteProject(project), "ref2", ""))

	output, err = git.RunGit(remoteProjectPath, []string{"show-ref"})
	refs := []string{}
	for _, line := range strings.Split(output.Stdout, "\n") {
		if line == "" {
			continue
		}
		refs = append(refs, strings.Fields(line)[1])
	}
	assert.Assert(t, util.UnorderedContains(refs, []string{"refs/heads/ref1", "refs/heads/ref2"}))

	// Test that an error is thrown if we try to create a remote that already exists.
	assert.ErrorContains(t, harness.CreateRemoteRef(GetRemoteProject(project), "ref2", ""), "already exists")
}

func TestAddFiles_simple(t *testing.T) {
	harnessConfig := simpleHarnessConfig
	harness := &RepoHarness{}
	defer harness.Teardown()
	err := harness.Initialize(&harnessConfig)
	assert.NilError(t, err)

	for _, project := range simpleHarnessConfig.Manifest.Projects {
		files, ok := simpleFiles[project.Path]
		if !ok {
			continue
		}
		_, err := harness.AddFiles(GetRemoteProject(project), "main", files)
		assert.NilError(t, err)
	}

	// Check that all files were added to remotes.
	for projectPath, files := range simpleFiles {
		project, err := harness.manifest.GetProjectByPath(projectPath)
		assert.NilError(t, err)
		tmpDir, err := ioutil.TempDir(harness.harnessRoot, "tmp-clone-dir")

		err = git.Clone(harness.GetRemotePath(GetRemoteProject(*project)), tmpDir)
		// Explicitly checkout main to avoid COIL issues with bots.
		assert.NilError(t, git.Checkout(tmpDir, "main"))

		assert.NilError(t, err)

		for _, file := range files {
			// Check that file exists.
			filePath := filepath.Join(tmpDir, file.Name)
			_, err = os.Stat(filePath)
			assert.NilError(t, err)
			// Check file contents.
			fileContents, err := ioutil.ReadFile(filePath)
			assert.NilError(t, err)
			assert.Assert(t, reflect.DeepEqual(file.Contents, fileContents))
		}
		os.RemoveAll(tmpDir)
	}
}

// Tests a few specific things:
// Creating a file in a branch other than main
// Creating a nested file (e.g. a/b/c.txt)
func TestAddFile(t *testing.T) {
	harnessConfig := simpleHarnessConfig
	harness := &RepoHarness{}
	defer harness.Teardown()
	assert.NilError(t, harness.Initialize(&harnessConfig))

	project := harness.manifest.Projects[0]

	projectPath := harness.GetRemotePath(GetRemoteProject(project))
	remoteRef := git.RemoteRef{
		Remote: project.RemoteName,
		Ref:    "foo1",
	}
	file := File{Name: "docs/README", Contents: []byte("foo1")}
	_, err := harness.AddFile(GetRemoteProject(project), remoteRef.Ref, file)
	assert.NilError(t, err)

	// Check that file was added to remote.
	tmpDir, err := ioutil.TempDir(harness.harnessRoot, "tmp-clone-dir")

	assert.NilError(t, git.Init(tmpDir, false))
	assert.NilError(t, git.AddRemote(tmpDir, project.RemoteName, projectPath))
	assert.NilError(t, git.CreateTrackingBranch(tmpDir, "tmp", remoteRef))

	// Check that file exists.
	filePath := filepath.Join(tmpDir, file.Name)
	_, err = os.Stat(filePath)
	assert.NilError(t, err)
	// Check file contents.
	fileContents, err := ioutil.ReadFile(filePath)
	assert.NilError(t, err)
	assert.Assert(t, reflect.DeepEqual(file.Contents, fileContents))
}

func TestReadFile(t *testing.T) {
	harnessConfig := simpleHarnessConfig
	harness := &RepoHarness{}
	defer harness.Teardown()
	assert.NilError(t, harness.Initialize(&harnessConfig))
	project := harness.manifest.Projects[0]

	remoteRef := git.RemoteRef{
		Remote: project.RemoteName,
		Ref:    "foo1",
	}
	file := File{Name: "docs/README", Contents: []byte("foo1")}
	_, err := harness.AddFile(GetRemoteProject(project), remoteRef.Ref, file)
	assert.NilError(t, err)

	// Testing ReadFile by assuming correctness of Initialize and AddFile is
	// obviously not ideal, but there's not really a better way to do it
	// without essentially reimplementing AddFile inline.
	contents, err := harness.ReadFile(GetRemoteProject(project), remoteRef.Ref, "docs/README")
	assert.NilError(t, err)
	assert.StringsEqual(t, string(contents), "foo1")

	_, err = harness.ReadFile(GetRemoteProject(project), remoteRef.Ref, "docs/MISSING")
	assert.Assert(t, err != nil)
}

func TestTeardown(t *testing.T) {
	// Pretend that harness has been initialized and harness root has been created.
	tmpDir := "harness_root"
	tmpDir, err := ioutil.TempDir("", tmpDir)
	defer os.RemoveAll(tmpDir)

	harness := RepoHarness{
		harnessRoot: tmpDir,
	}
	// Check that harness root exists.
	_, err = os.Stat(tmpDir)
	assert.NilError(t, err)
	// Perform teardown.
	assert.NilError(t, harness.Teardown())
	_, err = os.Stat(tmpDir)
	// Root no longer exists.
	assert.Assert(t, os.IsNotExist(err))
	assert.StringsEqual(t, harness.harnessRoot, "")
}

func TestGetRemotePath(t *testing.T) {
	harness := &RepoHarness{
		harnessRoot: "foo/",
	}

	project := simpleHarnessConfig.Manifest.Projects[0]
	expectedPath := filepath.Join(harness.harnessRoot, project.RemoteName, project.Name)
	assert.StringsEqual(t, harness.GetRemotePath(GetRemoteProject(project)), expectedPath)
}

func TestAssertProjectBranches(t *testing.T) {
	harness := &RepoHarness{
		harnessRoot: "foo",
	}
	project := repo.Project{
		RemoteName: "bar",
		Name:       "baz",
	}
	projectPath := "foo/bar/baz"

	branches := []string{"main", "branch"}
	stdout := ""
	for _, branch := range branches {
		stdout += fmt.Sprintf("aaa refs/heads/%s\n", branch)
	}

	git.CommandRunnerImpl = cmd.FakeCommandRunner{
		ExpectedCmd: []string{"git", "show-ref"},
		ExpectedDir: projectPath,
		Stdout:      stdout,
	}

	assert.NilError(t, harness.AssertProjectBranches(GetRemoteProject(project), branches))
	assert.ErrorContains(t, harness.AssertProjectBranches(GetRemoteProject(project), []string{"bad"}), "mismatch")
	// Also test AssertProjectBranchesMissing
	assert.ErrorContains(t, harness.AssertProjectBranchesMissing(GetRemoteProject(project), branches), "mismatch")
	assert.NilError(t, harness.AssertProjectBranchesMissing(GetRemoteProject(project), []string{"bad1", "bad2"}))

	// Set command runner back to the real one. Most tests in this package do not mock git.
	git.CommandRunnerImpl = cmd.RealCommandRunner{}
}

func TestAssertProjectBranchesExact(t *testing.T) {
	harness := &RepoHarness{
		harnessRoot: "foo",
	}
	project := repo.Project{
		RemoteName: "bar",
		Name:       "baz",
	}
	projectPath := "foo/bar/baz"

	branches := []string{"main", "branch"}
	stdout := ""
	for _, branch := range branches {
		stdout += fmt.Sprintf("aaa refs/heads/%s\n", branch)
	}

	git.CommandRunnerImpl = cmd.FakeCommandRunner{
		ExpectedCmd: []string{"git", "show-ref"},
		ExpectedDir: projectPath,
		Stdout:      stdout,
	}

	assert.NilError(t, harness.AssertProjectBranchesExact(GetRemoteProject(project), branches))
	assert.ErrorContains(t, harness.AssertProjectBranchesExact(GetRemoteProject(project), append(branches, "extra")), "mismatch")

	// Set command runner back to the real one. Most tests in this package do not mock git.
	git.CommandRunnerImpl = cmd.RealCommandRunner{}
}

// createFooBarBaz creates foo bar baz file structure, the greatest file structure on earth
func createFooBarBaz(t *testing.T, root, bazContents string) {
	assert.NilError(t, os.Mkdir(filepath.Join(root, "foo"), 0755))
	assert.NilError(t, os.Mkdir(filepath.Join(root, "foo", "bar"), 0755))
	assert.NilError(t, ioutil.WriteFile(filepath.Join(root, "foo", "bar", "baz"), []byte(bazContents), 0666))
}

// checkFooBarBaz checks the foo bar baz file structure, the greatest file structure on earth
func checkFooBarBaz(t *testing.T, root, bazContents string) {
	_, err := os.Stat(filepath.Join(root, "foo"))
	assert.NilError(t, err)
	_, err = os.Stat(filepath.Join(root, "foo", "bar"))
	assert.NilError(t, err)
	snapshotBazPath := filepath.Join(root, "foo", "bar", "baz")
	_, err = os.Stat(snapshotBazPath)
	assert.NilError(t, err)
	// Check contents of bar/baz.
	contents, err := ioutil.ReadFile(snapshotBazPath)
	assert.NilError(t, err)
	assert.StringsEqual(t, string(contents), bazContents)
}

func TestSnapshot(t *testing.T) {
	root, err := ioutil.TempDir("", "assert_test")
	assert.NilError(t, err)
	defer os.RemoveAll(root)
	harness := &RepoHarness{
		harnessRoot: root,
	}
	assert.NilError(t, os.Mkdir(filepath.Join(harness.harnessRoot, "snapshots"), 0777))

	// Create a hierachy of files.
	fooRoot, err := ioutil.TempDir(harness.harnessRoot, "snapshot_test")
	assert.NilError(t, err)
	bazContents := "foo, bar and baz, oh my!"
	createFooBarBaz(t, fooRoot, bazContents)

	// Create snapshot and verify accuracy.
	snapshotDir, err := harness.Snapshot(fooRoot)
	assert.NilError(t, err)
	checkFooBarBaz(t, snapshotDir, bazContents)
}

func TestAssertProjectBranchEqual(t *testing.T) {
	root, err := ioutil.TempDir("", "assert_test")
	assert.NilError(t, err)
	defer os.RemoveAll(root)
	harness := &RepoHarness{
		harnessRoot: root,
	}

	local, err := ioutil.TempDir(harness.harnessRoot, "")
	assert.NilError(t, err)
	remote, err := ioutil.TempDir(harness.harnessRoot, "")
	assert.NilError(t, err)

	project := repo.Project{
		Name: filepath.Base(local),
	}

	// Initialize remote repo and make a commit.
	assert.NilError(t, git.Init(remote, false))
	// Explicitly checkout main to avoid COIL issues with bots.
	assert.NilError(t, git.CreateBranch(remote, "main"))
	assert.NilError(t, ioutil.WriteFile(filepath.Join(remote, "foo"), []byte("foo"), 0644))
	_, err = git.CommitAll(remote, "init commit")
	assert.NilError(t, err)
	// Clone remote so that we have two identical repos.
	assert.NilError(t, git.Clone(remote, local))

	assert.NilError(t, harness.AssertProjectBranchEqual(GetRemoteProject(project), "main", remote))
	// Now, make commit to local.
	assert.NilError(t, ioutil.WriteFile(filepath.Join(local, "bar"), []byte("bar"), 0644))
	_, err = git.CommitAll(local, "addl commit")
	assert.NilError(t, err)
	assert.ErrorContains(t, harness.AssertProjectBranchEqual(GetRemoteProject(project), "main", remote), "mismatch")
}

func TestAssertProjectBranchHasAncestor(t *testing.T) {
	root, err := ioutil.TempDir("", "assert_test")
	assert.NilError(t, err)
	defer os.RemoveAll(root)
	harness := &RepoHarness{
		harnessRoot: root,
	}

	local, err := ioutil.TempDir(harness.harnessRoot, "")
	assert.NilError(t, err)
	remote, err := ioutil.TempDir(harness.harnessRoot, "")
	assert.NilError(t, err)

	project := repo.Project{
		Name: filepath.Base(local),
	}

	// Initialize remote repo and make a commit.
	assert.NilError(t, git.Init(remote, false))
	// Explicitly checkout main to avoid COIL issues with bots.
	assert.NilError(t, git.CreateBranch(remote, "main"))
	assert.NilError(t, ioutil.WriteFile(filepath.Join(remote, "foo"), []byte("foo"), 0644))
	_, err = git.CommitAll(remote, "init commit")
	assert.NilError(t, err)
	// Clone remote so that we have two identical repos.
	assert.NilError(t, git.Clone(remote, local))

	assert.NilError(t, harness.AssertProjectBranchHasAncestor(GetRemoteProject(project), "main", remote, project.Revision))

	// Now, make commit to local. We should still be good.
	assert.NilError(t, ioutil.WriteFile(filepath.Join(local, "bar"), []byte("bar"), 0644))
	_, err = git.CommitAll(local, "addl commit")
	assert.NilError(t, err)
	assert.NilError(t, harness.AssertProjectBranchHasAncestor(GetRemoteProject(project), "main", remote, project.Revision))

	// But if we make a commit to remote, our local repo will no longer descend from it.
	assert.NilError(t, ioutil.WriteFile(filepath.Join(remote, "baz"), []byte("baz"), 0644))
	_, err = git.CommitAll(remote, "addl commit")
	assert.NilError(t, err)
	assert.ErrorContains(t, harness.AssertProjectBranchHasAncestor(GetRemoteProject(project), "main", remote, project.Revision), "does not descend")
}
