// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"net/url"

	"go.chromium.org/luci/config/validation"
)

func validateConfig(c *validation.Context, cfg *Config) {
	validateMonorailHostname(c, cfg.MonorailHostname)
}

func validateMonorailHostname(c *validation.Context, hostname string) {
	c.Enter("monorail_hostname")
	if hostname == "" {
		c.Errorf("empty value is not allowed")
	} else if _, err := url.Parse(hostname); err != nil {
		c.Errorf("invalid hostname: %s", hostname)
	}
	c.Exit()
}
