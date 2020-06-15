// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"net/http"
	"net/url"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/appengine/gaesecrets"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config/server/cfgcache"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/secrets"
)

const configFile = "config.cfg"

// unique type to prevent assignment.
type contextKeyType struct{}

// unique key used to store and retrieve context.
var (
	contextKey        = contextKeyType{}
	secretInDatastore = "hwid"
)

var cachedCfg = cfgcache.Register(&cfgcache.Entry{
	Path: configFile,
	Type: (*Config)(nil),
})

// Import fetches the most recent config and stores it in the datastore.
//
// Must be called periodically to make sure Get and Middleware use the freshest
// config.
func Import(c context.Context) error {
	_, err := cachedCfg.Update(c, nil)
	return err
}

// Get returns the config in c, or panics.
// See also Use and Middleware.
func Get(c context.Context) *Config {
	return c.Value(contextKey).(*Config)
}

// Middleware loads the service config and installs it into the context.
func Middleware(c *router.Context, next router.Handler) {
	msg, err := cachedCfg.Get(c.Context, nil)
	if err != nil {
		logging.WithError(err).Errorf(c.Context, "could not load application config")
		http.Error(c.Writer, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Make a copy, since we are going to mutate it below.
	var cfg Config
	proto.Merge(&cfg, msg)

	// We store HWID server secret in datastore in production. The fallback to
	// config file is only for local development.
	ctx := gaesecrets.Use(c.Context, &gaesecrets.Config{
		NoAutogenerate: true,
	})
	secret, err := secrets.GetSecret(ctx, secretInDatastore)
	if err == nil {
		// The HWID must be a valid plain text string. No control characters.
		s := string(secret.Current)
		if s != url.QueryEscape(s) {
			logging.WithError(err).Errorf(c.Context, "wrong hwid secret configured: '%v'", url.QueryEscape(s))
			http.Error(c.Writer, "Internal server error", http.StatusInternalServerError)
			return
		}
		cfg.HwidSecret = string(secret.Current)
	} else {
		logging.Infof(c.Context, "Cannot get HWID server secret from datastore: %s", err.Error())
	}

	c.Context = Use(c.Context, &cfg)
	next(c)
}

// Use installs cfg into c.
func Use(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, contextKey, cfg)
}
