// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"path"

	"go.chromium.org/luci/config/validation"
)

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
	if repoConfig.BenignFilePattern != nil {
		c.Enter("benign_file_pattern")
		validateBenignFilePattern(c, repoConfig.BenignFilePattern)
		c.Exit()
	}
}

func validateBenignFilePattern(c *validation.Context, benignFilePattern *BenignFilePattern) {
	m := benignFilePattern.FileExtensionMap
	for ext, paths := range m {
		if ext != path.Ext(ext) && ext != "*" {
			c.Errorf("invalid file extension %s", ext)
		}

		for _, p := range paths.Paths {
			// This two match statements validate that it's a valid-enough
			// path. They should not error when trying to match on it.
			if _, err := path.Match(p, "test"); err != nil {
				c.Errorf("invalid path %s: %s", p, err)
			}
			if _, err := path.Match(p, "src/"); err != nil {
				c.Errorf("invalid path %s: %s", p, err)
			}

			if pExt := path.Ext(p); ext != pExt && p[len(p)-1] != '/' && p[len(p)-1] != '*' {
				c.Errorf("the extension of path %s does not match the extension %s", p, ext)
			}
		}
	}
}
