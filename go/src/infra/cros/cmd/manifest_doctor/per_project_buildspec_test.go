// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"testing"
	"time"

	"infra/cros/internal/assert"
	gerrit "infra/cros/internal/gerrit"
	"infra/cros/internal/gs"

	lgs "go.chromium.org/luci/common/gcloud/gs"
	"go.chromium.org/luci/hardcoded/chromeinfra"
)

const (
	unpinnedLocalManifestXML = `
<manifest>
  <remote name="cros-internal"
          fetch="https://chrome-internal.googlesource.com"
          review="https://chrome-internal-review.googlesource.com" />
  <project name="foo"
	   path="src/foo"
	   remote="cros-internal" />
  <project name="bar"
	   path="src/bar"
	   remote="cros-internal" />
</manifest>
`

	buildspecXML = `
<manifest>
  <remote name="cros-internal"
          fetch="https://chrome-internal.googlesource.com"
          review="https://chrome-internal-review.googlesource.com" />
  <project name="foo"
	   path="src/foo"
	   revision="revision-foo"
	   remote="cros-internal" />
  <project name="baz"
	   path="src/baz"
	   revision="revision-baz"
	   remote="cros-internal" />
</manifest>
`

	pinnedLocalManifestXML = `<manifest>
  <remote fetch="https://chrome-internal.googlesource.com" name="cros-internal" review="https://chrome-internal-review.googlesource.com"></remote>
  <default></default>
  <project path="src/foo" name="foo" revision="revision-foo" remote="cros-internal"></project>
  <project path="src/bar" name="bar" remote="cros-internal"></project>
</manifest>`
)

var (
	application = GetApplication(chromeinfra.DefaultAuthOptions())
)

type testConfig struct {
	projects   map[string][]string
	otherRepos []string
	// Map between buildspec name and whether or not to expect a GS write.
	buildspecs              map[string]bool
	branches                []string
	buildspecsExists        bool
	expectedForce           bool
	watchPaths              map[string][]string
	allProjects             []string
	noLocalManifestProjects []string
	expectedSetTTL          map[string]time.Duration
	dryRun                  bool
}

func (tc *testConfig) setUpPPBTest(t *testing.T) (*gs.FakeClient, gerrit.Client) {
	t.Helper()

	projects := map[string]string{}
	for prog, projs := range tc.projects {
		projects["chromeos/program/"+prog] = fmt.Sprintf("chromeos-%s", prog)
		for _, proj := range projs {
			projects["chromeos/project/"+prog+"/"+proj] = fmt.Sprintf("chromeos-%s-%s", prog, proj)
		}
	}
	for _, repo := range tc.otherRepos {
		projects[repo] = gsBuildspecPath(repo).Bucket()
	}
	// Mock tip-of-branch (branch) manifest file requests.
	expectedDownloads := map[gerrit.ExpectedPathParams]*string{}
	for project := range projects {
		contents := unpinnedLocalManifestXML
		for _, branch := range tc.branches {
			expectedDownloads[gerrit.ExpectedPathParams{
				Host:    chromeInternalHost,
				Project: project,
				Path:    "local_manifest.xml",
				Ref:     branch,
			}] = &contents
		}
	}
	for _, project := range tc.noLocalManifestProjects {
		for _, branch := range tc.branches {
			expectedDownloads[gerrit.ExpectedPathParams{
				Host:    chromeInternalHost,
				Project: project,
				Path:    "local_manifest.xml",
				Ref:     branch,
			}] = nil
		}
	}

	gc := &gerrit.MockClient{
		T: t,
		ExpectedProjects: map[string][]string{
			chromeInternalHost: tc.allProjects,
		},
		ExpectedBranches: map[string]map[string]map[string]string{
			// Mock manifest-internal branches request.
			chromeInternalHost: {
				"chromeos/manifest-internal": {
					"refs/heads/main":                "deadcafe",
					"refs/heads/release-R93-13816.B": "deadbeef",
					"refs/heads/release-R94-13904.B": "beefcafe",
				},
			},
		},
		ExpectedDownloads: expectedDownloads,
	}

	// Mock external and internal buildspec file requests.
	expectedReads := map[string][]byte{}
	expectedWrites := map[string][]byte{}
	for buildspec := range tc.buildspecs {
		expectedReads[string(lgs.MakePath(externalBuildspecsGSBucketDefault, buildspec))] = []byte("")
		expectedReads[string(lgs.MakePath(internalBuildspecsGSBucketDefault, buildspec))] = []byte(buildspecXML)
	}

	expectedBucketLists := make(map[string][]string)

	for buildspec, expectWrite := range tc.buildspecs {
		relpath := fmt.Sprintf("buildspecs/%s", buildspec)
		if expectWrite {
			for _, bucket := range projects {
				expectedWrites[string(lgs.MakePath(bucket, relpath))] = []byte(pinnedLocalManifestXML)
			}
		}
		list := []string{}
		if tc.buildspecsExists {
			list = []string{"buildspecs/" + buildspec}
		}
		expectedBucketLists[relpath] = list
		expectedBucketLists[relpath] = list
	}
	expectedLists := make(map[string]map[string][]string)
	for _, bucket := range projects {
		expectedLists[bucket] = expectedBucketLists
	}
	if tc.dryRun {
		expectedWrites = make(map[string][]byte)
	}
	// Set up gs.List expectations.
	expectedLists[externalBuildspecsGSBucketDefault] = map[string][]string{}
	expectedLists[internalBuildspecsGSBucketDefault] = map[string][]string{}

	if tc.watchPaths != nil {
		for prefix, files := range tc.watchPaths {
			expectedLists[externalBuildspecsGSBucketDefault][prefix] = files
			expectedLists[internalBuildspecsGSBucketDefault][prefix] = files
		}
	}

	f := &gs.FakeClient{
		T:              t,
		ExpectedReads:  expectedReads,
		ExpectedWrites: expectedWrites,
		ExpectedLists:  expectedLists,
		ExpectedSetTTL: tc.expectedSetTTL,
	}
	return f, gc
}

func TestCreateProjectBuildspec(t *testing.T) {
	t.Parallel()
	ttl := 90
	tc := testConfig{
		projects: map[string][]string{
			"galaxy": {"milkyway"},
		},
		buildspecs: map[string]bool{
			"full/buildspecs/93/13811.0.0.xml": true,
		},
		branches: []string{"refs/heads/release-R93-13816.B"},
		// Test --ttl feature.
		expectedSetTTL: map[string]time.Duration{
			"gs://chromeos-galaxy/buildspecs/full/buildspecs/93/13811.0.0.xml":          time.Duration(ttl * 24 * int(time.Hour)),
			"gs://chromeos-galaxy-milkyway/buildspecs/full/buildspecs/93/13811.0.0.xml": time.Duration(ttl * 24 * int(time.Hour)),
		},
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		buildspec:                  "full/buildspecs/93/13811.0.0.xml",
		projects:                   []string{"galaxy/milkyway"},
		ttl:                        ttl,
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}
func TestCreateProjectBuildspecDryRun(t *testing.T) {
	t.Parallel()
	tc := testConfig{
		projects: map[string][]string{
			"galaxy": {"milkyway"},
		},
		buildspecs: map[string]bool{
			"full/buildspecs/93/13811.0.0.xml": true,
		},
		branches: []string{"refs/heads/release-R93-13816.B"},
		dryRun:   true,
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		buildspec:                  "full/buildspecs/93/13811.0.0.xml",
		projects:                   []string{"galaxy/milkyway"},
		push:                       false,
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}

// Specifically test 96 to check that the tool properly accounts for the
// missing 95.
func TestCreateProjectBuildspecToT(t *testing.T) {
	t.Parallel()
	tc := testConfig{
		projects: map[string][]string{
			"galaxy": {"milkyway"},
		},
		buildspecs: map[string]bool{
			"full/buildspecs/96/13811.0.0-rc2.xml": true,
		},
		branches: []string{"refs/heads/main"},
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		buildspec:                  "full/buildspecs/96/13811.0.0-rc2.xml",
		projects:                   []string{"galaxy/milkyway"},
		push:                       true,
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}

func TestCreateProjectBuildspecForce(t *testing.T) {
	t.Parallel()
	tc := testConfig{
		projects: map[string][]string{
			"galaxy": {"milkyway"},
		},
		buildspecs: map[string]bool{
			"full/buildspecs/93/13811.0.0.xml": true,
		},
		branches:         []string{"refs/heads/release-R93-13816.B"},
		buildspecsExists: true,
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		buildspec:                  "full/buildspecs/93/13811.0.0.xml",
		projects:                   []string{"galaxy/milkyway"},
		force:                      true,
		push:                       true,
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}
func TestCreateProjectBuildspecExistsNoForce(t *testing.T) {
	t.Parallel()
	// File shouldn't be written to GS if force is not set.
	tc := testConfig{
		projects: map[string][]string{
			"galaxy": {"milkyway"},
		},
		buildspecs: map[string]bool{
			"full/buildspecs/93/13811.0.0.xml": false,
		},
		branches:         []string{"refs/heads/release-R93-13816.B"},
		buildspecsExists: true,
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		buildspec:                  "full/buildspecs/93/13811.0.0.xml",
		projects:                   []string{"galaxy/milkyway"},
		force:                      false,
		push:                       true,
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}

func TestCreateProjectBuildspecMultiple(t *testing.T) {
	t.Parallel()
	watchPaths := map[string][]string{
		"full/buildspecs/": {
			"full/buildspecs/93/",
			"full/buildspecs/94/13010.0.0-rc1.xml",
			"full/buildspecs/94/13011.0.0-rc1.xml",
		},
		"buildspecs/": {
			"full/buildspecs/94/13010.0.0.xml",
			"full/buildspecs/94/13011.0.0.xml",
		},
	}

	tc := testConfig{
		projects: map[string][]string{
			"galaxy": {"milkyway"},
		},
		buildspecs: map[string]bool{
			"full/buildspecs/94/13010.0.0-rc1.xml": true,
			"full/buildspecs/94/13011.0.0-rc1.xml": true,
			"buildspecs/94/13010.0.0.xml":          true,
			"buildspecs/94/13011.0.0.xml":          true,
		},
		watchPaths: watchPaths,
		branches:   []string{"refs/heads/release-R94-13904.B"},
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		watchPaths:                 []string{"full/buildspecs/", "buildspecs/"},
		minMilestone:               94,
		projects:                   []string{"galaxy/milkyway"},
		push:                       true,
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}

func TestCreateProjectBuildspecMultipleProgram(t *testing.T) {
	t.Parallel()
	watchPaths := map[string][]string{
		"full/buildspecs/": {
			"full/buildspecs/93/",
			"full/buildspecs/94/13010.0.0-rc1.xml",
			"full/buildspecs/94/13011.0.0-rc1.xml",
		},
		"buildspecs/": {
			"full/buildspecs/94/13010.0.0.xml",
			"full/buildspecs/94/13011.0.0.xml",
		},
	}

	tc := testConfig{
		projects: map[string][]string{
			"galaxy": {"milkyway", "andromeda"},
		},
		buildspecs: map[string]bool{
			"full/buildspecs/94/13010.0.0-rc1.xml": true,
			"full/buildspecs/94/13011.0.0-rc1.xml": true,
			"buildspecs/94/13010.0.0.xml":          true,
			"buildspecs/94/13011.0.0.xml":          true,
		},
		watchPaths: watchPaths,
		branches:   []string{"refs/heads/release-R94-13904.B"},
		allProjects: []string{
			"chromeos/project/galaxy/milkyway",
			"chromeos/project/galaxy/andromeda",
			"chromeos/project/galaxy/missing",
			"chromeos/foo",
		},
		// Test that a project missing a local manifest file does not doom
		// the overall run, if wildcards are in use.
		noLocalManifestProjects: []string{
			"chromeos/project/galaxy/missing",
		},
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		watchPaths:                 []string{"full/buildspecs/", "buildspecs/"},
		minMilestone:               94,
		projects:                   []string{"galaxy/*"},
		push:                       true,
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}

func TestCreateProjectBuildspecOtherRepos(t *testing.T) {
	t.Parallel()
	tc := testConfig{
		projects: map[string][]string{
			"galaxy": {"milkyway"},
		},
		otherRepos: []string{"chromeos-vendor-qti-camx"},
		buildspecs: map[string]bool{
			"full/buildspecs/93/13811.0.0.xml": true,
		},
		branches: []string{"refs/heads/release-R93-13816.B"},
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		buildspec:                  "full/buildspecs/93/13811.0.0.xml",
		otherRepos:                 []string{"chromeos-vendor-qti-camx"},
		projects:                   []string{"galaxy/milkyway"},
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}
