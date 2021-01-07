// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"path"
	"strconv"

	"go.chromium.org/luci/config/validation"
)

var validTimeUnits = map[string]bool{"s": true, "m": true, "h": true, "d": true}

func validateConfig(c *validation.Context, cfg *Config) {
	for key, hostConfig := range cfg.HostConfigs {
		c.Enter("host_config %s", key)
		validateHostConfig(c, hostConfig)
		c.Exit()
	}
}

func validateHostConfig(c *validation.Context, hostConfig *HostConfig) {
	for key, repoConfig := range hostConfig.RepoConfigs {
		c.Enter("repo_config %s", key)
		validateRepoConfig(c, repoConfig)
		c.Exit()
	}
}

func validateRepoConfig(c *validation.Context, repoConfig *RepoConfig) {
	if repoConfig.CleanRevertPattern != nil {
		c.Enter("clean_revert_pattern")
		validateCleanRevertPattern(c, repoConfig.CleanRevertPattern)
		c.Exit()
	}
}

func validateCleanRevertPattern(c *validation.Context, cleanRevertPattern *CleanRevertPattern) {
	tw := cleanRevertPattern.TimeWindow
	unit := tw[len(tw)-1:]
	_, err := strconv.Atoi(tw[:len(tw)-1])
	if err != nil || !validTimeUnits[unit] {
		c.Errorf("invalid time_window %s: %s", tw, err)
	}

	for _, p := range cleanRevertPattern.ExcludedPaths {
		// This two match statements validate that it's a valid-enough
		// path. They should not error when trying to match on it.
		if _, err := path.Match(p, "test"); err != nil {
			c.Errorf("invalid path %s: %s", p, err)
		}
		if _, err := path.Match(p, "src/"); err != nil {
			c.Errorf("invalid path %s: %s", p, err)
		}
	}
}
