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
)

// unique key used to store and retrieve context.
var contextKey = "qscheduler-swarming luci-config key"

// Provider returns the current non-nil config when called.
type Provider func() *Config

// Get returns the config in c if it exists, or nil.
func Get(c context.Context) *Config {
	if p, _ := c.Value(&contextKey).(Provider); p != nil {
		return p()
	}
	return nil
}

// Use installs a config provider into c.
func Use(c context.Context, p Provider) context.Context {
	return context.WithValue(c, &contextKey, p)
}
