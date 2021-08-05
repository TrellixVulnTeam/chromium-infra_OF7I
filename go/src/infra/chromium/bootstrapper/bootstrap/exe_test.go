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
)

func TestExeBootstrapper(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("ExeBootstrapper", t, func() {
		build := &buildbucketpb.Build{
			Input: &buildbucketpb.Build_Input{
				Properties: &structpb.Struct{},
			},
		}

		pkg := &fakecipd.Package{
			Refs:      map[string]string{},
			Instances: map[string]*fakecipd.PackageInstance{},
		}
		ctx := cipd.UseCipdClientFactory(ctx, fakecipd.Factory(map[string]*fakecipd.Package{
			"fake-package": pkg,
		}))

		cipdRoot := t.TempDir()
		cipdClient, err := cipd.NewClient(ctx, cipdRoot)
		PanicOnError(err)

		bootstrapper := NewExeBootstrapper(cipdClient)

		Convey("DeployExe", func() {

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
			input := getInput(build)

			Convey("returns the cmd for the executable", func() {
				cmd, err := bootstrapper.DeployExe(ctx, input)

				So(err, ShouldBeNil)
				So(cmd, ShouldResemble, []string{filepath.Join(cipdRoot, "fake-package", "fake-exe")})
			})

		})

	})
}
