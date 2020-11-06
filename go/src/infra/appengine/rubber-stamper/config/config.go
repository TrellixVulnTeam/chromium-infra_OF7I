// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"net/http"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/server/cfgcache"
	"go.chromium.org/luci/config/validation"
	"go.chromium.org/luci/server/router"
	"google.golang.org/protobuf/proto"
)

// A string with unique address used to store and retrieve config from context.
var ctxKeyConfig = "rubber-stamper.config"

// contextState is stored in the context under &ctxKeyConfig.
type contextState struct {
	config   *Config
	revision string
}

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

// Middleware loads the service config and installs it into the context.
func Middleware(c *router.Context, next router.Handler) {
	var meta config.Meta
	cfg, err := cachedCfg.Get(c.Context, &meta)
	if err != nil {
		logging.WithError(err).Errorf(c.Context, "could not load application config")
		http.Error(c.Writer, "Internal server error", http.StatusInternalServerError)
		return
	}
	c.Context = setConfig(c.Context, cfg.(*Config), meta.Revision)
	next(c)
}

// Get returns the config stored in the context.
func Get(c context.Context) *Config {
	return c.Value(&ctxKeyConfig).(*contextState).config
}

// GetConfigRevision returns the revision of the current config.
func GetConfigRevision(c context.Context) string {
	return c.Value(&ctxKeyConfig).(*contextState).revision
}

// setConfig installs cfg into c.
func setConfig(c context.Context, cfg *Config, rev string) context.Context {
	return context.WithValue(c, &ctxKeyConfig, &contextState{
		config:   cfg,
		revision: rev,
	})
}
