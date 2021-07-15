// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build linux

package main

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"infra/cros/internal/assert"
	"infra/cros/internal/gerrit"
	"infra/cros/internal/gs"

	"github.com/golang/mock/gomock"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/common/proto/gitiles/mock_gitiles"
)

func checkFiles(t *testing.T, path string, expected map[string]string) {
	for filename, expectedContents := range expected {
		data, err := ioutil.ReadFile(filepath.Join(path, filename))
		assert.NilError(t, err)
		assert.StringsEqual(t, string(data), expectedContents)
	}
	// Make sure there are no extraneous files.
	files, err := ioutil.ReadDir(path)
	assert.NilError(t, err)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		_, ok := expected[file.Name()]
		assert.Assert(t, ok)
	}
}

func TestSetupProject(t *testing.T) {
	// Mock Gitiles controller
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)

	branch := "mybranch"
	expectedFiles := map[string]string{
		"foo_program.xml": "chromeos/program/foo",
		"bar_project.xml": "chromeos/project/foo/bar",
		"baz_chipset.xml": "chromeos/overlays/chipset-baz-private",
	}

	for _, projectName := range expectedFiles {
		req := &gitilespb.DownloadFileRequest{
			Project:    projectName,
			Path:       "local_manifest.xml",
			Committish: branch,
		}
		gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(req)).Return(
			&gitilespb.DownloadFileResponse{
				Contents: projectName,
			},
			nil,
		)
	}

	gerrit.MockGitiles = gitilesMock

	dir, err := ioutil.TempDir("", "setup_project")
	defer os.RemoveAll(dir)
	assert.NilError(t, err)
	localManifestDir := filepath.Join(dir, ".repo/local_manifests/")
	assert.NilError(t, os.MkdirAll(localManifestDir, os.ModePerm))

	b := setupProject{
		chromeosCheckoutPath: dir,
		program:              "foo",
		localManifestBranch:  branch,
		project:              "bar",
		chipset:              "baz",
	}
	ctx := context.Background()
	assert.NilError(t, b.setupProject(ctx, nil, nil))
	checkFiles(t, localManifestDir, expectedFiles)
}

func TestSetupProject_allProjects(t *testing.T) {
	// Mock Gitiles controller
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)

	gitilesMock.EXPECT().Projects(gomock.Any(), gomock.Any()).Return(
		&gitilespb.ProjectsResponse{
			Projects: []string{
				"chromeos/project/foo/bar1",
				"chromeos/project/foo/bar2",
			},
		},
		nil,
	)

	branch := "mybranch"
	expectedFiles := map[string]string{
		"foo_program.xml":  "chromeos/program/foo",
		"bar1_project.xml": "chromeos/project/foo/bar1",
		"bar2_project.xml": "chromeos/project/foo/bar2",
		"baz_chipset.xml":  "chromeos/overlays/chipset-baz-private",
	}

	for _, projectName := range expectedFiles {
		req := &gitilespb.DownloadFileRequest{
			Project:    projectName,
			Path:       "local_manifest.xml",
			Committish: branch,
		}
		gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(req)).Return(
			&gitilespb.DownloadFileResponse{
				Contents: projectName,
			},
			nil,
		)
	}

	gerrit.MockGitiles = gitilesMock

	dir, err := ioutil.TempDir("", "setup_project")
	defer os.RemoveAll(dir)
	assert.NilError(t, err)
	localManifestDir := filepath.Join(dir, ".repo/local_manifests/")
	assert.NilError(t, os.MkdirAll(localManifestDir, os.ModePerm))

	b := setupProject{
		chromeosCheckoutPath: dir,
		program:              "foo",
		localManifestBranch:  branch,
		allProjects:          true,
		project:              "bar",
		chipset:              "baz",
	}
	ctx := context.Background()
	assert.NilError(t, b.setupProject(ctx, nil, nil))
	checkFiles(t, localManifestDir, expectedFiles)
}

func TestSetupProject_buildspecs(t *testing.T) {
	buildspec := "90/13811.0.0.xml"

	// Mock Gitiles controller
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)

	gitilesMock.EXPECT().Projects(gomock.Any(), gomock.Any()).Return(
		&gitilespb.ProjectsResponse{
			Projects: []string{
				"chromeos/project/foo/bar1",
				"chromeos/project/foo/bar2",
			},
		},
		nil,
	)

	gsSuffix := "/buildspecs/" + buildspec
	expectedDownloads := map[string][]byte{
		"gs://chromeos-foo" + gsSuffix:      []byte("chromeos/program/foo"),
		"gs://chromeos-foo-bar1" + gsSuffix: []byte("chromeos/project/foo/bar1"),
		"gs://chromeos-foo-bar2" + gsSuffix: []byte("chromeos/project/foo/bar2"),
	}

	f := &gs.FakeClient{
		T:                 t,
		ExpectedDownloads: expectedDownloads,
	}

	gerrit.MockGitiles = gitilesMock

	dir, err := ioutil.TempDir("", "setup_project")
	defer os.RemoveAll(dir)
	assert.NilError(t, err)
	localManifestDir := filepath.Join(dir, ".repo/local_manifests/")
	assert.NilError(t, os.MkdirAll(localManifestDir, os.ModePerm))

	b := setupProject{
		chromeosCheckoutPath: dir,
		program:              "foo",
		allProjects:          true,
		project:              "bar",
		buildspec:            buildspec,
	}
	ctx := context.Background()
	assert.NilError(t, b.setupProject(ctx, nil, f))
	expectedFiles := map[string]string{
		"foo_program.xml":  "chromeos/program/foo",
		"bar1_project.xml": "chromeos/project/foo/bar1",
		"bar2_project.xml": "chromeos/project/foo/bar2",
	}
	checkFiles(t, localManifestDir, expectedFiles)
}
