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
	"infra/chromium/util"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestGetBootstrapConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("BuildBootstrapper.GetBootstrapConfig", t, func() {

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

		bootstrapper := NewBuildBootstrapper(gitiles.NewClient(ctx), gerrit.NewClient(ctx))

		Convey("fails", func() {

			Convey("if unable to get revision", func() {
				input := getInput(build)
				topLevelGitiles.Refs["refs/heads/top-level"] = ""

				properties, err := bootstrapper.GetBootstrapConfig(ctx, input)

				So(err, ShouldNotBeNil)
				So(properties, ShouldBeNil)
			})

			Convey("if unable to get file", func() {
				input := getInput(build)

				properties, err := bootstrapper.GetBootstrapConfig(ctx, input)

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

				properties, err := bootstrapper.GetBootstrapConfig(ctx, input)

				So(err, ShouldErrLike, "failed to get target ref for config change")
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

				properties, err := bootstrapper.GetBootstrapConfig(ctx, input)

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

				properties, err := bootstrapper.GetBootstrapConfig(ctx, input)

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

				properties, err := bootstrapper.GetBootstrapConfig(ctx, input)

				So(err, ShouldNotBeNil)
				So(PatchRejected.In(err), ShouldBeTrue)
				So(properties, ShouldBeNil)
			})

		})

		Convey("returns config", func() {

			Convey("with buildProperties from input", func() {
				topLevelGitiles.Refs["refs/heads/top-level"] = "top-level-top-level-head"
				topLevelGitiles.Revisions["top-level-top-level-head"] = &fakegitiles.Revision{
					Files: map[string]*string{
						"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{}`),
					},
				}
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
				build.Input.Properties.Fields["test_property"] = structpb.NewStringValue("foo")
				input := getInput(build)

				config, err := bootstrapper.GetBootstrapConfig(ctx, input)

				So(err, ShouldBeNil)
				So(config.buildProperties, ShouldResembleProtoJSON, `{
					"test_property": "foo"
				}`)
			})

			Convey("for top-level project", func() {

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

				Convey("returns config with properties from top level ref when no commit or change for project", func() {
					topLevelGitiles.Refs["refs/heads/top-level"] = "top-level-top-level-head"
					topLevelGitiles.Revisions["top-level-top-level-head"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
								"test_property": "top-level-head-value"
							}`),
						},
					}
					input := getInput(build)

					config, err := bootstrapper.GetBootstrapConfig(ctx, input)

					So(err, ShouldBeNil)
					So(config.commit.GitilesCommit, ShouldResembleProtoJSON, `{
						"host": "chromium.googlesource.com",
						"project": "top/level",
						"ref": "refs/heads/top-level",
						"id": "top-level-top-level-head"
					}`)
					So(config.change, ShouldBeNil)
					So(config.builderProperties, ShouldResembleProtoJSON, `{
						"test_property": "top-level-head-value"
					}`)
				})

				Convey("returns config with properties from commit ref when commit for project without ID", func() {
					build.Input.GitilesCommit = &buildbucketpb.GitilesCommit{
						Host:    "chromium.googlesource.com",
						Project: "top/level",
						Ref:     "refs/heads/some-branch",
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

					config, err := bootstrapper.GetBootstrapConfig(ctx, input)

					So(err, ShouldBeNil)
					So(config.commit.GitilesCommit, ShouldResembleProtoJSON, `{
						"host": "chromium.googlesource.com",
						"project": "top/level",
						"ref": "refs/heads/some-branch",
						"id": "top-level-some-branch-head"
					}`)
					So(config.change, ShouldBeNil)
					So(config.builderProperties, ShouldResembleProtoJSON, `{
						"test_property": "some-branch-head-value"
					}`)
				})

				Convey("returns config with properties from commit revision when commit for project with ID", func() {
					build.Input.GitilesCommit = &buildbucketpb.GitilesCommit{
						Host:    "chromium.googlesource.com",
						Project: "top/level",
						Ref:     "refs/heads/some-branch",
						Id:      "some-branch-revision",
					}
					topLevelGitiles.Revisions["some-branch-revision"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
								"test_property": "some-branch-revision-value"
							}`),
						},
					}
					input := getInput(build)

					config, err := bootstrapper.GetBootstrapConfig(ctx, input)

					So(err, ShouldBeNil)
					So(config.commit.GitilesCommit, ShouldResembleProtoJSON, `{
						"host": "chromium.googlesource.com",
						"project": "top/level",
						"ref": "refs/heads/some-branch",
						"id": "some-branch-revision"
					}`)
					So(config.change, ShouldBeNil)
					So(config.builderProperties, ShouldResembleProtoJSON, `{
						"test_property": "some-branch-revision-value"
					}`)
				})

				Convey("returns config with properties from target ref and patch applied when change for project", func() {
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
							},
						},
					}
					topLevelGitiles.Refs["refs/heads/some-branch"] = "top-level-some-branch-head"
					topLevelGitiles.Revisions["top-level-some-branch-head"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
								"test_property": "some-branch-head-value",
								"test_property2": "some-branch-head-value2",
								"test_property3": "some-branch-head-value3",
								"test_property4": "some-branch-head-value4",
								"test_property5": "some-branch-head-value5"
							}`),
						},
					}
					topLevelGitiles.Revisions["cl-base"] = &fakegitiles.Revision{
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
								"test_property": "some-branch-head-value",
								"test_property2": "some-branch-head-value2",
								"test_property3": "some-branch-head-value3",
								"test_property4": "some-branch-head-value4",
								"test_property5": "some-branch-head-old-value5"
							}`),
						},
					}
					topLevelGitiles.Revisions["cl-revision"] = &fakegitiles.Revision{
						Parent: "cl-base",
						Files: map[string]*string{
							"infra/config/fake-bucket/fake-builder/properties.textpb": strPtr(`{
								"test_property": "some-branch-head-new-value",
								"test_property2": "some-branch-head-value2",
								"test_property3": "some-branch-head-value3",
								"test_property4": "some-branch-head-value4",
								"test_property5": "some-branch-head-old-value5"
							}`),
						},
					}
					input := getInput(build)

					config, err := bootstrapper.GetBootstrapConfig(ctx, input)

					So(err, ShouldBeNil)
					So(config.commit.GitilesCommit, ShouldResembleProtoJSON, `{
						"host": "chromium.googlesource.com",
						"project": "top/level",
						"ref": "refs/heads/some-branch",
						"id": "top-level-some-branch-head"
					}`)
					So(config.change.GerritChange, ShouldResembleProtoJSON, `{
						"host": "chromium-review.googlesource.com",
						"project": "top/level",
						"change": 2345,
						"patchset": 1
					}`)
					So(config.builderProperties, ShouldResembleProtoJSON, `{
						"test_property": "some-branch-head-new-value",
						"test_property2": "some-branch-head-value2",
						"test_property3": "some-branch-head-value3",
						"test_property4": "some-branch-head-value4",
						"test_property5": "some-branch-head-value5"
					}`)
					So(config.skipAnalysisReasons, ShouldResemble, []string{
						"properties file infra/config/fake-bucket/fake-builder/properties.textpb is affected by CL",
					})
				})

			})

			Convey("for properties-optional bootstrapping", func() {
				inputOpts := InputOptions{PropertiesOptional: true}
				delete(build.Input.Properties.Fields, "$bootstrap/properties")
				input, err := inputOpts.NewInput(build)
				util.PanicOnError(err)

				config, err := bootstrapper.GetBootstrapConfig(ctx, input)

				So(err, ShouldBeNil)
				So(config.commit, ShouldBeNil)
				So(config.change, ShouldBeNil)
				So(config.builderProperties, ShouldBeNil)
			})
		})

	})
}

func TestUpdateBuild(t *testing.T) {
	t.Parallel()

	Convey("BootstrapConfig.UpdateBuild", t, func() {

		Convey("updates build with gitiles commit, builder properties, $build/chromium_bootstrap module properties and build properties", func() {
			config := &BootstrapConfig{
				commit: &gitilesCommit{&buildbucketpb.GitilesCommit{
					Host:    "fake-host",
					Project: "fake-project",
					Ref:     "fake-ref",
					Id:      "fake-revision",
				}},
				buildProperties: jsonToStruct(`{
					"foo": "build-foo-value",
					"bar": "build-bar-value"
				}`),
				builderProperties: jsonToStruct(`{
					"foo": "builder-foo-value",
					"baz": "builder-baz-value"
				}`),
				skipAnalysisReasons: []string{
					"skip-analysis-reason1",
					"skip-analysis-reason2",
				},
			}
			exe := &BootstrappedExe{
				Source: &BootstrappedExe_Cipd{
					Cipd: &Cipd{
						Server:           "fake-cipd-server",
						Package:          "fake-cipd-package",
						RequestedVersion: "fake-cipd-ref",
						ActualVersion:    "fake-cipd-instance-id",
					},
				},
				Cmd: []string{"fake-exe"},
			}
			build := &buildbucketpb.Build{
				Input: &buildbucketpb.Build_Input{
					GitilesCommit: &buildbucketpb.GitilesCommit{
						Host:    "fake-host",
						Project: "fake-project",
						Ref:     "fake-ref",
					},
				},
			}

			err := config.UpdateBuild(build, exe)

			So(err, ShouldBeNil)
			So(build, ShouldResembleProtoJSON, `{
				"input": {
					"gitiles_commit": {
						"host": "fake-host",
						"project": "fake-project",
						"ref": "fake-ref",
						"id": "fake-revision"
					},
					"properties": {
						"$build/chromium_bootstrap": {
							"commits": [
								{
									"host": "fake-host",
									"project": "fake-project",
									"ref": "fake-ref",
									"id": "fake-revision"
								}
							],
							"exe": {
								"cipd": {
									"server": "fake-cipd-server",
									"package": "fake-cipd-package",
									"requested_version": "fake-cipd-ref",
									"actual_version": "fake-cipd-instance-id"
								},
								"cmd": ["fake-exe"]
							},
							"skip_analysis_reasons": [
								"skip-analysis-reason1",
								"skip-analysis-reason2"
							]
						},
						"foo": "build-foo-value",
						"bar": "build-bar-value",
						"baz": "builder-baz-value"
					}
				}
			}`)
		})

		Convey("updates build with $build/chromium_bootstrap module properties and build properties for properties optional bootstrapping", func() {
			config := &BootstrapConfig{
				buildProperties: jsonToStruct(`{
					"foo": "build-foo-value",
					"bar": "build-bar-value"
				}`),
			}
			exe := &BootstrappedExe{
				Source: &BootstrappedExe_Cipd{
					Cipd: &Cipd{
						Server:           "fake-cipd-server",
						Package:          "fake-cipd-package",
						RequestedVersion: "fake-cipd-ref",
						ActualVersion:    "fake-cipd-instance-id",
					},
				},
				Cmd: []string{"fake-exe"},
			}
			build := &buildbucketpb.Build{
				Input: &buildbucketpb.Build_Input{},
			}

			err := config.UpdateBuild(build, exe)

			So(err, ShouldBeNil)
			So(build, ShouldResembleProtoJSON, `{
				"input": {
					"properties": {
						"$build/chromium_bootstrap": {
							"exe": {
								"cipd": {
									"server": "fake-cipd-server",
									"package": "fake-cipd-package",
									"requested_version": "fake-cipd-ref",
									"actual_version": "fake-cipd-instance-id"
								},
								"cmd": ["fake-exe"]
							}
						},
						"foo": "build-foo-value",
						"bar": "build-bar-value"
					}
				}
			}`)
		})

		Convey("does not update gitiles commit for different repo", func() {
			config := &BootstrapConfig{
				commit: &gitilesCommit{&buildbucketpb.GitilesCommit{
					Host:    "fake-host",
					Project: "fake-project",
					Ref:     "fake-ref",
					Id:      "fake-revision",
				}},
				buildProperties: jsonToStruct("{}"),
			}
			exe := &BootstrappedExe{
				Source: &BootstrappedExe_Cipd{
					Cipd: &Cipd{
						Server:           "fake-cipd-server",
						Package:          "fake-cipd-package",
						RequestedVersion: "fake-cipd-ref",
						ActualVersion:    "fake-cipd-instance-id",
					},
				},
				Cmd: []string{"fake-exe"},
			}
			build := &buildbucketpb.Build{
				Input: &buildbucketpb.Build_Input{
					GitilesCommit: &buildbucketpb.GitilesCommit{
						Host:    "fake-host",
						Project: "fake-other-project",
						Ref:     "fake-ref",
					},
				},
			}

			err := config.UpdateBuild(build, exe)

			So(err, ShouldBeNil)
			So(build.Input.GitilesCommit, ShouldResembleProtoJSON, `{
				"host": "fake-host",
				"project": "fake-other-project",
				"ref": "fake-ref"
			}`)

		})

	})

}
