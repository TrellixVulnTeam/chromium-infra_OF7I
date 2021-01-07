// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package config

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/config/validation"
)

var sampleConfigStr = `
	host_configs {
		key: "test-host"
		value: {
			repo_configs {
				key: "dummy"
				value: {
					benign_file_pattern {
						paths: "a/b.txt"
						paths: "a/*/c.txt"
						paths: "d/*.txt"
						paths: "z/*"
					}
					clean_revert_pattern {
						time_window: "7d"
						excluded_paths: "a/b/*"
					}
				}
			}
		}
  	}
`

func createConfig() *Config {
	var cfg Config
	So(proto.UnmarshalText(sampleConfigStr, &cfg), ShouldBeNil)
	return &cfg
}

func TestConfigValidator(t *testing.T) {
	validate := func(cfg *Config) error {
		c := validation.Context{Context: context.Background()}
		validateConfig(&c, cfg)
		return c.Finalize()
	}

	Convey("sampleConfigStr is valid", t, func() {
		cfg := createConfig()
		So(validate(cfg), ShouldBeNil)
	})

	Convey("validateConfig catches errors", t, func() {
		cfg := createConfig()
		Convey("validateCleanRevertPattern catches errors", func() {
			crp := cfg.HostConfigs["test-host"].RepoConfigs["dummy"].CleanRevertPattern
			Convey("invalid time window value", func() {
				crp.TimeWindow = "a1s"
				So(validate(cfg), ShouldErrLike, "invalid time_window a1s")
			})
			Convey("invalid time window unit", func() {
				crp.TimeWindow = "12t"
				So(validate(cfg), ShouldErrLike, "invalid time_window 12t")
			})
			Convey("invalid path", func() {
				crp.ExcludedPaths[0] = "\\"
				So(validate(cfg), ShouldErrLike, "invalid path \\")
			})
		})
	})
}
