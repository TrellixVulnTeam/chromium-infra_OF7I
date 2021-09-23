// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
//go:build linux
// +build linux

package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/repo"
	rh "infra/cros/internal/repoharness"
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

type fakeFirestoreClient struct {
	t *testing.T
	// Data to return for readFirestoreData calls
	readData localManifestBranchMetadata
	// Data to return for writeFirestoreData calls for various branches
	expectedWriteData map[string]localManifestBranchMetadata
}

func (f *fakeFirestoreClient) readFirestoreData(_ context.Context, _ string) (localManifestBranchMetadata, bool) {
	// Deep copy!
	mapCopy := map[string]string{}
	for k, v := range f.readData.PathToPrevSHA {
		mapCopy[k] = v
	}
	data := f.readData
	f.readData.PathToPrevSHA = mapCopy
	return data, true
}

func (f *fakeFirestoreClient) writeFirestoreData(_ context.Context, branch string, _ bool, bm localManifestBranchMetadata) {
	f.t.Helper()
	expected, ok := f.expectedWriteData[branch]
	if !ok {
		f.t.Fatalf("unexpected call to writeFirestoreData for branch %s", branch)
	}
	assert.Assert(f.t, reflect.DeepEqual(expected, bm))
}

func setUp(t *testing.T) (*rh.RepoHarness, rh.RemoteProject, *fakeFirestoreClient) {
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
	expectedWriteData := map[string]localManifestBranchMetadata{}
	for _, branch := range branches {
		expectedSHA := "-1"
		if branch != ignoreBranch {
			expectedSHA = branchSHAs[branch]
		}
		expectedWriteData[branch] = localManifestBranchMetadata{
			PathToPrevSHA: map[string]string{
				projects[0].Path: expectedSHA,
				projects[1].Path: expectedSHA,
			},
		}
	}
	firestoreClient := &fakeFirestoreClient{
		t: t,
		readData: localManifestBranchMetadata{
			PathToPrevSHA: map[string]string{
				projects[0].Path: "-1",
				projects[1].Path: "-1",
			},
		},
		expectedWriteData: expectedWriteData,
	}

	return harness, remoteManifestProject, firestoreClient
}

func TestBranchLocalManifests(t *testing.T) {
	t.Parallel()
	harness, remoteManifestProject, fsClient := setUp(t)
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
	assert.NilError(t, b.BranchLocalManifests(ctx, fsClient))
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
	t.Parallel()
	r, remoteManifestProject, fsClient := setUp(t)
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
	assert.NilError(t, b.BranchLocalManifests(ctx, fsClient))
	assert.NilError(t, r.AssertNoRemoteDiff())
}

func TestBranchLocalManifests_specificBranches(t *testing.T) {
	t.Parallel()
	harness, remoteManifestProject, fsClient := setUp(t)
	defer harness.Teardown()

	branch := "stabilize-13851.B"
	checkout, err := harness.Checkout(remoteManifestProject, branch, "default.xml")
	assert.NilError(t, err)

	ctx := context.Background()
	b := localManifestBrancher{
		chromeosCheckoutPath: checkout,
		projects:             []string{"foo/", "bar/"},
		specificBranches:     []string{branch},
		push:                 true,
		workerCount:          2,
	}
	assert.NilError(t, b.BranchLocalManifests(ctx, fsClient))
	assert.NilError(t, harness.ProcessSubmitRefs())

	manifest := harness.Manifest()
	project, err := manifest.GetProjectByName("foo")
	assert.NilError(t, err)

	localManifest, err := harness.ReadFile(
		rh.GetRemoteProject(*project), branch, "local_manifest.xml")

	expected := fmt.Sprintf(localManifestXML, branch)
	assert.StringsEqual(t, string(localManifest), expected)
}
