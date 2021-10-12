// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package resultcollector

import (
	"testing"

	rdbpb "go.chromium.org/luci/resultdb/proto/v1"
	"go.chromium.org/luci/server/tq"

	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSchedule(t *testing.T) {
	Convey(`TestSchedule`, t, func() {
		ctx, skdr := tq.TestingContext(testutil.TestingContext(), nil)
		RegisterTasksClass()

		inv := &rdbpb.Invocation{
			Name:  "invocations/build-87654321",
			Realm: "chromium:ci",
		}
		task := &taskspb.CollectTestResults{
			Resultdb: &taskspb.ResultDB{
				Invocation: inv,
				Host:       "results.api.cr.dev",
			},
			Builder:                   "Linux Tests",
			IsPreSubmit:               false,
			ContributedToClSubmission: false,
		}
		So(Schedule(ctx, inv, task.Resultdb.Host, task.Builder, false, false), ShouldBeNil)
		So(skdr.Tasks().Payloads()[0], ShouldResembleProto, task)
	})
}
