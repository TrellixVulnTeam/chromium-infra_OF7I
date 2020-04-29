// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// UFS service will read local config file instead of luci-config service,
// as LUCI Config library is not compatible with GAEv2 now.

package config

import "context"

const configFile = "config.cfg"

// unique key used to store and retrieve context.
var contextKey = "ufs luci-config key"

// Provider returns the current non-nil config when called.
type Provider func() *Config

// Get returns the config in c if it exists, or nil.
func Get(c context.Context) *Config {
	return c.Value(&contextKey).(*Config)
}

// Use installs a config into c.
func Use(c context.Context, cfg *Config) context.Context {
	return context.WithValue(c, &contextKey, cfg)
}
