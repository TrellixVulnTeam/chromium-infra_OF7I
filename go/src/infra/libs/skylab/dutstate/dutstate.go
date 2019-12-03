// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package dutstate provides utils related to the DUT state cache file
// and the autotest host info file.
package dutstate

import (
	"path/filepath"

	"go.chromium.org/luci/common/data/stringset"
)

const (
	hostInfoSubDir     = "host_info_store"
	hostInfoFileSuffix = ".store"
	dutStateSubDir     = "swarming_state"
	dutStateFileSuffix = ".json"
)

// ResultsDir constructs the path to the task results dir.
// A swarming task may have multiple attempts ("runs").
// The swarming task ID always ends in "0", e.g. "123456789abcdef0".
// The corresponding runs will have IDs ending in "1", "2", etc., e.g.
// "123456789abcdef1".
// All runs are nested under the same subdir.
func ResultsDir(autotestDir string, runID string) string {
	taskID := runID[:len(runID)-1] + "0"
	subDir := "swarming-" + taskID
	return filepath.Join(autotestDir, "results", subDir, runID[len(runID)-1:])
}

// HostInfoFilePath constructs the path to the autotest host info store.
func HostInfoFilePath(resultsDir string, dutName string) string {
	return filepath.Join(resultsDir, hostInfoSubDir, dutName+hostInfoFileSuffix)
}

// CacheFilePath constructs the path to the state cache file.
func CacheFilePath(autotestDir string, dutID string) string {
	return filepath.Join(autotestDir, dutStateSubDir, dutID+dutStateFileSuffix)
}

// ProvisionableLabelSet provides the whitelist of labels that may change
// during provision. Only these labels can appear in the DUT state file.
func ProvisionableLabelSet() stringset.Set {
	return stringset.NewFromSlice("cros-version", "fwro-version", "fwrw-version")
}

// ProvisionableAttributeSet provides the whitelist of attributest that may
// change during provision. Only these labels can appear in the DUT state file.
func ProvisionableAttributeSet() stringset.Set {
	return stringset.NewFromSlice("job_repo_url", "outlet_changed")
}
