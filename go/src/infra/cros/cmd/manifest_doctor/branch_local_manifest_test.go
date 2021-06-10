// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build linux

package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/gerrit"
	"infra/cros/internal/repo"
	rh "infra/cros/internal/repoharness"

	"cloud.google.com/go/firestore"
	"github.com/golang/mock/gomock"
	"go.chromium.org/luci/common/proto/gitiles/mock_gitiles"
)

const (
	referenceManifestXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <default remote="cros-internal" revision="123"/>
  <remote fetch="%[2]s" name="cros-internal" />

  <project name="manifest-internal" path="manifest-internal/" revision="%[1]s" />
  <project name="foo" path="foo/" revision="%[1]s" />
  <project name="bar" path="bar/" revision="%[1]s" />
  <project name="bar" path="baz/" revision="%[1]s" />
</manifest>
`

	localManifestXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <!-- LOCAL MANIFEST -->
  <default remote="cros-internal" revision="123"/>
  <remote name="cros-internal" />

  <project name="foo" path="foo/" revision="%[1]s" />
  <project name="bar" path="bar/" revision="%[1]s" />
</manifest>
`
)

var (
	ignoreBranch = "stabilize-13852.B"
	branches     = []string{
		"main",
		"release-R89-13729.B",
		"release-R90-13816.B",
		"release-R91-13904.B",
		"stabilize-13851.B",
		// Extra branch that we will NOT create a local_manifest.xml for, to
		// ensure that the tool ignores such branches gracefully.
		ignoreBranch,
	}
)

func setUp(t *testing.T) (*rh.RepoHarness, rh.RemoteProject) {
	config := &rh.Config{
		Manifest: repo.Manifest{
			Default: repo.Default{
				RemoteName: "cros",
			},
			Remotes: []repo.Remote{
				{Name: "cros"},
			},
			Projects: []repo.Project{
				{Path: "manifest-internal/", Name: "manifest-internal"},
				{Path: "foo/", Name: "foo"},
				{Path: "bar/", Name: "bar"},
				{Path: "baz/", Name: "baz"},
			},
		},
	}
	harness := &rh.RepoHarness{}
	err := harness.Initialize(config)
	assert.NilError(t, err)

	remoteManifestProject := rh.GetRemoteProject(config.Manifest.Projects[0])
	// foo and bar
	projects := []repo.Project{
		config.Manifest.Projects[1],
		config.Manifest.Projects[2],
	}

	fetchLocation := harness.Manifest().Remotes[0].Fetch

	localManifestFile := rh.File{
		Name:     "local_manifest.xml",
		Contents: []byte(fmt.Sprintf(localManifestXML, "main")),
	}

	// Mock Gitiles controller
	ctl := gomock.NewController(t)
	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)
	gerrit.MockGitiles = gitilesMock

	branchSHAs := make(map[string]string)
	for _, branch := range branches {
		// Create appropriate branches and files in our sentinel repository,
		// manifest-internal.
		if branch != "main" {
			assert.NilError(t, harness.CreateRemoteRef(remoteManifestProject, branch, ""))
		}
		referenceManifest := rh.File{
			Name:     "default.xml",
			Contents: []byte(fmt.Sprintf(referenceManifestXML, branch, fetchLocation)),
		}
		branchSHAs[branch], err = harness.AddFile(remoteManifestProject, branch, referenceManifest)
		assert.NilError(t, err)

		for _, project := range projects {
			remoteProject := rh.GetRemoteProject(project)
			// Create corresponding branches in project directories.
			if branch != "main" {
				assert.NilError(t, harness.CreateRemoteRef(remoteProject, branch, ""))
			}
			// Create local_manifest.xml files in appropriate branches.
			if branch != ignoreBranch {
				_, err = harness.AddFile(remoteProject, branch, localManifestFile)
				assert.NilError(t, err)
			}
		}
	}

	// Mock readFirestoreData and writeFirestoreData
	readFirestoreData = func(ctx context.Context, dsClient *firestore.Client, branch string) (localManifestBranchMetadata, bool) {
		// Set SHA to something out of bounds for our mocked gitiles call to force updates.
		bm := localManifestBranchMetadata{
			PathToPrevSHA: map[string]string{
				projects[0].Path: "-1",
				projects[1].Path: "-1",
			},
		}
		return bm, true
	}
	writeFirestoreData = func(ctx context.Context, dsClient *firestore.Client, branch string, docExists bool, bm localManifestBranchMetadata) {
		expectedSHA := "-1"
		if branch != ignoreBranch {
			expectedSHA = branchSHAs[branch]
		}
		expected := localManifestBranchMetadata{
			PathToPrevSHA: map[string]string{
				projects[0].Path: expectedSHA,
				projects[1].Path: expectedSHA,
			},
		}
		assert.Assert(t, reflect.DeepEqual(expected, bm))
	}

	return harness, remoteManifestProject
}

func TestBranchLocalManifests(t *testing.T) {
	harness, remoteManifestProject := setUp(t)
	defer harness.Teardown()

	checkout, err := harness.Checkout(remoteManifestProject, "main", "default.xml")
	assert.NilError(t, err)

	ctx := context.Background()
	b := localManifestBrancher{
		chromeosCheckoutPath: checkout,
		projects:             []string{"foo/", "bar/"},
		minMilestone:         90,
		push:                 true,
		workerCount:          2,
	}
	assert.NilError(t, b.BranchLocalManifests(ctx, nil))
	assert.NilError(t, harness.ProcessSubmitRefs())

	checkBranches := map[string]bool{
		"release-R90-13816.B": true,
		"release-R91-13904.B": true,
		"stabilize-13851.B":   true,
	}
	manifest := harness.Manifest()
	for _, branch := range branches {
		if branch == "stabilize-13852.B" {
			continue
		}

		project, err := manifest.GetProjectByName("foo")
		assert.NilError(t, err)

		localManifest, err := harness.ReadFile(
			rh.GetRemoteProject(*project), branch, "local_manifest.xml")

		var expected string
		if branched, ok := checkBranches[branch]; branched && ok {
			expected = fmt.Sprintf(localManifestXML, branch)
		} else {
			expected = fmt.Sprintf(localManifestXML, "main")
		}
		assert.StringsEqual(t, string(localManifest), expected)
	}
}

func TestBranchLocalManifestsDryRun(t *testing.T) {
	r, remoteManifestProject := setUp(t)
	defer r.Teardown()

	assert.NilError(t, r.SnapshotRemotes())

	checkout, err := r.Checkout(remoteManifestProject, "main", "default.xml")
	assert.NilError(t, err)

	ctx := context.Background()
	b := localManifestBrancher{
		chromeosCheckoutPath: checkout,
		projects:             []string{"foo/"},
		minMilestone:         90,
		push:                 false,
		workerCount:          1,
	}
	assert.NilError(t, b.BranchLocalManifests(ctx, nil))
	assert.NilError(t, r.AssertNoRemoteDiff())
}
