// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bb

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/common/logging/memlogger"

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
