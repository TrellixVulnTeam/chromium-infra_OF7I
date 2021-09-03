// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package config implements app-level configs for Weetbix.
package config

import (
	"context"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/server/cfgcache"
	"go.chromium.org/luci/config/validation"
)

// Cached service config.
var cachedCfg = cfgcache.Register(&cfgcache.Entry{
	Path: "config.cfg",
	Type: (*Config)(nil),
	Validator: func(ctx *validation.Context, msg proto.Message) error {
		validateConfig(ctx, msg.(*Config))
		return nil
	},
})

// Update fetches the config and puts it into the datastore.
func Update(ctx context.Context) error {
	_, err := cachedCfg.Update(ctx, nil)
	return err
}

// Get returns the config stored in the cachedCfg.
func Get(ctx context.Context) (*Config, error) {
	cfg, err := cachedCfg.Get(ctx, nil)
	return cfg.(*Config), err
}

// SetTestConfig set test configs in the cachedCfg.
func SetTestConfig(ctx context.Context, cfg *Config) error {
	return cachedCfg.Set(ctx, cfg, &config.Meta{})
}
