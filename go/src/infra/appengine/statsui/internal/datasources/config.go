// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package datasources

import (
	"gopkg.in/yaml.v2"
)

// Config is used to configure the data source client
type Config struct {
	Sources map[string]SourceConfig `yaml:"sources"`
}

// SourceConfig specifies the configuration for a single data source
type SourceConfig struct {
	// Valid keys map to the Period enum in service.proto
	// i.e. DAY, WEEK
	Queries map[string]string `yaml:"queries,flow"`
}

func UnmarshallConfig(yamlConfig []byte) (*Config, error) {
	sources := Config{}
	err := yaml.Unmarshal(yamlConfig, &sources)
	if err != nil {
		return nil, err
	}
	return &sources, nil
}
