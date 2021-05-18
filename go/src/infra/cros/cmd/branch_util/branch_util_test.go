// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build linux

package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/branch"
	mv "infra/cros/internal/chromeosversion"
	gerrit "infra/cros/internal/gerrit"
	"infra/cros/internal/git"
	"infra/cros/internal/repo"
	rh "infra/cros/internal/repoharness"
	"infra/cros/internal/testutil"

	"infra/cros/cmd/branch_util/test"

	"github.com/golang/mock/gomock"
	"github.com/maruel/subcommands"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/common/proto/gitiles/mock_gitiles"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

const (
	defaultFileName  = "default.xml"
	officialFileName = "official.xml"
	externalFileName = "external.xml"
	internalFileName = "internal.xml"
	remotesFileName  = "_remotes.xml"

	defaultXML = `
  <default revision="refs/heads/main" remote="cros" sync-j="8"/>
`
	remoteExternalXML = `
  <remote name="cros" revision="refs/heads/main" fetch="%s"/>
`
	remoteInternalXML = `
  <remote name="cros-internal" revision="refs/heads/main" fetch="%s"/>
`
	projectsExternalXML = `
  <project path="src/repohooks" name="chromiumos/repohooks"
           groups="minilayout,firmware,buildtools,labtools,crosvm" />
  <repo-hooks in-project="chromiumos/repohooks" enabled-list="pre-upload" />

  <!--This comment should persist.-->
  <project name="chromiumos/manifest" path="manifest"/>

  <new-element name="this should persist" />

  <project name="chromiumos/overlays/chromiumos-overlay"
           path="src/third_party/chromiumos-overlay"/>

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
`
	projectsInternalXML = `
  <project name="chromeos/manifest-internal"
           path="manifest-internal"
           remote="cros-internal"
           upstream="refs/heads/main"/>

  <project name="chromeos/explicit-pinned"
           path="src/explicit-pinned"
           revision="refs/heads/explicit-pinned"
           remote="cros-internal">
    <annotation name="branch-mode" value="pin"/>
  </project>

  <project name="chromeos/explicit-branch"
           path="src/explicit-branch"
           remote="cros-internal">
    <annotation name="branch-mode" value="create"/>
  </project>

  <project name="chromeos/explicit-tot"
           path="src/explicit-tot"
           remote="cros-internal">
    <annotation name="branch-mode" value="tot"/>
  </project>
`
	includeRemotesXML = `
  <include name="_remotes.xml"/>
`
	includeExternalXML = `
  <include name="external.xml"/>
`
	includeInternalXML = `
  <include name="internal.xml"/>
`
)

const (
	existingBranchName = "old-branch"

	defaultBranchedXML = `
  <default remote="cros" sync-j="8"/>
`
	remoteExternalBranchedXML = `
  <remote name="cros" fetch="%s"/>
`
	remoteInternalBranchedXML = `
  <remote name="cros-internal" fetch="%s"/>
`

	projectsExternalBranchedXML = `
  <project path="src/repohooks" name="chromiumos/repohooks"
           groups="minilayout,firmware,buildtools,labtools,crosvm"
           revision="refs/heads/%[1]s" />
  <repo-hooks in-project="chromiumos/repohooks" enabled-list="pre-upload" />

  <!--This comment should persist.-->
  <project name="chromiumos/manifest"
           path="manifest"
           revision="refs/heads/%[1]s"/>

  <new-element name="this should persist" />

  <project name="chromiumos/overlays/chromiumos-overlay"
           path="src/third_party/chromiumos-overlay"
           revision="refs/heads/%[1]s"/>

  <project name="external/implicit-pinned"
           path="src/third_party/implicit-pinned"
           revision="refs/heads/implicit-pinned"/>

  <!--This comment should also persist.-->
  <project name="chromiumos/multicheckout"
           path="src/third_party/multicheckout-a"
           revision="refs/heads/%[1]s-multicheckout-a"/>

  <project name="chromiumos/multicheckout"
           path="src/third_party/multicheckout-b"
           revision="refs/heads/%[1]s-multicheckout-b"/>
`

	projectsInternalBranchedXML = `
  <project name="chromeos/manifest-internal"
           path="manifest-internal"
           remote="cros-internal"
           revision="refs/heads/%[1]s"
           upstream="refs/heads/%[1]s"/>

  <project name="chromeos/explicit-pinned"
           path="src/explicit-pinned"
           revision="refs/heads/explicit-pinned"
           remote="cros-internal">
    <annotation name="branch-mode" value="pin"/>
  </project>

  <project name="chromeos/explicit-branch"
           path="src/explicit-branch"
           remote="cros-internal"
           revision="refs/heads/%[1]s">
    <annotation name="branch-mode" value="create"/>
  </project>

  <project name="chromeos/explicit-tot"
           path="src/explicit-tot"
           remote="cros-internal"
           revision="refs/heads/main">
    <annotation name="branch-mode" value="tot"/>
  </project>
`
)

var (
	manifestProject = rh.RemoteProject{
		RemoteName:  "cros",
		ProjectName: "chromiumos/manifest",
	}
	manifestInternalProject = rh.RemoteProject{
		RemoteName:  "cros-internal",
		ProjectName: "chromeos/manifest-internal",
	}
	application = getApplication(chromeinfra.DefaultAuthOptions())
)

func getManifestFiles(crosFetch, crosInternalFetch string) (
	manifestFiles, manifestInternalFiles map[string]string, fullTotXML string) {
	// We use a fake value here because if a <	remote> tag has fetch="",
	// it will default to the actual gerrit remote.
	if crosFetch == "" {
		crosFetch = "placeholder"
	}
	if crosInternalFetch == "" {
		crosInternalFetch = "placeholder"
	}
	remoteInternalXML := fmt.Sprintf(remoteInternalXML, crosInternalFetch)
	remoteExternalXML := fmt.Sprintf(remoteExternalXML, crosFetch)

	manifestFiles = map[string]string{
		remotesFileName: manifestXML(remoteExternalXML),
		externalFileName: manifestXML(
			defaultXML, includeRemotesXML, projectsExternalXML),
		defaultFileName: manifestXML(includeExternalXML),
	}

	manifestInternalFiles = map[string]string{
		remotesFileName:  manifestXML(remoteExternalXML, remoteInternalXML),
		externalFileName: manifestFiles[externalFileName],
		internalFileName: manifestXML(
			defaultXML, includeRemotesXML, projectsInternalXML),
		officialFileName: manifestXML(
			includeInternalXML, includeExternalXML),
		defaultFileName: manifestXML(
			includeInternalXML, includeExternalXML),
	}

	fullTotXML = manifestXML(
		defaultXML,
		remoteExternalXML,
		remoteInternalXML,
		projectsExternalXML,
		projectsInternalXML,
	)
	return
}

func getBranchedManifestFiles(branch, crosFetch, crosInternalFetch string) (
	manifestBranchedFiles map[string]string,
	manifestInternalBranchedFiles map[string]string,
	fullBranchedXML string) {
	// We use a fake value here because if a <	remote> tag has fetch="",
	// it will default to the actual gerrit remot
	if crosFetch == "" {
		crosFetch = "placeholder"
	}
	if crosInternalFetch == "" {
		crosInternalFetch = "placeholder"
	}
	remoteInternalXML := fmt.Sprintf(remoteInternalBranchedXML, crosInternalFetch)
	remoteExternalXML := fmt.Sprintf(remoteExternalBranchedXML, crosFetch)
	projectsExternalBranchedXML := fmt.Sprintf(projectsExternalBranchedXML, branch)
	projectsInternalBranchedXML := fmt.Sprintf(projectsInternalBranchedXML, branch)

	manifestBranchedFiles = map[string]string{
		remotesFileName: manifestXML(remoteExternalXML),
		externalFileName: manifestXML(
			defaultBranchedXML, includeRemotesXML, projectsExternalBranchedXML),
		defaultFileName: manifestXML(includeExternalXML),
	}

	manifestInternalBranchedFiles = map[string]string{
		remotesFileName:  manifestXML(remoteExternalXML, remoteInternalXML),
		externalFileName: manifestBranchedFiles[externalFileName],
		internalFileName: manifestXML(
			defaultXML, includeRemotesXML, projectsInternalBranchedXML),
		officialFileName: manifestXML(
			includeInternalXML, includeExternalXML),
		defaultFileName: manifestXML(
			includeInternalXML, includeExternalXML),
	}

	fullBranchedXML = manifestXML(
		defaultBranchedXML,
		remoteExternalXML,
		remoteInternalXML,
		projectsExternalBranchedXML,
		projectsInternalBranchedXML,
	)
	return
}

func getExistingBranchManifestFiles(crosFetch, crosInternalFetch string) (
	manifestBranchedFiles map[string]string,
	manifestInternalBranchedFiles map[string]string,
	fullBranchedXML string) {
	return getBranchedManifestFiles(existingBranchName, crosFetch, crosInternalFetch)
}

func manifestXML(chunks ...string) string {
	return fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?>
		<manifest>
		%s
		</manifest>`,
		strings.Join(chunks, ""))
}

func getDefaultConfig() test.CrosRepoHarnessConfig {
	_, _, fullTotXML := getManifestFiles("", "")

	var manifest repo.Manifest
	err := xml.Unmarshal([]byte(fullTotXML), &manifest)
	if err != nil {
		log.Fatalf("failed to parse manifest: %s", err.Error())
	}
	config := test.CrosRepoHarnessConfig{
		Manifest:       manifest,
		VersionProject: "chromiumos/overlays/chromiumos-overlay",
	}
	return config
}

func fullManifestPath(r *test.CrosRepoHarness) string {
	return filepath.Join(r.Harness.HarnessRoot(), "manifest.xml")
}

func fullBranchedManifestPath(r *test.CrosRepoHarness) string {
	return filepath.Join(r.Harness.HarnessRoot(), "manifest-branched.xml")
}

func addManifestFiles(t *testing.T,
	r *test.CrosRepoHarness,
	project rh.RemoteProject,
	branch string,
	files map[string]string) {

	filesToAdd := []rh.File{}
	for file, contents := range files {
		filesToAdd = append(filesToAdd, rh.File{
			Name:     file,
			Contents: []byte(contents),
		})
	}
	_, err := r.Harness.AddFiles(project, branch, filesToAdd)
	assert.NilError(t, err)
}

func setUp(t *testing.T) *test.CrosRepoHarness {
	config := getDefaultConfig()
	var r test.CrosRepoHarness
	assert.NilError(t, r.Initialize(&config))

	// Write version.
	version := mv.VersionInfo{
		ChromeBranch:      12,
		BuildNumber:       3,
		BranchBuildNumber: 0,
		PatchNumber:       0,
	}
	assert.NilError(t, r.SetVersion("", version))

	// Write full tot manifest to a file so that it can be passed to cros branch.
	// CrosRepoHarness modifies the manifest it is passed (e.g. updates fetch attributes),
	// so use the updated version.
	manifest := r.Harness.Manifest()
	manifestData, err := xml.Marshal(manifest)
	assert.NilError(t, err)
	assert.NilError(t, ioutil.WriteFile(fullManifestPath(&r), manifestData, 0777))

	// Because we use a hierachy of manifest files, we need to update the fetch attributes
	// in _remotes.xml.
	crosFetchVal := manifest.GetRemoteByName("cros").Fetch
	crosInternalFetchVal := manifest.GetRemoteByName("cros-internal").Fetch
	manifestFiles, manifestInternalFiles, _ := getManifestFiles(crosFetchVal, crosInternalFetchVal)

	manifestBranchedFiles, manifestInternalBranchedFiles, fullBranchedXML :=
		getExistingBranchManifestFiles(crosFetchVal, crosInternalFetchVal)

	// Add manifest files to remote.
	addManifestFiles(t, &r, manifestProject, "main", manifestFiles)

	// Add manifest-internal files to remote.
	addManifestFiles(t, &r, manifestInternalProject, "main", manifestInternalFiles)

	// Create existing branch on remote.
	var branchManifest *repo.Manifest
	assert.NilError(t, xml.Unmarshal([]byte(fullBranchedXML), &branchManifest))
	branchManifest.ResolveImplicitLinks()
	// Write full branched manifest to file so that it can be passed to cros branch in
	// *Nonmain tests.
	assert.NilError(t, ioutil.WriteFile(fullBranchedManifestPath(&r), []byte(fullBranchedXML), 0777))

	// Create Ref for each project.
	for _, project := range branchManifest.Projects {
		projectBranch := git.StripRefs(project.Revision)
		err = r.Harness.CreateRemoteRef(rh.GetRemoteProject(project), projectBranch, "")
		if err != nil && strings.Contains(err.Error(), "already exists") {
			continue
		}
		assert.NilError(t, err)
	}
	// Set version file.
	version = mv.VersionInfo{
		ChromeBranch:      12,
		BuildNumber:       2,
		BranchBuildNumber: 1,
		PatchNumber:       0,
	}
	assert.NilError(t, r.SetVersion(existingBranchName, version))
	// Commit manifest files.
	addManifestFiles(t, &r, manifestProject, existingBranchName, manifestBranchedFiles)
	addManifestFiles(t, &r, manifestInternalProject, existingBranchName, manifestInternalBranchedFiles)

	assert.NilError(t, r.Harness.SnapshotRemotes())

	return &r
}

// Get the keys of a map[string]string.
func getKeys(m map[string]string) []string {
	keys := make([]string, len(m))
	idx := 0
	for k := range m {
		keys[idx] = k
		idx++
	}
	return keys
}

func assertManifestsRepaired(t *testing.T, r *test.CrosRepoHarness, branch string) {
	manifestFiles, manifestInternalFiles, _ := getManifestFiles("", "")
	assert.NilError(t, r.AssertManifestProjectRepaired(
		manifestProject, branch, getKeys(manifestFiles)))
	assert.NilError(t, r.AssertManifestProjectRepaired(
		manifestInternalProject, branch, getKeys(manifestInternalFiles)))
}

func assertCommentsPersist(t *testing.T, r *test.CrosRepoHarness,
	sourceFiles func(string, string) (map[string]string, map[string]string, string), branch string) {
	manifestFiles, manifestInternalFiles, _ := sourceFiles("", "")
	assert.NilError(t, r.AssertCommentsPersist(manifestProject, branch, manifestFiles))
	assert.NilError(t, r.AssertCommentsPersist(manifestInternalProject, branch, manifestInternalFiles))
}

func assertMinimalManifestChanges(t *testing.T, r *test.CrosRepoHarness, branch string) {
	// Ensure that the created manifests differ minimally from the expected manifests (as produced by
	// getBranchedManifestFiles).
	expectedManifestFiles, expectedManifestInternalFiles, _ := getBranchedManifestFiles(branch, "", "")
	assert.NilError(t, r.AssertMinimalManifestChanges(manifestProject, branch, expectedManifestFiles))
	assert.NilError(t, r.AssertMinimalManifestChanges(manifestInternalProject, branch, expectedManifestInternalFiles))
}

func assertNoRemoteDiff(t *testing.T, r *test.CrosRepoHarness) {
	assert.NilError(t, r.Harness.AssertNoRemoteDiff())
}

type fakeCreateRemoteBranchesAPI struct {
	t      *testing.T
	r      *test.CrosRepoHarness
	dryRun bool
	force  bool
}

func (f *fakeCreateRemoteBranchesAPI) CreateRemoteBranchesAPI(
	_ *http.Client, branches []branch.GerritProjectBranch, dryRun bool, _ float64) error {
	if f.dryRun {
		return nil
	}
	// Create the appropriate branches on the appropriate remote projects.
	for _, projectBranch := range branches {
		manifest := f.r.Harness.Manifest()
		project, err := manifest.GetProjectByName(projectBranch.Project)
		assert.NilError(f.t, err)

		remoteProject := rh.GetRemoteProject(*project)
		if f.force {
			err = f.r.Harness.CreateRemoteRefForce(remoteProject, projectBranch.Branch, projectBranch.SrcRef)
		} else {
			err = f.r.Harness.CreateRemoteRef(remoteProject, projectBranch.Branch, projectBranch.SrcRef)
		}
		assert.NilError(f.t, err)
	}
	return nil
}

// setUpCreate creates the neccessary mocks we need to test the create-v2 function
func setUpCreate(t *testing.T, dryRun, force, useBranch bool) (*test.CrosRepoHarness, error) {
	r := setUp(t)

	// Get manifest contents for return
	manifestPath := fullManifestPath(r)
	buildspecName := "buildspecs/12/3.0.0.xml"
	if useBranch {
		manifestPath = fullBranchedManifestPath(r)
		buildspecName = "buildspecs/12/2.1.0.xml"
	}
	manifestFile, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	manifest := string(manifestFile)

	// Get version file contents for return
	versionProject := rh.GetRemoteProject(test.DefaultVersionProject)
	versionBranch := "main"
	if useBranch {
		versionBranch = existingBranchName
	}
	crosVersionFile, err := r.Harness.ReadFile(versionProject, versionBranch, mv.VersionFileProjectPath)
	assert.NilError(t, err)

	// Mock Gitiles controller
	ctl := gomock.NewController(t)

	// Mock manifest request
	reqManifest := &gitilespb.DownloadFileRequest{
		Project:    "chromeos/manifest-versions",
		Path:       buildspecName,
		Committish: "HEAD",
		Format:     gitilespb.DownloadFileRequest_TEXT,
	}

	// Mock version file request
	reqVersionFile := &gitilespb.DownloadFileRequest{
		Project:    "chromiumos/overlays/chromiumos-overlay",
		Path:       "chromeos/config/chromeos_version.sh",
		Committish: "refs/heads/" + versionBranch,
		Format:     gitilespb.DownloadFileRequest_TEXT,
	}

	// Mock out calls to gerrit.DownloadFileFromGitiles.
	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)
	gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(reqManifest)).Return(
		&gitilespb.DownloadFileResponse{
			Contents: manifest,
		},
		nil,
	)
	gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(reqVersionFile)).Return(
		&gitilespb.DownloadFileResponse{
			Contents: string(crosVersionFile),
		},
		nil,
	)
	gerrit.MockGitiles = gitilesMock

	// Mock out call to CreateRemoteBranchesAPI.
	f := &fakeCreateRemoteBranchesAPI{t: t, r: r, dryRun: dryRun, force: force}
	CreateRemoteBranchesAPI = f.CreateRemoteBranchesAPI

	return r, nil
}

func TestCreate(t *testing.T) {
	r, err := setUpCreate(t, false, false, false)
	defer r.Teardown()
	assert.NilError(t, err)

	branch := "new-branch"
	s := &branchApplication{application, nil, nil}
	ret := subcommands.Run(s, []string{
		"create", "--push",
		"--custom", branch,
		"--buildspec-manifest", "12/3.0.0.xml",
		// We don't really care about this check as ACLs are still enforced
		// (just a less elegant failure), and it's one less thing to mock.
		"--skip-group-check",
		// Test with two workers for kicks.
		"-j", "2",
	})
	assert.Assert(t, ret == 0)

	manifest := r.Harness.Manifest()
	assert.NilError(t, r.AssertCrosBranches([]string{branch}))
	assert.NilError(t, r.AssertCrosBranchFromManifest(manifest, branch, ""))
	assertManifestsRepaired(t, r, branch)
	newBranchVersion := mv.VersionInfo{
		ChromeBranch:      12,
		BuildNumber:       3,
		BranchBuildNumber: 1,
		PatchNumber:       0,
	}
	assert.NilError(t, r.AssertCrosVersion(branch, newBranchVersion))
	mainVersion := mv.VersionInfo{
		ChromeBranch:      12,
		BuildNumber:       4,
		BranchBuildNumber: 0,
		PatchNumber:       0,
	}
	assert.NilError(t, r.AssertCrosVersion("main", mainVersion))

	assertCommentsPersist(t, r, getManifestFiles, branch)
	// Check that manifests were minmally changed (e.g. element ordering preserved).
	// This check is meaningful because the manifests are created using the branch_util
	// tool which reads in, unmarshals, and modifies the manifests from getManifestFiles.
	// The expected manifests (which the branched manifests are being compared to)
	// are simply strings produced by getBranchedManifestFiles.
	assertMinimalManifestChanges(t, r, branch)
}

// Branch off of old-branch to make sure that the source version is being
// bumped in the correct branch.
// Covers crbug.com/1744928.
func TestCreateReleaseNonmain(t *testing.T) {
	r, err := setUpCreate(t, false, false, true)
	defer r.Teardown()
	assert.NilError(t, err)

	manifest := r.Harness.Manifest()
	branch := "release-R12-2.1.B"

	s := &branchApplication{application, nil, nil}
	ret := subcommands.Run(s, []string{
		"create", "--push",
		"--buildspec-manifest", "12/2.1.0.xml",
		"--release",
		// We don't really care about this check as ACLs are still enforced
		// (just a less elegant failure), and it's one less thing to mock.
		"--skip-group-check",
	})
	assert.Assert(t, ret == 0)

	assert.NilError(t, r.AssertCrosBranches([]string{branch}))

	crosFetchVal := manifest.GetRemoteByName("cros").Fetch
	crosInternalFetchVal := manifest.GetRemoteByName("cros-internal").Fetch
	_, _, fullBranchedXML := getBranchedManifestFiles(existingBranchName, crosFetchVal, crosInternalFetchVal)
	var branchManifest *repo.Manifest
	assert.NilError(t, xml.Unmarshal([]byte(fullBranchedXML), &branchManifest))
	branchManifest.ResolveImplicitLinks()

	assert.NilError(t, r.AssertCrosBranchFromManifest(*branchManifest, branch, "old-branch"))
	assertManifestsRepaired(t, r, branch)
	newBranchVersion := mv.VersionInfo{
		ChromeBranch:      12,
		BuildNumber:       2,
		BranchBuildNumber: 1,
		PatchNumber:       1,
	}
	assert.NilError(t, r.AssertCrosVersion(branch, newBranchVersion))
	sourceVersion := mv.VersionInfo{
		ChromeBranch:      13,
		BuildNumber:       3,
		BranchBuildNumber: 0,
		PatchNumber:       0,
	}
	assert.NilError(t, r.AssertCrosVersion("old-branch", sourceVersion))

	assertCommentsPersist(t, r, getExistingBranchManifestFiles, branch)
}
func TestCreateDryRun(t *testing.T) {
	r, err := setUpCreate(t, true, false, false)
	defer r.Teardown()
	assert.NilError(t, err)

	branch := "new-branch"
	s := &branchApplication{application, nil, nil}
	ret := subcommands.Run(s, []string{
		"create",
		"--buildspec-manifest", "12/3.0.0.xml",
		"--custom", branch,
		// We don't really care about this check as ACLs are still enforced
		// (just a less elegant failure), and it's one less thing to mock.
		"--skip-group-check",
	})
	assert.Assert(t, ret == 0)
	assertNoRemoteDiff(t, r)
}

// Test create overwrites existing branches when --force is set.
func TestCreateOverwrite(t *testing.T) {
	r, err := setUpCreate(t, false, true, false)
	defer r.Teardown()
	assert.NilError(t, err)

	manifest := r.Harness.Manifest()

	branch := "old-branch"
	s := &branchApplication{application, nil, nil}
	ret := subcommands.Run(s, []string{
		"create", "--push",
		"--force",
		"--buildspec-manifest", "12/3.0.0.xml",
		"--custom", branch,
		// We don't really care about this check as ACLs are still enforced
		// (just a less elegant failure), and it's one less thing to mock.
		"--skip-group-check",
	})
	assert.Assert(t, ret == 0)

	assert.NilError(t, r.AssertCrosBranches([]string{branch}))
	assert.NilError(t, r.AssertCrosBranchFromManifest(manifest, branch, ""))
	assertManifestsRepaired(t, r, branch)
	newBranchVersion := mv.VersionInfo{
		ChromeBranch:      12,
		BuildNumber:       3,
		BranchBuildNumber: 1,
		PatchNumber:       0,
	}
	assert.NilError(t, r.AssertCrosVersion(branch, newBranchVersion))
	mainVersion := mv.VersionInfo{
		ChromeBranch:      12,
		BuildNumber:       4,
		BranchBuildNumber: 0,
		PatchNumber:       0,
	}
	assert.NilError(t, r.AssertCrosVersion("main", mainVersion))

	assertCommentsPersist(t, r, getManifestFiles, branch)
}

// Test create dies when it tries to overwrite without --force.
func TestCreateOverwriteMissingForce(t *testing.T) {
	r, err := setUpCreate(t, false, false, false)
	defer r.Teardown()
	assert.NilError(t, err)

	manifest := r.Harness.Manifest()

	branch := "old-branch"
	var stderrBuf bytes.Buffer
	stderrLog := log.New(&stderrBuf, "", log.LstdFlags|log.Lmicroseconds)
	s := &branchApplication{application, nil, stderrLog}
	ret := subcommands.Run(s, []string{
		"create", "--push",
		"--buildspec-manifest", "12/3.0.0.xml",
		"--custom", branch,
		// We don't really care about this check as ACLs are still enforced
		// (just a less elegant failure), and it's one less thing to mock.
		"--skip-group-check",
	})
	assert.Assert(t, ret != 0)
	assert.Assert(t, strings.Contains(stderrBuf.String(), "rerun with --force"))

	// Check that no remotes change.
	for _, remote := range manifest.Remotes {
		remotePath := filepath.Join(r.Harness.HarnessRoot(), remote.Name)
		remoteSnapshot, err := r.Harness.GetRecentRemoteSnapshot(remote.Name)
		assert.NilError(t, err)
		assert.NilError(t, testutil.AssertContentsEqual(remoteSnapshot, remotePath))
	}
}

// Test create dies when given a version that was already branched.
func TestCreatExistingVersion(t *testing.T) {
	r, err := setUpCreate(t, false, false, false)
	defer r.Teardown()
	assert.NilError(t, err)

	// Our set up uses branch 12.3.0.0. A branch created from this version must
	// end in 12-3.B. We create a branch with that suffix so that the tool
	// will think 12.3.0.0 has already been branched.
	// We just need to add a branch to the manifest internal repo because
	// the tool checks if a branch exists for a version by looking at
	// branches in the manifest internal repo.
	assert.NilError(t,
		r.Harness.CreateRemoteRef(manifestInternalProject, "release-R12-3.B", ""))
	// Snapshot of manifestInternalProject is stale -- need to update.
	assert.NilError(t, r.Harness.SnapshotRemotes())

	var stderrBuf bytes.Buffer
	stderrLog := log.New(&stderrBuf, "", log.LstdFlags|log.Lmicroseconds)
	s := &branchApplication{application, nil, stderrLog}
	ret := subcommands.Run(s, []string{
		"create-v1", "--push",
		"--buildspec-manifest", "12/3.0.0.xml",
		"--stabilize",
		// We don't really care about this check as ACLs are still enforced
		// (just a less elegant failure), and it's one less thing to mock.
		"--skip-group-check",
	})
	assert.Assert(t, ret != 0)
	assert.Assert(t, strings.Contains(stderrBuf.String(), "already branched 3.0.0"))
	assertNoRemoteDiff(t, r)
}

func TestRename(t *testing.T) {
	r := setUp(t)
	defer r.Teardown()

	localRoot, err := ioutil.TempDir("", "test_rename")
	defer os.RemoveAll(localRoot)
	assert.NilError(t, err)
	manifest := r.Harness.Manifest()
	manifestDir := r.Harness.GetRemotePath(manifestInternalProject)

	oldBranch := existingBranchName // "old-branch"
	newBranch := "new-branch"

	s := &branchApplication{application, nil, nil}
	ret := subcommands.Run(s, []string{
		"rename", "--push",
		"--manifest-url", manifestDir,
		oldBranch, newBranch,
	})
	assert.Assert(t, ret == 0)

	assert.NilError(t, r.AssertCrosBranches([]string{newBranch}))
	assert.NilError(t, r.AssertCrosBranchesMissing([]string{oldBranch}))

	// Get manifest for oldBranch.
	crosFetchVal := manifest.GetRemoteByName("cros").Fetch
	crosInternalFetchVal := manifest.GetRemoteByName("cros-internal").Fetch
	_, _, fullBranchedXML := getExistingBranchManifestFiles(crosFetchVal, crosInternalFetchVal)
	var branchManifest *repo.Manifest
	assert.NilError(t, xml.Unmarshal([]byte(fullBranchedXML), &branchManifest))
	branchManifest.ResolveImplicitLinks()

	assert.NilError(t, r.AssertCrosBranchFromManifest(*branchManifest, newBranch, oldBranch))
	assertManifestsRepaired(t, r, newBranch)
	newBranchVersion := mv.VersionInfo{
		ChromeBranch:      12,
		BuildNumber:       2,
		BranchBuildNumber: 1,
		PatchNumber:       0,
	}
	assert.NilError(t, r.AssertCrosVersion(newBranch, newBranchVersion))

	assertCommentsPersist(t, r, getExistingBranchManifestFiles, newBranch)
	assertMinimalManifestChanges(t, r, newBranch)
}

func TestRenameDryRun(t *testing.T) {
	r := setUp(t)
	defer r.Teardown()

	localRoot, err := ioutil.TempDir("", "test_rename")
	defer os.RemoveAll(localRoot)
	assert.NilError(t, err)
	manifestDir := r.Harness.GetRemotePath(manifestInternalProject)

	oldBranch := "old-branch"
	newBranch := "new-branch"

	s := &branchApplication{application, nil, nil}
	ret := subcommands.Run(s, []string{
		"rename",
		"--manifest-url", manifestDir,
		oldBranch, newBranch,
	})
	assert.Assert(t, ret == 0)

	assertNoRemoteDiff(t, r)
}

// Test rename successfully force overwrites.
func TestRenameOverwrite(t *testing.T) {
	r := setUp(t)
	defer r.Teardown()

	localRoot, err := ioutil.TempDir("", "test_rename")
	defer os.RemoveAll(localRoot)
	assert.NilError(t, err)
	manifest := r.Harness.Manifest()
	manifestDir := r.Harness.GetRemotePath(manifestInternalProject)

	oldBranch := existingBranchName // "old-branch"
	newBranch := "main"

	newBranchVersion := mv.VersionInfo{
		ChromeBranch:      12,
		BuildNumber:       3,
		BranchBuildNumber: 0,
		PatchNumber:       0,
	}
	assert.NilError(t, r.AssertCrosVersion(newBranch, newBranchVersion))
	oldBranchVersion := mv.VersionInfo{
		ChromeBranch:      12,
		BuildNumber:       2,
		BranchBuildNumber: 1,
		PatchNumber:       0,
	}
	assert.NilError(t, r.AssertCrosVersion(oldBranch, oldBranchVersion))

	assertCommentsPersist(t, r, getManifestFiles, newBranch)

	// Gah! Turns out we actually wanted what's in oldBranch. Let's try force renaming
	// oldBranch to main, overwriting the existing contents of main in the process.
	s := &branchApplication{application, nil, nil}
	ret := subcommands.Run(s, []string{
		"rename", "--push", "--force",
		"--manifest-url", manifestDir,
		oldBranch, "main",
	})
	assert.Assert(t, ret == 0)

	assert.NilError(t, r.AssertCrosBranches([]string{newBranch}))
	assert.NilError(t, r.AssertCrosBranchesMissing([]string{oldBranch}))

	// Get manifest for oldBranch.
	crosFetchVal := manifest.GetRemoteByName("cros").Fetch
	crosInternalFetchVal := manifest.GetRemoteByName("cros-internal").Fetch
	_, _, fullBranchedXML := getExistingBranchManifestFiles(crosFetchVal, crosInternalFetchVal)
	var branchManifest *repo.Manifest
	assert.NilError(t, xml.Unmarshal([]byte(fullBranchedXML), &branchManifest))
	branchManifest.ResolveImplicitLinks()

	assert.NilError(t, r.AssertCrosBranchFromManifest(*branchManifest, newBranch, oldBranch))
	assertManifestsRepaired(t, r, newBranch)
	assert.NilError(t, r.AssertCrosVersion(newBranch, oldBranchVersion))

	assertCommentsPersist(t, r, getExistingBranchManifestFiles, newBranch)
}

// Test rename dies if it tries to overwrite without --force.
func TestRenameOverwriteMissingForce(t *testing.T) {
	r := setUp(t)
	defer r.Teardown()

	localRoot, err := ioutil.TempDir("", "test_rename")
	defer os.RemoveAll(localRoot)
	assert.NilError(t, err)
	manifestDir := r.Harness.GetRemotePath(manifestInternalProject)

	oldBranch := "old-branch"

	var stderrBuf bytes.Buffer
	stderrLog := log.New(&stderrBuf, "", log.LstdFlags|log.Lmicroseconds)
	s := &branchApplication{application, nil, stderrLog}
	ret := subcommands.Run(s, []string{
		"rename", "--push",
		"--manifest-url", manifestDir,
		"main", oldBranch,
	})
	assert.Assert(t, ret != 0)
	assert.Assert(t, strings.Contains(stderrBuf.String(), "rerun with --force"))
	assertNoRemoteDiff(t, r)
}

func TestDelete(t *testing.T) {
	r := setUp(t)
	defer r.Teardown()

	localRoot, err := ioutil.TempDir("", "test_delete")
	defer os.RemoveAll(localRoot)
	assert.NilError(t, err)

	branchToDelete := "old-branch"

	manifestDir := r.Harness.GetRemotePath(manifestInternalProject)

	s := &branchApplication{application, nil, nil}
	ret := subcommands.Run(s, []string{
		"delete", "--push", "--force",
		"--manifest-url", manifestDir,
		branchToDelete,
	})
	assert.Assert(t, ret == 0)

	assert.NilError(t, r.AssertCrosBranchesMissing([]string{branchToDelete}))
}

// Test delete does not modify remote repositories without --push.
func TestDeleteDryRun(t *testing.T) {
	r := setUp(t)
	defer r.Teardown()

	localRoot, err := ioutil.TempDir("", "test_delete")
	defer os.RemoveAll(localRoot)
	assert.NilError(t, err)

	branchToDelete := "old-branch"

	manifestDir := r.Harness.GetRemotePath(manifestInternalProject)

	var outputBuf bytes.Buffer
	stdoutLog := log.New(&outputBuf, "stdout: ", log.LstdFlags|log.Lmicroseconds)
	stderrLog := log.New(&outputBuf, "stderr: ", log.LstdFlags|log.Lmicroseconds)

	s := &branchApplication{application, stdoutLog, stderrLog}
	ret := subcommands.Run(s, []string{
		"delete", "--force",
		"--manifest-url", manifestDir,
		branchToDelete,
	})
	assert.Assert(t, ret == 0)
	assertNoRemoteDiff(t, r)
}

// Test delete does not modify remote when --push set without --force.
func TestDeleteMissingForce(t *testing.T) {
	r := setUp(t)
	defer r.Teardown()

	localRoot, err := ioutil.TempDir("", "test_delete")
	defer os.RemoveAll(localRoot)
	assert.NilError(t, err)

	branchToDelete := "old-branch"

	manifestDir := r.Harness.GetRemotePath(manifestInternalProject)
	var stderrBuf bytes.Buffer
	stderrLog := log.New(&stderrBuf, "", log.LstdFlags|log.Lmicroseconds)
	s := &branchApplication{application, nil, stderrLog}
	ret := subcommands.Run(s, []string{
		"delete", "--push",
		"--manifest-url", manifestDir,
		branchToDelete,
	})
	assert.Assert(t, ret != 0)
	assert.Assert(t, strings.Contains(stderrBuf.String(), "Must set --force to delete remote branches."))
	assertNoRemoteDiff(t, r)
}
