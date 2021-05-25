// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build linux

package main

import (
	"context"
	"fmt"
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
</manifest>
`

	localManifestXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <!-- LOCAL MANIFEST -->
  <default remote="cros-internal" revision="123"/>
  <remote name="cros-internal" />

  <project name="foo" path="foo/" revision="%s" />
</manifest>
`
)

var (
	branches = []string{
		"main",
		"release-R89-13729.B",
		"release-R90-13816.B",
		"release-R91-13904.B",
		"stabilize-13851.B",
		// Extra branch that we will NOT create a local_manifest.xml for, to
		// ensure that the tool ignores such branches gracefully.
		"stabilize-13852.B",
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
			},
		},
	}
	harness := &rh.RepoHarness{}
	err := harness.Initialize(config)
	assert.NilError(t, err)

	remoteManifestProject := rh.GetRemoteProject(config.Manifest.Projects[0])
	fooProject := config.Manifest.Projects[1]
	remoteFooProject := rh.GetRemoteProject(fooProject)

	fetchLocation := harness.Manifest().Remotes[0].Fetch

	localManifestFile := rh.File{
		Name:     "local_manifest.xml",
		Contents: []byte(fmt.Sprintf(localManifestXML, "main")),
	}

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
		_, err := harness.AddFile(remoteManifestProject, branch, referenceManifest)
		assert.NilError(t, err)

		// Create corresponding branches in project directories.
		if branch != "main" {
			assert.NilError(t, harness.CreateRemoteRef(remoteFooProject, branch, ""))
		}
		// Create local_manifest.xml files in appropriate branches.
		if branch != "stabilize-13852.B" {
			_, err = harness.AddFile(remoteFooProject, branch, localManifestFile)
			assert.NilError(t, err)
		}
	}

	return harness, remoteManifestProject
}

func TestBranchLocalManifests(t *testing.T) {
	harness, remoteManifestProject := setUp(t)
	defer harness.Teardown()

	checkout, err := harness.Checkout(remoteManifestProject, "main", "default.xml")
	assert.NilError(t, err)

	ctx := context.Background()
	assert.NilError(t, BranchLocalManifests(ctx, nil, checkout, []string{"foo/"}, 90, false))
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
	assert.NilError(t, BranchLocalManifests(ctx, nil, checkout, []string{"foo/"}, 90, true))
	assert.NilError(t, r.AssertNoRemoteDiff())
}
