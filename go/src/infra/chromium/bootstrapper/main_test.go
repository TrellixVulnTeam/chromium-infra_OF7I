// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"errors"
	"infra/chromium/bootstrapper/cipd"
	fakecipd "infra/chromium/bootstrapper/fakes/cipd"
	fakegitiles "infra/chromium/bootstrapper/fakes/gitiles"
	"infra/chromium/bootstrapper/gitiles"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	. "go.chromium.org/luci/common/testing/assertions"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func strPtr(s string) *string {
	return &s
}

func createInput(buildJson string) io.Reader {
	build := &buildbucketpb.Build{}
	if err := protojson.Unmarshal([]byte(buildJson), build); err != nil {
		panic(err)
	}
	buildProtoBytes, err := proto.Marshal(build)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(buildProtoBytes)
}

type reader struct {
	readFn func([]byte) (int, error)
}

func (r reader) Read(p []byte) (n int, err error) {
	return r.readFn(p)
}

func TestPerformBootstrap(t *testing.T) {
	t.Parallel()

	project := &fakegitiles.Project{
		Refs:  map[string]string{},
		Files: map[fakegitiles.FileRevId]*string{},
	}

	pkg := &fakecipd.Package{
		Refs:      map[string]string{},
		Instances: map[string]*fakecipd.PackageInstance{},
	}

	ctx := context.Background()

	ctx = gitiles.UseGitilesClientFactory(ctx, fakegitiles.Factory(map[string]*fakegitiles.Host{
		"fake-host": {
			Projects: map[string]*fakegitiles.Project{
				"fake-project": project,
			},
		},
	}))

	ctx = cipd.UseCipdClientFactory(ctx, fakecipd.Factory(map[string]*fakecipd.Package{
		"fake-package": pkg,
	}))

	Convey("performBootstrap", t, func() {

		cipdRoot := t.TempDir()

		Convey("fails if reading input fails", func() {
			input := reader{func(p []byte) (int, error) {
				return 0, errors.New("test read failure")
			}}

			cmd, err := performBootstrap(ctx, input, cipdRoot, "")

			So(err, ShouldNotBeNil)
			So(cmd, ShouldBeNil)
		})

		Convey("fails if unmarshalling build fails", func() {
			input := strings.NewReader("invalid-proto")

			cmd, err := performBootstrap(ctx, input, cipdRoot, "")

			So(err, ShouldNotBeNil)
			So(cmd, ShouldBeNil)
		})

		Convey("fails if bootstrap fails", func() {
			input := createInput(`{}`)

			cmd, err := performBootstrap(ctx, input, cipdRoot, "")

			So(err, ShouldNotBeNil)
			So(cmd, ShouldBeNil)
		})

		Convey("succeeds for valid input", func() {
			input := createInput(`{
				"input": {
					"properties": {
						"$bootstrap": {
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
						}
					}
				}
			}`)
			project.Refs["fake-ref"] = "fake-revision"
			project.Files[fakegitiles.FileRevId{
				Revision: "fake-revision",
				Path:     "fake-properties-file",
			}] = strPtr(`{
				"foo": "bar"
			}`)

			cmd, err := performBootstrap(ctx, input, cipdRoot, "fake-output-path")

			So(err, ShouldBeNil)
			So(cmd, ShouldNotBeNil)
			So(cmd.Args, ShouldResemble, []string{
				filepath.Join(cipdRoot, "fake-package", "fake-exe"),
				"--output",
				"fake-output-path",
			})
			contents, _ := ioutil.ReadAll(cmd.Stdin)
			build := &buildbucketpb.Build{}
			proto.Unmarshal(contents, build)
			So(build, ShouldResembleProtoJSON, `{
				"input": {
					"properties": {
						"$build/chromium_bootstrap": {
							"commits": [
								{
									"host": "fake-host",
									"project": "fake-project",
									"ref": "fake-ref",
									"id": "fake-revision"
								}
							]
						},
						"foo": "bar"
					}
				}
			}`)
		})

	})
}
