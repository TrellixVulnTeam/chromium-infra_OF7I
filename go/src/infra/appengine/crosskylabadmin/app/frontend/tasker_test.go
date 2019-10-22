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

func TestRunTaskByDUTName(t *testing.T) {
	Convey("with run repair job with DUT name", t, func() {
		tf, validate := newTestFixture(t)
		defer validate()
		expectTaskCreationForDUT(tf, "task1", "host1")
		at := worker.AdminTaskForType(tf.C, fleet.TaskType_Repair)
		taskURL, err := runTaskByDUTName(tf.C, at, tf.MockSwarming, "host1")
		So(err, ShouldBeNil)
		So(taskURL, ShouldContainSubstring, "task1")
	})
}

// expectTaskCreationByDUTName sets up the expectations for a single task creation based on DUT name.
func expectTaskCreationForDUT(tf testFixture, taskID string, hostname string) *gomock.Call {
	m := &createTaskArgsMatcher{
		DutName: hostname,
	}
	return tf.MockSwarming.EXPECT().CreateTask(gomock.Any(), gomock.Any(), m).Return(taskID, nil)
}
