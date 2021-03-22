// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.
// +build linux

package atutil

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"infra/cros/internal/assert"
)

func TestAddCrosVersionLabelToHostInfoFile(t *testing.T) {
	tmpDir := "hostinfo_tmp_dir"
	tmpDir, err := ioutil.TempDir("", tmpDir)
	defer os.RemoveAll(tmpDir)
	assert.NilError(t, err)
	tmpPath := filepath.Join(tmpDir, "host_info_pre.json")

	// We're modifying host_info_pre.json, so need to copy it to a tmp file.
	fileContents, err := ioutil.ReadFile("test_data/host_info_pre.json")
	assert.NilError(t, err)
	err = ioutil.WriteFile(tmpPath, fileContents, 0644)
	assert.NilError(t, err)

	assert.NilError(t, AddCrosVersionLabelToHostInfoFile(tmpPath, "foo"))

	// Check that file matches expectation.
	preData, err := ioutil.ReadFile(tmpPath)
	assert.NilError(t, err)
	postData, err := ioutil.ReadFile("test_data/host_info_post.json")
	assert.NilError(t, err)
	if strings.Compare(string(preData), string(postData)) != 0 {
		t.Fatalf("file mismatch.\ngot: %v\n\nexpected: %v", string(preData), string(postData))
	}
}
