// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package site

import (
	"os"
	"path/filepath"
	"testing"
)

// NOTE: not thread safe
func testSecretsDir(t *testing.T) {
	oldEnvVar := os.Getenv("XDG_CACHE_HOME")
	defer func() {
		os.Setenv("XDG_CACHE_HOME", oldEnvVar)
	}()
	os.Setenv("XDG_CACHE_HOME", "FAKE_DIRECTORY")
	if SecretsDir() != filepath.Join("FAKE_DIRECTORY", "stable_version2", "auth") {
		t.Errorf("invalid SecretsDir")
	}
}
