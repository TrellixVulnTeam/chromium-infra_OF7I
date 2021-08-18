// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build !windows

package main

import (
	"path/filepath"
	"testing"

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
	program string
	project string
	// Map between buildspec name and whether or not to expect a GS write.
	buildspecs       map[string]bool
	branches         []string
	buildspecsExists bool
	expectedForce    bool
	watchPaths       map[string]map[string][]string
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
	)

	// Mock tip-of-branch (branch) manifest file requests.
	projects := []string{
		"chromeos/program/" + tc.program,
		"chromeos/project/" + tc.program + "/" + tc.project,
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

	// Mock external buildspec file request.
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
		)

		// Mock buildspec file requests.
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
		)
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

	expectedLists := make(map[string][]string)
	expectedWrites := make(map[string][]byte)

	for buildspec, expectWrite := range tc.buildspecs {
		if expectWrite {
			expectedWrites["gs://chromeos-galaxy/buildspecs/"+buildspec] = []byte(pinnedLocalManifestXML)
			expectedWrites["gs://chromeos-galaxy-milkyway/buildspecs/"+buildspec] = []byte(pinnedLocalManifestXML)
		}
		list := []string{}
		if tc.buildspecsExists {
			list = []string{"buildspecs/" + buildspec}
		}
		expectedLists["buildspecs/"+buildspec] = list
		expectedLists["buildspecs/"+buildspec] = list
	}

	f := &gs.FakeClient{
		T:              t,
		ExpectedWrites: expectedWrites,
		ExpectedLists: map[string]map[string][]string{
			"chromeos-galaxy":          expectedLists,
			"chromeos-galaxy-milkyway": expectedLists,
		},
	}
	return f, gc
}

func TestCreateProjectBuildspec(t *testing.T) {
	t.Parallel()
	tc := testConfig{
		program: "galaxy",
		project: "milkyway",
		buildspecs: map[string]bool{
			"full/buildspecs/93/13811.0.0.xml": true,
		},
		branches: []string{"refs/heads/release-R93-13816.B"},
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		buildspec: "full/buildspecs/93/13811.0.0.xml",
		projects:  []string{tc.program + "/" + tc.project},
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}

// Specifically test 96 to check that the tool properly accounts for the
// missing 95.
func TestCreateProjectBuildspecToT(t *testing.T) {
	t.Parallel()
	tc := testConfig{
		program: "galaxy",
		buildspecs: map[string]bool{
			"full/buildspecs/96/13811.0.0-rc2.xml": true,
		},
		project:  "milkyway",
		branches: []string{"refs/heads/main"},
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		buildspec: "full/buildspecs/96/13811.0.0-rc2.xml",
		projects:  []string{tc.program + "/" + tc.project},
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}

func TestCreateProjectBuildspecForce(t *testing.T) {
	t.Parallel()
	tc := testConfig{
		program: "galaxy",
		project: "milkyway",
		buildspecs: map[string]bool{
			"full/buildspecs/93/13811.0.0.xml": true,
		},
		branches:         []string{"refs/heads/release-R93-13816.B"},
		buildspecsExists: true,
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		buildspec: "full/buildspecs/93/13811.0.0.xml",
		projects:  []string{tc.program + "/" + tc.project},
		force:     true,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}
func TestCreateProjectBuildspecExistsNoForce(t *testing.T) {
	t.Parallel()
	// File shouldn't be written to GS if force is not set.
	tc := testConfig{
		program: "galaxy",
		project: "milkyway",
		buildspecs: map[string]bool{
			"full/buildspecs/93/13811.0.0.xml": false,
		},
		branches:         []string{"refs/heads/release-R93-13816.B"},
		buildspecsExists: true,
	}
	f, gc := tc.setUpPPBTest(t)

	b := projectBuildspec{
		buildspec: "full/buildspecs/93/13811.0.0.xml",
		projects:  []string{tc.program + "/" + tc.project},
		force:     false,
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
		program: "galaxy",
		project: "milkyway",
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
		projects:     []string{tc.program + "/" + tc.project},
		force:        true,
	}
	assert.NilError(t, b.CreateBuildspecs(f, gc))
}
