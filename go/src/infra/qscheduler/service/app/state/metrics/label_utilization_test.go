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
package metrics

import (
	"testing"

	"go.chromium.org/luci/common/data/stringset"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCompute(t *testing.T) {
	sID := "s1"
	c := newUtilizationCounter()
	Convey("Label utilization is correctly computed, and previously known labels are retained.", t, func() {
		utils := c.Compute(sID, labelDigest{})
		So(utils, ShouldNotBeNil)
		So(utils, ShouldBeEmpty)

		utils = c.Compute(sID, labelDigest{
			IdleWorkerLabels:      []stringset.Set{stringset.NewFromSlice("unused_1", "used_1")},
			RunningWorkerLabels:   []stringset.Set{stringset.NewFromSlice("unused_1", "used_2")},
			WaitingTaskBaseLabels: [][]string{{"used_1"}},
			RunningTaskBaseLabels: [][]string{{"used_2"}},
		})
		So(utils, ShouldResemble, map[string]*labelUtilization{
			"used_1": {IdleBots: 1, WaitingRequests: 1},
			"used_2": {RunningBots: 1, RunningRequests: 1},
		})

		utils = c.Compute(sID, labelDigest{})
		So(utils, ShouldResemble, map[string]*labelUtilization{
			"used_1": {},
			"used_2": {},
		})
	})
}
