// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	fakegitiles "infra/chromium/bootstrapper/fakes/gitiles"
	"infra/chromium/bootstrapper/gitiles"
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"
	. "github.com/smartystreets/goconvey/convey"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestPropertyBootstrapper(t *testing.T) {
	t.Parallel()

	Convey("PropertyBootstrapper", t, func() {
		ctx := context.Background()

		build := &buildbucketpb.Build{
			Input: &buildbucketpb.Build_Input{
				Properties: &structpb.Struct{},
			},
		}

		topLevelProject := &fakegitiles.Project{
			Refs:  map[string]string{},
			Files: map[fakegitiles.FileRevId]*string{},
		}

		ctx = gitiles.UseGitilesClientFactory(ctx, fakegitiles.Factory(map[string]*fakegitiles.Host{
			"chromium.googlesource.com": {
				Projects: map[string]*fakegitiles.Project{
					"top/level": topLevelProject,
				},
			},
		}))

		bootstrapper := NewPropertyBootstrapper(gitiles.NewClient(ctx))

		Convey("ComputeBootstrappedProperties", func() {

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

			Convey("fails", func() {

				Convey("if unable to get revision", func() {
					input := getInput(build)
					topLevelProject.Refs["refs/heads/top-level"] = ""

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldNotBeNil)
					So(properties, ShouldBeNil)
				})

				Convey("if unable to get file", func() {
					input := getInput(build)
					topLevelProject.Refs["refs/heads/top-level"] = "top-level-top-level-head"
					topLevelProject.Files[fakegitiles.FileRevId{
						Revision: "top-level-top-level-head",
						Path:     "infra/config/fake-bucket/fake-builder/properties.textpb",
					}] = nil

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldNotBeNil)
					So(properties, ShouldBeNil)
				})

				Convey("if the properties file is invalid", func() {
					input := getInput(build)
					topLevelProject.Refs["refs/heads/top-level"] = "top-level-top-level-head"
					topLevelProject.Files[fakegitiles.FileRevId{
						Revision: "top-level-top-level-head",
						Path:     "infra/config/fake-bucket/fake-builder/properties.textpb",
					}] = strPtr("")

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldNotBeNil)
					So(properties, ShouldBeNil)
				})

			})

			Convey("returns properties", func() {

				Convey("with properties from the properties file", func() {
					topLevelProject.Refs["refs/heads/top-level"] = "top-level-top-level-head"
					topLevelProject.Files[fakegitiles.FileRevId{
						Revision: "top-level-top-level-head",
						Path:     "infra/config/fake-bucket/fake-builder/properties.textpb",
					}] = strPtr(`{
						"$build/baz": {
							"quux": "quuz"
						},
						"foo": "bar"
					}`)
					input := getInput(build)

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldBeNil)
					So(getValueAtPath(properties, "$build/baz"), ShouldResembleProtoJSON, `{"quux": "quuz"}`)
					So(getValueAtPath(properties, "foo"), ShouldResembleProtoJSON, `"bar"`)
					So(properties.Fields, ShouldNotContainKey, "$bootstrap")
				})

				Convey("with build properties merged with properties from the properties file", func() {
					topLevelProject.Refs["refs/heads/top-level"] = "top-level-top-level-head"
					topLevelProject.Files[fakegitiles.FileRevId{
						Revision: "top-level-top-level-head",
						Path:     "infra/config/fake-bucket/fake-builder/properties.textpb",
					}] = strPtr(`{
						"$build/baz": {
							"quux": "quuz"
						},
						"foo": "bar"
					}`)
					setPropertiesFromJson(build, map[string]string{
						"$build/baz": `{
							"quuy": "quuw"
						}`,
						"shaz": `1337.0`,
					})
					input := getInput(build)

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldBeNil)
					So(getValueAtPath(properties, "$build/baz"), ShouldResembleProtoJSON, `{"quuy": "quuw"}`)
					So(getValueAtPath(properties, "foo"), ShouldResembleProtoJSON, `"bar"`)
					So(getValueAtPath(properties, "shaz"), ShouldResembleProtoJSON, `1337`)
					So(properties.Fields, ShouldNotContainKey, "$bootstrap")
				})

				Convey("for top level project with commits in $build/chromium_bootstrap property", func() {

					Convey("for commit ref when build has gitiles commit without ID for top level project", func() {
						build.Input.GitilesCommit = &buildbucketpb.GitilesCommit{
							Host:    "chromium.googlesource.com",
							Project: "top/level",
							Ref:     "refs/heads/some-branch",
						}
						topLevelProject.Refs["refs/heads/some-branch"] = "top-level-some-branch-head"
						topLevelProject.Files[fakegitiles.FileRevId{
							Revision: "top-level-some-branch-head",
							Path:     "infra/config/fake-bucket/fake-builder/properties.textpb",
						}] = strPtr(`{"test_property": "some-branch-head-value"}`)
						input := getInput(build)

						properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

						So(err, ShouldBeNil)
						So(getValueAtPath(properties, "test_property"), ShouldResembleProtoJSON, `"some-branch-head-value"`)
						So(getValueAtPath(properties, "$build/chromium_bootstrap", "commits"), ShouldResembleProtoJSON, `[
							{
								"host": "chromium.googlesource.com",
								"project": "top/level",
								"ref": "refs/heads/some-branch",
								"id": "top-level-some-branch-head"
							}
						]`)
					})

					Convey("for commit revision when build has gitiles commit with ID for top level project", func() {
						build.Input.GitilesCommit = &buildbucketpb.GitilesCommit{
							Host:    "chromium.googlesource.com",
							Project: "top/level",
							Ref:     "refs/heads/some-branch",
							Id:      "some-branch-revision",
						}
						topLevelProject.Files[fakegitiles.FileRevId{
							Revision: "some-branch-revision",
							Path:     "infra/config/fake-bucket/fake-builder/properties.textpb",
						}] = strPtr(`{"test_property": "some-branch-revision-value"}`)
						input := getInput(build)

						properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

						So(err, ShouldBeNil)
						So(getValueAtPath(properties, "test_property"), ShouldResembleProtoJSON, `"some-branch-revision-value"`)
						So(getValueAtPath(properties, "$build/chromium_bootstrap", "commits"), ShouldResembleProtoJSON, `[
							{
								"host": "chromium.googlesource.com",
								"project": "top/level",
								"ref": "refs/heads/some-branch",
								"id": "some-branch-revision"
							}
						]`)
					})

					Convey("for top level ref when build does not have gitiles commit or gerrit change for top level project", func() {
						build.Input.GitilesCommit = &buildbucketpb.GitilesCommit{
							Host:    "chromium.googlesource.com",
							Project: "unrelated",
							Ref:     "refs/heads/irrelevant",
						}
						input := getInput(build)
						topLevelProject.Refs["refs/heads/top-level"] = "top-level-top-level-head"
						topLevelProject.Files[fakegitiles.FileRevId{
							Revision: "top-level-top-level-head",
							Path:     "infra/config/fake-bucket/fake-builder/properties.textpb",
						}] = strPtr(`{"test_property": "top-level-head-value"}`)

						properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

						So(err, ShouldBeNil)
						So(getValueAtPath(properties, "test_property"), ShouldResembleProtoJSON, `"top-level-head-value"`)
						So(getValueAtPath(properties, "$build/chromium_bootstrap", "commits"), ShouldResembleProtoJSON, `[
							{
								"host": "chromium.googlesource.com",
								"project": "top/level",
								"ref": "refs/heads/top-level",
								"id": "top-level-top-level-head"
							}
						]`)
					})

				})

			})

		})

	})
}
