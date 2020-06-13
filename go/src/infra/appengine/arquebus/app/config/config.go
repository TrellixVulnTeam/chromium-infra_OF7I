// Copyright 2019 The LUCI Authors.
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

// Package config implements interface for app-level configs for Arquebus.
package config

import (
	"context"
	"net/http"

	"google.golang.org/protobuf/proto"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config"
	"go.chromium.org/luci/config/server/cfgcache"
	"go.chromium.org/luci/config/validation"
	"go.chromium.org/luci/server/router"

	"infra/appengine/arquebus/app/util"
)

// A string with unique address used to store and retrieve context values.
var ctxKeyConfig = "arquebus.config"

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
//
// It is then used by all requests that go through Middleware.
func Update(c context.Context) error {
	_, err := cachedCfg.Update(c, nil)
	return err
}

// Get returns the config stored in the context.
func Get(c context.Context) *Config {
	return c.Value(&ctxKeyConfig).(*contextState).config
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
	c.Context = SetConfig(c.Context, cfg.(*Config), meta.Revision)
	next(c)
}

// SetConfig installs cfg into c.
func SetConfig(c context.Context, cfg *Config, rev string) context.Context {
	return context.WithValue(c, &ctxKeyConfig, &contextState{
		config:   cfg,
		revision: rev,
	})
}

// GetConfigRevision returns the revision of the current config.
func GetConfigRevision(c context.Context) string {
	return c.Value(&ctxKeyConfig).(*contextState).revision
}

// IsEqual returns whether the IssueQuery objects are equal.
func (lhs *IssueQuery) IsEqual(rhs *IssueQuery) bool {
	// IssueQuery is a proto-generated struct.
	return (lhs.Q == rhs.Q &&
		util.EqualSortedLists(lhs.ProjectNames, rhs.ProjectNames))
}
