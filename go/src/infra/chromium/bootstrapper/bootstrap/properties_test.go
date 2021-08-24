// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	fakegerrit "infra/chromium/bootstrapper/fakes/gerrit"
	fakegitiles "infra/chromium/bootstrapper/fakes/gitiles"
	"infra/chromium/bootstrapper/gerrit"
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

		topLevelGitiles := &fakegitiles.Project{
			Refs:      map[string]string{},
			Revisions: map[string]*fakegitiles.Revision{},
		}
		ctx = gitiles.UseGitilesClientFactory(ctx, fakegitiles.Factory(map[string]*fakegitiles.Host{
			"chromium.googlesource.com": {
				Projects: map[string]*fakegitiles.Project{
					"top/level": topLevelGitiles,
				},
			},
		}))

		topLevelGerrit := &fakegerrit.Project{
			Changes: map[int64]*fakegerrit.Change{},
		}
		ctx = gerrit.UseGerritClientFactory(ctx, fakegerrit.Factory(map[string]*fakegerrit.Host{
			"chromium-review.googlesource.com": {
				Projects: map[string]*fakegerrit.Project{
					"top/level": topLevelGerrit,
				},
			},
		}))

		bootstrapper := NewPropertyBootstrapper(gitiles.NewClient(ctx), gerrit.NewClient(ctx))

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
					topLevelGitiles.Refs["refs/heads/top-level"] = ""

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldNotBeNil)
					So(properties, ShouldBeNil)
				})

				Convey("if unable to get file", func() {
					input := getInput(build)

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldNotBeNil)
					So(properties, ShouldBeNil)
				})

				Convey("if unable to get target ref of change", func() {
					build.Input.GerritChanges = append(build.Input.GerritChanges, &buildbucketpb.GerritChange{
						Host:     "chromium-review.googlesource.com",
						Project:  "top/level",
						Change:   2345,
						Patchset: 1,
					})
					topLevelGerrit.Changes[2345] = nil
					input := getInput(build)

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldErrLike, "failed to get target ref for config change")
					So(properties, ShouldBeNil)
				})

				Convey("if unable to get files affected by patchset", func() {
					build.Input.GerritChanges = append(build.Input.GerritChanges, &buildbucketpb.GerritChange{
						Host:     "chromium-review.googlesource.com",
						Project:  "top/level",
						Change:   2345,
						Patchset: 2, // non-existent patchset
					})
					input := getInput(build)

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldErrLike, "failed to determine if properties file infra/config/fake-bucket/fake-builder/properties.textpb was affected")
					So(properties, ShouldBeNil)
				})

				Convey("if the properties file is invalid", func() {
					input := getInput(build)
					topLevelGitiles.Refs["refs/heads/top-level"] = "top-level-top-level-head"
					topLevelGitiles.Revisions["top-level-top-level-head"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(""),
						},
					}

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldNotBeNil)
					So(properties, ShouldBeNil)
				})

				Convey("if unable to get diff for properties file", func() {
					build.Input.GerritChanges = append(build.Input.GerritChanges, &buildbucketpb.GerritChange{
						Host:     "chromium-review.googlesource.com",
						Project:  "top/level",
						Change:   2345,
						Patchset: 1,
					})
					topLevelGerrit.Changes[2345] = &fakegerrit.Change{
						Ref: "top-level-some-branch-head",
						Patchsets: map[int32]*fakegerrit.Patchset{
							1: {
								Revision: "cl-revision",
								AffectedFiles: map[string]*fakegerrit.AffectedFile{
									"infra/config/fake-bucket/fake-builder/properties.textpb": nil,
								},
							},
						},
					}
					topLevelGitiles.Refs["top-level-some-branch-head"] = "top-level-some-branch-head-revision"
					topLevelGitiles.Revisions["top-level-some-branch-head-revision"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr("{}"),
						},
					}
					topLevelGitiles.Revisions["cl-revision"] = &fakegitiles.Revision{
						Parent: "non-existent-base",
					}
					topLevelGitiles.Revisions["non-existent-base"] = nil
					input := getInput(build)

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldErrLike, "failed to get diff for infra/config/fake-bucket/fake-builder/properties.textpb")
					So(properties, ShouldBeNil)
				})

				Convey("if patch for properties file does not apply", func() {
					build.Input.GerritChanges = append(build.Input.GerritChanges, &buildbucketpb.GerritChange{
						Host:     "chromium-review.googlesource.com",
						Project:  "top/level",
						Change:   2345,
						Patchset: 1,
					})
					topLevelGerrit.Changes[2345] = &fakegerrit.Change{
						Ref: "top-level-some-branch-head",
						Patchsets: map[int32]*fakegerrit.Patchset{
							1: {
								Revision: "cl-revision",
								AffectedFiles: map[string]*fakegerrit.AffectedFile{
									"infra/config/fake-bucket/fake-builder/properties.textpb": nil,
								},
							},
						},
					}
					topLevelGitiles.Refs["top-level-some-branch-head"] = "top-level-some-branch-head-revision"
					topLevelGitiles.Revisions["top-level-some-branch-head-revision"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
								"test_property": "foo"
							}`),
						},
					}
					topLevelGitiles.Revisions["cl-revision"] = &fakegitiles.Revision{
						Parent: "cl-base",
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
								"test_property": "bar"
							}`),
						},
					}
					topLevelGitiles.Revisions["cl-base"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr("{}"),
						},
					}
					input := getInput(build)

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldNotBeNil)
					So(PatchRejected.In(err), ShouldBeTrue)
					So(properties, ShouldBeNil)
				})

			})

			Convey("returns properties", func() {

				Convey("with properties from the properties file", func() {
					topLevelGitiles.Refs["refs/heads/top-level"] = "top-level-top-level-head"
					topLevelGitiles.Revisions["top-level-top-level-head"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
								"$build/baz": {
									"quux": "quuz"
								},
								"foo": "bar"
							}`),
						},
					}
					input := getInput(build)

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldBeNil)
					So(getValueAtPath(properties, "$build/baz"), ShouldResembleProtoJSON, `{"quux": "quuz"}`)
					So(getValueAtPath(properties, "foo"), ShouldResembleProtoJSON, `"bar"`)
					So(properties.Fields, ShouldNotContainKey, "$bootstrap")
				})

				Convey("with build properties merged with properties from the properties file", func() {
					topLevelGitiles.Refs["refs/heads/top-level"] = "top-level-top-level-head"
					topLevelGitiles.Revisions["top-level-top-level-head"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
								"$build/baz": {
									"quux": "quuz"
								},
								"foo": "bar"
							}`),
						},
					}
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

				Convey("with patch for modified properties file applied", func() {
					build.Input.GerritChanges = append(build.Input.GerritChanges, &buildbucketpb.GerritChange{
						Host:     "chromium-review.googlesource.com",
						Project:  "top/level",
						Change:   2345,
						Patchset: 1,
					})
					topLevelGerrit.Changes[2345] = &fakegerrit.Change{
						Ref: "refs/heads/some-branch",
						Patchsets: map[int32]*fakegerrit.Patchset{
							1: {
								Revision: "cl-revision",
								AffectedFiles: map[string]*fakegerrit.AffectedFile{
									"infra/config/fake-bucket/fake-builder/properties.textpb": nil,
								},
							},
						},
					}
					topLevelGitiles.Refs["refs/heads/some-branch"] = "top-level-some-branch-head"
					topLevelGitiles.Revisions["top-level-some-branch-head"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
									"test_property": "some-branch-head-value"
								}`),
						},
					}
					topLevelGitiles.Revisions["cl-base"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
									"test_property": "some-branch-head-value"
								}`),
						},
					}
					topLevelGitiles.Revisions["cl-revision"] = &fakegitiles.Revision{
						Parent: "cl-base",
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
									"test_property": "some-branch-head-value",
									"test_property2": "some-branch-head-value2"
								}`),
						},
					}
					input := getInput(build)

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, input)

					So(err, ShouldBeNil)
					So(getValueAtPath(properties, "test_property"), ShouldResembleProtoJSON, `"some-branch-head-value"`)
					So(getValueAtPath(properties, "test_property2"), ShouldResembleProtoJSON, `"some-branch-head-value2"`)
				})

				Convey("for top level project with commits in $build/chromium_bootstrap property", func() {

					Convey("for commit ref when build has gitiles commit without ID for top level project", func() {
						build.Input.GitilesCommit = &buildbucketpb.GitilesCommit{
							Host:    "chromium.googlesource.com",
							Project: "top/level",
							Ref:     "refs/heads/some-branch",
						}
						topLevelGitiles.Refs["refs/heads/some-branch"] = "top-level-some-branch-head"
						topLevelGitiles.Revisions["top-level-some-branch-head"] = &fakegitiles.Revision{
							Files: map[string]*string{
								"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{"test_property": "some-branch-head-value"}`),
							},
						}
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
						topLevelGitiles.Revisions["some-branch-revision"] = &fakegitiles.Revision{
							Files: map[string]*string{
								"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{"test_property": "some-branch-revision-value"}`),
							},
						}
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

					Convey("for target ref when build has gerrit change for top level project", func() {
						build.Input.GerritChanges = append(build.Input.GerritChanges, &buildbucketpb.GerritChange{
							Host:     "chromium-review.googlesource.com",
							Project:  "top/level",
							Change:   2345,
							Patchset: 1,
						})
						topLevelGerrit.Changes[2345] = &fakegerrit.Change{
							Ref: "refs/heads/some-branch",
							Patchsets: map[int32]*fakegerrit.Patchset{
								1: {
									AffectedFiles: map[string]*fakegerrit.AffectedFile{
										"foo/bar": nil,
									},
								},
							},
						}
						topLevelGitiles.Refs["refs/heads/some-branch"] = "top-level-some-branch-head"
						topLevelGitiles.Revisions["top-level-some-branch-head"] = &fakegitiles.Revision{
							Files: map[string]*string{
								"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
									"test_property": "some-branch-head-value"
								}`),
							},
						}
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

					Convey("for top level ref when build does not have gitiles commit or gerrit change for top level project", func() {
						build.Input.GitilesCommit = &buildbucketpb.GitilesCommit{
							Host:    "chromium.googlesource.com",
							Project: "unrelated",
							Ref:     "refs/heads/irrelevant",
						}
						input := getInput(build)
						topLevelGitiles.Refs["refs/heads/top-level"] = "top-level-top-level-head"
						topLevelGitiles.Revisions["top-level-top-level-head"] = &fakegitiles.Revision{
							Files: map[string]*string{
								"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{"test_property": "top-level-head-value"}`),
							},
						}

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
