// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/server/cfgcache"
	"go.chromium.org/luci/config/validation"
	"go.chromium.org/luci/server/router"

	cpb "infra/appengine/cr-audit-commits/app/proto"
)

// ctxKeyConfig is a string with unique address used to store and retrieve
// context values.
var ctxKeyConfig = "cr_audit_commits.config"

type contextState struct {
	config   *cpb.Config
	revision string
}

var cachedCfg = cfgcache.Register(&cfgcache.Entry{
	Path: "config.cfg",
	Type: (*cpb.Config)(nil),
	Validator: func(ctx *validation.Context, msg proto.Message) error {
		validateConfig(ctx, msg.(*cpb.Config))
		return nil
	},
})

// Update is called by a cron job, which fetches the config and puts it into
// the datastore.
func Update(c *router.Context) {
	ctx := c.Context
	if _, err := cachedCfg.Update(ctx, nil); err != nil {
		logging.WithError(err).Errorf(ctx, "Failed to update application config")
	}
}

// Middleware loads the service config and installs it into the context.
func Middleware(c *router.Context, next router.Handler) {
	var meta config.Meta
	cfg, err := cachedCfg.Get(c.Context, &meta)
	if err != nil {
		logging.WithError(err).Errorf(c.Context, "Could not load application config")
		// TODO: should http error after putting the config in LUCI-config
		return
	}
	c.Context = context.WithValue(c.Context, &ctxKeyConfig, &contextState{
		config:   cfg.(*cpb.Config),
		revision: meta.Revision,
	})
	next(c)
}

// Get returns the config stored in the context.
func Get(c context.Context) *cpb.Config {
	return c.Value(&ctxKeyConfig).(*contextState).config
}

// GetConfigRevision returns the revision of the current config.
func GetConfigRevision(c context.Context) string {
	return c.Value(&ctxKeyConfig).(*contextState).revision
}
