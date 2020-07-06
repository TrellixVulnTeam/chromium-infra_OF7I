// Copyright 2018 The LUCI Authors.
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

package frontend

import (
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/app/frontend/internal/worker"
)

func TestRunTaskByBotID(t *testing.T) {
	Convey("with run repair job with BOT id", t, func() {
		tf, validate := newTestFixture(t)
		defer validate()

		expectTaskCreationForDUT(tf, "task1", "bot_id", 5, 15)
		at := worker.AdminTaskForType(tf.C, fleet.TaskType_Repair)
		taskURL, err := runTaskByBotID(tf.C, at, tf.MockSwarming, "bot_id", 5, 15)
		So(err, ShouldBeNil)
		So(taskURL, ShouldContainSubstring, "task1")
	})
	Convey("with run audit job with BOT id and custom experation and execution times", t, func() {
		tf, validate := newTestFixture(t)
		defer validate()
		expectTaskCreationForDUT(tf, "task1", "bot_id", 7200, 7200)
		at := worker.AuditTaskWithActions(tf.C, "action1,action2")
		So(len(at.Cmd), ShouldEqual, 7)
		So(at.Cmd[0], ShouldEqual, "/opt/infra-tools/skylab_swarming_worker")
		So(at.Cmd[1], ShouldEqual, "-actions")
		So(at.Cmd[2], ShouldEqual, "action1,action2")
		So(at.Cmd[3], ShouldEqual, "-logdog-annotation-url")
		So(at.Cmd[5], ShouldEqual, "-task-name")
		So(at.Cmd[6], ShouldEqual, "admin_audit")
		taskURL, err := runTaskByBotID(tf.C, at, tf.MockSwarming, "bot_id", 7200, 7200)
		So(err, ShouldBeNil)
		So(taskURL, ShouldContainSubstring, "task1")
	})
}

// expectTaskCreationByDUTName sets up the expectations for a single task creation based on DUT name.
func expectTaskCreationForDUT(tf testFixture, taskID, botID string, expSec, execTimeoutSecs int) *gomock.Call {
	m := &createTaskArgsMatcher{
		BotID:                botID,
		ExpirationSecs:       int64(expSec),
		ExecutionTimeoutSecs: int64(execTimeoutSecs),
	}
	return tf.MockSwarming.EXPECT().CreateTask(gomock.Any(), gomock.Any(), m).Return(taskID, nil)
}
