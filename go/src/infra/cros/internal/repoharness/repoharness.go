// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package repoharness contains code for a test harness that allows for
// easy faking of a repo checkout.
package repoharness

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"infra/cros/internal/cmd"
	"infra/cros/internal/git"
	"infra/cros/internal/repo"
	test_util "infra/cros/internal/testutil"

	"github.com/otiai10/copy"
	"go.chromium.org/luci/common/errors"
)

var (
	// CommandRunnerImpl is the command runner impl currently being used by the
	// package. Exists for testing purposes.
	CommandRunnerImpl cmd.CommandRunner = cmd.RealCommandRunner{}

	// ForSubmitRefRegexp matches refs of the form refs/for/...%submit
	ForSubmitRefRegexp = regexp.MustCompile(`^refs\/for\/(?P<name>.+)%submit$`)
)

// File contains information about a file.
type File struct {
	Name     string
	Contents []byte
	Perm     os.FileMode
}

const (
	readWritePerms = 0666
	dirPerms       = 0777
)

// RemoteProject identifies a remote project.
type RemoteProject struct {
	RemoteName  string
	ProjectName string
}

// GetRemoteProject returns the corresponding remote project for
// a given project.
func GetRemoteProject(project repo.Project) RemoteProject {
	return RemoteProject{
		RemoteName:  project.RemoteName,
		ProjectName: project.Name,
	}
}

// Config for a RepoHarness.
type Config struct {
	// Initialize() will create a test harness with
	// the appropriate remote repos and a local repo.
	// Both remote and local repos will have the appropriate
	// projects created (with initialized git repos inside them).
	Manifest repo.Manifest
}

// RepoHarness is a test harness that fakes out an entire repo checkout
// based on a supplied manifest.
type RepoHarness struct {
	// Manifest that defines the harness configuration.
	manifest repo.Manifest
	// Root directory of the whole harness setup.
	harnessRoot string
	// Path of local checkout, or empty (if does not exist).
	localCheckout string
	// Most recent snapshot of each remote.
	recentRemoteSnapshots map[string]string
}

// Manifest returns the manifest associated with the harness.
func (r *RepoHarness) Manifest() repo.Manifest {
	return r.manifest
}

// HarnessRoot returns the path to the root directory of the test harness.
func (r *RepoHarness) HarnessRoot() string {
	return r.harnessRoot
}

func (r *RepoHarness) runCommand(cmd []string, cwd string) error {
	if cwd == "" {
		cwd = r.harnessRoot
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var stdoutBuf, stderrBuf bytes.Buffer
	err := CommandRunnerImpl.RunCommand(ctx, &stdoutBuf, &stderrBuf, cwd, cmd[0], cmd[1:]...)
	if err != nil {
		return errors.Annotate(err, "error running command (stderr: %s)", stderrBuf.String()).Err()
	}
	return nil
}

// Initialize creates a RepoHarness based on the given config.
func (r *RepoHarness) Initialize(config *Config) error {
	// Protect client, because we really don't want repo to try and hit chromium.googlesource.com.
	// Unless client has explicitly messed with the manifest to make it work, this won't work
	// and will fail fairly slowly.
	if len(config.Manifest.Remotes) == 0 {
		return fmt.Errorf("no remotes specified in config manifest")
	}
	if config.Manifest.Default.RemoteName == "" {
		config.Manifest.Default.RemoteName = config.Manifest.Remotes[0].Name
	}
	if config.Manifest.Default.Revision == "" {
		config.Manifest.Default.Revision = "refs/heads/main"
	}

	var err error
	// Set up root directory for harness instance.
	r.harnessRoot, err = ioutil.TempDir("", "repo_harness")
	if err != nil {
		return errors.Annotate(err, "failed to create harness root dir").Err()
	}
	// Create snapshots/ dir.
	if err = os.Mkdir(filepath.Join(r.harnessRoot, "snapshots"), dirPerms); err != nil {
		return errors.Annotate(err, "failed to create harness snapshots dir").Err()
	}

	// Resolve implicit links in the manifest. We do this so that each project has
	// an explicit remote listed.
	config.Manifest.ResolveImplicitLinks()
	r.manifest = config.Manifest

	// Initialize remote repositories.
	for _, remote := range r.manifest.Remotes {
		remoteName := remote.Name
		// Create directory.
		err = os.Mkdir(filepath.Join(r.harnessRoot, remoteName), dirPerms)
		if err != nil {
			return errors.Annotate(err, "failed to create tmp dir for remote %s", remoteName).Err()
		}
	}
	// Update Fetch attribute in manifest remotes.
	for i, remote := range r.manifest.Remotes {
		r.manifest.Remotes[i].Fetch = "file://" + filepath.Join(r.harnessRoot, remote.Name)
	}

	// Initialize projects on remotes.
	for _, project := range r.manifest.Projects {
		remoteName := project.RemoteName
		projectPath := filepath.Join(r.harnessRoot, remoteName, project.Name)
		projectLabel := fmt.Sprintf("project %s (remote %s)", project.Name, remoteName)

		// Project could already exist due to multiple checkouts. If it does, skip
		// initialization/main branch setup.
		if _, err = os.Stat(projectPath); err != nil {
			// Create project directory.
			if err = os.MkdirAll(projectPath, dirPerms); err != nil {
				return errors.Annotate(err, "failed to create dir for %s", projectLabel).Err()
			}
			// Initialize bare repo in project directory.
			if err = git.Init(projectPath, true); err != nil {
				return errors.Annotate(err, "failed to init git repo for %s", projectLabel).Err()
			}

			// Make an initial commit so that the "main" branch is not unborn.
			if err = r.CreateRemoteRef(GetRemoteProject(project), "main", ""); err != nil {
				return errors.Annotate(err, "failed to init git repo for %s", projectLabel).Err()
			}
		}
		// If revision is set, create that branch too.
		if project.Revision != "" && !strings.HasPrefix(project.Revision, "refs/heads/") {
			return fmt.Errorf("revisions must be of the form refs/heads/<branch>")
		}

		revision := git.StripRefs(project.Revision)
		if revision != "" && revision != "main" {
			// Creating the revision ref from a fresh repo/commit and not from refs/heads/main is
			// kind of nice because it removes some false positives from AssertCrosBranchFromManifest
			// -- if a multicheckout branch is created from refs/heads/main instead of its set
			// revision, the assert would still pass if the revision itself descends from refs/heads/main.
			if err = r.CreateRemoteRef(GetRemoteProject(project), revision, ""); err != nil {
				return errors.Annotate(err, "failed to init git repo for %s", projectLabel).Err()
			}
		}
	}
	return err
}

func (r *RepoHarness) assertInitialized() error {
	if r.harnessRoot == "" {
		return fmt.Errorf("repo harness needs to be initialized")
	}
	return nil
}

// Teardown tears down a repo harness.
func (r *RepoHarness) Teardown() error {
	if r.harnessRoot != "" {
		root := r.harnessRoot
		r.harnessRoot = ""
		return os.RemoveAll(root)
	}
	return fmt.Errorf("harness was never initialized")
}

// CreateRemoteRef creates a remote ref for a specific project.
// Otherwise, a temporary local checkout will be created and an empty commit
// will be used to create the remote ref.
func (r *RepoHarness) CreateRemoteRef(project RemoteProject, ref string, commit string) error {
	return r.createRemoteRefHelper(project, ref, commit, false)
}

func (r *RepoHarness) CreateRemoteRefForce(project RemoteProject, ref string, commit string) error {
	return r.createRemoteRefHelper(project, ref, commit, true)
}

func (r *RepoHarness) createRemoteRefHelper(project RemoteProject, ref string, commit string, force bool) error {
	projectLabel := fmt.Sprintf("%s/%s", project.RemoteName, project.ProjectName)
	remoteProjectPath := r.GetRemotePath(project)

	var repoPath string
	var err error
	remoteRef := git.RemoteRef{
		Ref: git.NormalizeRef(ref),
	}

	if commit == "" {
		// Set up tmp local repo and make empty commit.
		repoPath, err = ioutil.TempDir(r.harnessRoot, "tmp-repo")
		defer os.RemoveAll(repoPath)
		errs := []error{
			err,
			git.Init(repoPath, false),
		}
		for _, err := range errs {
			if err != nil {
				return errors.Annotate(err, "failed to make temp local repo").Err()
			}
		}
		commitMsg := fmt.Sprintf("empty commit for ref %s %s", remoteRef.Remote, remoteRef.Ref)
		commit, err = git.CommitEmpty(repoPath, commitMsg)
		if err != nil {
			return errors.Annotate(err, "failed to make empty commit").Err()
		}

		if err = git.AddRemote(repoPath, project.RemoteName, remoteProjectPath); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				err = nil
			} else {
				return errors.Annotate(err, "failed to add remote %s to tmp repo", project.RemoteName).Err()
			}
		}
		remoteRef.Remote = project.RemoteName
	} else {
		repoPath = remoteProjectPath
		remoteRef.Remote = remoteProjectPath
	}

	if !force {
		remoteExists, err := git.RemoteHasBranch(repoPath, remoteRef.Remote, remoteRef.Ref)
		if err != nil {
			return errors.Annotate(err, "failed to ls-remote remote %s", remoteRef.Remote).Err()
		}
		if remoteExists {
			return fmt.Errorf("remote ref %s already exists", ref)
		}
	}

	if err := git.PushRef(repoPath, commit, remoteRef, git.ForceIf(force)); err != nil {
		return errors.Annotate(err, "failed to add remote ref %s %s:%s", projectLabel, commit, remoteRef.Ref).Err()
	}
	return nil
}

// AddFile adds a file to the specified branch in the specified remote project.
// Returns the sha1 of the commit that adds the file.
func (r *RepoHarness) AddFile(project RemoteProject, branch string, file File) (string, error) {
	return r.AddFiles(project, branch, []File{file})
}

// AddFiles adds files to the specified branch in the specified remote project.
// Returns a map with the sha1's of the commits.
func (r *RepoHarness) AddFiles(project RemoteProject, branch string, files []File) (string, error) {
	if err := r.assertInitialized(); err != nil {
		return "", err
	}
	remote := r.manifest.GetRemoteByName(project.RemoteName)
	if remote == nil {
		return "", fmt.Errorf("remote %s does not exist in manifest", project.RemoteName)
	}

	projectLabel := fmt.Sprintf("%s", project.ProjectName)

	// Populate project in specified remote with files. Because the remote repository is bare,
	// we need to write/commit the files locally and then push them to the remote.
	// We do this using a temp checkout of the appropriate remote.
	tmpRepo, err := ioutil.TempDir(r.harnessRoot, "tmp-repo")
	defer os.RemoveAll(tmpRepo)

	projectPath := r.GetRemotePath(project)
	remoteRef := git.RemoteRef{
		Remote: project.RemoteName,
		Ref:    branch,
	}

	errs := []error{
		err,
		git.Init(tmpRepo, false),
		git.AddRemote(tmpRepo, project.RemoteName, projectPath),
		git.CreateTrackingBranch(tmpRepo, "tmp", remoteRef),
	}

	for _, file := range files {
		filePath := filepath.Join(tmpRepo, file.Name)
		// Set file perms to default value if not specified.
		if file.Perm == 0 {
			file.Perm = readWritePerms
		}

		errs = append(errs,
			os.MkdirAll(filepath.Dir(filePath), dirPerms),
			ioutil.WriteFile(filePath, file.Contents, file.Perm))
	}

	commit, err := git.CommitAll(tmpRepo, "add files")
	errs = append(errs, err, git.PushRef(tmpRepo, "tmp", remoteRef))

	for _, err = range errs {
		if err != nil {
			return "", errors.Annotate(err, "failed to add files to %s", projectLabel).Err()
		}
	}

	return commit, nil
}

// ProcessSubmitRefs makes sure that refs/heads/... refs accurately reflect
// corresponding refs/for/...%submit refs, if they exist.
func (r *RepoHarness) ProcessSubmitRefs() error {
	temporaryBranchName := "temporary_submit_branch"
	for _, project := range r.manifest.Projects {
		remoteName := project.RemoteName
		projectPath := filepath.Join(r.harnessRoot, remoteName, project.Name)

		refMap, err := git.Refs(projectPath)
		if err != nil {
			return err
		}

		// Process each refs/for/...%submit ref.
		for ref := range refMap {
			match := ForSubmitRefRegexp.FindStringSubmatch(ref)
			if match == nil {
				continue
			}

			// Create a branch pointing to the refs/for/...%submit ref.
			err := git.RunGitIgnoreOutput(projectPath, []string{"branch", temporaryBranchName, ref, "--force"})
			if err != nil {
				return err
			}

			// The remote repositories are bare git repos (as they should be),
			// which means that we can't rebase the refs/for/...%submit ref
			// directly onto refs/heads/.... To get around this, we create a branch
			// (which is externally visible) on the remote, pull that into a local
			// checkout, do the needed rebasing, and then push directly to the
			// corresponding refs/heads/... ref.
			tmpRepo, err := ioutil.TempDir(r.harnessRoot, "tmp-repo")
			defer os.RemoveAll(tmpRepo)
			if err != nil {
				return err
			}

			remoteForRef := git.RemoteRef{
				Remote: project.RemoteName,
				Ref:    temporaryBranchName,
			}
			remoteHeadRef := git.RemoteRef{
				Remote: project.RemoteName,
				Ref:    match[1],
			}
			// Replay (using rebase) the changes in refs/for/...%submit onto
			// refs/heads/..., and push the changes to the remote heads ref.
			if err := git.Init(tmpRepo, false); err != nil {
				return err
			}
			if err := git.AddRemote(tmpRepo, project.RemoteName, projectPath); err != nil {
				return err
			}
			if err := git.CreateTrackingBranch(tmpRepo, "for", remoteForRef); err != nil {
				return err
			}
			if err := git.CreateTrackingBranch(tmpRepo, "head", remoteHeadRef); err != nil {
				return err
			}
			if err := git.Checkout(tmpRepo, "for"); err != nil {
				return err
			}
			if err := git.RunGitIgnoreOutput(tmpRepo, []string{"rebase", "head"}); err != nil {
				return err
			}
			if err := git.PushRef(tmpRepo, "for", remoteHeadRef, git.Force()); err != nil {
				return err
			}
		}
	}
	return nil
}

// ReadFile reads a file from a remote.
func (r *RepoHarness) ReadFile(project RemoteProject, branch, filePath string) ([]byte, error) {
	if err := r.assertInitialized(); err != nil {
		return []byte{}, err
	}
	tmpRepo, err := ioutil.TempDir(r.harnessRoot, "tmp-repo")
	defer os.RemoveAll(tmpRepo)
	if err != nil {
		return nil, err
	}

	remotePath := r.GetRemotePath(project)
	ref := git.StripRefs(branch)
	refspec := fmt.Sprintf("%s/%s", "remote", ref)
	// Checkout just the file we need.
	errs := []error{
		git.Init(tmpRepo, false),
		git.AddRemote(tmpRepo, "remote", remotePath),
		git.RunGitIgnoreOutput(tmpRepo, []string{"fetch", "remote", ref, "--depth", "1"}),
		git.RunGitIgnoreOutput(tmpRepo, []string{"checkout", refspec, "--", filePath}),
	}
	contents, err := ioutil.ReadFile(filepath.Join(tmpRepo, filePath))
	errs = append(errs, err)

	for _, err = range errs {
		if err != nil {
			return []byte{}, errors.Annotate(err, "failed to read file %s from %s/%s", filePath, project.RemoteName, branch).Err()
		}
	}

	return contents, nil
}

// Checkout creates a local checkout of the project using the specified manifest
// project/branch/file, if one does not already exist.
// Returns a path to the local checkout, and a potential error.
func (r *RepoHarness) Checkout(manifestProject RemoteProject, branch, manifestFile string) (string, error) {
	if r.localCheckout != "" {
		return r.localCheckout, nil
	}

	checkoutPath := filepath.Join(r.harnessRoot, "my_checkout")
	if err := os.Mkdir(checkoutPath, dirPerms); err != nil {
		return "", errors.Annotate(err, "failed to create dir %s", checkoutPath).Err()
	}

	// Create local checkout of manifest repo so that we can sync to a manifest using `repo init`.
	manifestRepoCheckout := filepath.Join(r.harnessRoot, "manifest-repo")
	if err := os.Mkdir(manifestRepoCheckout, dirPerms); err != nil {
		return "", errors.Annotate(err, "failed to create dir %s", manifestRepoCheckout).Err()
	}
	if err := git.Clone(r.GetRemotePath(manifestProject), manifestRepoCheckout, git.SingleBranch(), git.Branch(branch)); err != nil {
		return "", errors.Annotate(err, "failed to clone remote manifest project").Err()
	}

	repoPath, cleanup, err := repo.EnsureRepoTool()
	defer cleanup()
	if err != nil {
		return "", errors.Annotate(err, "failed to install repo tool").Err()
	}

	initArgs := repo.InitArgs{
		ManifestURL:    manifestRepoCheckout,
		ManifestBranch: branch,
		ManifestFile:   manifestFile,
	}
	ctx := context.Background()
	if err := repo.Init(ctx, checkoutPath, repoPath, initArgs); err != nil {
		return "", errors.Annotate(err, "failed to repo init").Err()
	}
	if err := repo.Sync(ctx, checkoutPath, repoPath); err != nil {
		return "", errors.Annotate(err, "failed to repo sync").Err()
	}

	r.localCheckout = checkoutPath
	return checkoutPath, nil
}

// Snapshot recursively copies a directory's contents to a temp dir.
func (r *RepoHarness) Snapshot(path string) (string, error) {
	snapshotRoot := filepath.Join(r.harnessRoot, "snapshots/")
	snapshotDir, err := ioutil.TempDir(snapshotRoot, "snapshot")
	if err != nil {
		return "", err
	}
	if err = copy.Copy(path, snapshotDir); err != nil {
		return "", err
	}
	return snapshotDir, nil
}

// SnapshotRemotes takes a snapshot of the current state of each remote and stores them
// within the harness struct.
func (r *RepoHarness) SnapshotRemotes() error {
	// Take snapshot of each project in its current state.
	r.recentRemoteSnapshots = make(map[string]string)
	for _, remote := range r.Manifest().Remotes {
		remotePath := filepath.Join(r.HarnessRoot(), remote.Name)
		var err error
		r.recentRemoteSnapshots[remote.Name], err = r.Snapshot(remotePath)
		if err != nil {
			return errors.Annotate(err, "error taking snapshot of remote %s", remote.Name).Err()
		}
	}

	return nil
}

// GetRecentRemoteSnapshot returns the path of the most recent snapshot for a particular remote.
func (r *RepoHarness) GetRecentRemoteSnapshot(remote string) (string, error) {
	remoteSnapshot, ok := r.recentRemoteSnapshots[remote]
	if !ok {
		return "", fmt.Errorf("snapshot does not exist for remote %s", remote)
	}
	return remoteSnapshot, nil
}

func (r *RepoHarness) AssertNoRemoteDiff() error {
	manifest := r.Manifest()
	for _, remote := range manifest.Remotes {
		remotePath := filepath.Join(r.HarnessRoot(), remote.Name)
		remoteSnapshot, err := r.GetRecentRemoteSnapshot(remote.Name)
		if err != nil {
			return err
		}
		if err := test_util.AssertContentsEqual(remoteSnapshot, remotePath); err != nil {
			return err
		}
	}
	return nil
}

// GetRemotePath returns the path to the remote project repo.
func (r *RepoHarness) GetRemotePath(project RemoteProject) string {
	return filepath.Join(r.harnessRoot, project.RemoteName, project.ProjectName)
}

// AssertProjectBranches asserts that the remote project has the correct branches.
func (r *RepoHarness) AssertProjectBranches(project RemoteProject, branches []string) error {
	if err := r.assertInitialized(); err != nil {
		return err
	}
	gitRepo := r.GetRemotePath(project)
	return test_util.AssertGitBranches(gitRepo, branches)
}

// AssertProjectBranchesExact asserts that the remote project has only the correct branches.
func (r *RepoHarness) AssertProjectBranchesExact(project RemoteProject, branches []string) error {
	if err := r.assertInitialized(); err != nil {
		return err
	}
	gitRepo := r.GetRemotePath(project)
	return test_util.AssertGitBranchesExact(gitRepo, branches)
}

// AssertProjectBranchesMissing asserts that the remote project does not have the specified branches.
func (r *RepoHarness) AssertProjectBranchesMissing(project RemoteProject, branches []string) error {
	if err := r.assertInitialized(); err != nil {
		return err
	}
	gitRepo := r.GetRemotePath(project)
	assert := test_util.AssertGitBranchesExact(gitRepo, branches)
	if assert != nil && strings.Contains(assert.Error(), "mismatch") {
		return nil
	}
	return fmt.Errorf("project branch mismatch, some of %v existed", branches)
}

// AssertProjectBranchEqual asserts that the specified branch in the project matches
// the corresponding branch in the given snapshot.
func (r *RepoHarness) AssertProjectBranchEqual(project RemoteProject, branch, snapshotPath string) error {
	if err := r.assertInitialized(); err != nil {
		return err
	}
	expected, err := git.GetGitRepoRevision(snapshotPath, branch)
	if err != nil {
		return err
	}
	actual, err := git.GetGitRepoRevision(r.GetRemotePath(project), branch)
	if err != nil {
		return err
	}
	if expected != actual {
		return fmt.Errorf("mismatch for branch %s: project at revision %s, snapshot at revision %s", branch, actual, expected)
	}
	return nil
}

// AssertProjectBranchHasAncestor asserts that the specified branch in the project descends
// from the given snapshot.
func (r *RepoHarness) AssertProjectBranchHasAncestor(project RemoteProject, branch, snapshotPath, snapshotBranch string) error {
	ancestor, err := git.GetGitRepoRevision(snapshotPath, snapshotBranch)
	if err != nil {
		return err
	}
	descendent, err := git.GetGitRepoRevision(r.GetRemotePath(project), branch)
	if err != nil {
		return err
	}

	ok, err := git.IsReachable(r.GetRemotePath(project), ancestor, descendent)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("branch %s does not descend from snapshot at %s:%s", branch, snapshotPath, snapshotBranch)
	}
	return nil
}
