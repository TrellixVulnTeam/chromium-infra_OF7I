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

		opts := InputOptions{}

		Convey("fails if no required bootstrap properties are not set", func() {
			input, err := opts.NewInput(build)

			So(err, ShouldErrLike, "the following required properties are not set: $bootstrap/exe, $bootstrap/properties")
			So(input, ShouldBeNil)
		})

		Convey("fails validating $bootstrap/properties", func() {
			setBootstrapExeProperties(build, `{
				"exe": {
					"cipd_package": "fake-package",
					"cipd_version": "fake-version",
					"cmd": ["fake-exe"]
				}
			}`)

			Convey("for incorrectly typed $bootstrap/properties", func() {
				setBootstrapPropertiesProperties(build, `{"foo": "bar"}`)

				input, err := opts.NewInput(build)

				So(err, ShouldErrLike, `unknown field "foo"`)
				So(input, ShouldBeNil)
			})

			Convey("for invalid $bootstrap/properties", func() {
				setBootstrapPropertiesProperties(build, "{}")
				input, err := opts.NewInput(build)

				So(err, ShouldErrLike, "none of the config_project fields in $bootstrap/properties is set")
				So(input, ShouldBeNil)
			})

		})

		Convey("fails validating $bootstrap/exe", func() {
			setBootstrapPropertiesProperties(build, `{
				"top_level_project": {
					"repo": {
						"host": "chromium.googlesource.com",
						"project": "top/level"
					},
					"ref": "refs/heads/top-level"
				},
				"properties_file": "infra/config/fake-bucket/fake-builder/properties.textpb"
			}`)

			Convey("for incorrectly typed $bootstrap/exe", func() {
				setBootstrapExeProperties(build, `{"foo": "bar"}`)

				input, err := opts.NewInput(build)

				So(err, ShouldErrLike, `unknown field "foo"`)
				So(input, ShouldBeNil)
			})

			Convey("for invalid $bootstrap/exe", func() {
				setBootstrapExeProperties(build, "{}")

				input, err := opts.NewInput(build)

				So(err, ShouldErrLike, "$bootstrap/exe.exe is not set")
				So(input, ShouldBeNil)
			})

		})

		Convey("succeeds", func() {
			setBootstrapPropertiesProperties(build, `{
				"top_level_project": {
					"repo": {
						"host": "chromium.googlesource.com",
						"project": "top/level"
					},
					"ref": "refs/heads/top-level"
				},
				"properties_file": "infra/config/fake-bucket/fake-builder/properties.textpb"
			}`)
			setBootstrapExeProperties(build, `{
				"exe": {
					"cipd_package": "fake-package",
					"cipd_version": "fake-version",
					"cmd": ["fake-exe"]
				}
			}`)
			build.Input.Properties.Fields["foo"] = structpb.NewStringValue("bar")

			Convey("for well-formed properties", func() {
				input, err := opts.NewInput(build)

				So(err, ShouldBeNil)
				So(input.commits, ShouldBeEmpty)
				So(input.changes, ShouldBeEmpty)
				So(input.buildProperties, ShouldResembleProtoJSON, `{
					"foo": "bar"
				}`)
				So(input.propsProperties, ShouldResembleProtoJSON, `{
					"top_level_project": {
						"repo": {
							"host": "chromium.googlesource.com",
							"project": "top/level"
						},
						"ref": "refs/heads/top-level"
					},
					"properties_file": "infra/config/fake-bucket/fake-builder/properties.textpb"
				}`)
				So(input.exeProperties, ShouldResembleProtoJSON, `{
					"exe": {
						"cipd_package": "fake-package",
						"cipd_version": "fake-version",
						"cmd": ["fake-exe"]
					}
				}`)
				So(input.casRecipeBundle, ShouldBeNil)
				// Make sure the build wasn't modified
				So(build.Input.Properties.Fields, ShouldContainKey, "$bootstrap/properties")
				So(build.Input.Properties.Fields, ShouldContainKey, "$bootstrap/exe")
			})

			Convey("without $bootstrap/properties if PropertiesOptional is set", func() {
				opts := InputOptions{PropertiesOptional: true}
				delete(build.Input.Properties.Fields, "$bootstrap/properties")

				input, err := opts.NewInput(build)

				So(err, ShouldBeNil)
				So(input.propertiesOptional, ShouldBeTrue)
				So(input.propsProperties, ShouldBeNil)
			})

			Convey("with $bootstrap/properties if PropertiesOptional is set", func() {
				opts := InputOptions{PropertiesOptional: true}

				input, err := opts.NewInput(build)

				So(err, ShouldBeNil)
				So(input.propertiesOptional, ShouldBeTrue)
				So(input.propsProperties, ShouldResembleProtoJSON, `{
					"top_level_project": {
						"repo": {
							"host": "chromium.googlesource.com",
							"project": "top/level"
						},
						"ref": "refs/heads/top-level"
					},
					"properties_file": "infra/config/fake-bucket/fake-builder/properties.textpb"
				}`)
			})

			Convey("with commits set if build has commit", func() {
				build.Input.GitilesCommit = &buildbucketpb.GitilesCommit{
					Host:    "fake-host",
					Project: "fake-project",
					Ref:     "fake-ref",
					Id:      "fake-revision",
				}

				input, err := opts.NewInput(build)

				So(err, ShouldBeNil)
				So(input.commits, ShouldResembleProto, []*buildbucketpb.GitilesCommit{build.Input.GitilesCommit})
				// Make sure we can't modify the build through aliased protos
				So(input.commits[0], ShouldNotEqual, build.Input.GitilesCommit)
			})

			Convey("with commits set if $bootstrap/trigger has commits", func() {
				setBootstrapTriggerProperties(build, `{
					"commits": [
						{
							"host": "fake-host1",
							"project": "fake-project1",
							"ref": "fake-ref1",
							"id": "fake-revision1"
						},
						{
							"host": "fake-host2",
							"project": "fake-project2",
							"ref": "fake-ref2",
							"id": "fake-revision2"
						}
					]
				}`)

				input, err := opts.NewInput(build)

				So(err, ShouldBeNil)
				So(input.commits, ShouldResembleProto, []*buildbucketpb.GitilesCommit{
					{
						Host:    "fake-host1",
						Project: "fake-project1",
						Ref:     "fake-ref1",
						Id:      "fake-revision1",
					},
					{
						Host:    "fake-host2",
						Project: "fake-project2",
						Ref:     "fake-ref2",
						Id:      "fake-revision2",
					},
				})
			})

			Convey("with commits set if build has commit and $bootstrap/trigger has commits", func() {
				build.Input.GitilesCommit = &buildbucketpb.GitilesCommit{
					Host:    "fake-host1",
					Project: "fake-project1",
					Ref:     "fake-ref1a",
				}
				setBootstrapTriggerProperties(build, `{
					"commits": [
						{
							"host": "fake-host1",
							"project": "fake-project1",
							"ref": "fake-ref1b",
							"id": "fake-revision1"
						},
						{
							"host": "fake-host2",
							"project": "fake-project2",
							"ref": "fake-ref2",
							"id": "fake-revision2"
						}
					]
				}`)

				input, err := opts.NewInput(build)

				So(err, ShouldBeNil)
				So(input.commits, ShouldResembleProto, []*buildbucketpb.GitilesCommit{
					{
						Host:    "fake-host1",
						Project: "fake-project1",
						Ref:     "fake-ref1a",
					},
					{
						Host:    "fake-host1",
						Project: "fake-project1",
						Ref:     "fake-ref1b",
						Id:      "fake-revision1",
					},
					{
						Host:    "fake-host2",
						Project: "fake-project2",
						Ref:     "fake-ref2",
						Id:      "fake-revision2",
					},
				})
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

				input, err := opts.NewInput(build)

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

				input, err := opts.NewInput(build)

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
