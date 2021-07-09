// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	"fmt"
	"infra/chromium/bootstrapper/cipd"
	fakecipd "infra/chromium/bootstrapper/fakes/cipd"
	fakegitiles "infra/chromium/bootstrapper/fakes/gitiles"
	"infra/chromium/bootstrapper/gitiles"
	"path/filepath"
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"
	. "github.com/smartystreets/goconvey/convey"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/luciexe/exe"
	"google.golang.org/protobuf/encoding/protojson"
)

func strPtr(s string) *string {
	return &s
}

func setPropertiesFromJson(build *buildbucketpb.Build, propsJson map[string]string) {
	props := make(map[string]interface{}, len(propsJson))
	for key, p := range propsJson {
		s := &structpb.Value{}
		if err := protojson.Unmarshal([]byte(p), s); err != nil {
			panic(err)
		}
		props[key] = s
	}
	if err := exe.WriteProperties(build.Input.Properties, props); err != nil {
		panic(err)
	}
}

func setBootstrapProperties(build *buildbucketpb.Build, propsJson string) {
	setPropertiesFromJson(build, map[string]string{
		"$bootstrap": propsJson,
	})
}

func getBootstrapper(build *buildbucketpb.Build) *Bootstrapper {
	bootstrapper, err := NewBootstrapper(build)
	if err != nil {
		panic(err)
	}
	return bootstrapper
}

func getValueAtPath(s *structpb.Struct, path ...string) *structpb.Value {
	if len(path) < 1 {
		panic("at least one path element must be provided")
	}
	original := s
	for i, p := range path[:len(path)-1] {
		value, ok := s.Fields[p]
		if !ok {
			panic(fmt.Sprintf("path %s is not present in struct %v", path[:i+1], original))
		}
		s = value.GetStructValue()
		if s == nil {
			panic(fmt.Sprintf("path %s is not present in struct %v", path[:i+2], original))
		}
	}
	value, ok := s.Fields[path[len(path)-1]]
	if !ok {
		panic(fmt.Sprintf("path %s is not present in struct %v", path, original))
	}
	return value
}

func TestBootstrapper(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("Bootstrapper", t, func() {
		build := &buildbucketpb.Build{
			Input: &buildbucketpb.Build_Input{
				Properties: &structpb.Struct{},
			},
		}

		Convey("NewBootstrapper", func() {

			Convey("fails for missing $bootstrap", func() {
				bootstrapper, err := NewBootstrapper(build)

				So(err, ShouldErrLike, "none of the config_project fields in $bootstrap is set")
				So(bootstrapper, ShouldBeNil)
			})

			Convey("fails for incorrectly typed $bootstrap", func() {
				setBootstrapProperties(build, `{"foo": "bar"}`)

				bootstrapper, err := NewBootstrapper(build)

				So(err, ShouldErrLike, `unknown field "foo"`)
				So(bootstrapper, ShouldBeNil)
			})

			Convey("fails for invalid $bootstrap", func() {
				setBootstrapProperties(build, "{}")

				bootstrapper, err := NewBootstrapper(build)

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

				bootstrapper, err := NewBootstrapper(build)

				So(err, ShouldBeNil)
				So(bootstrapper, ShouldNotBeNil)
			})

		})

		Convey("ComputeBootstrappedProperties", func() {
			topLevelProject := &fakegitiles.Project{
				Refs:  map[string]string{},
				Files: map[fakegitiles.FileRevId]*string{},
			}

			ctx := gitiles.UseGitilesClientFactory(ctx, fakegitiles.Factory(map[string]*fakegitiles.Host{
				"chromium.googlesource.com": {
					Projects: map[string]*fakegitiles.Project{
						"top/level": topLevelProject,
					},
				},
			}))
			gitilesClient := gitiles.NewClient(ctx)

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
					bootstrapper := getBootstrapper(build)
					topLevelProject.Refs["refs/heads/top-level"] = ""

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, gitilesClient)

					So(err, ShouldNotBeNil)
					So(properties, ShouldBeNil)
				})

				Convey("if unable to get file", func() {
					bootstrapper := getBootstrapper(build)
					topLevelProject.Refs["refs/heads/top-level"] = "top-level-top-level-head"
					topLevelProject.Files[fakegitiles.FileRevId{
						Revision: "top-level-top-level-head",
						Path:     "infra/config/fake-bucket/fake-builder/properties.textpb",
					}] = nil

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, gitilesClient)

					So(err, ShouldNotBeNil)
					So(properties, ShouldBeNil)
				})

				Convey("if the properties file is invalid", func() {
					bootstrapper := getBootstrapper(build)
					topLevelProject.Refs["refs/heads/top-level"] = "top-level-top-level-head"
					topLevelProject.Files[fakegitiles.FileRevId{
						Revision: "top-level-top-level-head",
						Path:     "infra/config/fake-bucket/fake-builder/properties.textpb",
					}] = strPtr("")

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, gitilesClient)

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
					bootstrapper := getBootstrapper(build)

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, gitilesClient)

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
					bootstrapper := getBootstrapper(build)

					properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, gitilesClient)

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
						bootstrapper := getBootstrapper(build)

						properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, gitilesClient)

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
						bootstrapper := getBootstrapper(build)

						properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, gitilesClient)

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
						bootstrapper := getBootstrapper(build)
						topLevelProject.Refs["refs/heads/top-level"] = "top-level-top-level-head"
						topLevelProject.Files[fakegitiles.FileRevId{
							Revision: "top-level-top-level-head",
							Path:     "infra/config/fake-bucket/fake-builder/properties.textpb",
						}] = strPtr(`{"test_property": "top-level-head-value"}`)

						properties, err := bootstrapper.ComputeBootstrappedProperties(ctx, gitilesClient)

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

		Convey("SetupExe", func() {

			pkg := &fakecipd.Package{
				Refs:      map[string]string{},
				Instances: map[string]*fakecipd.PackageInstance{},
			}

			cipdRoot := t.TempDir()
			ctx := cipd.UseCipdClientFactory(ctx, fakecipd.Factory(map[string]*fakecipd.Package{
				"fake-package": pkg,
			}))
			recipeClient, err := cipd.NewClient(ctx, cipdRoot)
			if err != nil {
				panic(err)
			}

			setBootstrapProperties(build, `{
				"top_level_project": {
					"repo": {
						"host": "fake-host",
						"project": "fake-project"
					},
					"ref": "fake-ref"
				},
				"properties_file": "fake-properties-file",
				"exe": {
					"cipd_package": "fake-package",
					"cipd_version": "fake-version",
					"cmd": ["fake-exe"]
				}
		}`)
			bootstrapper := getBootstrapper(build)

			Convey("returns the cmd for the executable", func() {
				cmd, err := bootstrapper.SetupExe(ctx, recipeClient)

				So(err, ShouldBeNil)
				So(cmd, ShouldResemble, []string{filepath.Join(cipdRoot, "fake-package", "fake-exe")})
			})

		})

	})
}
