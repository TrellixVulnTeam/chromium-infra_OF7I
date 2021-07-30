// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build !windows

package main

import (
	"context"
	"infra/cros/internal/assert"
	"infra/cros/internal/gerrit"
	"infra/cros/internal/gs"
	"testing"

	"github.com/golang/mock/gomock"
	gitilespb "go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/common/proto/gitiles/mock_gitiles"
)

const (
	internalManifestXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="cros" fetch="https://chromium.googlesource.com">
    <annotation name="public" value="true"/>
  </remote>
  <remote name="cros-internal" fetch="https://chrome-internal.googlesource.com">
    <annotation name="public" value="false"/>
  </remote>
  <default remote="cros" revision="refs/heads/main" sync-j="8"/>

  <project remote="cros-internal" name="foo" path="foo/" revision="123" />
  <project remote="cros" name="bar" path="bar/" revision="456" />
  <project name="baz" path="baz/" revision="789" />
</manifest>`

	internalManifestXMLNoAnnotations = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote name="cros" fetch="https://chromium.googlesource.com"/>
  <remote name="cros-internal" fetch="https://chrome-internal.googlesource.com"/>
  <default remote="cros" revision="refs/heads/main" sync-j="8"/>

  <project remote="cros-internal" name="foo" path="foo/" revision="123" />
  <project remote="cros" name="bar" path="bar/" revision="456" />
  <project name="baz" path="baz/" revision="789" />
</manifest>`

	externalManifestXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote fetch="https://chromium.googlesource.com" name="cros">
    <annotation name="public" value="true"></annotation>
  </remote>
  <default remote="cros" revision="refs/heads/main" sync-j="8"></default>
  <project path="bar/" name="bar" revision="456" remote="cros"></project>
  <project path="baz/" name="baz" revision="789"></project>
</manifest>`
)

func TestPublicBuildspec(t *testing.T) {
	t.Parallel()
	expectedLists := map[string]map[string][]string{
		"buildspecs-internal": {
			"test/": {"test/foo.xml"},
		},
		"buildspecs-external": {
			"test/": {},
		},
	}
	expectedReads := map[string][]byte{
		"gs://buildspecs-internal/test/foo.xml": []byte(internalManifestXML),
	}
	expectedWrites := map[string][]byte{
		"gs://buildspecs-external/test/foo.xml": []byte(externalManifestXML),
	}
	f := &gs.FakeClient{
		T:              t,
		ExpectedLists:  expectedLists,
		ExpectedReads:  expectedReads,
		ExpectedWrites: expectedWrites,
	}
	b := publicBuildspec{
		push:       true,
		watchPaths: []string{"test/"},
	}
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f, nil))
}

func TestPublicBuildspecDryRun(t *testing.T) {
	t.Parallel()
	expectedLists := map[string]map[string][]string{
		"buildspecs-internal": {
			"test/": {"test/foo.xml"},
		},
		"buildspecs-external": {
			"test/": {},
		},
	}
	expectedReads := map[string][]byte{
		"gs://buildspecs-internal/test/foo.xml": []byte(internalManifestXML),
	}
	f := &gs.FakeClient{
		T:              t,
		ExpectedLists:  expectedLists,
		ExpectedReads:  expectedReads,
		ExpectedWrites: map[string][]byte{},
	}
	b := publicBuildspec{
		push:       false,
		watchPaths: []string{"test/"},
	}
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f, nil))
}

func TestPublicBuildspecNoAnnotations(t *testing.T) {
	t.Parallel()
	expectedLists := map[string]map[string][]string{
		"buildspecs-internal": {
			"test/": {"test/foo.xml"},
		},
		"buildspecs-external": {
			"test/": {},
		},
	}
	expectedReads := map[string][]byte{
		"gs://buildspecs-internal/test/foo.xml": []byte(internalManifestXMLNoAnnotations),
	}
	expectedWrites := map[string][]byte{
		"gs://buildspecs-external/test/foo.xml": []byte(externalManifestXML),
	}
	f := &gs.FakeClient{
		T:              t,
		ExpectedLists:  expectedLists,
		ExpectedReads:  expectedReads,
		ExpectedWrites: expectedWrites,
	}

	// Mock Gitiles controller
	ctl := gomock.NewController(t)
	t.Cleanup(ctl.Finish)
	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)
	reqLocalManifest := &gitilespb.DownloadFileRequest{
		Project:    "chromeos/manifest-internal",
		Path:       "default.xml",
		Committish: "HEAD",
	}
	gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(reqLocalManifest)).Return(
		&gitilespb.DownloadFileResponse{
			Contents: internalManifestXML,
		},
		nil,
	)

	mockMap := map[string]gitilespb.GitilesClient{
		chromeInternalHost: gitilesMock,
	}
	gc := gerrit.NewTestClient(mockMap)

	b := publicBuildspec{
		push:       true,
		watchPaths: []string{"test/"},
	}
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f, gc))

}

func TestPublicBuildspecNoAnnotations_missingAtToT(t *testing.T) {
	t.Parallel()
	expectedLists := map[string]map[string][]string{
		"buildspecs-internal": {
			"test/": {"test/foo.xml"},
		},
		"buildspecs-external": {
			"test/": {},
		},
	}
	expectedReads := map[string][]byte{
		"gs://buildspecs-internal/test/foo.xml": []byte(internalManifestXMLNoAnnotations),
	}
	f := &gs.FakeClient{
		T:             t,
		ExpectedLists: expectedLists,
		ExpectedReads: expectedReads,
	}

	// Mock Gitiles controller
	ctl := gomock.NewController(t)
	t.Cleanup(ctl.Finish)
	gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)
	reqLocalManifest := &gitilespb.DownloadFileRequest{
		Project:    "chromeos/manifest-internal",
		Path:       "default.xml",
		Committish: "HEAD",
	}
	gitilesMock.EXPECT().DownloadFile(gomock.Any(), gerrit.DownloadFileRequestEq(reqLocalManifest)).Return(
		&gitilespb.DownloadFileResponse{
			Contents: internalManifestXMLNoAnnotations,
		},
		nil,
	)

	mockMap := map[string]gitilespb.GitilesClient{
		chromeInternalHost: gitilesMock,
	}
	gc := gerrit.NewTestClient(mockMap)

	b := publicBuildspec{
		push:       true,
		watchPaths: []string{"test/"},
	}
	assert.ErrorContains(t, b.CreatePublicBuildspecs(context.Background(), f, gc), "could not get public status")
}
