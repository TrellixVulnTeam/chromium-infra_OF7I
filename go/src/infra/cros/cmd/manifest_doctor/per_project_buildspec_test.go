// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build !windows

package main

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"infra/cros/internal/assert"
	gerrit "infra/cros/internal/gerrit"
	"infra/cros/internal/gs"

	"github.com/golang/mock/gomock"
	gitpb "go.chromium.org/luci/common/proto/git"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/common/proto/gitiles/mock_gitiles"
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
	projects map[string][]string
	// Map between buildspec name and whether or not to expect a GS write.
	buildspecs       map[string]bool
	branches         []string
	buildspecsExists bool
	expectedForce    bool
	watchPaths       map[string]map[string][]string
	allProjects      []string
	expectedSetTTL   map[string]time.Duration
	dryRun           bool
}

func namesToFiles(files []string) []*gitpb.File {
	res := make([]*gitpb.File, len(files))
	for i, file := range files {
		res[i] = &gitpb.File{
			Path: file,
		}
	}
	return res
}

func (tc *testConfig) setUpPPBTest(t *testing.T) (*gs.FakeClient, *gerrit.Client) {
	t.Helper()
	// Mock Gitiles controller
	ctl := gomock.NewController(t)
	t.Cleanup(ctl.Finish)
	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)

	// Mock Projects request.
	if tc.allProjects != nil {
		gitilesMock.EXPECT().Projects(gomock.Any(), gomock.Any()).Return(
			&gitilespb.ProjectsResponse{
				Projects: tc.allProjects,
			},
			nil,
		)
	}

	// Mock manifest-internal branches request.
	request := &gitilespb.RefsRequest{
		Project:  "chromeos/manifest-internal",
		RefsPath: "refs/heads",
	}
	response := make(map[string]string)
	response["refs/heads/main"] = "deadcafe"
	response["refs/heads/release-R93-13816.B"] = "deadbeef"
	response["refs/heads/release-R94-13904.B"] = "beefcafe"
	gitilesMock.EXPECT().Refs(gomock.Any(), gerrit.RefsRequestEq(request)).Return(
		&gitilespb.RefsResponse{
			Revisions: response,
		},
		nil,
	).AnyTimes()

	// Mock tip-of-branch (branch) manifest file requests.
	for prog, projs := range tc.projects {
		for _, proj := range projs {
			projects := []string{
				"chromeos/program/" + prog,
				"chromeos/project/" + prog + "/" + proj,
			}
			for _, project := range projects {
				for _, branch := range tc.branches {
					reqLocalManifest := &gitilespb.DownloadFileRequest{
						Project:    project,
						Path:       "local_manifest.xml",
						Committish: branch,
					}
					gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(reqLocalManifest)).Return(
						&gitilespb.DownloadFileResponse{
							Contents: unpinnedLocalManifestXML,
						},
						nil,
					)
				}
			}
		}
	}

	// Mock external and internal buildspec file requests.
	for buildspec := range tc.buildspecs {
		reqExternalBuildspec := &gitilespb.DownloadFileRequest{
			Project:    "chromiumos/manifest-versions",
			Path:       buildspec,
			Committish: "HEAD",
		}
		gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(reqExternalBuildspec)).Return(
			&gitilespb.DownloadFileResponse{
				Contents: "",
			},
			nil,
		).AnyTimes()

		reqBuildspecs := &gitilespb.DownloadFileRequest{
			Project:    "chromeos/manifest-versions",
			Path:       buildspec,
			Committish: "HEAD",
		}
		gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(reqBuildspecs)).Return(
			&gitilespb.DownloadFileResponse{
				Contents: buildspecXML,
			},
			nil,
		).AnyTimes()
	}

	// Set up gerrit.List expectations.
	if tc.watchPaths != nil {
		for watchPath, subpaths := range tc.watchPaths {
			subdirs := []string{}
			for subpath, files := range subpaths {
				subdirs = append(subdirs, subpath)
				if files == nil {
					continue
				}
				reqList := &gitilespb.ListFilesRequest{
					Project:    "chromiumos/manifest-versions",
					Path:       filepath.Join(watchPath, subpath),
					Committish: "HEAD",
				}
				gitilesMock.EXPECT().ListFiles(gomock.Any(), gerrit.ListFilesRequestEq(reqList)).Return(
					&gitilespb.ListFilesResponse{
						Files: namesToFiles(files),
					},
					nil,
				)
			}
			reqList := &gitilespb.ListFilesRequest{
				Project:    "chromiumos/manifest-versions",
				Path:       watchPath,
				Committish: "HEAD",
			}
			gitilesMock.EXPECT().ListFiles(gomock.Any(), gerrit.ListFilesRequestEq(reqList)).Return(
				&gitilespb.ListFilesResponse{
					Files: namesToFiles(subdirs),
				},
				nil,
			)
		}
	}

	mockMap := map[string]gitilespb.GitilesClient{
		chromeInternalHost: gitilesMock,
		chromeExternalHost: gitilesMock,
	}
	gc := gerrit.NewTestClient(mockMap)

	expectedWrites := make(map[string][]byte)
	expectedBucketLists := make(map[string][]string)

	for buildspec, expectWrite := range tc.buildspecs {
		relpath := fmt.Sprintf("buildspecs/%s", buildspec)
		if expectWrite {
			for prog, projs := range tc.projects {
				for _, proj := range projs {
					program_bucket := fmt.Sprintf("gs://chromeos-%s/", prog)
					project_bucket := fmt.Sprintf("gs://chromeos-%s-%s/", prog, proj)
					expectedWrites[program_bucket+relpath] = []byte(pinnedLocalManifestXML)
					expectedWrites[project_bucket+relpath] = []byte(pinnedLocalManifestXML)
				}
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
	for prog, projs := range tc.projects {
		for _, proj := range projs {
			expectedLists[fmt.Sprintf("chromeos-%s", prog)] = expectedBucketLists
			expectedLists[fmt.Sprintf("chromeos-%s-%s", prog, proj)] = expectedBucketLists
		}
	}
	if tc.dryRun {
		expectedWrites = make(map[string][]byte)
	}

	f := &gs.FakeClient{
		T:              t,
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
		buildspec: "full/buildspecs/93/13811.0.0.xml",
		projects:  []string{"galaxy/milkyway"},
		ttl:       ttl,
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
		buildspec: "full/buildspecs/93/13811.0.0.xml",
		projects:  []string{"galaxy/milkyway"},
		push:      false,
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
		buildspec: "full/buildspecs/96/13811.0.0-rc2.xml",
		projects:  []string{"galaxy/milkyway"},
		push:      true,
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
		buildspec: "full/buildspecs/93/13811.0.0.xml",
		projects:  []string{"galaxy/milkyway"},
		force:     true,
		push:      true,
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
		buildspec: "full/buildspecs/93/13811.0.0.xml",
		projects:  []string{"galaxy/milkyway"},
		force:     false,
		push:      true,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}

func TestCreateProjectBuildspecMultiple(t *testing.T) {
	t.Parallel()
	watchPaths := map[string]map[string][]string{
		"full/buildspecs/": {
			"93": nil,
			"94": {
				"13010.0.0-rc1.xml",
				"13011.0.0-rc1.xml",
			},
		},
		"buildspecs/": {
			"93": nil,
			"94": {
				"13010.0.0.xml",
				"13011.0.0.xml",
			},
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
		watchPaths:   []string{"full/buildspecs/", "buildspecs/"},
		minMilestone: 94,
		projects:     []string{"galaxy/milkyway"},
		push:         true,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}

func TestCreateProjectBuildspecMultipleProgram(t *testing.T) {
	t.Parallel()
	watchPaths := map[string]map[string][]string{
		"full/buildspecs/": {
			"93": nil,
			"94": {
				"13010.0.0-rc1.xml",
				"13011.0.0-rc1.xml",
			},
		},
		"buildspecs/": {
			"93": nil,
			"94": {
				"13010.0.0.xml",
				"13011.0.0.xml",
			},
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
			"chromeos/foo",
		},
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		watchPaths:   []string{"full/buildspecs/", "buildspecs/"},
		minMilestone: 94,
		projects:     []string{"galaxy/*"},
		push:         true,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}
