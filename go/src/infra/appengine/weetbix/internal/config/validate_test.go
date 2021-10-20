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
	"google.golang.org/protobuf/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

var sampleConfigStr = `
	monorail_hostname: "monorail-test.appspot.com"
	chunk_gcs_bucket: "my-chunk-bucket"
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

	Convey("monorail hostname", t, func() {
		Convey("must be specified", func() {
			cfg := createConfig()
			cfg.MonorailHostname = ""
			So(validate(cfg), ShouldErrLike, "empty value is not allowed")
		})
		Convey("must be correctly formed", func() {
			cfg := createConfig()
			cfg.MonorailHostname = "monorail host"
			So(validate(cfg), ShouldErrLike, `invalid hostname: "monorail host"`)
		})
	})
	Convey("chunk GCS bucket", t, func() {
		Convey("must be specified", func() {
			cfg := createConfig()
			cfg.ChunkGcsBucket = ""
			So(validate(cfg), ShouldErrLike, "empty value is not allowed")
		})
		Convey("must be correctly formed", func() {
			cfg := createConfig()
			cfg.ChunkGcsBucket = "my bucket"
			So(validate(cfg), ShouldErrLike, `invalid bucket: "my bucket"`)
		})
	})
}

func TestProjectConfigValidator(t *testing.T) {
	t.Parallel()

	validate := func(cfg *ProjectConfig) error {
		c := validation.Context{Context: context.Background()}
		ValidateProjectConfig(&c, cfg)
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
		cfg := createProjectConfig()
		Convey("must be specified", func() {
			cfg.Monorail = nil
			So(validate(cfg), ShouldErrLike, "monorail must be specified")
		})

		Convey("project must be specified", func() {
			cfg.Monorail.Project = ""
			So(validate(cfg), ShouldErrLike, "empty value is not allowed")
		})

		Convey("illegal monorail project", func() {
			// Project does not satisfy regex.
			cfg.Monorail.Project = "-my-project"
			So(validate(cfg), ShouldErrLike, "project is not a valid monorail project")
		})

		Convey("negative priority field ID", func() {
			cfg.Monorail.PriorityFieldId = -1
			So(validate(cfg), ShouldErrLike, "value must be non-negative")
		})

		Convey("field value with negative field ID", func() {
			cfg.Monorail.DefaultFieldValues = []*MonorailFieldValue{
				{
					FieldId: -1,
					Value:   "",
				},
			}
			So(validate(cfg), ShouldErrLike, "value must be non-negative")
		})

		Convey("priorities", func() {
			priorities := cfg.Monorail.Priorities
			Convey("at least one must be specified", func() {
				cfg.Monorail.Priorities = nil
				So(validate(cfg), ShouldErrLike, "at least one monorail priority must be specified")
			})

			Convey("priority value is empty", func() {
				priorities[0].Priority = ""
				So(validate(cfg), ShouldErrLike, "empty value is not allowed")
			})

			Convey("threshold is not specified", func() {
				priorities[0].Threshold = nil
				So(validate(cfg), ShouldErrLike, "impact thresolds must be specified")
			})

			Convey("last priority", func() {
				lastPriority := priorities[len(priorities)-1]
				bugFilingThres := cfg.BugFilingThreshold
				Convey("unexpected failures 1d must be set if set on bug-filing threshold", func() {
					bugFilingThres.UnexpectedFailures_1D = proto.Int64(100)
					lastPriority.Threshold.UnexpectedFailures_1D = nil
					So(validate(cfg), ShouldErrLike, "unexpected_failures_1d threshold must be set, with a value of at most 100")
				})

				Convey("unexpected failures 1d must be satisfied by the bug-filing threshold", func() {
					bugFilingThres.UnexpectedFailures_1D = proto.Int64(100)
					lastPriority.Threshold.UnexpectedFailures_1D = proto.Int64(101)
					So(validate(cfg), ShouldErrLike, "value must be at most 100")
				})

				Convey("unexpected failures 3d must be satisfied by the bug-filing threshold", func() {
					bugFilingThres.UnexpectedFailures_3D = proto.Int64(300)
					lastPriority.Threshold.UnexpectedFailures_3D = proto.Int64(301)
					So(validate(cfg), ShouldErrLike, "value must be at most 300")
				})

				Convey("unexpected failures 7d must be satisfied by the bug-filing threshold", func() {
					bugFilingThres.UnexpectedFailures_7D = proto.Int64(700)
					lastPriority.Threshold.UnexpectedFailures_7D = proto.Int64(701)
					So(validate(cfg), ShouldErrLike, "value must be at most 700")
				})
			})
			// Other thresholding validation cases tested under bug-filing threshold and are
			// not repeated given the implementation is shared.
		})

		Convey("priority hysteresis", func() {
			Convey("value too high", func() {
				cfg.Monorail.PriorityHysteresisPercent = 1001
				So(validate(cfg), ShouldErrLike, "value must not exceed 1000 percent")
			})
			Convey("value is negative", func() {
				cfg.Monorail.PriorityHysteresisPercent = -1
				So(validate(cfg), ShouldErrLike, "value must not be negative")
			})
		})
	})
	Convey("bug filing threshold", t, func() {
		cfg := createProjectConfig()
		threshold := cfg.BugFilingThreshold
		So(threshold, ShouldNotBeNil)

		Convey("must be specified", func() {
			cfg.BugFilingThreshold = nil
			So(validate(cfg), ShouldErrLike, "impact thresolds must be specified")
		})

		Convey("unexpected failures 1d is negative", func() {
			threshold.UnexpectedFailures_1D = proto.Int64(-1)
			So(validate(cfg), ShouldErrLike, "value must be non-negative")
		})

		Convey("unexpected failures 3d is negative", func() {
			threshold.UnexpectedFailures_3D = proto.Int64(-1)
			So(validate(cfg), ShouldErrLike, "value must be non-negative")
		})

		Convey("unexpected failures 7d is negative", func() {
			threshold.UnexpectedFailures_7D = proto.Int64(-1)
			So(validate(cfg), ShouldErrLike, "value must be non-negative")
		})
	})
}
