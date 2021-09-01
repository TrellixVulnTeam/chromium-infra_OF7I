// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bootstrap

import (
	"context"
	"path/filepath"
	"testing"

	"infra/chromium/bootstrapper/cipd"
	fakecipd "infra/chromium/bootstrapper/fakes/cipd"
	. "infra/chromium/bootstrapper/util"

	structpb "github.com/golang/protobuf/ptypes/struct"
	. "github.com/smartystreets/goconvey/convey"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	. "go.chromium.org/luci/common/testing/assertions"
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

		bootstrapper := NewExeBootstrapper(cipdClient)

		build := &buildbucketpb.Build{
			Input: &buildbucketpb.Build_Input{
				Properties: &structpb.Struct{},
			},
		}

		Convey("without led_recipe_bundle property", func() {
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
					"cipd_version": "fake-ref",
					"cmd": ["fake-exe"]
				}
			}`)
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
				So(exe, ShouldNotBeNil)
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

		cipdRoot := t.TempDir()
		cipdClient, err := cipd.NewClient(ctx, cipdRoot)
		PanicOnError(err)

		bootstrapper := NewExeBootstrapper(cipdClient)

		exe := &BootstrappedExe{
			Source: &BootstrappedExe_Cipd{
				Cipd: &Cipd{
					Server:        "https://chrome-infra-packages.appspot.com",
					Package:       "fake-package",
					ActualVersion: "fake-instance-id",
				},
			},
			Cmd: []string{"fake-exe"},
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
			So(cmd, ShouldResemble, []string{filepath.Join(cipdRoot, "fake-package", "fake-exe")})
		})

	})
}
