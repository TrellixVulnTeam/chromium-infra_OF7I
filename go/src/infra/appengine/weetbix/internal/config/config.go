// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package config implements app-level configs for Weetbix.
package config

import (
	"context"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/errors"
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

// Update fetches the latest config and puts it into the datastore.
func Update(ctx context.Context) error {
	var errs []error
	if _, err := cachedCfg.Update(ctx, nil); err != nil {
		errs = append(errs, err)
	}
	if err := updateProjects(ctx); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errors.NewMultiError(errs...)
	}
	return nil
}

// Get returns the service-level config.
func Get(ctx context.Context) (*Config, error) {
	cfg, err := cachedCfg.Get(ctx, nil)
	return cfg.(*Config), err
}

// SetTestConfig set test configs in the cachedCfg.
func SetTestConfig(ctx context.Context, cfg *Config) error {
	return cachedCfg.Set(ctx, cfg, &config.Meta{})
}
