// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"infra/chromium/bootstrapper/bootstrap"
	"infra/chromium/bootstrapper/cipd"
	fakecipd "infra/chromium/bootstrapper/fakes/cipd"
	fakegitiles "infra/chromium/bootstrapper/fakes/gitiles"
	"infra/chromium/bootstrapper/gitiles"
	. "infra/chromium/util"

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
	PanicOnError(protojson.Unmarshal([]byte(buildJson), build))
	buildProtoBytes, err := proto.Marshal(build)
	PanicOnError(err)
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
		Refs:      map[string]string{},
		Revisions: map[string]*fakegitiles.Revision{},
	}

	pkg := &fakecipd.Package{
		Refs:      map[string]string{},
		Instances: map[string]*fakecipd.PackageInstance{},
	}

	opts := options{
		outputPath: "fake-output-path",
		cipdRoot:   "fake-cipd-root",
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

		Convey("fails if reading input fails", func() {
			input := reader{func(p []byte) (int, error) {
				return 0, errors.New("test read failure")
			}}

			cmd, exeInput, err := performBootstrap(ctx, input, opts)

			So(err, ShouldNotBeNil)
			So(cmd, ShouldBeNil)
			So(exeInput, ShouldBeNil)
		})

		Convey("fails if unmarshalling build fails", func() {
			input := strings.NewReader("invalid-proto")

			cmd, exeInput, err := performBootstrap(ctx, input, opts)

			So(err, ShouldNotBeNil)
			So(cmd, ShouldBeNil)
			So(exeInput, ShouldBeNil)
		})

		Convey("fails if bootstrap fails", func() {
			input := createInput(`{}`)

			cmd, exeInput, err := performBootstrap(ctx, input, opts)

			So(err, ShouldNotBeNil)
			So(cmd, ShouldBeNil)
			So(exeInput, ShouldBeNil)
		})

		input := createInput(`{
			"input": {
				"properties": {
					"$bootstrap/properties": {
						"top_level_project": {
							"repo": {
								"host": "fake-host",
								"project": "fake-project"
							},
							"ref": "fake-ref"
						},
						"properties_file": "fake-properties-file"
					},
					"$bootstrap/exe": {
						"exe": {
							"cipd_package": "fake-package",
							"cipd_version": "fake-version",
							"cmd": ["fake-exe"]
						}
					}
				}
			}
		}`)

		Convey("fails if determining executable fails", func() {
			project.Refs["fake-ref"] = "fake-revision"
			project.Revisions["fake-revision"] = &fakegitiles.Revision{
				Files: map[string]*string{
					"fake-properties-file": strPtr(`{
						"foo": "bar"
					}`),
				},
			}
			pkg.Refs["fake-version"] = ""

			cmd, exeInput, err := performBootstrap(ctx, input, opts)

			So(err, ShouldNotBeNil)
			So(cmd, ShouldBeNil)
			So(exeInput, ShouldBeNil)
		})

		Convey("succeeds for valid input", func() {
			project.Refs["fake-ref"] = "fake-revision"
			project.Revisions["fake-revision"] = &fakegitiles.Revision{
				Files: map[string]*string{
					"fake-properties-file": strPtr(`{
						"foo": "bar"
					}`),
				},
			}
			pkg.Refs["fake-version"] = "fake-instance-id"

			cmd, exeInput, err := performBootstrap(ctx, input, opts)

			So(err, ShouldBeNil)
			So(cmd, ShouldResemble, []string{
				filepath.Join(opts.cipdRoot, "bootstrapped-exe", "fake-exe"),
				"--output",
				opts.outputPath,
			})
			build := &buildbucketpb.Build{}
			proto.Unmarshal(exeInput, build)
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
									"server": "https://chrome-infra-packages.appspot.com",
									"package": "fake-package",
									"requested_version": "fake-version",
									"actual_version": "fake-instance-id"
								},
								"cmd": ["fake-exe"]
							}
						},
						"foo": "bar"
					}
				}
			}`)
		})

		Convey("succeeds for properties-optional without $bootstrap/properties", func() {
			input := createInput(`{
				"input": {
					"properties": {
						"$bootstrap/exe": {
							"exe": {
								"cipd_package": "fake-package",
								"cipd_version": "fake-version",
								"cmd": ["fake-exe"]
							}
						}
					}
				}
			}`)
			opts.propertiesOptional = true

			cmd, exeInput, err := performBootstrap(ctx, input, opts)

			So(err, ShouldBeNil)
			So(cmd, ShouldResemble, []string{
				filepath.Join(opts.cipdRoot, "bootstrapped-exe", "fake-exe"),
				"--output",
				opts.outputPath,
			})
			build := &buildbucketpb.Build{}
			proto.Unmarshal(exeInput, build)
			So(build, ShouldResembleProtoJSON, `{
				"input": {
					"properties": {
						"$build/chromium_bootstrap": {
							"exe": {
								"cipd": {
									"server": "https://chrome-infra-packages.appspot.com",
									"package": "fake-package",
									"requested_version": "fake-version",
									"actual_version": "fake-instance-id"
								},
								"cmd": ["fake-exe"]
							}
						}
					}
				}
			}`)
		})

	})
}

func testBootstrapFn(bootstrapErr error) bootstrapFn {
	return func(ctx context.Context, input io.Reader, opts options) ([]string, []byte, error) {
		if bootstrapErr != nil {
			return nil, nil, bootstrapErr
		}
		return []string{"fake", "command"}, []byte("fake-contents"), nil
	}
}

func testExecuteCmdFn(cmdErr error) executeCmdFn {
	return func(ctx context.Context, cmd []string, input []byte) error {
		if cmdErr != nil {
			return cmdErr
		}
		return nil
	}
}

type buildUpdateRecord struct {
	build *buildbucketpb.Build
}

func testUpdateBuildFn(updateErr error) (*buildUpdateRecord, updateBuildFn) {
	update := &buildUpdateRecord{}
	return update, func(ctx context.Context, build *buildbucketpb.Build) error {
		update.build = build
		if updateErr != nil {
			return updateErr
		}
		return nil
	}
}

func TestBootstrapMain(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	Convey("bootstrapMain", t, func() {

		getOptions := func() options { return options{} }
		performBootstrap := testBootstrapFn(nil)
		execute := testExecuteCmdFn(nil)
		record, updateBuild := testUpdateBuildFn(nil)

		Convey("does not update build on success", func() {
			err := bootstrapMain(ctx, getOptions, performBootstrap, execute, updateBuild)

			So(err, ShouldBeNil)
			So(record.build, ShouldBeNil)
		})

		Convey("does not update build on exe failure", func() {
			cmdErr := &exec.ExitError{
				ProcessState: &os.ProcessState{},
				Stderr:       []byte("test cmd failure"),
			}
			execute := testExecuteCmdFn(cmdErr)

			err := bootstrapMain(ctx, getOptions, performBootstrap, execute, updateBuild)

			So(err, ShouldErrLike, cmdErr)
			So(record.build, ShouldBeNil)
		})

		Convey("updates build when failing to execute cmd", func() {
			cmdErr := errors.New("test cmd execution failure")
			execute := testExecuteCmdFn(cmdErr)

			err := bootstrapMain(ctx, getOptions, performBootstrap, execute, updateBuild)

			So(err, ShouldErrLike, cmdErr)
			So(record.build, ShouldResembleProtoJSON, `{
				"status": "INFRA_FAILURE",
				"summary_markdown": "<pre>test cmd execution failure</pre>"
			}`)
		})

		Convey("updates build for bootstrap failure", func() {
			bootstrapErr := errors.New("test bootstrap failure")
			performBootstrap := testBootstrapFn(bootstrapErr)

			err := bootstrapMain(ctx, getOptions, performBootstrap, execute, updateBuild)

			So(err, ShouldErrLike, bootstrapErr)
			So(record.build, ShouldResembleProtoJSON, `{
				"status": "INFRA_FAILURE",
				"summary_markdown": "<pre>test bootstrap failure</pre>"
			}`)

			Convey("with failure_type set for patch rejected failure", func() {
				bootstrapErr := bootstrap.PatchRejected.Apply(bootstrapErr)
				performBootstrap := testBootstrapFn(bootstrapErr)

				err := bootstrapMain(ctx, getOptions, performBootstrap, execute, updateBuild)

				So(err, ShouldErrLike, bootstrapErr)
				So(record.build, ShouldResembleProtoJSON, `{
					"status": "INFRA_FAILURE",
					"summary_markdown": "<pre>test bootstrap failure</pre>",
					"output": {
						"properties": {
							"failure_type": "PATCH_FAILURE"
						}
					}
				}`)
			})

		})

		Convey("returns original error if updating build fails", func() {
			bootstrapErr := errors.New("test bootstrap failure")
			performBootstrap := testBootstrapFn(bootstrapErr)
			updateBuildErr := errors.New("test update build failure")
			_, updateBuild := testUpdateBuildFn(updateBuildErr)

			err := bootstrapMain(ctx, getOptions, performBootstrap, execute, updateBuild)

			So(err, ShouldErrLike, bootstrapErr)
		})

	})
}
