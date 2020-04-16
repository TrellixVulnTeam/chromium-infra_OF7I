// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"flag"
	"io/ioutil"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
)

const configPath = "app/config/config.cfg"

// Loader periodically rereads the config file from disk (GKE) and injects
// it into the request context.
//
// Intended for GKE environment where the config is distributed as k8s ConfigMap
// object.
type Loader struct {
	ConfigPath string // path to the config file, set via -config-path

	lastGood atomic.Value
}

// RegisterFlags registers CLI flags.
func (l *Loader) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&l.ConfigPath, "config-path", configPath, "Path to the ufs config file")
}

// Load loads the config from local static config file.
func (l *Loader) Load() (*Config, error) {
	if l.ConfigPath == "" {
		return nil, errors.Reason("-config-path is required").Err()
	}

	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, errors.Annotate(err, "failed to open the config file").Err()
	}
	cfg := &Config{}
	if err := proto.UnmarshalText(string(b), cfg); err != nil {
		return nil, errors.Annotate(err, "invalid Config proto message").Err()
	}
	l.lastGood.Store(cfg)
	return cfg, nil
}

// Config returns last good config or nil.
func (l *Loader) Config() *Config {
	cfg, _ := l.lastGood.Load().(*Config)
	return cfg
}

// ReloadLoop periodically reloads the config file until the context is
// canceled.
func (l *Loader) ReloadLoop(c context.Context) {
	for {
		if r := <-clock.After(c, time.Minute); r.Err != nil {
			return // the context is canceled, the server is closing
		}
		prevCfg := l.Config()
		newCfg, err := l.Load()
		if err != nil {
			logging.WithError(err).Errorf(c, "Failed to reload the config, using the cached one")
		} else if prevCfg != nil && !proto.Equal(prevCfg, newCfg) {
			logging.Infof(c, "Reloaded the config")
		}
	}
}
