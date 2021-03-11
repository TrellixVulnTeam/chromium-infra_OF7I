// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package generator

import (
	"fmt"
	"sort"

	"go.chromium.org/chromiumos/infra/proto/go/testplans"
	bbproto "go.chromium.org/luci/buildbucket/proto"
)

type testSuite struct {
	skylabBoard string
	build       *bbproto.Build
	tsc         *testplans.TestSuiteCommon
	isVM        bool
}

func tsc(buildResult buildResult, skylabBoard string, tsc *testplans.TestSuiteCommon, isVM bool, m map[testGroup][]testSuite) map[testGroup][]testSuite {
	ts := testSuite{
		tsc:         tsc,
		build:       buildResult.build,
		skylabBoard: skylabBoard,
		isVM:        isVM,
	}
	for _, tsg := range tsc.GetTestSuiteGroups() {
		g := testGroup(tsg.GetTestSuiteGroup())
		m[g] = append(m[g], ts)
	}
	return m
}

func (ts testSuite) String() string {
	return ts.tsc.GetDisplayName()
}

// groupAndSort groups known test suites by the test testGroup(s) they're in, then
// sorts each testGroup by the preference that the test plan generator should show
// toward elements in that testGroup. The first element in each testGroup is the one
// that the test plan generator is encouraged to schedule against first.
// This all supports oneof-based testing. See go/cq-oneof
func groupAndSort(buildResult []buildResult, boardPriorityList *testplans.BoardPriorityList) (map[testGroup][]testSuite, error) {
	m := make(map[testGroup][]testSuite)
	for _, br := range buildResult {
		req := br.perTargetTestReqs
		for _, t := range req.GetHwTestCfg().GetHwTest() {
			m = tsc(br, t.GetSkylabBoard(), t.GetCommon(), false, m)
		}
		for _, t := range req.GetDirectTastVmTestCfg().GetTastVmTest() {
			m = tsc(br, br.buildID.buildTarget, t.GetCommon(), true, m)
		}
		for _, t := range req.GetVmTestCfg().GetVmTest() {
			m = tsc(br, br.buildID.buildTarget, t.GetCommon(), true, m)
		}
	}

	boardPriorities := make(map[string]*testplans.BoardPriority)
	for _, boardPriority := range boardPriorityList.BoardPriorities {
		boardPriorities[boardPriority.SkylabBoard] = boardPriority
	}

	for _, suites := range m {
		for _, s := range suites {
			if s.tsc.GetDisplayName() == "" {
				// Display name is required in the generator as a key for referring to
				// each suite. All suites in the config therefore must have a display name.
				return nil, fmt.Errorf("missing display name for test suite %v", s)
			}
		}
		sort.Slice(suites, func(i, j int) bool {
			if suites[i].tsc.GetCritical().GetValue() != suites[j].tsc.GetCritical().GetValue() {
				// critical test suites at the front
				return suites[i].tsc.GetCritical().GetValue()
			}
			if suites[i].build.GetCritical() != suites[j].build.GetCritical() {
				// critical builds at the front
				return suites[i].build.GetCritical() == bbproto.Trinary_YES
			}
			if suites[i].isVM != suites[j].isVM {
				// always prefer VM tests
				return suites[i].isVM
			}
			if !suites[i].isVM && !suites[j].isVM {
				// then prefer the board with the least oversubscription
				if boardPriorities[suites[i].skylabBoard].GetPriority() != boardPriorities[suites[j].skylabBoard].GetPriority() {
					return boardPriorities[suites[i].skylabBoard].GetPriority() < boardPriorities[suites[j].skylabBoard].GetPriority()
				}
			}
			// finally sort by name, just for a stable sort
			return suites[i].tsc.GetDisplayName() < suites[j].tsc.GetDisplayName()
		})
	}
	return m, nil
}
