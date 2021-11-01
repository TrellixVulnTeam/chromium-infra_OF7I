// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"go.chromium.org/luci/config/validation"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

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
			So(validate(cfg), ShouldErrLike, "empty chunk_gcs_bucket is not allowed")
		})
		Convey("must be correctly formed", func() {
			cfg := createConfig()
			cfg.ChunkGcsBucket = "my bucket"
			So(validate(cfg), ShouldErrLike, `invalid chunk_gcs_bucket: "my bucket"`)
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
			So(validate(cfg), ShouldErrLike, "empty project is not allowed")
		})

		Convey("illegal monorail project", func() {
			// Project does not satisfy regex.
			cfg.Monorail.Project = "-my-project"
			So(validate(cfg), ShouldErrLike, `invalid project: "-my-project"`)
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

	Convey("realm config", t, func() {
		cfg := createProjectConfig()
		So(len(cfg.Realms), ShouldEqual, 1)
		realm := cfg.Realms[0]

		Convey("realm name", func() {
			Convey("must be specified", func() {
				realm.Name = ""
				So(validate(cfg), ShouldErrLike, "empty realm_name is not allowed")
			})
			Convey("invalid", func() {
				realm.Name = "chromium:ci"
				So(validate(cfg), ShouldErrLike, `invalid realm_name: "chromium:ci"`)
			})
			Convey("valid", func() {
				realm.Name = "ci"
				So(validate(cfg), ShouldBeNil)
			})
		})

		Convey("TestVariantAnalysisConfig", func() {
			tvCfg := realm.TestVariantAnalysis
			So(tvCfg, ShouldNotBeNil)
			utCfg := tvCfg.UpdateTestVariantTask
			So(utCfg, ShouldNotBeNil)
			Convey("UpdateTestVariantTask", func() {
				Convey("interval", func() {
					Convey("empty not allowed", func() {
						utCfg.UpdateTestVariantTaskInterval = nil
						So(validate(cfg), ShouldErrLike, `empty interval is not allowed`)
					})
					Convey("must be greater than 0", func() {
						utCfg.UpdateTestVariantTaskInterval = durationpb.New(-time.Hour)
						So(validate(cfg), ShouldErrLike, `interval is less than 0`)
					})
				})

				Convey("duration", func() {
					Convey("empty not allowed", func() {
						utCfg.TestVariantStatusUpdateDuration = nil
						So(validate(cfg), ShouldErrLike, `empty duration is not allowed`)
					})
					Convey("must be greater than 0", func() {
						utCfg.TestVariantStatusUpdateDuration = durationpb.New(-time.Hour)
						So(validate(cfg), ShouldErrLike, `duration is less than 0`)
					})
				})
			})

			bqExports := tvCfg.BqExports
			So(len(bqExports), ShouldEqual, 1)
			bqe := bqExports[0]
			So(bqe, ShouldNotBeNil)
			Convey("BqExport", func() {
				table := bqe.Table
				So(table, ShouldNotBeNil)
				Convey("BigQueryTable", func() {
					Convey("cloud project", func() {
						Convey("should npt be empty", func() {
							table.CloudProject = ""
							So(validate(cfg), ShouldErrLike, "empty cloud_project is not allowed")
						})
						Convey("not end with hyphen", func() {
							table.CloudProject = "project-"
							So(validate(cfg), ShouldErrLike, `invalid cloud_project: "project-"`)
						})
						Convey("not too short", func() {
							table.CloudProject = "p"
							So(validate(cfg), ShouldErrLike, `invalid cloud_project: "p"`)
						})
						Convey("must start with letter", func() {
							table.CloudProject = "0project"
							So(validate(cfg), ShouldErrLike, `invalid cloud_project: "0project"`)
						})
					})

					Convey("dataset", func() {
						Convey("should npt be empty", func() {
							table.Dataset = ""
							So(validate(cfg), ShouldErrLike, "empty dataset is not allowed")
						})
						Convey("should be valid", func() {
							table.Dataset = "data-set"
							So(validate(cfg), ShouldErrLike, `invalid dataset: "data-set"`)
						})
					})

					Convey("table", func() {
						Convey("should npt be empty", func() {
							table.Table = ""
							So(validate(cfg), ShouldErrLike, "empty table_name is not allowed")
						})
						Convey("should be valid", func() {
							table.Table = "table/name"
							So(validate(cfg), ShouldErrLike, `invalid table_name: "table/name"`)
						})
					})
				})
			})
		})
	})
}
