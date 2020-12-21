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
						file_extension_map {
							key: ".txt"
							value: {
								paths: "a/b.txt",
								paths: "a/*/c.txt",
								paths: "d/*"
							}
						}
						file_extension_map {
							key: ""
							value: {
								paths: "a/b"
							}
						}
						file_extension_map {
							key: "*"
							value: {
								paths: "z/*"
							}
						}
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
		Convey("validateBenignFilePattern catches errors", func() {
			m := cfg.HostConfigs["test-host"].RepoConfigs["dummy"].BenignFilePattern.FileExtensionMap
			Convey("invalid file extension", func() {
				m["a.txt"] = &Paths{}
				So(validate(cfg), ShouldErrLike, "invalid file extension a.txt")
			})
			Convey("invalid path", func() {
				m[".txt"].Paths[0] = "\\"
				So(validate(cfg), ShouldErrLike, "invalid path \\")
			})
			Convey("invalid path with extension", func() {
				m[".txt"].Paths[0] = "a/b.md"
				err := validate(cfg)
				So(err, ShouldErrLike, "the extension of path a/b.md does not match the extension .txt")
			})
			Convey("invalid path with extension *", func() {
				m["*"].Paths[0] = "a/b.md"
				err := validate(cfg)
				So(err, ShouldErrLike, "the extension of path a/b.md does not match the extension *")
			})
		})
	})
}
