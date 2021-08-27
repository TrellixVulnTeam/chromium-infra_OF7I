// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"io/ioutil"
	"testing"

	"go.chromium.org/luci/config/validation"
	"google.golang.org/protobuf/encoding/prototext"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

var sampleConfigStr = `
	monorail_hostname: "monorail-test.appspot.com"
`

// createConfig returns a new valid Config for testing.
func createConfig() *Config {
	var cfg Config
	So(prototext.Unmarshal([]byte(sampleConfigStr), &cfg), ShouldBeNil)
	return &cfg
}

func TestConfigValidator(t *testing.T) {
	t.Parallel()

	validate := func(cfg *Config) error {
		c := validation.Context{Context: context.Background()}
		validateConfig(&c, cfg)
		return c.Finalize()
	}

	Convey("config template is valid", t, func() {
		content, err := ioutil.ReadFile(
			"../../configs/services/chops-weetbix-dev/config-template.cfg",
		)
		So(err, ShouldBeNil)
		cfg := &Config{}
		So(prototext.Unmarshal(content, cfg), ShouldBeNil)
		So(validate(cfg), ShouldBeNil)
	})

	Convey("valid config is valid", t, func() {
		cfg := createConfig()
		So(validate(cfg), ShouldBeNil)

	})

	Convey("empty monorail_hostname is not valid", t, func() {
		cfg := createConfig()
		cfg.MonorailHostname = ""
		So(validate(cfg), ShouldErrLike, "empty value is not allowed")
	})
}
