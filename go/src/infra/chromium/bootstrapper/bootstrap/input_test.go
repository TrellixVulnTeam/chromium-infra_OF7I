// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestInput(t *testing.T) {
	t.Parallel()

	Convey("NewInput", t, func() {
		build := &buildbucketpb.Build{
			Input: &buildbucketpb.Build_Input{
				Properties: &structpb.Struct{},
			},
		}

		Convey("fails for missing $bootstrap", func() {
			bootstrapper, err := NewInput(build)

			So(err, ShouldErrLike, "none of the config_project fields in $bootstrap is set")
			So(bootstrapper, ShouldBeNil)
		})

		Convey("fails for incorrectly typed $bootstrap", func() {
			setBootstrapProperties(build, `{"foo": "bar"}`)

			bootstrapper, err := NewInput(build)

			So(err, ShouldErrLike, `unknown field "foo"`)
			So(bootstrapper, ShouldBeNil)
		})

		Convey("fails for invalid $bootstrap", func() {
			setBootstrapProperties(build, "{}")

			bootstrapper, err := NewInput(build)

			So(err, ShouldErrLike, "none of the config_project fields in $bootstrap is set")
			So(bootstrapper, ShouldBeNil)
		})

		Convey("returns bootstrapper for well-formed $bootstrap", func() {
			setBootstrapProperties(build, `{
				"top_level_project": {
					"repo": {
						"host": "chromium.googlesource.com",
						"project": "top/level"
					},
					"ref": "refs/heads/top-level"
				},
				"properties_file": "infra/config/fake-bucket/fake-builder/properties.textpb",
				"exe": {
					"cipd_package": "fake-package",
					"cipd_version": "fake-version",
					"cmd": ["fake-exe"]
				}
			}`)

			bootstrapper, err := NewInput(build)

			So(err, ShouldBeNil)
			So(bootstrapper, ShouldNotBeNil)
		})

	})
}
