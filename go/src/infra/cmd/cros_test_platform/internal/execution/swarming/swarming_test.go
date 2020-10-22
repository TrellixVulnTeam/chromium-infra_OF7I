// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"context"
	"fmt"
	"testing"

	"infra/libs/skylab/worker"

	. "github.com/smartystreets/goconvey/convey"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/memlogger"

	"infra/libs/skylab/inventory"
	"infra/libs/skylab/request"
)

// fakeSwarming implements skylab_api.Swarming
type fakeSwarming struct {
	botExists           bool
	createCalls         []*swarming_api.SwarmingRpcsNewTaskRequest
	taskIDToIsolateHash map[string]string
	taskIDToTaskState   map[string]string
}

func newFakeSwarming() *fakeSwarming {
	return &fakeSwarming{
		botExists:           true,
		taskIDToIsolateHash: make(map[string]string),
		taskIDToTaskState:   make(map[string]string),
	}
}

func (f *fakeSwarming) CreateTask(ctx context.Context, req *swarming_api.SwarmingRpcsNewTaskRequest) (*swarming_api.SwarmingRpcsTaskRequestMetadata, error) {
	f.createCalls = append(f.createCalls, req)
	return &swarming_api.SwarmingRpcsTaskRequestMetadata{
		TaskId: "foo-id",
	}, nil
}

func (f *fakeSwarming) GetResults(ctx context.Context, IDs []string) ([]*swarming_api.SwarmingRpcsTaskResult, error) {
	var outputsRef *swarming_api.SwarmingRpcsFilesRef
	if len(IDs) != 1 {
		panic(fmt.Sprintf("got %d results instead of one.", len(IDs)))
	}
	ID := IDs[0]
	if h, ok := f.taskIDToIsolateHash[ID]; ok {
		outputsRef = &swarming_api.SwarmingRpcsFilesRef{
			Isolated: h,
		}
	}
	return []*swarming_api.SwarmingRpcsTaskResult{
		{
			TaskId:     ID,
			State:      f.taskIDToTaskState[ID],
			OutputsRef: outputsRef,
		},
	}, nil
}

func (f *fakeSwarming) BotExists(ctx context.Context, dims []*swarming_api.SwarmingRpcsStringPair) (bool, error) {
	return f.botExists, nil
}

func (f *fakeSwarming) GetTaskURL(taskID string) string {
	return ""
}

func (f *fakeSwarming) setCannedBotExistsResponse(b bool) {
	f.botExists = b
}

func (f *fakeSwarming) setTaskState(ID string, state string) {
	f.taskIDToTaskState[ID] = state
}

func TestNonExistentBot(t *testing.T) {
	Convey("When arguments ask for a non-existent bot", t, func() {
		swarming := newFakeSwarming()
		swarming.setCannedBotExistsResponse(false)
		skylab := &rawSwarmingSkylabClient{
			swarmingClient: swarming,
		}
		var ml memlogger.MemLogger
		ctx := setLogger(context.Background(), &ml)
		board := "foo-board"
		args := &request.Args{
			SchedulableLabels: &inventory.SchedulableLabels{
				Board: &board,
			},
		}
		expectedRejectedTaskDims := map[string]string{
			"label-board": "foo-board",
			"pool":        "ChromeOSSkylab",
		}
		Convey("the validation fails.", func() {
			botExists, rejectedTaskDims, err := skylab.ValidateArgs(ctx, args)
			So(err, ShouldBeNil)
			So(rejectedTaskDims, ShouldResemble, expectedRejectedTaskDims)
			So(botExists, ShouldBeFalse)
			So(loggerOutput(ml, logging.Warning), ShouldContainSubstring, "foo-board")
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
		swarming.setCannedBotExistsResponse(true)
		skylab := &rawSwarmingSkylabClient{
			swarmingClient: swarming,
		}
		Convey("the validation passes.", func() {
			botExists, rejectedTaskDims, err := skylab.ValidateArgs(context.Background(), &request.Args{})
			So(err, ShouldBeNil)
			So(rejectedTaskDims, ShouldBeNil)
			So(botExists, ShouldBeTrue)
		})
	})
}

func TestLaunch(t *testing.T) {
	Convey("When a task is launched", t, func() {
		swarming := newFakeSwarming()
		skylab := &rawSwarmingSkylabClient{
			swarmingClient: swarming,
		}
		task, err := skylab.LaunchTask(context.Background(), &request.Args{
			Cmd: worker.Command{
				TaskName: "foo-name",
			},
		})
		So(err, ShouldBeNil)
		Convey("the Swarming client is called with the correct args", func() {
			So(task, ShouldNotBeNil)
			So(swarming.createCalls, ShouldHaveLength, 1)
			So(swarming.createCalls[0].Name, ShouldEqual, "foo-name")
		})
	})
}
