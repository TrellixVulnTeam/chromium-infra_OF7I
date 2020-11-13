// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"

	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/server/cfgcache"
	"go.chromium.org/luci/config/validation"
	"google.golang.org/protobuf/proto"
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
func Update(c context.Context) error {
	_, err := cachedCfg.Update(c, nil)
	return err
}

// Get returns the config stored in the cachedCfg.
func Get(c context.Context) (*Config, error) {
	cfg, err := cachedCfg.Get(c, nil)
	return cfg.(*Config), err
}

// SetTestConfig set test configs in the cachedCfg.
func SetTestConfig(ctx context.Context, cfg *Config) error {
	return cachedCfg.Set(ctx, cfg, &config.Meta{})
}
