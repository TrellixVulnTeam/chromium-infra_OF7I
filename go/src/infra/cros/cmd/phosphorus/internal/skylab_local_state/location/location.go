// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package location provides utils for manipulating local file paths and
// URLs.
package location

import (
	"path/filepath"
)

// ResultsDir constructs the path to the task results dir.
// A swarming task may have multiple attempts ("runs").
// The swarming task ID always ends in "0", e.g. "123456789abcdef0".
// The corresponding runs will have IDs ending in "1", "2", etc., e.g.
// "123456789abcdef1".
// All runs are nested under the same subdir.
func ResultsDir(autotestDir string, runID string, testID string) string {
	return filepath.Join(autotestDir, "results", resultsSubDir(runID), runID[len(runID)-1:], testID)
}

func resultsSubDir(runID string) string {
	taskID := runID[:len(runID)-1] + "0"
	return "swarming-" + taskID
}

const (
	hostInfoSubDir     = "host_info_store"
	hostInfoFileSuffix = ".store"
)

// HostInfoFilePath constructs the path to the autotest host info store.
func HostInfoFilePath(resultsDir string, dutName string) string {
	return filepath.Join(resultsDir, hostInfoSubDir, dutName+hostInfoFileSuffix)
}
