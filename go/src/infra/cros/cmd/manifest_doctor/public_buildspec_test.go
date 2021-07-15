// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build !windows

package main

import (
	"context"
	"infra/cros/internal/assert"
	"infra/cros/internal/gs"
	"testing"
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
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f))

}

func TestPublicBuildspecDryRun(t *testing.T) {
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
	assert.NilError(t, b.CreatePublicBuildspecs(context.Background(), f))

}
