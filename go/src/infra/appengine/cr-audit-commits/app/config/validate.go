// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"go.chromium.org/luci/config/validation"

	cpb "infra/appengine/cr-audit-commits/app/proto"
)

func validateConfig(c *validation.Context, cfg *cpb.Config) {
	validateRefConfigs(c, cfg.RefConfigs)
}

func validateRefConfigs(c *validation.Context, RefConfigs map[string]*cpb.RefConfig) {
	// TODO: validate RefConfigs
}
