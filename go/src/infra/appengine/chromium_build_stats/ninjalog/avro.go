// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ninjalog

import (
	_ "embed"
	"fmt"
	"sync"

	goavro "github.com/linkedin/goavro/v2"
	"sigs.k8s.io/yaml"
)

//go:embed avro_schema.yaml
var yamlSchema []byte

var codecOnce sync.Once
var codec *goavro.Codec
var codecErr error

// AVROCodec returns codec used to write ninja log with AVRO format.
func AVROCodec() (*goavro.Codec, error) {
	codecOnce.Do(func() {
		jsonSchema, err := yaml.YAMLToJSON(yamlSchema)
		if err != nil {
			codecErr = fmt.Errorf("failed to convert %s: %w", yamlSchema, err)
			return
		}

		codec, err = goavro.NewCodec(string(jsonSchema))
		if err != nil {
			codecErr = fmt.Errorf("failed to create codec: %w", err)
			return
		}
	})

	return codec, codecErr
}
