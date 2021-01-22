// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
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

	if cfg.DefaultTimeWindow != "" {
		validateTimeWindow(c, cfg.DefaultTimeWindow)
	} else {
		c.Errorf("empty default_time_window")
	}
}

func validateHostConfig(c *validation.Context, hostConfig *HostConfig) {
	for key, repoConfig := range hostConfig.RepoConfigs {
		c.Enter("repo_config %s", key)
		validateRepoConfig(c, repoConfig)
		c.Exit()
	}

	if hostConfig.CleanRevertTimeWindow != "" {
		validateTimeWindow(c, hostConfig.CleanRevertTimeWindow)
	}
}

func validateRepoConfig(c *validation.Context, repoConfig *RepoConfig) {
	if repoConfig.CleanRevertPattern != nil {
		c.Enter("clean_revert_pattern")
		validateCleanRevertPattern(c, repoConfig.CleanRevertPattern)
		c.Exit()
	}
	if repoConfig.CleanCherryPickPattern != nil {
		c.Enter("clean_cherry_pick_pattern")
		validateCleanCherryPickPattern(c, repoConfig.CleanCherryPickPattern)
		c.Exit()
	}
}

func validateCleanRevertPattern(c *validation.Context, cleanRevertPattern *CleanRevertPattern) {
	tw := cleanRevertPattern.TimeWindow
	if tw != "" {
		validateTimeWindow(c, tw)
	}
}

func validateCleanCherryPickPattern(c *validation.Context, cleanCherryPickPattern *CleanCherryPickPattern) {
	tw := cleanCherryPickPattern.TimeWindow
	if tw != "" {
		validateTimeWindow(c, tw)
	}
}

func validateTimeWindow(c *validation.Context, tw string) {
	unit := tw[len(tw)-1:]
	_, err := strconv.Atoi(tw[:len(tw)-1])
	if err != nil || !validTimeUnits[unit] {
		c.Errorf("invalid time_window %s: %s", tw, err)
	}
}
