// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
//go:build !windows
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

	internalManifestXMLDumped = `<?xml version="1.0" encoding="UTF-8"?>
<manifest>
  <remote fetch="https://chromium.googlesource.com" name="cros">
    <annotation name="public" value="true"></annotation>
  </remote>
  <remote fetch="https://chrome-internal.googlesource.com" name="cros-internal">
    <annotation name="public" value="false"></annotation>
  </remote>
  <default remote="cros" revision="refs/heads/main" sync-j="8"></default>
  <project path="foo/" name="foo" revision="123" remote="cros-internal"></project>
  <project path="bar/" name="bar" revision="456" remote="cros"></project>
  <project path="baz/" name="baz" revision="789"></project>
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
		"chromeos-manifest-versions": {
			"test/": {"test/foo.xml"},
		},
		"chromiumos-manifest-versions": {
			"test/": {},
		},
	}
	expectedReads := map[string][]byte{
		"gs://chromeos-manifest-versions/test/foo.xml": []byte(internalManifestXML),
	}
	expectedWrites := map[string][]byte{
		"gs://chromiumos-manifest-versions/test/foo.xml": []byte(externalManifestXML),
	}
	f := &gs.FakeClient{
		T:              t,
		ExpectedLists:  expectedLists,
		ExpectedReads:  expectedReads,
		ExpectedWrites: expectedWrites,
	}
	b := publicBuildspec{
		push:                       true,
		watchPaths:                 []string{"test/"},
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f, nil))
}

func TestPublicBuildspecDryRun(t *testing.T) {
	t.Parallel()
	expectedLists := map[string]map[string][]string{
		"chromeos-manifest-versions": {
			"test/": {"test/foo.xml"},
		},
		"chromiumos-manifest-versions": {
			"test/": {},
		},
	}
	expectedReads := map[string][]byte{
		"gs://chromeos-manifest-versions/test/foo.xml": []byte(internalManifestXML),
	}
	f := &gs.FakeClient{
		T:              t,
		ExpectedLists:  expectedLists,
		ExpectedReads:  expectedReads,
		ExpectedWrites: map[string][]byte{},
	}
	b := publicBuildspec{
		push:                       false,
		watchPaths:                 []string{"test/"},
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f, nil))
}

func TestPublicBuildspecNoAnnotations(t *testing.T) {
	t.Parallel()
	expectedLists := map[string]map[string][]string{
		"chromeos-manifest-versions": {
			"test/": {"test/foo.xml"},
		},
		"chromiumos-manifest-versions": {
			"test/": {},
		},
	}
	expectedReads := map[string][]byte{
		"gs://chromeos-manifest-versions/test/foo.xml": []byte(internalManifestXMLNoAnnotations),
	}
	expectedWrites := map[string][]byte{
		"gs://chromiumos-manifest-versions/test/foo.xml": []byte(externalManifestXML),
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
		push:                       true,
		watchPaths:                 []string{"test/"},
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f, gc))

}

func TestPublicBuildspecNoAnnotations_missingAtToT(t *testing.T) {
	t.Parallel()
	expectedLists := map[string]map[string][]string{
		"chromeos-manifest-versions": {
			"test/": {"test/foo.xml"},
		},
		"chromiumos-manifest-versions": {
			"test/": {},
		},
	}
	expectedReads := map[string][]byte{
		"gs://chromeos-manifest-versions/test/foo.xml": []byte(internalManifestXMLNoAnnotations),
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
		push:                       true,
		watchPaths:                 []string{"test/"},
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.ErrorContains(t, b.CreatePublicBuildspecs(context.Background(), f, gc), "could not get public status")
}

func legacyTest(t *testing.T, externalList []string, externalDownloads map[string]string,
	expectedGSWrites map[string][]byte) (*gs.FakeClient, gerrit.Client) {
	t.Helper()

	contents := internalManifestXML
	expectedDownloads := map[gerrit.ExpectedPathParams]*string{
		{
			Host:    chromeInternalHost,
			Project: "chromeos/manifest-versions",
			Path:    "test/foo.xml",
			Ref:     "HEAD",
		}: &contents,
	}
	if externalDownloads != nil {
		for path, contents := range externalDownloads {
			expectedDownloads[gerrit.ExpectedPathParams{
				Host:    chromeExternalHost,
				Project: "chromiumos/manifest-versions",
				Path:    path,
				Ref:     "HEAD",
			}] = &contents
		}
	}

	if externalList == nil {
		externalList = []string{}
	}
	gc := &gerrit.MockClient{
		T: t,
		ExpectedLists: map[gerrit.ExpectedPathParams][]string{
			{
				Host:    chromeInternalHost,
				Project: "chromeos/manifest-versions",
				Path:    "test/",
				Ref:     "HEAD",
			}: {"foo.xml"},
			{
				Host:    chromeExternalHost,
				Project: "chromiumos/manifest-versions",
				Path:    "test/",
				Ref:     "HEAD",
			}: externalList,
		},
		ExpectedDownloads: expectedDownloads,
	}

	expectedLists := map[string]map[string][]string{
		"chromiumos-manifest-versions": {
			"test/": {},
		},
	}
	expectedWrites := map[string][]byte{
		"gs://chromeos-manifest-versions/test/foo.xml":   []byte(internalManifestXMLDumped),
		"gs://chromiumos-manifest-versions/test/foo.xml": []byte(externalManifestXML),
	}
	if expectedGSWrites != nil {
		expectedWrites = expectedGSWrites
	}
	f := &gs.FakeClient{
		T:              t,
		ExpectedLists:  expectedLists,
		ExpectedWrites: expectedWrites,
	}
	return f, gc
}

func TestPublicBuildspecLegacy(t *testing.T) {
	t.Parallel()
	f, gc := legacyTest(t, nil, nil, nil)
	b := publicBuildspec{
		push:                       true,
		watchPaths:                 []string{"test/"},
		readFromManifestVersions:   true,
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f, gc))
}

// Check that dry run doesn't upload anything.
func TestPublicBuildspecLegacy_DryRun(t *testing.T) {
	t.Parallel()
	expectedGSWrites := map[string][]byte{}
	f, gc := legacyTest(t, nil, nil, expectedGSWrites)
	b := publicBuildspec{
		push:                       false,
		watchPaths:                 []string{"test/"},
		readFromManifestVersions:   true,
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f, gc))
}

// When we're reading from legacy (git) and the public buildspec already
// exists in the public manifest-versions repository, we should use that
// file instead of generating a new one.
func TestPublicBuildspecLegacy_ExternalExists(t *testing.T) {
	t.Parallel()
	externalList := []string{"foo.xml"}
	expectedExternalDownloads := map[string]string{
		"test/foo.xml": "foo",
	}
	expectedGSWrites := map[string][]byte{
		"gs://chromeos-manifest-versions/test/foo.xml":   []byte(internalManifestXMLDumped),
		"gs://chromiumos-manifest-versions/test/foo.xml": []byte("foo"),
	}
	f, gc := legacyTest(t, externalList, expectedExternalDownloads, expectedGSWrites)
	b := publicBuildspec{
		push:                       true,
		watchPaths:                 []string{"test/"},
		readFromManifestVersions:   true,
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f, gc))
}

// Check that dry run doesn't upload anything.
func TestPublicBuildspecLegacy_ExternalExists_DryRun(t *testing.T) {
	t.Parallel()
	externalList := []string{"foo.xml"}
	expectedExternalDownloads := map[string]string{
		"test/foo.xml": "foo",
	}
	expectedGSWrites := map[string][]byte{}
	f, gc := legacyTest(t, externalList, expectedExternalDownloads, expectedGSWrites)
	b := publicBuildspec{
		push:                       false,
		watchPaths:                 []string{"test/"},
		readFromManifestVersions:   true,
		internalBuildspecsGSBucket: internalBuildspecsGSBucketDefault,
		externalBuildspecsGSBucket: externalBuildspecsGSBucketDefault,
	}
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f, gc))
}
