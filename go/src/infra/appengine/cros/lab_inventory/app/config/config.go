// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"net/http"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/config/server/cfgclient"
	"go.chromium.org/luci/config/server/cfgclient/textproto"
	"go.chromium.org/luci/config/validation"
	"go.chromium.org/luci/server/router"
	"golang.org/x/net/context"
)

const configFile = "config.cfg"

// unique type to prevent assignment.
type contextKeyType struct{}

// unique key used to store and retrieve context.
var contextKey = contextKeyType{}

// Get returns the config in c, or panics.
// See also Use and Middleware.
func Get(c context.Context) *Config {
	return c.Value(contextKey).(*Config)
}

// Middleware loads the service config and installs it into the context.
func Middleware(c *router.Context, next router.Handler) {
	var cfg Config
	err := cfgclient.Get(
		c.Context,
		cfgclient.AsService,
		cfgclient.CurrentServiceConfigSet(c.Context),
		configFile,
		textproto.Message(&cfg),
		nil,
	)
	if err != nil {
		logging.WithError(err).Errorf(c.Context, "could not load application config")
		http.Error(c.Writer, "Internal server error", http.StatusInternalServerError)
		return
	}
	c.Context = Use(c.Context, &cfg)
	next(c)
}

// Use installs cfg into c.
func Use(c context.Context, cfg *Config) context.Context {
	return context.WithValue(c, contextKey, cfg)
}

// SetupValidation adds validation rules for configuration data pushed via luci-config.
func SetupValidation() {
	rules := &validation.Rules
	rules.Add("services/${appid}", configFile, validateConfig)
}

func validateConfig(c *validation.Context, configSet, path string, content []byte) error {
	cfg := &Config{}
	if err := proto.UnmarshalText(string(content), cfg); err != nil {
		c.Errorf("not a valid Config proto message: %s", err)
		return nil
	}
	return nil
}
