// Copyright 2018 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"net/http"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config/server/cfgcache"
	"go.chromium.org/luci/server/router"
)

// unique type to prevent assignment.
type contextKeyType struct{}

// unique key used to store and retrieve context.
var contextKey = contextKeyType{}

// defines how to fetch and cache the config.
var cachedCfg = cfgcache.Register(&cfgcache.Entry{
	Path: "config.cfg",
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
	} else {
		c.Context = Use(c.Context, msg.(*Config))
		next(c)
	}
}

// Use installs cfg into c.
func Use(c context.Context, cfg *Config) context.Context {
	return context.WithValue(c, contextKey, cfg)
}
