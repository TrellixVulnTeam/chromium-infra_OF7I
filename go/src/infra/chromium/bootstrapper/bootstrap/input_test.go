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
			input, err := NewInput(build)

			So(err, ShouldErrLike, "none of the config_project fields in $bootstrap is set")
			So(input, ShouldBeNil)
		})

		Convey("fails for incorrectly typed $bootstrap", func() {
			setBootstrapProperties(build, `{"foo": "bar"}`)

			input, err := NewInput(build)

			So(err, ShouldErrLike, `unknown field "foo"`)
			So(input, ShouldBeNil)
		})

		Convey("fails for invalid $bootstrap", func() {
			setBootstrapProperties(build, "{}")

			input, err := NewInput(build)

			So(err, ShouldErrLike, "none of the config_project fields in $bootstrap is set")
			So(input, ShouldBeNil)
		})

		Convey("succeeds", func() {
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
			build.Input.Properties.Fields["foo"] = structpb.NewStringValue("bar")

			Convey("for well-formed $bootstrap", func() {
				input, err := NewInput(build)

				So(err, ShouldBeNil)
				So(input.commit, ShouldBeNil)
				So(input.changes, ShouldBeEmpty)
				So(input.buildProperties, ShouldResembleProtoJSON, `{
					"foo": "bar"
				}`)
				So(input.properties, ShouldResembleProtoJSON, `{
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
				So(input.casRecipeBundle, ShouldBeNil)
				// Make sure the build wasn't modified
				So(build.Input.Properties.Fields, ShouldContainKey, "$bootstrap")
			})

			Convey("with commit set if build has commit", func() {
				build.Input.GitilesCommit = &buildbucketpb.GitilesCommit{
					Host:    "fake-host",
					Project: "fake-project",
					Ref:     "fake-ref",
					Id:      "fake-revision",
				}

				input, err := NewInput(build)

				So(err, ShouldBeNil)
				So(input.commit, ShouldResembleProto, build.Input.GitilesCommit)
				// Make sure we can't modify the build through aliased protos
				So(input.commit, ShouldNotEqual, build.Input.GitilesCommit)
			})

			Convey("with changes set if build has changes", func() {
				build.Input.GerritChanges = []*buildbucketpb.GerritChange{
					{
						Host:     "fake-host",
						Project:  "fake-project",
						Change:   1234,
						Patchset: 5,
					},
					{
						Host:     "fake-host2",
						Project:  "fake-project2",
						Change:   6789,
						Patchset: 10,
					},
				}

				input, err := NewInput(build)

				So(err, ShouldBeNil)
				So(input.changes, ShouldHaveLength, 2)
				So(input.changes, ShouldResembleProto, build.Input.GerritChanges)
				// Make sure we can't modify the build through aliased protos
				So(input.changes[0], ShouldNotEqual, build.Input.GerritChanges[0])
				So(input.changes[1], ShouldNotEqual, build.Input.GerritChanges[1])
			})

			Convey("with casRecipeBundle set if build has led_cas_recipe_bundle property", func() {
				build.Input.Properties.Fields["led_cas_recipe_bundle"] = structpb.NewStructValue(jsonToStruct(`{
					"cas_instance": "fake-instance",
					"digest": {
						"hash": "fake-hash",
						"size_bytes": 42
					}
				}`))

				input, err := NewInput(build)

				So(err, ShouldBeNil)
				So(input.casRecipeBundle, ShouldResembleProtoJSON, `{
					"cas_instance": "fake-instance",
					"digest": {
						"hash": "fake-hash",
						"size_bytes": 42
					}
				}`)
				So(input.buildProperties.Fields, ShouldNotContainKey, "led_cas_recipe_bundle")
				// Make sure the build wasn't modified
				So(build.Input.Properties.Fields, ShouldContainKey, "led_cas_recipe_bundle")
			})
		})

	})
}
