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
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	swarming "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/data/strpair"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/internal/app/clients"
	"infra/appengine/crosskylabadmin/internal/app/config"
	"infra/appengine/crosskylabadmin/internal/app/frontend/test"
	"infra/appengine/crosskylabadmin/internal/tq"
)

const repairQ = "repair-bots"
const repairLabstationQ = "repair-labstations"
const auditQ = "audit-bots"

func TestFlattenAndDuplicateBots(t *testing.T) {
	Convey("zero bots", t, func() {
		tf, validate := newTestFixture(t)
		defer validate()

		tf.MockSwarming.EXPECT().ListAliveBotsInPool(
			gomock.Any(), gomock.Eq(config.Get(tf.C).Swarming.BotPool), gomock.Any(),
		).AnyTimes().Return([]*swarming.SwarmingRpcsBotInfo{}, nil)

		bots, err := tf.MockSwarming.ListAliveBotsInPool(tf.C, config.Get(tf.C).Swarming.BotPool, strpair.Map{})
		So(err, ShouldBeNil)
		bots = flattenAndDedpulicateBots([][]*swarming.SwarmingRpcsBotInfo{bots})
		So(bots, ShouldBeEmpty)
	})

	Convey("multiple bots", t, func() {
		tf, validate := newTestFixture(t)
		defer validate()

		sbots := []*swarming.SwarmingRpcsBotInfo{
			test.BotForDUT("dut_1", "ready", ""),
			test.BotForDUT("dut_2", "repair_failed", ""),
		}
		tf.MockSwarming.EXPECT().ListAliveBotsInPool(
			gomock.Any(), gomock.Eq(config.Get(tf.C).Swarming.BotPool), gomock.Any(),
		).AnyTimes().Return(sbots, nil)

		bots, err := tf.MockSwarming.ListAliveBotsInPool(tf.C, config.Get(tf.C).Swarming.BotPool, strpair.Map{})
		So(err, ShouldBeNil)
		bots = flattenAndDedpulicateBots([][]*swarming.SwarmingRpcsBotInfo{bots})
		So(bots, ShouldHaveLength, 2)
	})

	Convey("duplicated bots", t, func() {
		tf, validate := newTestFixture(t)
		defer validate()

		sbots := []*swarming.SwarmingRpcsBotInfo{
			test.BotForDUT("dut_1", "ready", ""),
			test.BotForDUT("dut_1", "repair_failed", ""),
		}
		tf.MockSwarming.EXPECT().ListAliveBotsInPool(
			gomock.Any(), gomock.Eq(config.Get(tf.C).Swarming.BotPool), gomock.Any(),
		).AnyTimes().Return(sbots, nil)

		bots, err := tf.MockSwarming.ListAliveBotsInPool(tf.C, config.Get(tf.C).Swarming.BotPool, strpair.Map{})
		So(err, ShouldBeNil)
		bots = flattenAndDedpulicateBots([][]*swarming.SwarmingRpcsBotInfo{bots})
		So(bots, ShouldHaveLength, 1)
	})
}
func TestPushBotsForAdminTasks(t *testing.T) {
	Convey("Handling 4 different state of cros bots", t, func() {
		bot1 := test.BotForDUT("dut_1", "needs_repair", "label-os_type:OS_TYPE_CROS;id:id1")
		bot2 := test.BotForDUT("dut_2", "repair_failed", "label-os_type:OS_TYPE_CROS;id:id2")
		bot3 := test.BotForDUT("dut_3", "needs_reset", "label-os_type:OS_TYPE_JETSTREAM;id:id3")
		bot4 := test.BotForDUT("dut_4", "needs_manual_repair", "label-os_type:OS_TYPE_JETSTREAM;id:id4")
		bot5 := test.BotForDUT("dut_5", "needs_replacement", "label-os_type:OS_TYPE_JETSTREAM;id:id5")
		bot1LabStation := test.BotForDUT("dut_1l", "needs_repair", "label-os_type:OS_TYPE_LABSTATION;id:lab_id1")
		bot1SchedulingUnit := test.BotForDUT("dut1su", "needs_repair", "id:su_id1")
		appendPaths := func(paths map[string]*tq.Task) (arr []string) {
			for _, v := range paths {
				arr = append(arr, v.Path)
			}
			return arr
		}
		validateTasksInQueue := func(tasks tq.QueueData, qKey string, qPath string, botIDs []string) {
			fmt.Println(tasks)
			repairTasks, ok := tasks[qKey]
			So(ok, ShouldBeTrue)
			repairPaths := appendPaths(repairTasks)
			var expectedPaths []string
			for _, botID := range botIDs {
				expectedPaths = append(expectedPaths, fmt.Sprintf("/internal/task/%s/%s", qPath, botID))
			}
			sort.Strings(repairPaths)
			sort.Strings(expectedPaths)
			So(repairPaths, ShouldResemble, expectedPaths)
		}
		tf, validate := newTestFixture(t)
		defer validate()
		tqt := tq.GetTestable(tf.C)
		tqt.CreateQueue(repairQ)

		Convey("run needs_repair status", func() {
			tqt.ResetTasks()
			tf.MockSwarming.EXPECT().ListAliveIdleBotsInPool(
				gomock.Any(), gomock.Eq(config.Get(tf.C).Swarming.BotPool),
				gomock.Eq(strpair.Map{clients.DutStateDimensionKey: {"needs_repair"}}),
			).AnyTimes().Return([]*swarming.SwarmingRpcsBotInfo{bot1, bot3, bot1LabStation, bot1SchedulingUnit}, nil)
			expectDefaultPerBotRefresh(tf)

			request := fleet.PushBotsForAdminTasksRequest{
				TargetDutState: fleet.DutState_NeedsRepair,
			}
			res, err := tf.Tracker.PushBotsForAdminTasks(tf.C, &request)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)

			tasks := tqt.GetScheduledTasks()
			validateTasksInQueue(tasks, repairQ, "cros_repair", []string{"id1", "su_id1"})
		})
		Convey("run only for repair_failed status", func() {
			tqt.ResetTasks()
			tf.MockSwarming.EXPECT().ListAliveIdleBotsInPool(
				gomock.Any(), gomock.Eq(config.Get(tf.C).Swarming.BotPool),
				gomock.Eq(strpair.Map{clients.DutStateDimensionKey: {"repair_failed"}}),
			).AnyTimes().Return([]*swarming.SwarmingRpcsBotInfo{bot2}, nil)
			expectDefaultPerBotRefresh(tf)

			request := fleet.PushBotsForAdminTasksRequest{
				TargetDutState: fleet.DutState_RepairFailed,
			}
			res, err := tf.Tracker.PushBotsForAdminTasks(tf.C, &request)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)

			tasks := tqt.GetScheduledTasks()
			validateTasksInQueue(tasks, repairQ, "cros_repair", []string{"id2"})
		})
		Convey("run only for needs_manual_repair status", func() {
			tqt.ResetTasks()
			tf.MockSwarming.EXPECT().ListAliveIdleBotsInPool(
				gomock.Any(),
				gomock.Eq(config.Get(tf.C).Swarming.BotPool),
				gomock.Eq(strpair.Map{clients.DutStateDimensionKey: {"needs_manual_repair"}}),
			).AnyTimes().Return([]*swarming.SwarmingRpcsBotInfo{bot3, bot4}, nil)
			expectDefaultPerBotRefresh(tf)
			request := fleet.PushBotsForAdminTasksRequest{
				TargetDutState: fleet.DutState_NeedsManualRepair,
			}
			res, err := tf.Tracker.PushBotsForAdminTasks(tf.C, &request)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)

			tasks := tqt.GetScheduledTasks()
			validateTasksInQueue(tasks, repairQ, "cros_repair", []string{"id4"})
		})
		Convey("don't run for needs_replacement status", func() {
			tqt.ResetTasks()
			tf.MockSwarming.EXPECT().ListAliveIdleBotsInPool(
				gomock.Any(),
				gomock.Eq(config.Get(tf.C).Swarming.BotPool),
				gomock.Eq(strpair.Map{clients.DutStateDimensionKey: {"needs_replacement"}}),
			).AnyTimes().Return([]*swarming.SwarmingRpcsBotInfo{bot3, bot5}, nil)
			expectDefaultPerBotRefresh(tf)
			request := fleet.PushBotsForAdminTasksRequest{
				TargetDutState: fleet.DutState_NeedsReplacement,
			}
			res, err := tf.Tracker.PushBotsForAdminTasks(tf.C, &request)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)

			tasks := tqt.GetScheduledTasks()
			validateTasksInQueue(tasks, repairQ, "cros_repair", []string{})
		})
	})
}

func TestPushBotsForAdminAuditTasks(t *testing.T) {
	Convey("Handling types of cros bots", t, func() {
		bot3 := test.BotForDUT("dut_3", "needs_repair", "label-os_type:OS_TYPE_MOBLAB;id:id3")
		bot4 := test.BotForDUT("dut_4", "ready", "label-os_type:OS_TYPE_MOBLAB;id:id4")
		bot5 := test.BotForDUT("dut_5", "needs_deploy", "label-os_type:OS_TYPE_MOBLAB;id:id5")
		bot6 := test.BotForDUT("dut_6", "needs_reset", "label-os_type:OS_TYPE_MOBLAB;id:id6")
		bot6.State = "{\"storage_state\":[\"NEED_REPLACEMENT\"],\"servo_usb_state\":[\"NEED_REPLACEMENT\"], \"rpm_state\": [\"UNKNOWN\"]}"
		bot7 := test.BotForDUT("dut_7", "needs_replacement", "label-os_type:OS_TYPE_MOBLAB;id:id7")
		bot2LabStation := test.BotForDUT("dut_2l", "ready", "label-os_type:OS_TYPE_LABSTATION;id:lab_id2")
		bot1SchedulingUnit := test.BotForDUT("dut1su", "ready", "id:su_id1")
		appendPaths := func(paths map[string]*tq.Task) (arr []string) {
			for _, v := range paths {
				arr = append(arr, v.Path)
			}
			return arr
		}
		validateTasksInQueue := func(tasks tq.QueueData, qKey, qPath string, botIDs, actions []string) {
			fmt.Println(tasks)
			repairTasks, ok := tasks[qKey]
			So(ok, ShouldBeTrue)
			repairPaths := appendPaths(repairTasks)
			var expectedPaths []string
			actionStr := strings.Join(actions, "-")
			for _, botID := range botIDs {
				expectedPaths = append(expectedPaths, fmt.Sprintf("/internal/task/%s/%s/%s", qPath, botID, actionStr))
			}
			sort.Strings(repairPaths)
			sort.Strings(expectedPaths)
			So(repairPaths, ShouldResemble, expectedPaths)
		}
		tf, validate := newTestFixture(t)
		defer validate()
		tqt := tq.GetTestable(tf.C)
		tqt.CreateQueue(auditQ)

		Convey("fail to run when actions is not specified", func() {
			request := fleet.PushBotsForAdminAuditTasksRequest{}
			res, err := tf.Tracker.PushBotsForAdminAuditTasks(tf.C, &request)
			So(err, ShouldNotBeNil)
			So(res, ShouldBeNil)
		})
		Convey("run only Servo USB-key check for all DUTs", func() {
			tqt.ResetTasks()
			tf.MockSwarming.EXPECT().ListAliveBotsInPool(
				gomock.Any(), gomock.Eq(config.Get(tf.C).Swarming.BotPool),
				gomock.Eq(strpair.Map{}),
			).AnyTimes().Return([]*swarming.SwarmingRpcsBotInfo{bot3, bot4, bot5, bot6, bot7, bot2LabStation, bot1SchedulingUnit}, nil)
			expectDefaultPerBotRefresh(tf)

			actions := []string{"verify-servo-usb-drive"}
			request := fleet.PushBotsForAdminAuditTasksRequest{
				Task: fleet.AuditTask_ServoUSBKey,
			}
			res, err := tf.Tracker.PushBotsForAdminAuditTasks(tf.C, &request)
			So(err, ShouldBeNil)
			So(res, ShouldNotBeNil)

			tasks := tqt.GetScheduledTasks()
			validateTasksInQueue(tasks, auditQ, "audit", []string{"id3", "id4", "id6", "su_id1"}, actions)
		})
	})
}

func TestPushLabstationsForRepair(t *testing.T) {
	Convey("Handling labstation bots", t, func() {
		tf, validate := newTestFixture(t)
		defer validate()
		tqt := tq.GetTestable(tf.C)
		tqt.CreateQueue(repairLabstationQ)
		bot1 := test.BotForDUT("dut_1", "needs_repair", "label-os_type:OS_TYPE_LABSTATION;label-pool:labstation_main;id:lab_1")
		bot2 := test.BotForDUT("dut_2", "ready", "label-os_type:OS_TYPE_LABSTATION;label-pool:servo_verification;id:lab_2")
		bots := []*swarming.SwarmingRpcsBotInfo{bot1, bot2}
		tf.MockSwarming.EXPECT().ListAliveIdleBotsInPool(
			gomock.Any(), gomock.Eq(config.Get(tf.C).Swarming.BotPool), gomock.Any(),
		).AnyTimes().Return(bots, nil)
		expectDefaultPerBotRefresh(tf)
		_, err := tf.Tracker.PushRepairJobsForLabstations(tf.C, &fleet.PushRepairJobsForLabstationsRequest{})
		So(err, ShouldBeNil)

		tasks := tqt.GetScheduledTasks()
		repairTasks, ok := tasks[repairLabstationQ]
		So(ok, ShouldBeTrue)
		var repairPaths []string
		for _, v := range repairTasks {
			repairPaths = append(repairPaths, v.Path)
		}
		sort.Strings(repairPaths)
		expectedPaths := []string{
			"/internal/task/labstation_repair/lab_1",
			"/internal/task/labstation_repair/lab_2",
		}
		So(repairPaths, ShouldResemble, expectedPaths)
	})
}
