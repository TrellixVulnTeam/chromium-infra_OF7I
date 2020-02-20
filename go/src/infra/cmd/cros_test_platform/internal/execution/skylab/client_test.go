// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package skylab

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	swarming_api "go.chromium.org/luci/common/api/swarming/swarming/v1"

	"infra/libs/skylab/request"
)

// fakeSwarming implements skylab_api.Swarming
type fakeSwarming struct {
	botExists bool
}

func newFakeSwarming() *fakeSwarming {
	return &fakeSwarming{
		botExists: true,
	}
}

func (f *fakeSwarming) CreateTask(ctx context.Context, req *swarming_api.SwarmingRpcsNewTaskRequest) (*swarming_api.SwarmingRpcsTaskRequestMetadata, error) {
	return nil, nil
}

func (f *fakeSwarming) GetResults(ctx context.Context, IDs []string) ([]*swarming_api.SwarmingRpcsTaskResult, error) {
	return nil, nil
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
