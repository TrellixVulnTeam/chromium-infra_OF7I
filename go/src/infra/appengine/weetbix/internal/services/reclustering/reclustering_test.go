// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package reclustering

import (
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.chromium.org/luci/server/tq"
	_ "go.chromium.org/luci/server/tq/txn/spanner"

	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSchedule(t *testing.T) {
	Convey(`TestSchedule`, t, func() {
		ctx, skdr := tq.TestingContext(testutil.TestingContext(), nil)

		task := &taskspb.ReclusterChunks{
			Project:      "chromium",
			AttemptTime:  timestamppb.New(time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC)),
			StartChunkId: "",
			EndChunkId:   strings.Repeat("ff", 16),
		}
		expected := proto.Clone(task).(*taskspb.ReclusterChunks)
		So(Schedule(ctx, "chromium-20250101-120000-shard-1-of-1", task), ShouldBeNil)
		So(skdr.Tasks().Payloads()[0], ShouldResembleProto, expected)
	})
}
