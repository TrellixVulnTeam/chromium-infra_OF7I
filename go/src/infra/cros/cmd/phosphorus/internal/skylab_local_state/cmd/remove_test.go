// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveResultsParentDir(t *testing.T) {
	autotestDir := os.TempDir()
	resultsParentDir := filepath.Join(autotestDir, "results/swarming-fake_run_ID_0")
	defer os.RemoveAll(resultsParentDir)

	// Write test file in results parent directory
	err := os.MkdirAll(resultsParentDir, 0777)
	if err != nil {
		t.Fatalf("unexpected error calling os.Mkdir: %v", err)
	}
	err = os.WriteFile(filepath.Join(resultsParentDir, "foo.txt"), []byte("test data"), 0777)
	if err != nil {
		t.Fatalf("unexpected error calling os.WriteFile: %v", err)
	}
	matches, err := filepath.Glob(resultsParentDir + "/*")
	if err != nil {
		t.Fatalf("unexpected error calling filepath.Glob: %v", err)
	}
	if count := len(matches); count != 1 {
		t.Fatalf("unexpected error writing test file: expected 1 file in %s, found %d", resultsParentDir, count)
	}

	// Remove results parent directory
	err = removeResultsParentDir(autotestDir, "fake_run_ID_3")
	if err != nil {
		t.Fatalf("unexpected error calling removeResultsParentDir: %v", err)
	}

	// Confirm test file deleted
	matches, err = filepath.Glob(resultsParentDir + "/*")
	if err != nil {
		t.Fatalf("unexpected error calling filepath.Glob: %v", err)
	}
	if len(matches) > 0 {
		t.Errorf("expected all files in %s to be removed; found %v", resultsParentDir, matches)
	}
}
