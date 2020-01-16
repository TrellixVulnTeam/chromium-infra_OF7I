// Copyright 2019 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scheduler

import (
	"context"
	"testing"
	"time"

	"infra/qscheduler/qslib/protos/metrics"
	"infra/qscheduler/qslib/tutils"

	"github.com/google/go-cmp/cmp"
	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/luci/common/data/stringset"
)

func TestToMetricsSchedulerState(t *testing.T) {
	Convey("Given a state with some balances, accounts, and requests", t, func() {
		ctx := context.Background()
		tm := time.Unix(100, 0).UTC()
		s := New(tm)
		s.AddAccount(ctx, "aid2", NewAccountConfig(1, 1, []float32{2, 3, 4}, false, ""), []float32{1, 2, 3, 4, 5})
		s.AddAccount(ctx, "aid1", NewAccountConfig(1, 1, []float32{2, 3, 4}, false, ""), []float32{5, 4, 3, 2, 1})
		// req1 and req2 should go to the running tasks.
		s.AddRequest(ctx, NewTaskRequest("req1", "a1", stringset.NewFromSlice("provision 1", "provision 2"), stringset.NewFromSlice("base 2", "base 1"), tm), tm, nil, NullEventSink)
		s.AddRequest(ctx, NewTaskRequest("req2", "a1", stringset.NewFromSlice("provision 3", "provision 4"), stringset.NewFromSlice("base 4", "base 3"), tm), tm, nil, NullEventSink)
		// req3 and req4 should go to the queued tasks.
		s.AddRequest(ctx, NewTaskRequest("req3", "a1", stringset.NewFromSlice("provision 5", "provision 6"), stringset.NewFromSlice("base 6", "base 5"), tm), tm, nil, NullEventSink)
		s.AddRequest(ctx, NewTaskRequest("req4", "a1", stringset.NewFromSlice("provision 7", "provision 8"), stringset.NewFromSlice("base 8", "base 7"), tm), tm, nil, NullEventSink)

		// worker 1 should run req1.
		s.MarkIdle(ctx, "worker 1", stringset.NewFromSlice("base 1", "base 2"), tm, NullEventSink)
		// worker 2 should run req2.
		s.MarkIdle(ctx, "worker 2", stringset.NewFromSlice("base 3", "base 4"), tm, NullEventSink)
		// worker 3 and 4 should remain idle.
		s.MarkIdle(ctx, "worker 3", stringset.NewFromSlice("base foo", "base bar"), tm, NullEventSink)
		s.MarkIdle(ctx, "worker 4", stringset.NewFromSlice("base foo", "base bar"), tm, NullEventSink)

		s.RunOnce(ctx, NullEventSink)

		Convey("test the state is transformed to metrics.SchedulerState.", func() {
			pool := "foo_pool"
			accounts := []*metrics.Account{
				{
					Id:           &metrics.Account_ID{Name: "aid1"},
					Balance:      []float32{5, 4, 3, 2, 1},
					Pool:         &metrics.Pool{Id: pool},
					SnapshotTime: tutils.TimestampProto(tm),
				},
				{
					Id:           &metrics.Account_ID{Name: "aid2"},
					Balance:      []float32{1, 2, 3, 4, 5},
					Pool:         &metrics.Pool{Id: pool},
					SnapshotTime: tutils.TimestampProto(tm),
				},
			}

			tasks := []*metrics.Task{
				{
					Id:                  &metrics.Task_ID{Name: "req1"},
					AccountId:           &metrics.Account_ID{Name: "a1"},
					WorkerId:            &metrics.Worker_ID{Name: "worker 1"},
					BaseLabels:          []string{"base 1", "base 2"},
					ProvisionableLabels: []string{"provision 1", "provision 2"},
					Pool:                &metrics.Pool{Id: pool},
					SnapshotTime:        tutils.TimestampProto(tm),
				},
				{
					Id:                  &metrics.Task_ID{Name: "req2"},
					AccountId:           &metrics.Account_ID{Name: "a1"},
					WorkerId:            &metrics.Worker_ID{Name: "worker 2"},
					BaseLabels:          []string{"base 3", "base 4"},
					ProvisionableLabels: []string{"provision 3", "provision 4"},
					Pool:                &metrics.Pool{Id: pool},
					SnapshotTime:        tutils.TimestampProto(tm),
				},
				{
					Id:                  &metrics.Task_ID{Name: "req3"},
					AccountId:           &metrics.Account_ID{Name: "a1"},
					WorkerId:            &metrics.Worker_ID{Name: ""},
					BaseLabels:          []string{"base 5", "base 6"},
					ProvisionableLabels: []string{"provision 5", "provision 6"},
					Pool:                &metrics.Pool{Id: pool},
					SnapshotTime:        tutils.TimestampProto(tm),
				},
				{
					Id:                  &metrics.Task_ID{Name: "req4"},
					AccountId:           &metrics.Account_ID{Name: "a1"},
					WorkerId:            &metrics.Worker_ID{Name: ""},
					BaseLabels:          []string{"base 7", "base 8"},
					ProvisionableLabels: []string{"provision 7", "provision 8"},
					Pool:                &metrics.Pool{Id: pool},
					SnapshotTime:        tutils.TimestampProto(tm),
				},
			}

			workers := []*metrics.Worker{
				{
					Id:           &metrics.Worker_ID{Name: "worker 1"},
					TaskId:       &metrics.Task_ID{Name: "req1"},
					Labels:       []string{"base 1", "base 2"},
					Pool:         &metrics.Pool{Id: pool},
					SnapshotTime: tutils.TimestampProto(tm),
				},
				{
					Id:           &metrics.Worker_ID{Name: "worker 2"},
					TaskId:       &metrics.Task_ID{Name: "req2"},
					Labels:       []string{"base 3", "base 4"},
					Pool:         &metrics.Pool{Id: pool},
					SnapshotTime: tutils.TimestampProto(tm),
				},
				{
					Id:           &metrics.Worker_ID{Name: "worker 3"},
					TaskId:       &metrics.Task_ID{Name: ""},
					Labels:       []string{"base bar", "base foo"},
					Pool:         &metrics.Pool{Id: pool},
					SnapshotTime: tutils.TimestampProto(tm),
				},
				{
					Id:           &metrics.Worker_ID{Name: "worker 4"},
					TaskId:       &metrics.Task_ID{Name: ""},
					Labels:       []string{"base bar", "base foo"},
					Pool:         &metrics.Pool{Id: pool},
					SnapshotTime: tutils.TimestampProto(tm),
				},
			}

			want := &Snapshot{
				accounts,
				tasks,
				workers,
			}
			diff := cmp.Diff(s.state.snapshot(pool), want)
			So(diff, ShouldBeBlank)
		})
	})
}
