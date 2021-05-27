// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build !windows

package main

import (
	"reflect"
	"testing"

	"infra/cros/internal/assert"
	gerrit "infra/cros/internal/gerrit"
	"infra/cros/internal/repo"

	"github.com/golang/mock/gomock"
	lgs "go.chromium.org/luci/common/gcloud/gs"
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

	pinnedLocalManifestXML = `
<manifest>
	<remote fetch="https://chrome-internal.googlesource.com" name="cros-internal"></remote>
	<default></default>
	<project path="src/foo" name="foo" revision="revision-foo" remote="cros-internal"></project>
	<project path="src/bar" name="bar" remote="cros-internal"></project>
</manifest>
`
)

var (
	application = GetApplication(chromeinfra.DefaultAuthOptions())
)

type fakeGSClient struct {
	t              *testing.T
	expectedWrites map[string]*repo.Manifest
}

// WriteFileToGS writes the specified data to the specified gs path.
func (f *fakeGSClient) WriteFileToGS(gsPath lgs.Path, data []byte) error {
	expected, ok := f.expectedWrites[string(gsPath)]
	if !ok {
		f.t.Fatalf("unexpected write at %s", string(gsPath))
	}
	got, err := repo.ParseManifest(data)
	assert.NilError(f.t, err)
	if !reflect.DeepEqual(expected, got) {
		f.t.Fatalf("mismatch for write at %s: expected:\n%v\ngot:\n%v\n", string(gsPath), expected, got)
	}
	return nil
}

func TestCreateProjectBuildspec(t *testing.T) {
	project := "chromeos/project/galaxy/milkyway"
	buildspec := "90/13811.0.0.xml"
	releaseBranch := "refs/heads/release-R90-13816.B"

	// Mock Gitiles controller
	ctl := gomock.NewController(t)
	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)

	// Mock manifest-internal branches request.
	request := &gitilespb.RefsRequest{
		Project:  "chromeos/manifest-internal",
		RefsPath: "refs/heads",
	}
	response := make(map[string]string)
	response["refs/heads/main"] = "deadcafe"
	response["refs/heads/release-R90-13816.B"] = "deadbeef"
	response["refs/heads/release-R91-13904.B"] = "beefcafe"
	gitilesMock.EXPECT().Refs(gomock.Any(), gerrit.RefsRequestEq(request)).Return(
		&gitilespb.RefsResponse{
			Revisions: response,
		},
		nil,
	)

	// Mock tip-of-branch (releaseBranch) manifest file request.
	reqLocalManifest := &gitilespb.DownloadFileRequest{
		Project:    project,
		Path:       "local_manifest.xml",
		Committish: releaseBranch,
		Format:     gitilespb.DownloadFileRequest_TEXT,
	}
	gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(reqLocalManifest)).Return(
		&gitilespb.DownloadFileResponse{
			Contents: unpinnedLocalManifestXML,
		},
		nil,
	)

	// Mock external buildspec file request.
	reqExternalBuildspec := &gitilespb.DownloadFileRequest{
		Project:    "chromiumos/manifest-versions",
		Path:       "full/buildspecs/" + buildspec,
		Committish: "HEAD",
		Format:     gitilespb.DownloadFileRequest_TEXT,
	}
	gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(reqExternalBuildspec)).Return(
		&gitilespb.DownloadFileResponse{
			Contents: "",
		},
		nil,
	)

	// Mock buildspec file request.
	reqBuildspecs := &gitilespb.DownloadFileRequest{
		Project:    "chromeos/manifest-versions",
		Path:       "buildspecs/" + buildspec,
		Committish: "HEAD",
		Format:     gitilespb.DownloadFileRequest_TEXT,
	}
	gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(reqBuildspecs)).Return(
		&gitilespb.DownloadFileResponse{
			Contents: buildspecXML,
		},
		nil,
	)

	gerrit.MockGitiles = gitilesMock

	expected, err := repo.ParseManifest([]byte(pinnedLocalManifestXML))
	assert.NilError(t, err)
	expectedWrites := map[string]*repo.Manifest{
		"gs://chromeos-galaxy-milkyway/buildspecs/" + buildspec: expected,
	}
	f := &fakeGSClient{
		t:              t,
		expectedWrites: expectedWrites,
	}

	b := projectBuildspec{
		buildspec: buildspec,
		project:   project,
	}
	assert.NilError(t, b.CreateProjectBuildspec(nil, f))
}

func TestGsPath(t *testing.T) {
	got, err := gsPath("chromeos/project/galaxy/milkyway", "90/13811.0.0.xml")
	assert.NilError(t, err)
	assert.StringsEqual(t, got.Bucket(), "chromeos-galaxy-milkyway")
	assert.StringsEqual(t, got.Filename(), "buildspecs/90/13811.0.0.xml")
}
