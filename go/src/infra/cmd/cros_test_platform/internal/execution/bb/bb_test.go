// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bb

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/jsonpb"
	structpb "github.com/golang/protobuf/ptypes/struct"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	buildbucket_pb "go.chromium.org/luci/buildbucket/proto"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/memlogger"

	"infra/cmd/cros_test_platform/internal/execution/skylab"
	"infra/libs/skylab/request"
)

func newBBSkylabClient(bbc buildbucket_pb.BuildsClient) *bbSkylabClient {
	return &bbSkylabClient{
		bbClient:    bbc,
		knownBuilds: make(map[skylab.TaskReference]build),
	}
}

// fakeSwarming implements skylab_api.Swarming.
type fakeSwarming struct {
	botsByBoard map[string]bool
}

func newFakeSwarming() *fakeSwarming {
	return &fakeSwarming{
		botsByBoard: make(map[string]bool),
	}
}

func (f *fakeSwarming) CreateTask(context.Context, *swarming_api.SwarmingRpcsNewTaskRequest) (*swarming_api.SwarmingRpcsTaskRequestMetadata, error) {
	return nil, nil
}

func (f *fakeSwarming) GetResults(context.Context, []string) ([]*swarming_api.SwarmingRpcsTaskResult, error) {
	return nil, nil
}

func (f *fakeSwarming) BotExists(_ context.Context, dims []*swarming_api.SwarmingRpcsStringPair) (bool, error) {
	for _, dim := range dims {
		if dim.Key == "label-board" {
			return f.botsByBoard[dim.Value], nil
		}
	}
	return false, nil
}

func (f *fakeSwarming) GetTaskURL(string) string {
	return ""
}

func (f *fakeSwarming) addBot(board string) {
	f.botsByBoard[board] = true
}

func TestNonExistentBot(t *testing.T) {
	Convey("When arguments ask for a non-existent bot", t, func() {
		swarming := newFakeSwarming()
		swarming.addBot("existing-board")
		skylab := &bbSkylabClient{
			swarmingClient: swarming,
		}
		var ml memlogger.MemLogger
		ctx := setLogger(context.Background(), &ml)
		var args request.Args
		addBoard(&args, "nonexistent-board")
		Convey("the validation fails.", func() {
			exists, err := skylab.ValidateArgs(ctx, &args)
			So(err, ShouldBeNil)
			So(exists, ShouldBeFalse)
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
		skylab := &bbSkylabClient{
			swarmingClient: swarming,
		}
		var args request.Args
		addBoard(&args, "existing-board")
		Convey("the validation passes.", func() {
			exists, err := skylab.ValidateArgs(context.Background(), &args)
			So(err, ShouldBeNil)
			So(exists, ShouldBeTrue)
		})
	})
}

func addBoard(args *request.Args, board string) {
	args.SchedulableLabels.Board = &board
}

func TestLaunch(t *testing.T) {
	Convey("When a task is launched", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		bbClient := buildbucket_pb.NewMockBuildsClient(ctrl)
		skylab := newBBSkylabClient(bbClient)
		var args request.Args
		addTestName(&args, "foo-test")

		var gotRequest *buildbucket_pb.ScheduleBuildRequest
		bbClient.EXPECT().ScheduleBuild(
			gomock.Any(),
			gomock.Any(),
		).Do(
			func(_ context.Context, r *buildbucket_pb.ScheduleBuildRequest) {
				gotRequest = r
			},
		).Return(&buildbucket_pb.Build{Id: 42}, nil)

		task, err := skylab.LaunchTask(context.Background(), &args)
		So(err, ShouldBeNil)
		Convey("the BB client is called with the correct args.", func() {
			So(gotRequest, ShouldNotBeNil)
			So(gotRequest.Properties, ShouldNotBeNil)
			So(gotRequest.Properties.Fields, ShouldNotBeNil)
			So(gotRequest.Properties.Fields["request"], ShouldNotBeNil)
			req, err := structPBToTestRunnerRequest(gotRequest.Properties.Fields["request"])
			So(err, ShouldBeNil)
			So(req.GetTest().GetAutotest().GetName(), ShouldEqual, "foo-test")
			Convey("and the task reference is recorded correctly.", func() {
				// TODO(zamorzaev): remove this implementation check once
				// results tests are added.
				So(task, ShouldNotBeNil)
				So(skylab.knownBuilds[task], ShouldNotBeNil)
				So(skylab.knownBuilds[task].bbID, ShouldEqual, 42)
			})
		})
	})
}

func addTestName(args *request.Args, name string) {
	if args.TestRunnerRequest == nil {
		args.TestRunnerRequest = &skylab_test_runner.Request{}
	}
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
