// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/luciexe/exe"
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	bb "infra/chromium/compilator_watcher/internal/bb"
)

type stepNameStatusPair struct {
	stepName string
	status   buildbucket_pb.Status
}

func getSteps(stepPairs []stepNameStatusPair) []*buildbucket_pb.Step {
	steps := make([]*buildbucket_pb.Step, len(stepPairs))

	for i, pair := range stepPairs {
		steps[i] = &buildbucket_pb.Step{Name: pair.stepName, Status: pair.status}
	}
	return steps
}

func getBuildsWithSteps(
	stepPairs []stepNameStatusPair,
	outputFields map[string]*structpb.Value,
	buildStatus buildbucket_pb.Status,
) *buildbucket_pb.Build {
	return &buildbucket_pb.Build{
		Status:          buildStatus,
		Id:              12345,
		SummaryMarkdown: "",
		Steps:           getSteps(stepPairs),
		Output: &buildbucket_pb.Build_Output{
			Properties: &structpb.Struct{
				Fields: outputFields,
			},
		},
	}
}

func TestLuciEXEMain(t *testing.T) {
	t.Parallel()

	Convey("luciEXEMain", t, func() {
		now := time.Date(2021, 01, 01, 00, 00, 00, 00, time.UTC)
		ctx, clk := testclock.UseTime(context.Background(), now)

		clk.SetTimerCallback(func(amt time.Duration, timer clock.Timer) {
			tags := testclock.GetTags(timer)
			for _, tag := range tags {
				if tag == clock.ContextDeadlineTag {
					return
				}
			}
			clk.Add(amt)
		})

		input := &buildbucket_pb.Build{
			Output: &buildbucket_pb.Build_Output{
				Properties: &structpb.Struct{},
			},
		}
		sender := exe.BuildSender(func() {})

		swarmingProps := jsonToStruct(`{
			"all_test_binaries": "33b6bb7fa8f8fad5a82f048b1b2b582411638ae6311ee61d243be4bc82ec7bf8/87",
			"browser_tests": "b2e92f076208f8170d80d73c5723834f7a52af23ca2400a7fe56f3ea39a75713/86"
		}`)

		genericCompBuildOutputProps := jsonToStruct(`{
			"got_angle_revision": "701d51b101c8ce1a1a840a7b0dbe3f36dfc1eec9",
			"got_revision": "04d2ba64ba046c038f8995982ecde0a7f029da1e",
			"got_revision_cp": "refs/heads/main@{#964359}",
			"affected_files": {
				"first_100": ["src/chrome/browser/extensions/extension_message_bubble_controller_unittest.cc"],
				"total_count": 1}
		}`)

		genericCompBuildOutputPropsNoSwarming := copyPropertiesStruct(genericCompBuildOutputProps)
		genericCompBuildOutputPropsWSwarming := copyPropertiesStruct(genericCompBuildOutputProps)
		genericCompBuildOutputPropsWSwarming.GetFields()[swarmingOutputPropKey] = structpb.NewStructValue(swarmingProps)

		genericCompleteSteps := getSteps([]stepNameStatusPair{
			{
				stepName: "builder cache",
				status:   buildbucket_pb.Status_SUCCESS,
			},
			{
				stepName: "lookup GN args",
				status:   buildbucket_pb.Status_SUCCESS,
			},
			{
				stepName: "compile (with patch)",
				status:   buildbucket_pb.Status_SUCCESS,
			},
			{
				stepName: swarmingTriggerPropsStepName,
				status:   buildbucket_pb.Status_SUCCESS,
			},
			{
				stepName: "test_pre_run (with patch)",
				status:   buildbucket_pb.Status_SUCCESS,
			},
			{
				stepName: "check_network_annotations (with patch)",
				status:   buildbucket_pb.Status_SUCCESS,
			},
			{
				stepName: "gerrit changes",
				status:   buildbucket_pb.Status_SUCCESS,
			},
		})

		userArgs := []string{"-compilator-id", "12345", "-get-swarming-trigger-props"}

		Convey("fails if userArgs is empty", func() {
			var userArgs []string
			err := luciEXEMain(ctx, input, userArgs, sender)

			expectedErrText := "compilator-id is required (and 1 other error)"
			So(err, ShouldErrLike, expectedErrText)
			So(
				input.SummaryMarkdown,
				ShouldResemble,
				"Error while running compilator_watcher: "+expectedErrText)
		})
		Convey("fails if userArgs is missing compilator build ID", func() {
			userArgs := []string{"-get-swarming-trigger-props"}
			err := luciEXEMain(ctx, input, userArgs, sender)

			expectedErrText := "compilator-id is required"
			So(err, ShouldErrLike, expectedErrText)
			So(
				input.SummaryMarkdown,
				ShouldResemble,
				"Error while running compilator_watcher: "+expectedErrText)
		})
		Convey("fails if userArgs is missing phase", func() {
			userArgs := []string{"-compilator-id", "12345"}
			err := luciEXEMain(ctx, input, userArgs, sender)

			expectedErrText := "Exactly one of -get-swarming-trigger-props or -get-local-tests is required"
			So(err, ShouldErrLike, expectedErrText)
			So(
				input.SummaryMarkdown,
				ShouldResemble,
				"Error while running compilator_watcher: "+expectedErrText)
		})
		Convey("fails if both phases are passed in", func() {
			userArgs := []string{
				"-compilator-id", "12345", "-get-swarming-trigger-props", "-get-local-tests"}
			err := luciEXEMain(ctx, input, userArgs, sender)

			expectedErrText := "Exactly one of -get-swarming-trigger-props or -get-local-tests is required"
			So(err, ShouldErrLike, expectedErrText)
			So(
				input.SummaryMarkdown,
				ShouldResemble,
				"Error while running compilator_watcher: "+expectedErrText)
		})
		Convey("copies compilator build failure status and summary", func() {
			compBuild := &buildbucket_pb.Build{
				Status:          buildbucket_pb.Status_FAILURE,
				Id:              12345,
				SummaryMarkdown: "Compile failure",
				Output: &buildbucket_pb.Build_Output{
					Properties: &structpb.Struct{},
				},
			}
			ctx = context.WithValue(
				ctx,
				bb.FakeBuildsContextKey,
				[]bb.FakeGetBuildResponse{{Build: compBuild}})
			err := luciEXEMain(ctx, input, userArgs, sender)

			So(err, ShouldBeNil)
			So(input.Status, ShouldResemble, buildbucket_pb.Status_FAILURE)
			So(input.SummaryMarkdown, ShouldResemble, "Compile failure")

		})

		Convey("copies compilator output properties", func() {
			Convey("during swarmingPhase", func() {
				expectedSubBuildOutputProps := copyPropertiesStruct(genericCompBuildOutputPropsWSwarming)

				compBuild := &buildbucket_pb.Build{
					Status:          buildbucket_pb.Status_STARTED,
					Id:              12345,
					SummaryMarkdown: "",
					Steps:           genericCompleteSteps,
					Output: &buildbucket_pb.Build_Output{
						Properties: genericCompBuildOutputPropsWSwarming,
					},
				}

				ctx = context.WithValue(
					ctx,
					bb.FakeBuildsContextKey,
					[]bb.FakeGetBuildResponse{{Build: compBuild}})
				err := luciEXEMain(ctx, input, userArgs, sender)

				So(err, ShouldBeNil)
				So(input.GetOutput(), ShouldResembleProto, &buildbucket_pb.Build_Output{
					Properties: expectedSubBuildOutputProps,
				})
			})
			Convey("during localTestPhase", func() {
				userArgs := []string{"-compilator-id", "12345", "-get-local-tests"}
				expectedSubBuildOutputProps := copyPropertiesStruct(genericCompBuildOutputPropsNoSwarming)

				compBuild := &buildbucket_pb.Build{
					Status:          buildbucket_pb.Status_SUCCESS,
					Id:              12345,
					SummaryMarkdown: "",
					Steps:           genericCompleteSteps,
					Output: &buildbucket_pb.Build_Output{
						Properties: genericCompBuildOutputPropsNoSwarming,
					},
				}

				ctx = context.WithValue(
					ctx,
					bb.FakeBuildsContextKey,
					[]bb.FakeGetBuildResponse{{Build: compBuild}})
				err := luciEXEMain(ctx, input, userArgs, sender)

				So(err, ShouldBeNil)
				So(input.GetOutput(), ShouldResembleProto, &buildbucket_pb.Build_Output{
					Properties: expectedSubBuildOutputProps,
				})
			})
		})

		Convey("sets input Status to SUCCESS when compilator build outputs swarming props but is still running", func() {
			expectedSubBuildOutputProps := copyPropertiesStruct(genericCompBuildOutputPropsWSwarming)

			compBuild := &buildbucket_pb.Build{
				Status:          buildbucket_pb.Status_STARTED,
				Id:              12345,
				SummaryMarkdown: "",
				Steps:           genericCompleteSteps,
				Output:          &buildbucket_pb.Build_Output{Properties: genericCompBuildOutputPropsWSwarming},
			}
			ctx = context.WithValue(
				ctx,
				bb.FakeBuildsContextKey,
				[]bb.FakeGetBuildResponse{{Build: compBuild}})
			err := luciEXEMain(ctx, input, userArgs, sender)

			So(err, ShouldBeNil)
			So(input.Status, ShouldResemble, buildbucket_pb.Status_SUCCESS)
			So(input.GetOutput(), ShouldResembleProto, &buildbucket_pb.Build_Output{
				Properties: expectedSubBuildOutputProps})
		})

		Convey("exits after compilator build successfully ends with no swarming trigger properties", func() {
			expectedSubBuildOutputProps := copyPropertiesStruct(genericCompBuildOutputPropsNoSwarming)

			compBuild := &buildbucket_pb.Build{
				Status:          buildbucket_pb.Status_SUCCESS,
				Id:              12345,
				SummaryMarkdown: "",
				Output: &buildbucket_pb.Build_Output{
					Properties: genericCompBuildOutputPropsNoSwarming,
				},
			}
			ctx = context.WithValue(
				ctx,
				bb.FakeBuildsContextKey,
				[]bb.FakeGetBuildResponse{{Build: compBuild}})
			err := luciEXEMain(ctx, input, userArgs, sender)

			So(err, ShouldBeNil)
			So(input.Status, ShouldResemble, buildbucket_pb.Status_SUCCESS)

			So(input.GetOutput(), ShouldResembleProto, &buildbucket_pb.Build_Output{
				Properties: expectedSubBuildOutputProps,
			})
		})

		Convey("updates last step even if step name is the same", func() {
			compBuilds := []bb.FakeGetBuildResponse{
				{Build: getBuildsWithSteps([]stepNameStatusPair{
					{
						stepName: "lookup GN args",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "analyze",
						status:   buildbucket_pb.Status_STARTED,
					},
				}, genericCompBuildOutputProps.GetFields(), buildbucket_pb.Status_STARTED)},
				{Build: getBuildsWithSteps([]stepNameStatusPair{
					{
						stepName: "lookup GN args",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "analyze",
						status:   buildbucket_pb.Status_SUCCESS,
					},
				}, genericCompBuildOutputProps.GetFields(), buildbucket_pb.Status_SUCCESS)},
			}
			ctx = context.WithValue(
				ctx,
				bb.FakeBuildsContextKey,
				compBuilds)
			userArgs := []string{"-compilator-id", "12345", "-get-swarming-trigger-props"}
			err := luciEXEMain(ctx, input, userArgs, sender)
			So(err, ShouldBeNil)
			expectedSteps := getSteps([]stepNameStatusPair{
				{
					stepName: "lookup GN args",
					status:   buildbucket_pb.Status_SUCCESS,
				},
				{
					stepName: "analyze",
					status:   buildbucket_pb.Status_SUCCESS,
				},
			})
			So(input.GetSteps(), ShouldResembleProto, expectedSteps)
		})

		Convey("updates last step even if step name is the same but is hidden failing step", func() {
			Convey("and no steps have been copied yet", func() {
				compBuilds := []bb.FakeGetBuildResponse{
					{Build: getBuildsWithSteps([]stepNameStatusPair{
						{
							stepName: "report builders",
							status:   buildbucket_pb.Status_STARTED,
						},
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED)},
					{Build: getBuildsWithSteps([]stepNameStatusPair{
						{
							stepName: "report builders",
							status:   buildbucket_pb.Status_FAILURE,
						},
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_FAILURE)},
				}
				ctx = context.WithValue(
					ctx,
					bb.FakeBuildsContextKey,
					compBuilds)
				userArgs := []string{"-compilator-id", "12345", "-get-swarming-trigger-props"}
				err := luciEXEMain(ctx, input, userArgs, sender)
				So(err, ShouldBeNil)
				expectedSteps := getSteps([]stepNameStatusPair{
					{
						stepName: "report builders",
						status:   buildbucket_pb.Status_FAILURE,
					},
				})
				So(input.GetSteps(), ShouldResembleProto, expectedSteps)
			})
			Convey("and previous copied steps exist", func() {
				compBuilds := []bb.FakeGetBuildResponse{
					{Build: getBuildsWithSteps([]stepNameStatusPair{
						{
							stepName: "builder cache",
							status:   buildbucket_pb.Status_SUCCESS,
						},
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED)},
					{Build: getBuildsWithSteps([]stepNameStatusPair{
						{
							stepName: "builder cache",
							status:   buildbucket_pb.Status_SUCCESS,
						},
						{
							stepName: "gclient config",
							status:   buildbucket_pb.Status_STARTED,
						},
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED)},
					{Build: getBuildsWithSteps([]stepNameStatusPair{
						{
							stepName: "builder cache",
							status:   buildbucket_pb.Status_SUCCESS,
						},
						{
							stepName: "gclient config",
							status:   buildbucket_pb.Status_FAILURE,
						},
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_FAILURE)},
				}
				ctx = context.WithValue(
					ctx,
					bb.FakeBuildsContextKey,
					compBuilds)
				userArgs := []string{"-compilator-id", "12345", "-get-swarming-trigger-props"}
				err := luciEXEMain(ctx, input, userArgs, sender)
				So(err, ShouldBeNil)
				expectedSteps := getSteps([]stepNameStatusPair{
					{
						stepName: "builder cache",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "gclient config",
						status:   buildbucket_pb.Status_FAILURE,
					},
				})
				So(input.GetSteps(), ShouldResembleProto, expectedSteps)
			})
		})

		Convey("copies correct Steps according to phase", func() {
			compBuilds := []bb.FakeGetBuildResponse{
				{Build: getBuildsWithSteps([]stepNameStatusPair{
					{
						stepName: "setup_build",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "report builders",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "builder cache",
						status:   buildbucket_pb.Status_SUCCESS,
					},
				}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED)},
				{Build: getBuildsWithSteps([]stepNameStatusPair{
					{
						stepName: "setup_build",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "report builders",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "builder cache",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "gclient config",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "lookup GN args",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "compile (with patch)",
						status:   buildbucket_pb.Status_SUCCESS,
					},
				}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED)},
				{Build: getBuildsWithSteps([]stepNameStatusPair{
					{
						stepName: "setup_build",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "report builders",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "builder cache",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "gclient config",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "lookup GN args",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "compile (with patch)",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: swarmingTriggerPropsStepName,
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "test_pre_run (with patch)",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "check_network_annotations (with patch)",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "gerrit changes",
						status:   buildbucket_pb.Status_SUCCESS,
					},
				}, genericCompBuildOutputProps.GetFields(), buildbucket_pb.Status_SUCCESS)},
			}
			ctx = context.WithValue(
				ctx,
				bb.FakeBuildsContextKey,
				compBuilds)

			Convey("during swarmingPhase", func() {
				userArgs := []string{"-compilator-id", "12345", "-get-swarming-trigger-props"}
				err := luciEXEMain(ctx, input, userArgs, sender)
				So(err, ShouldBeNil)
				So(input.Status, ShouldResemble, buildbucket_pb.Status_SUCCESS)

				expectedSteps := getSteps([]stepNameStatusPair{
					{
						stepName: "builder cache",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "lookup GN args",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "compile (with patch)",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: swarmingTriggerPropsStepName,
						status:   buildbucket_pb.Status_SUCCESS,
					},
				})
				So(input.GetSteps(), ShouldResembleProto, expectedSteps)
			})
			Convey("during localTestPhase", func() {
				userArgs := []string{"-compilator-id", "12345", "-get-local-tests"}
				err := luciEXEMain(ctx, input, userArgs, sender)
				So(err, ShouldBeNil)

				So(input.GetStartTime(), ShouldResemble, timestamppb.New(now))
				So(input.Status, ShouldResemble, buildbucket_pb.Status_SUCCESS)

				expectedSteps := getSteps([]stepNameStatusPair{
					{
						stepName: "test_pre_run (with patch)",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "check_network_annotations (with patch)",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "gerrit changes",
						status:   buildbucket_pb.Status_SUCCESS,
					},
				})
				So(input.GetSteps(), ShouldResembleProto, expectedSteps)
			})
			Convey("and displays failed hidden steps", func() {
				compBuilds := []bb.FakeGetBuildResponse{
					{Build: getBuildsWithSteps([]stepNameStatusPair{
						{
							stepName: "setup_build",
							status:   buildbucket_pb.Status_SUCCESS,
						},
						{
							stepName: "report builders",
							status:   buildbucket_pb.Status_SUCCESS,
						},
						{
							stepName: "builder cache",
							status:   buildbucket_pb.Status_FAILURE,
						},
						{
							stepName: "lookup GN args",
							status:   buildbucket_pb.Status_FAILURE,
						},
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_FAILURE)},
				}
				ctx = context.WithValue(
					ctx,
					bb.FakeBuildsContextKey,
					compBuilds)

				err := luciEXEMain(ctx, input, userArgs, sender)
				So(err, ShouldBeNil)
				So(input.Status, ShouldResemble, buildbucket_pb.Status_FAILURE)

				expectedSteps := getSteps([]stepNameStatusPair{
					{
						stepName: "builder cache",
						status:   buildbucket_pb.Status_FAILURE,
					},
					{
						stepName: "lookup GN args",
						status:   buildbucket_pb.Status_FAILURE,
					},
				})
				So(input.GetSteps(), ShouldResembleProto, expectedSteps)
			})

			Convey("sets InfraFailure with summary for timeout", func() {
				// Force luciexe to timeout right after first build is retrieved in copySteps()
				userArgs = []string{
					"-compilator-id",
					"12345",
					"-get-swarming-trigger-props",
					"-compilator-polling-timeout-sec",
					"5",
					"-max-consecutive-get-build-timeouts",
					"3",
				}

				clk.SetTimerCallback(func(amt time.Duration, timer clock.Timer) {
					tags := testclock.GetTags(timer)
					for i := 0; i < len(tags); i++ {
						tag := tags[i]
						if tag == clock.ContextDeadlineTag {
							return
						}
					}
					clk.Add(5*time.Second + time.Millisecond)
				})

				ctx = context.WithValue(
					ctx,
					bb.FakeBuildsContextKey,
					compBuilds)

				err := luciEXEMain(ctx, input, userArgs, sender)
				So(err, ShouldNotBeNil)
				So(exe.InfraErrorTag.In(err), ShouldBeTrue)
				So(input.SummaryMarkdown, ShouldResemble, "Error while running compilator_watcher: Timeout waiting for compilator build")
			})

			Convey("handles timeouts from GetBuild", func() {
				userArgs := []string{"-compilator-id", "12345", "-get-swarming-trigger-props"}

				Convey("by allowing up to max N consecutive errs", func() {
					compBuilds := []bb.FakeGetBuildResponse{
						{Build: getBuildsWithSteps([]stepNameStatusPair{
							{
								stepName: "report builders",
								status:   buildbucket_pb.Status_STARTED,
							},
						}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED)},
						{Err: grpcStatus.Error(codes.DeadlineExceeded, "Gateway Timeout")},
						{Err: grpcStatus.Error(codes.DeadlineExceeded, "Gateway Timeout")},
						{Build: getBuildsWithSteps([]stepNameStatusPair{
							{
								stepName: "report builders",
								status:   buildbucket_pb.Status_FAILURE,
							},
						}, map[string]*structpb.Value{}, buildbucket_pb.Status_FAILURE)},
					}
					ctx = context.WithValue(
						ctx,
						bb.FakeBuildsContextKey,
						compBuilds)
					err := luciEXEMain(ctx, input, userArgs, sender)
					So(err, ShouldBeNil)
					expectedSteps := getSteps([]stepNameStatusPair{
						{
							stepName: "report builders",
							status:   buildbucket_pb.Status_FAILURE,
						},
					})
					So(input.GetSteps(), ShouldResembleProto, expectedSteps)

				})
				Convey("and raising err if the num of consecutive errs exceeds max number", func() {
					compBuilds := []bb.FakeGetBuildResponse{
						{Build: getBuildsWithSteps([]stepNameStatusPair{
							{
								stepName: "report builders",
								status:   buildbucket_pb.Status_STARTED,
							},
						}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED)},
						{Err: grpcStatus.Error(codes.DeadlineExceeded, "Gateway Timeout")},
						{Err: grpcStatus.Error(codes.DeadlineExceeded, "Gateway Timeout")},
						{Err: grpcStatus.Error(codes.DeadlineExceeded, "Gateway Timeout")},
						{Err: grpcStatus.Error(codes.DeadlineExceeded, "Gateway Timeout")},
					}
					ctx = context.WithValue(
						ctx,
						bb.FakeBuildsContextKey,
						compBuilds)
					err := luciEXEMain(ctx, input, userArgs, sender)
					So(err, ShouldNotBeNil)
					So(err, ShouldErrLike, "rpc error: code = DeadlineExceeded desc = Gateway Timeout")
				})
				Convey("and errs need to be consecutive", func() {
					compBuilds := []bb.FakeGetBuildResponse{
						{Err: grpcStatus.Error(codes.DeadlineExceeded, "Gateway Timeout")},
						{Err: grpcStatus.Error(codes.DeadlineExceeded, "Gateway Timeout")},
						{Build: getBuildsWithSteps([]stepNameStatusPair{
							{
								stepName: "report builders",
								status:   buildbucket_pb.Status_STARTED,
							},
						}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED)},
						{Err: grpcStatus.Error(codes.DeadlineExceeded, "Gateway Timeout")},
						{Err: grpcStatus.Error(codes.DeadlineExceeded, "Gateway Timeout")},
						{Build: getBuildsWithSteps([]stepNameStatusPair{
							{
								stepName: "report builders",
								status:   buildbucket_pb.Status_FAILURE,
							},
						}, map[string]*structpb.Value{}, buildbucket_pb.Status_FAILURE)},
					}
					ctx = context.WithValue(
						ctx,
						bb.FakeBuildsContextKey,
						compBuilds)
					err := luciEXEMain(ctx, input, userArgs, sender)
					So(err, ShouldBeNil)
					expectedSteps := getSteps([]stepNameStatusPair{
						{
							stepName: "report builders",
							status:   buildbucket_pb.Status_FAILURE,
						},
					})
					So(input.GetSteps(), ShouldResembleProto, expectedSteps)
				})
			})
		})
	})
}
