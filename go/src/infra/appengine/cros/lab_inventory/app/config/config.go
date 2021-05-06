// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"net/http"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/appengine/gaesecrets"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config/server/cfgcache"
	"go.chromium.org/luci/server/router"
	"go.chromium.org/luci/server/secrets"

	"infra/appengine/cros/lab_inventory/app/external"
)

const configFile = "config.cfg"

// unique type to prevent assignment.
type contextKeyType struct{}

// unique key used to store and retrieve context.
var (
	contextKey        = contextKeyType{}
	secretInDatastore = "hwid"
)

var cachedCfg *cfgcache.Entry

func init() {
	cachedCfg = cfgcache.Register(&cfgcache.Entry{
		Path: configFile,
		Type: (*Config)(nil),
	})
}

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

// Interceptor is to be used to append config to context in grpc handlers.
func Interceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx, err := appendConfigToContext(ctx)
	if err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

// Middleware is to be used to append config to context in cron handlers.
func Middleware(c *router.Context, next router.Handler) {
	ctx, err := appendConfigToContext(c.Context)
	if err != nil {
		http.Error(c.Writer, "Internal server error", http.StatusInternalServerError)
		return
	}
	c.Context = ctx
	next(c)
}

// appendConfigToContext appends a copy of the config to the context and returns the updated context.
func appendConfigToContext(ctx context.Context) (context.Context, error) {
	msg, err := cachedCfg.Get(ctx, nil)
	if err != nil {
		logging.WithError(err).Errorf(ctx, "could not load application config")
		return nil, err
	}

	// Make a copy, since we are going to mutate it below.
	var cfg Config
	proto.Merge(&cfg, msg)

	// We store HWID server secret in datastore in production. The fallback to
	// config file is only for local development.
	ctx = gaesecrets.Use(ctx, &gaesecrets.Config{})
	secret, err := secrets.StoredSecret(ctx, secretInDatastore)
	if err == nil {
		// The HWID must be a valid plain text string. No control characters.
		s := string(secret.Current)
		if s != url.QueryEscape(s) {
			logging.WithError(err).Errorf(ctx, "wrong hwid secret configured: '%v'", url.QueryEscape(s))
			return nil, err
		}
		cfg.HwidSecret = string(secret.Current)
	} else {
		logging.Infof(ctx, "Cannot get HWID server secret from datastore: %s", err.Error())
	}

	ctx = Use(ctx, &cfg)
	ctx = external.WithServerInterface(ctx)

	return ctx, nil
}

// Use installs cfg into c.
func Use(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, contextKey, cfg)
}
