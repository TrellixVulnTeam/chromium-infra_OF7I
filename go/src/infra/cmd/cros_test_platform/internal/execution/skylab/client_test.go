// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform"
	"go.chromium.org/chromiumos/infra/proto/go/test_platform/skylab_test_runner"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/isolated"

	"infra/cmd/cros_test_platform/internal/execution/isolate"
	"infra/libs/skylab/request"
	"infra/libs/skylab/worker"
)

type fakeResultStore map[string]map[string]*skylab_test_runner.Result

func (s fakeResultStore) AddResult(isolatedHash string, filePath string, result *skylab_test_runner.Result) {
	s[isolatedHash] = map[string]*skylab_test_runner.Result{
		filePath: result,
	}
}

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

func (f *fakeSwarming) setTaskIsolatedHash(taskID string, isolatedHash string) {
	f.taskIDToIsolateHash[taskID] = isolatedHash
}

type fakeGetter struct {
	resultStore fakeResultStore
}

func (g *fakeGetter) GetFile(_ context.Context, hex isolated.HexDigest, filePath string) ([]byte, error) {
	r, ok := g.resultStore[string(hex)][filePath]
	if !ok {
		panic(fmt.Sprintf("fake getter could not get file with hash %s and path %s.", hex, filePath))
	}
	m := &jsonpb.Marshaler{}
	s, err := m.MarshalToString(r)
	if err != nil {
		panic(fmt.Sprintf("error when marshalling %#v: %s", r, err))
	}
	return []byte(s), nil
}

func fakeGetterFactory(s fakeResultStore) isolate.GetterFactory {
	return func(_ context.Context, _ string) (isolate.Getter, error) {
		return &fakeGetter{
			resultStore: s,
		}, nil
	}
}

func TestNonExistentBot(t *testing.T) {
	Convey("When arguments ask for a non-existent bot", t, func() {
		swarming := newFakeSwarming()
		swarming.setCannedBotExistsResponse(false)
		skylab := &Client{
			Swarming: swarming,
		}
		Convey("the validation fails.", func() {
			exists, err := skylab.ValidateArgs(context.Background(), &request.Args{})
			So(err, ShouldBeNil)
			So(exists, ShouldBeFalse)
		})
	})
}

func TestExistingBot(t *testing.T) {
	Convey("When arguments ask for an existing bot", t, func() {
		swarming := newFakeSwarming()
		swarming.setCannedBotExistsResponse(true)
		skylab := &Client{
			Swarming: swarming,
		}
		Convey("the validation passes.", func() {
			exists, err := skylab.ValidateArgs(context.Background(), &request.Args{})
			So(err, ShouldBeNil)
			So(exists, ShouldBeTrue)
		})
	})
}

func TestLaunch(t *testing.T) {
	Convey("When a task is launched", t, func() {
		swarming := newFakeSwarming()
		skylab := &Client{
			Swarming: swarming,
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

func TestCompletedTask(t *testing.T) {
	Convey("When a task is launched and completes", t, func() {
		ctx := context.Background()
		swarming := newFakeSwarming()
		i := fakeResultStore{}
		i.AddResult("foo-isolated", "results.json", &skylab_test_runner.Result{})
		skylab := &Client{
			Swarming:      swarming,
			IsolateGetter: fakeGetterFactory(i),
		}
		task, err := skylab.LaunchTask(ctx, &request.Args{})
		So(err, ShouldBeNil)
		swarming.setTaskState(task.SwarmingTaskID(), "COMPLETED")
		swarming.setTaskIsolatedHash(task.SwarmingTaskID(), "foo-isolated")

		Convey("the task results are reported correctly.", func() {
			res, err := task.FetchResults(ctx)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
			So(res.Result, ShouldNotBeNil)
		})
	})
}

func TestUnfinishedTask(t *testing.T) {
	Convey("When a task is launched and is killed", t, func() {
		ctx := context.Background()
		swarming := newFakeSwarming()
		skylab := &Client{
			Swarming: swarming,
		}
		task, err := skylab.LaunchTask(ctx, &request.Args{})
		So(err, ShouldBeNil)
		swarming.setTaskState(task.SwarmingTaskID(), "KILLED")
		swarming.setTaskIsolatedHash(task.SwarmingTaskID(), "ignored-isolated")

		Convey("no results are reported.", func() {
			res, err := task.FetchResults(ctx)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_ABORTED)
			So(res.Result, ShouldBeNil)
		})
	})
}

func TestMissingIsolate(t *testing.T) {
	Convey("When a task is launched, completes and is missing an isolated output", t, func() {
		ctx := context.Background()
		swarming := newFakeSwarming()
		skylab := &Client{
			Swarming:      swarming,
			IsolateGetter: fakeGetterFactory(nil),
		}
		task, err := skylab.LaunchTask(ctx, &request.Args{})
		So(err, ShouldBeNil)
		swarming.setTaskState(task.SwarmingTaskID(), "COMPLETED")

		Convey("no results are reported.", func() {
			res, err := task.FetchResults(ctx)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)
			So(res.LifeCycle, ShouldEqual, test_platform.TaskState_LIFE_CYCLE_COMPLETED)
			So(res.Result, ShouldBeNil)
		})
	})
}
