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

func TestServiceConfigValidator(t *testing.T) {
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

	Convey("empty monorail_hostname is not valid", t, func() {
		cfg := createConfig()
		cfg.MonorailHostname = ""
		So(validate(cfg), ShouldErrLike, "empty value is not allowed")
	})
}

func TestProjectConfigValidator(t *testing.T) {
	t.Parallel()

	validate := func(cfg *ProjectConfig) error {
		c := validation.Context{Context: context.Background()}
		validateProjectConfig(&c, cfg)
		return c.Finalize()
	}

	Convey("config template is valid", t, func() {
		content, err := ioutil.ReadFile(
			"../../configs/projects/chromium/chops-weetbix-dev-template.cfg",
		)
		So(err, ShouldBeNil)
		cfg := &ProjectConfig{}
		So(prototext.Unmarshal(content, cfg), ShouldBeNil)
		So(validate(cfg), ShouldBeNil)
	})

	Convey("valid config is valid", t, func() {
		cfg := createProjectConfig()
		So(validate(cfg), ShouldBeNil)
	})

	Convey("monorail", t, func() {
		Convey("empty project is not valid", func() {
			cfg := createProjectConfig()
			cfg.Monorail.Project = ""
			So(validate(cfg), ShouldErrLike, "empty value is not allowed")
		})

		Convey("illegal project is not valid", func() {
			cfg := createProjectConfig()
			// Project does not satisfy regex.
			cfg.Monorail.Project = "-my-project"
			So(validate(cfg), ShouldErrLike, "project is not a valid monorail project")
		})

		Convey("negative priority field ID is not valid", func() {
			cfg := createProjectConfig()
			cfg.Monorail.PriorityFieldId = -1
			So(validate(cfg), ShouldErrLike, "value must be non-negative")
		})

		Convey("field value with negative field ID is not valid", func() {
			cfg := createProjectConfig()
			cfg.Monorail.DefaultFieldValues = []*MonorailFieldValue{
				{
					FieldId: -1,
					Value:   "",
				},
			}
			So(validate(cfg), ShouldErrLike, "value must be non-negative")
		})
	})
}
