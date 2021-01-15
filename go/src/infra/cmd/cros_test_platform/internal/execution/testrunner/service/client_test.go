// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package service

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	structpb "github.com/golang/protobuf/ptypes/struct"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/memlogger"

	"infra/libs/skylab/inventory"
	"infra/libs/skylab/request"
)

// fakeSwarming implements skylab_api.Swarming.
type fakeSwarming struct {
	botsByBoard map[string]bool
}

func newFakeSwarming() *fakeSwarming {
	return &fakeSwarming{
		botsByBoard: make(map[string]bool),
	}
}

// BotExists implements swarmingClient interface.
func (f *fakeSwarming) BotExists(_ context.Context, dims []*swarming_api.SwarmingRpcsStringPair) (bool, error) {
	for _, dim := range dims {
		if dim.Key == "label-board" {
			return f.botsByBoard[dim.Value], nil
		}
	}
	return false, nil
}

func (f *fakeSwarming) addBot(board string) {
	f.botsByBoard[board] = true
}

func TestNonExistentBot(t *testing.T) {
	Convey("When arguments ask for a non-existent bot", t, func() {
		swarming := newFakeSwarming()
		swarming.addBot("existing-board")
		skylab := &clientImpl{
			swarmingClient: swarming,
		}
		var ml memlogger.MemLogger
		ctx := setLogger(context.Background(), &ml)
		var args request.Args
		args.SchedulableLabels = &inventory.SchedulableLabels{}
		addBoard(&args, "nonexistent-board")
		expectedRejectedTaskDims := map[string]string{
			"label-board": "nonexistent-board",
			"pool":        "ChromeOSSkylab",
		}
		Convey("the validation fails.", func() {
			botExists, rejectedTaskDims, err := skylab.ValidateArgs(ctx, &args)
			So(err, ShouldBeNil)
			So(rejectedTaskDims, ShouldResemble, expectedRejectedTaskDims)
			So(botExists, ShouldBeFalse)
			So(loggerOutput(ml, logging.Warning), ShouldContainSubstring, "nonexistent-board")
		})
	})
}

func setLogger(ctx context.Context, l logging.Logger) context.Context {
	return logging.SetFactory(ctx, func(context.Context) logging.Logger {
		return l
	})
}

func loggerOutput(ml memlogger.MemLogger, level logging.Level) string {
	out := ""
	for _, m := range ml.Messages() {
		if m.Level == level {
			out = out + m.Msg
		}
	}
	return out
}

func TestExistingBot(t *testing.T) {
	Convey("When arguments ask for an existing bot", t, func() {
		swarming := newFakeSwarming()
		swarming.addBot("existing-board")
		skylab := &clientImpl{
			swarmingClient: swarming,
		}
		var args request.Args
		args.SchedulableLabels = &inventory.SchedulableLabels{}
		addBoard(&args, "existing-board")
		Convey("the validation passes.", func() {
			botExists, rejectedTaskDims, err := skylab.ValidateArgs(context.Background(), &args)
			So(err, ShouldBeNil)
			So(rejectedTaskDims, ShouldBeNil)
			So(botExists, ShouldBeTrue)
		})
	})
}

func addBoard(args *request.Args, board string) {
	args.SchedulableLabels.Board = &board
}

func TestLaunchRequest(t *testing.T) {
	Convey("When a task is launched", t, func() {
		tf, cleanup := newTestFixture(t)
		defer cleanup()

		setBuilder(tf.skylab, "foo-project", "foo-bucket", "foo-builder-name")
		args := newArgs()
		addTestName(args, "foo-test")

		var gotRequest *buildbucket_pb.ScheduleBuildRequest
		tf.bb.EXPECT().ScheduleBuild(
			gomock.Any(),
			gomock.Any(),
		).Do(
			func(_ context.Context, r *buildbucket_pb.ScheduleBuildRequest) {
				gotRequest = r
			},
		).Return(&buildbucket_pb.Build{Id: 42}, nil)

		t, err := tf.skylab.LaunchTask(tf.ctx, args)
		So(err, ShouldBeNil)
		Convey("the BB client is called with the correct args", func() {
			So(gotRequest, ShouldNotBeNil)
			So(gotRequest.Properties, ShouldNotBeNil)
			So(gotRequest.Properties.Fields, ShouldNotBeNil)
			So(gotRequest.Properties.Fields["request"], ShouldNotBeNil)
			req, err := structPBToTestRunnerRequest(gotRequest.Properties.Fields["request"])
			So(err, ShouldBeNil)
			So(req.GetTest().GetAutotest().GetName(), ShouldEqual, "foo-test")
			Convey("and the URL is formatted correctly.", func() {
				So(tf.skylab.URL(t), ShouldEqual,
					"https://ci.chromium.org/p/foo-project/builders/foo-bucket/foo-builder-name/b42")
			})
		})
	})
}

func setBuilder(skylab *clientImpl, project string, bucket string, builder string) {
	skylab.builder = &buildbucket_pb.BuilderID{
		Project: project,
		Bucket:  bucket,
		Builder: builder,
	}
}

func addTestName(args *request.Args, name string) {
	if args.TestRunnerRequest.Test == nil {
		args.TestRunnerRequest.Test = &skylab_test_runner.Request_Test{
			Harness: &skylab_test_runner.Request_Test_Autotest_{
				Autotest: &skylab_test_runner.Request_Test_Autotest{},
			},
		}
	}
	args.TestRunnerRequest.Test.GetAutotest().Name = name
}

func structPBToTestRunnerRequest(from *structpb.Value) (*skylab_test_runner.Request, error) {
	m := jsonpb.Marshaler{}
	json, err := m.MarshalToString(from)
	if err != nil {
		return nil, errors.Annotate(err, "structPBToTestRunnerRequest").Err()
	}
	var req skylab_test_runner.Request
	if err := jsonpb.UnmarshalString(json, &req); err != nil {
		return nil, errors.Annotate(err, "structPBToTestRunnerRequest").Err()
	}
	return &req, nil
}

func TestFetchRequest(t *testing.T) {
	Convey("When a task is launched and completes", t, func() {
		tf, cleanup := newTestFixture(t)
		defer cleanup()

		tf.bb.EXPECT().ScheduleBuild(
			gomock.Any(),
			gomock.Any(),
		).Return(&buildbucket_pb.Build{Id: 42}, nil)

		var gotRequest *buildbucket_pb.GetBuildRequest
		tf.bb.EXPECT().GetBuild(
			gomock.Any(),
			gomock.Any(),
		).Do(
			func(_ context.Context, r *buildbucket_pb.GetBuildRequest) {
				gotRequest = r
			},
		).Return(&buildbucket_pb.Build{}, nil)

		task, err := tf.skylab.LaunchTask(tf.ctx, newArgs())
		So(err, ShouldBeNil)
		Convey("as the results are fetched", func() {
			_, err := tf.skylab.FetchResults(tf.ctx, task)
			So(err, ShouldBeNil)
			Convey("the BB client is called with the correct args.", func() {
				So(gotRequest.Id, ShouldEqual, 42)
				So(gotRequest.Fields, ShouldNotBeNil)
				So(gotRequest.Fields.Paths, ShouldContain, "id")
				So(gotRequest.Fields.Paths, ShouldContain, "infra.swarming.task_id")
				So(gotRequest.Fields.Paths, ShouldContain, "output.properties")
				So(gotRequest.Fields.Paths, ShouldContain, "status")
			})
		})
	})
}

func TestCompletedTask(t *testing.T) {
	Convey("When a task is launched and completes", t, func() {
		tf, cleanup := newTestFixture(t)
		defer cleanup()

		tf.bb.EXPECT().ScheduleBuild(
			gomock.Any(),
			gomock.Any(),
		).Return(&buildbucket_pb.Build{Id: 42}, nil)

		tf.bb.EXPECT().GetBuild(
			gomock.Any(),
			gomock.Any(),
		).Return(&buildbucket_pb.Build{
			Id: 42,
			Infra: &buildbucket_pb.BuildInfra{
				Swarming: &buildbucket_pb.BuildInfra_Swarming{
					TaskId: "foo-swarming-task-id",
				},
			},
			Status: buildbucket_pb.Status_SUCCESS,
			Output: outputProperty("foo-test-case"),
		}, nil)

		task, err := tf.skylab.LaunchTask(tf.ctx, newArgs())
		So(err, ShouldBeNil)
		Convey("the task results are reported correctly.", func() {
			res, err := tf.skylab.FetchResults(tf.ctx, task)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
			So(res.Result, ShouldNotBeNil)
			So(res.Result.GetAutotestResult().GetTestCases(), ShouldHaveLength, 1)
			So(res.Result.GetAutotestResult().GetTestCases()[0].GetName(), ShouldEqual, "foo-test-case")
			So(tf.skylab.SwarmingTaskID(task), ShouldEqual, "foo-swarming-task-id")
		})
	})
}

func TestCompletedTaskMissingResults(t *testing.T) {
	Convey("When a task is launched, completes and has no results", t, func() {
		tf, cleanup := newTestFixture(t)
		defer cleanup()

		tf.bb.EXPECT().ScheduleBuild(
			gomock.Any(),
			gomock.Any(),
		).Return(&buildbucket_pb.Build{Id: 42}, nil)

		tf.bb.EXPECT().GetBuild(
			gomock.Any(),
			gomock.Any(),
		).Return(&buildbucket_pb.Build{
			Id:     42,
			Status: buildbucket_pb.Status_SUCCESS,
		}, nil)

		task, err := tf.skylab.LaunchTask(tf.ctx, newArgs())
		So(err, ShouldBeNil)
		Convey("an error is returned.", func() {
			_, err := tf.skylab.FetchResults(tf.ctx, task)
			// This error is bubbled up as a non-zero exit code of the binary
			// which is interpreted as an INFRA_FAILURE by the recipe.
			So(err, ShouldNotBeNil)
		})
	})
}

func TestAbortedTask(t *testing.T) {
	Convey("When a task is launched and reports an infra failure", t, func() {
		tf, cleanup := newTestFixture(t)
		defer cleanup()

		tf.bb.EXPECT().ScheduleBuild(
			gomock.Any(),
			gomock.Any(),
		).Return(&buildbucket_pb.Build{Id: 42}, nil)

		tf.bb.EXPECT().GetBuild(
			gomock.Any(),
			gomock.Any(),
		).Return(&buildbucket_pb.Build{
			Id:     42,
			Status: buildbucket_pb.Status_INFRA_FAILURE,
			Output: outputProperty("foo-test-case"),
		}, nil)

		task, err := tf.skylab.LaunchTask(tf.ctx, newArgs())
		So(err, ShouldBeNil)
		Convey("results are ignored.", func() {
			res, err := tf.skylab.FetchResults(tf.ctx, task)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_ABORTED)
			So(res.Result, ShouldBeNil)
		})
	})
}

type testFixture struct {
	ctx    context.Context
	bb     *buildbucket_pb.MockBuildsClient
	skylab *clientImpl
}

func newTestFixture(t *testing.T) (*testFixture, func()) {
	ctrl := gomock.NewController(t)
	bb := buildbucket_pb.NewMockBuildsClient(ctrl)
	return &testFixture{
		ctx: context.Background(),
		bb:  bb,
		skylab: &clientImpl{
			bbClient:   bb,
			knownTasks: make(map[TaskReference]*task),
		},
	}, ctrl.Finish
}

func newArgs() *request.Args {
	return &request.Args{
		TestRunnerRequest: &skylab_test_runner.Request{},
	}
}

func outputProperty(testCase string) *buildbucket_pb.Build_Output {
	res := &skylab_test_runner.Result{
		Harness: &skylab_test_runner.Result_AutotestResult{
			AutotestResult: &skylab_test_runner.Result_Autotest{
				TestCases: []*skylab_test_runner.Result_Autotest_TestCase{
					{
						Name:    testCase,
						Verdict: skylab_test_runner.Result_Autotest_TestCase_VERDICT_PASS,
					},
				},
			},
		},
	}
	m, _ := proto.Marshal(res)
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(m)
	w.Close()
	return &buildbucket_pb.Build_Output{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"compressed_result": {
					Kind: &structpb.Value_StringValue{
						StringValue: base64.StdEncoding.EncodeToString(b.Bytes()),
					},
				},
			},
		},
	}
}
