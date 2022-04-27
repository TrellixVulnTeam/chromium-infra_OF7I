// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	"path/filepath"
	"testing"

	"infra/chromium/bootstrapper/cas"
	"infra/chromium/bootstrapper/cipd"
	fakecas "infra/chromium/bootstrapper/fakes/cas"
	fakecipd "infra/chromium/bootstrapper/fakes/cipd"
	. "infra/chromium/util"

	. "github.com/smartystreets/goconvey/convey"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	. "go.chromium.org/luci/common/testing/assertions"
	apipb "go.chromium.org/luci/swarming/proto/api"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestGetBootstrappedExeInfo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("ExeBootstrapper.GetBootstrappedExeInfo", t, func() {
		pkg := &fakecipd.Package{
			Refs: map[string]string{},
		}
		ctx := cipd.UseCipdClientFactory(ctx, fakecipd.Factory(map[string]*fakecipd.Package{
			"fake-package": pkg,
		}))

		cipdRoot := t.TempDir()
		cipdClient, err := cipd.NewClient(ctx, cipdRoot)
		PanicOnError(err)

		bootstrapper := NewExeBootstrapper(cipdClient, nil)

		build := &buildbucketpb.Build{
			Input: &buildbucketpb.Build_Input{
				Properties: &structpb.Struct{},
			},
		}
		setBootstrapPropertiesProperties(build, `{
			"top_level_project": {
				"repo": {
					"host": "fake-host",
					"project": "fake-project"
				},
				"ref": "fake-ref"
			},
			"properties_file": "fake-properties-file"
		}`)
		setBootstrapExeProperties(build, `{
			"exe": {
				"cipd_package": "fake-package",
				"cipd_version": "fake-ref",
				"cmd": ["fake-exe"]
			}
		}`)

		Convey("without led_cas_recipe_bundle property", func() {
			input := getInput(build)

			Convey("fails if resolving version fails", func() {
				pkg.Refs["fake-ref"] = ""

				exe, err := bootstrapper.GetBootstrappedExeInfo(ctx, input)

				So(err, ShouldNotBeNil)
				So(exe, ShouldBeNil)
			})

			Convey("returns info for CIPD bootstrapped exe", func() {
				pkg.Refs["fake-ref"] = "fake-instance-id"

				exe, err := bootstrapper.GetBootstrappedExeInfo(ctx, input)

				So(err, ShouldBeNil)
				So(exe, ShouldResembleProtoJSON, `{
					"cipd": {
						"server": "https://chrome-infra-packages.appspot.com",
						"package": "fake-package",
						"requested_version": "fake-ref",
						"actual_version": "fake-instance-id"
					},
					"cmd": ["fake-exe"]
				}`)
			})

		})

		Convey("with led_cas_recipe_bundle_property returns info for CAS bootstrapped exe", func() {
			build.Input.Properties.Fields["led_cas_recipe_bundle"] = structpb.NewStructValue(jsonToStruct(`{
				"cas_instance": "fake-instance",
				"digest": {
					"hash": "fake-hash",
					"size_bytes": 42
				}
			}`))
			input := getInput(build)

			exe, err := bootstrapper.GetBootstrappedExeInfo(ctx, input)

			So(err, ShouldBeNil)
			So(exe, ShouldResembleProtoJSON, `{
				"cas": {
					"cas_instance": "fake-instance",
					"digest": {
						"hash": "fake-hash",
						"size_bytes": 42
					}
				},
				"cmd": ["fake-exe"]
			}`)
		})

	})
}

func TestDeployExe(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("ExeBootstrapper.DeployExe", t, func() {
		pkg := &fakecipd.Package{
			Instances: map[string]*fakecipd.PackageInstance{},
		}
		ctx := cipd.UseCipdClientFactory(ctx, fakecipd.Factory(map[string]*fakecipd.Package{
			"fake-package": pkg,
		}))

		ctx = cas.UseCasClientFactory(ctx, fakecas.Factory(map[string]*fakecas.Instance{
			"non-existent-instance": nil,
		}))

		execRoot := t.TempDir()

		cipdClient, err := cipd.NewClient(ctx, execRoot)
		PanicOnError(err)

		casClient := cas.NewClient(ctx, execRoot)

		bootstrapper := NewExeBootstrapper(cipdClient, casClient)

		Convey("for CIPD exe", func() {
			exe := &BootstrappedExe{
				Source: &BootstrappedExe_Cipd{
					Cipd: &Cipd{
						Server:        "https://chrome-infra-packages.appspot.com",
						Package:       "fake-package",
						ActualVersion: "fake-instance-id",
					},
				},
				Cmd: []string{"fake-exe", "foo", "bar"},
			}

			Convey("fails if downloading the package fails", func() {
				pkg.Instances["fake-instance-id"] = nil

				cmd, err := bootstrapper.DeployExe(ctx, exe)

				So(err, ShouldNotBeNil)
				So(cmd, ShouldBeNil)
			})

			Convey("returns the cmd for the executable", func() {
				cmd, err := bootstrapper.DeployExe(ctx, exe)

				So(err, ShouldBeNil)
				So(cmd, ShouldResemble, []string{filepath.Join(execRoot, "bootstrapped-exe", "fake-exe"), "foo", "bar"})
			})

		})

		Convey("for CAS exe", func() {
			exe := &BootstrappedExe{
				Source: &BootstrappedExe_Cas{
					Cas: &apipb.CASReference{
						CasInstance: "fake-cas-instance",
						Digest: &apipb.Digest{
							Hash:      "fake-hash",
							SizeBytes: 42,
						},
					},
				},
				Cmd: []string{"fake-exe", "foo", "bar"},
			}

			Convey("returns the cmd for the executable", func() {
				cmd, err := bootstrapper.DeployExe(ctx, exe)

				So(err, ShouldBeNil)
				So(cmd, ShouldResemble, []string{filepath.Join(execRoot, "fake-exe"), "foo", "bar"})
			})
		})

	})
}
