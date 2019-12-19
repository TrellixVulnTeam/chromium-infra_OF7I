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
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/taskqueue"
	swarming "go.chromium.org/luci/common/api/swarming/swarming/v1"
	"go.chromium.org/luci/common/data/strpair"

	fleet "infra/appengine/crosskylabadmin/api/fleet/v1"
	"infra/appengine/crosskylabadmin/app/clients"
	"infra/appengine/crosskylabadmin/app/config"
	swarming_utils "infra/appengine/crosskylabadmin/app/frontend/internal/swarming"
	"infra/appengine/crosskylabadmin/app/frontend/test"
)

const repairQ = "repair-bots"
const resetQ = "reset-bots"
const repairLabstationQ = "repair-labstations"

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
		tf, validate := newTestFixture(t)
		defer validate()
		tqt := taskqueue.GetTestable(tf.C)
		tqt.CreateQueue(repairQ)
		tqt.CreateQueue(resetQ)
		bot1 := test.BotForDUT("dut_1", "needs_repair", "label-os_type:OS_TYPE_CROS")
		bot2 := test.BotForDUT("dut_2", "repair_failed", "label-os_type:OS_TYPE_CROS")
		bot3 := test.BotForDUT("dut_3", "needs_reset", "label-os_type:OS_TYPE_JETSTREAM")
		bots := []*swarming.SwarmingRpcsBotInfo{
			test.BotForDUT("dut_0", "ready", ""),
			test.BotForDUT("dut_l", "needs_repair", "label-os_type:OS_TYPE_LABSTATION"),
			bot1,
			bot2,
			bot3,
		}
		getDutName := func(bot *swarming.SwarmingRpcsBotInfo) string {
			h, err := swarming_utils.ExtractSingleValuedDimension(swarming_utils.DimensionsMap(bot.Dimensions), clients.DutNameDimensionKey)
			if err != nil {
				t.Fatalf("fail to extract dut_name for bot %s", bot1.BotId)
			}
			return h
		}
		h1 := getDutName(bot1)
		h2 := getDutName(bot2)
		h3 := getDutName(bot3)
		tf.MockSwarming.EXPECT().ListAliveIdleBotsInPool(
			gomock.Any(), gomock.Eq(config.Get(tf.C).Swarming.BotPool), gomock.Any(),
		).AnyTimes().Return(bots, nil)
		expectDefaultPerBotRefresh(tf)
		_, err := tf.Tracker.PushBotsForAdminTasks(tf.C, &fleet.PushBotsForAdminTasksRequest{})
		So(err, ShouldBeNil)

		tasks := tqt.GetScheduledTasks()
		fmt.Println(tasks)
		repairTasks, ok := tasks[repairQ]
		So(ok, ShouldBeTrue)
		var repairPaths []string
		for _, v := range repairTasks {
			repairPaths = append(repairPaths, v.Path)
		}
		sort.Strings(repairPaths)
		expectedPaths := []string{
			fmt.Sprintf("/internal/task/cros_repair/%s", h1),
			fmt.Sprintf("/internal/task/cros_repair/%s", h2),
		}
		sort.Strings(expectedPaths)
		So(repairPaths, ShouldResemble, expectedPaths)

		resetTasks, ok := tasks[resetQ]
		So(ok, ShouldBeTrue)
		var resetPaths []string
		for _, v := range resetTasks {
			resetPaths = append(resetPaths, v.Path)
		}
		expectedPaths = []string{
			fmt.Sprintf("/internal/task/reset/%s", h3),
		}
		So(resetPaths, ShouldResemble, expectedPaths)
	})
}

func TestPushLabstationsForRepair(t *testing.T) {
	Convey("Handling labstation bots", t, func() {
		tf, validate := newTestFixture(t)
		defer validate()
		tqt := taskqueue.GetTestable(tf.C)
		tqt.CreateQueue(repairLabstationQ)
		bot1 := test.BotForDUT("dut_1", "needs_repair", "label-os_type:OS_TYPE_LABSTATION;label-pool:labstation_main")
		bot2 := test.BotForDUT("dut_2", "ready", "label-os_type:OS_TYPE_LABSTATION;label-pool:labstation_main")
		bot3 := test.BotForDUT("dut_2", "ready", "label-os_type:OS_TYPE_LABSTATION;label-pool:DUT_POOL_QUOTA")
		bots := []*swarming.SwarmingRpcsBotInfo{
			bot1,
			bot2,
			bot3,
		}
		getDutName := func(bot *swarming.SwarmingRpcsBotInfo) string {
			h, err := swarming_utils.ExtractSingleValuedDimension(swarming_utils.DimensionsMap(bot.Dimensions), clients.DutNameDimensionKey)
			if err != nil {
				t.Fatalf("fail to extract dut_name for bot %s", bot1.BotId)
			}
			return h
		}
		h1 := getDutName(bot1)
		h2 := getDutName(bot2)
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
			fmt.Sprintf("/internal/task/labstation_repair/%s", h1),
			fmt.Sprintf("/internal/task/labstation_repair/%s", h2),
		}
		sort.Strings(expectedPaths)
		So(repairPaths, ShouldResemble, expectedPaths)
	})
}
