// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"go.chromium.org/luci/common/errors"
	"google.golang.org/protobuf/encoding/prototext"

	configpb "infra/appengine/weetbix/internal/config/proto"
)

var sampleConfigStr = `
	monorail_hostname: "monorail-test.appspot.com"
	chunk_gcs_bucket: "my-chunk-bucket"
	reclustering_workers: 50
	reclustering_interval_minutes: 5
`

// CreatePlaceholderConfig returns a new valid Config for testing.
func CreatePlaceholderConfig() (*configpb.Config, error) {
	var cfg configpb.Config
	err := prototext.Unmarshal([]byte(sampleConfigStr), &cfg)
	if err != nil {
		return nil, errors.Annotate(err, "Marshaling a test config").Err()
	}
	return &cfg, nil
}
