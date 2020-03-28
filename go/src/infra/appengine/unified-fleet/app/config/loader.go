// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
)

const configPath = "app/config/config.cfg"

// Load loads the config from local static config file.
func Load() (*Config, error) {
	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, errors.Annotate(err, "failed to open the config file").Err()
	}
	cfg := &Config{}
	if err := proto.UnmarshalText(string(b), cfg); err != nil {
		return nil, errors.Annotate(err, "invalid Config proto message").Err()
	}
	return cfg, nil
}
