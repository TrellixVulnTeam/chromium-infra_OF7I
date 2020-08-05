// Copyright 2020 The Chromium Authors. All Rights Reserved.
// Use of this source code is governed by the Apache v2.0 license that can be
// found in the LICENSE file.

// Package config implements interface for app-level configs for cr-rev.
package config

import (
	"context"

	"go.chromium.org/luci/config/server/cfgcache"
)

// Cached service config.
var cachedCfg = cfgcache.Register(&cfgcache.Entry{
	Path: "cr-rev.cfg",
	Type: (*Config)(nil),
})

// Set fetches the config and puts it into the datastore.
func Set(ctx context.Context) error {
	_, err := cachedCfg.Update(ctx, nil)
	return err
}

// Get returns the config stored in the context.
func Get(ctx context.Context) (*Config, error) {
	cfg, err := cachedCfg.Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	return cfg.(*Config), nil

}
