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

		swarmingProps := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"all_test_binaries": {
					Kind: &structpb.Value_StringValue{
						StringValue: "fhueowahfueoah",
					},
				},
				"browser_tests": {
					Kind: &structpb.Value_StringValue{
						StringValue: "fhuf8eaj9f0eja90eowahfueoah",
					},
				},
			},
		}

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
				[]*buildbucket_pb.Build{compBuild})
			err := luciEXEMain(ctx, input, userArgs, sender)

			So(err, ShouldBeNil)
			So(input.Status, ShouldResemble, buildbucket_pb.Status_FAILURE)
			So(input.SummaryMarkdown, ShouldResemble, "Compile failure")

		})

		Convey("sets input Status to SUCCESS when compilator build outputs swarming props but is still running", func() {
			compBuild := &buildbucket_pb.Build{
				Status:          buildbucket_pb.Status_STARTED,
				Id:              12345,
				SummaryMarkdown: "",
				Output: &buildbucket_pb.Build_Output{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"swarming_trigger_properties": {
								Kind: &structpb.Value_StructValue{
									StructValue: swarmingProps,
								},
							},
						},
					},
				},
			}
			ctx = context.WithValue(
				ctx,
				bb.FakeBuildsContextKey,
				[]*buildbucket_pb.Build{compBuild})
			err := luciEXEMain(ctx, input, userArgs, sender)

			So(err, ShouldBeNil)
			So(input.Status, ShouldResemble, buildbucket_pb.Status_SUCCESS)
			So(input.GetOutput(), ShouldResembleProto, &buildbucket_pb.Build_Output{
				Properties: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"swarming_trigger_properties": {
							Kind: &structpb.Value_StructValue{
								StructValue: swarmingProps,
							},
						},
					},
				},
			})
		})

		Convey("exits after compilator build successfully ends with no swarming trigger properties", func() {
			compBuild := &buildbucket_pb.Build{
				Status:          buildbucket_pb.Status_SUCCESS,
				Id:              12345,
				SummaryMarkdown: "",
				Output: &buildbucket_pb.Build_Output{
					Properties: &structpb.Struct{},
				},
			}
			ctx = context.WithValue(
				ctx,
				bb.FakeBuildsContextKey,
				[]*buildbucket_pb.Build{compBuild})
			err := luciEXEMain(ctx, input, userArgs, sender)

			So(err, ShouldBeNil)
			So(input.Status, ShouldResemble, buildbucket_pb.Status_SUCCESS)

			So(input.GetOutput(), ShouldResembleProto, &buildbucket_pb.Build_Output{
				Properties: &structpb.Struct{},
			})
		})

		Convey("updates last step even if step name is the same", func() {
			compBuilds := []*buildbucket_pb.Build{
				getBuildsWithSteps([]stepNameStatusPair{
					{
						stepName: "lookup GN args",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "analyze",
						status:   buildbucket_pb.Status_STARTED,
					},
				}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED),
				getBuildsWithSteps([]stepNameStatusPair{
					{
						stepName: "lookup GN args",
						status:   buildbucket_pb.Status_SUCCESS,
					},
					{
						stepName: "analyze",
						status:   buildbucket_pb.Status_SUCCESS,
					},
				}, map[string]*structpb.Value{}, buildbucket_pb.Status_SUCCESS),
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
				compBuilds := []*buildbucket_pb.Build{
					getBuildsWithSteps([]stepNameStatusPair{
						{
							stepName: "report builders",
							status:   buildbucket_pb.Status_STARTED,
						},
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED),
					getBuildsWithSteps([]stepNameStatusPair{
						{
							stepName: "report builders",
							status:   buildbucket_pb.Status_FAILURE,
						},
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_FAILURE),
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
				compBuilds := []*buildbucket_pb.Build{
					getBuildsWithSteps([]stepNameStatusPair{
						{
							stepName: "builder cache",
							status:   buildbucket_pb.Status_SUCCESS,
						},
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED),
					getBuildsWithSteps([]stepNameStatusPair{
						{
							stepName: "builder cache",
							status:   buildbucket_pb.Status_SUCCESS,
						},
						{
							stepName: "gclient config",
							status:   buildbucket_pb.Status_STARTED,
						},
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED),
					getBuildsWithSteps([]stepNameStatusPair{
						{
							stepName: "builder cache",
							status:   buildbucket_pb.Status_SUCCESS,
						},
						{
							stepName: "gclient config",
							status:   buildbucket_pb.Status_FAILURE,
						},
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_FAILURE),
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
			compBuilds := []*buildbucket_pb.Build{
				getBuildsWithSteps([]stepNameStatusPair{
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
				}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED),
				getBuildsWithSteps([]stepNameStatusPair{
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
				}, map[string]*structpb.Value{}, buildbucket_pb.Status_STARTED),
				getBuildsWithSteps([]stepNameStatusPair{
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
				}, map[string]*structpb.Value{
					"swarming_trigger_properties": {
						Kind: &structpb.Value_StructValue{StructValue: swarmingProps},
					},
				}, buildbucket_pb.Status_SUCCESS),
			}
			ctx = context.WithValue(
				ctx,
				bb.FakeBuildsContextKey,
				compBuilds)

			expectedOutputFields := map[string]*structpb.Value{
				"swarming_trigger_properties": {
					Kind: &structpb.Value_StructValue{
						StructValue: swarmingProps,
					},
				},
			}

			Convey("during swarmingPhase", func() {
				userArgs := []string{"-compilator-id", "12345", "-get-swarming-trigger-props"}
				err := luciEXEMain(ctx, input, userArgs, sender)
				So(err, ShouldBeNil)

				expectedOutputProps := &buildbucket_pb.Build_Output{
					Properties: &structpb.Struct{
						Fields: expectedOutputFields,
					},
				}

				So(input.GetOutput(), ShouldResembleProto, expectedOutputProps)
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

				So(input.GetOutput(), ShouldResembleProto, &buildbucket_pb.Build_Output{
					Properties: &structpb.Struct{},
				})

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
				compBuilds := []*buildbucket_pb.Build{
					getBuildsWithSteps([]stepNameStatusPair{
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
					}, map[string]*structpb.Value{}, buildbucket_pb.Status_FAILURE),
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
		})
	})
}
