// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package generator

import (
	"testing"

	"github.com/golang/protobuf/ptypes/wrappers"
	"go.chromium.org/chromiumos/infra/proto/go/testplans"
	bbproto "go.chromium.org/luci/buildbucket/proto"
)

func groupsObjs(groups []string) []*testplans.TestSuiteCommon_TestSuiteGroup {
	newGroups := make([]*testplans.TestSuiteCommon_TestSuiteGroup, len(groups))
	for i, g := range groups {
		newGroups[i] = &testplans.TestSuiteCommon_TestSuiteGroup{TestSuiteGroup: g}
	}
	return newGroups
}

func hwBuildResult(buildTarget, builderName, skylabBoard string, criticalSuite, criticalBuild bool, groups []string) buildResult {
	crit := bbproto.Trinary_NO
	if criticalBuild {
		crit = bbproto.Trinary_YES
	}
	return buildResult{
		buildID: buildID{buildTarget: buildTarget, builderName: builderName},
		build:   &bbproto.Build{Critical: crit},
		perTargetTestReqs: &testplans.PerTargetTestRequirements{
			TargetCriteria: &testplans.TargetCriteria{
				BuilderName: builderName,
			},
			HwTestCfg: &testplans.HwTestCfg{HwTest: []*testplans.HwTestCfg_HwTest{
				{
					Common: &testplans.TestSuiteCommon{
						DisplayName:     builderName + ".bvt-tast-cq",
						Critical:        &wrappers.BoolValue{Value: criticalSuite},
						TestSuiteGroups: groupsObjs(groups),
					},
					SkylabBoard:     skylabBoard,
					HwTestSuiteType: testplans.HwTestCfg_TAST},
			}},
		},
	}
}

func vmBuildResult(buildTarget, builderName string, criticalSuite, criticalBuild bool, groups []string) buildResult {
	crit := bbproto.Trinary_NO
	if criticalBuild {
		crit = bbproto.Trinary_YES
	}
	return buildResult{
		buildID: buildID{buildTarget: buildTarget, builderName: builderName},
		build:   &bbproto.Build{Critical: crit},
		perTargetTestReqs: &testplans.PerTargetTestRequirements{
			TargetCriteria: &testplans.TargetCriteria{
				BuilderName: builderName,
			},
			DirectTastVmTestCfg: &testplans.TastVmTestCfg{TastVmTest: []*testplans.TastVmTestCfg_TastVmTest{
				{
					Common: &testplans.TestSuiteCommon{
						DisplayName:     builderName + ".bvt-tast-vm-cq",
						Critical:        &wrappers.BoolValue{Value: criticalSuite},
						TestSuiteGroups: groupsObjs(groups),
					},
				}},
			},
		}}
}

func Test_SortOrder(t *testing.T) {
	br := []buildResult{
		// This will be the second ranked suite, since it's hw (vm goes first) and coral is a very common board.
		hwBuildResult("coral-arc-r", "coral-arc-r-cq", "coral", true, true, []string{"testGroup 1", "testGroup 5"}),
		hwBuildResult("sarien", "sarien-cq", "sarien", true, true, []string{"testGroup 1"}),
		// This will be last overall, since it's a noncritical suite.
		hwBuildResult("ocean", "ocean-cq", "ocean", false, true, []string{"testGroup 1"}),
		hwBuildResult("ocean-bark-r", "ocean-bark-r-cq", "ocean", true, false, []string{"testGroup 1"}),
		hwBuildResult("coral", "coral-cq", "coral", true, true, []string{"testGroup 1"}),
		// This will be the overall top priority, since it's a critical VM test suite.
		vmBuildResult("betty", "betty-arc-b-cq", true, true, []string{"testGroup 1"}),
		vmBuildResult("betty", "betty-shark-cq", false, true, []string{"testGroup 1"}),
	}

	boardPriorityList := &testplans.BoardPriorityList{
		BoardPriorities: []*testplans.BoardPriority{
			{SkylabBoard: "ocean", Priority: -100},
			{SkylabBoard: "coral", Priority: -6},
			{SkylabBoard: "sarien", Priority: 2},
			{SkylabBoard: "eve", Priority: 5},
		},
	}

	r, err := groupAndSort(br, boardPriorityList)
	if err != nil {
		t.Error(err)
	}

	group5 := r["testGroup 5"]
	if len(group5) != 1 {
		t.Errorf("wanted %v suites in testGroup 5, got %v", 1, len(group5))
	}
	if group5[0].tsc.GetDisplayName() != "coral-arc-r-cq.bvt-tast-cq" {
		t.Errorf("wanted %v as suite in testGroup 5, got %v", "coral-arc-r-cq.bvt-tast-cq", group5[0].tsc.GetDisplayName())
	}

	group1 := r["testGroup 1"]
	if len(group1) != 7 {
		t.Errorf("wanted %v suites in testGroup 1, got %v", 7, len(group1))
	}
	if group1[0].tsc.GetDisplayName() != "betty-arc-b-cq.bvt-tast-vm-cq" {
		t.Errorf("wanted %v as first suite in testGroup 1, got %v", "betty-arc-b-cq.bvt-tast-vm-cq", group1[0].tsc.GetDisplayName())
	}
	if group1[1].tsc.GetDisplayName() != "coral-arc-r-cq.bvt-tast-cq" {
		t.Errorf("wanted %v as second suite in testGroup 1, got %v", "coral-arc-r-cq.bvt-tast-cq", group1[1].tsc.GetDisplayName())
	}
	if group1[len(group1)-1].tsc.GetDisplayName() != "ocean-cq.bvt-tast-cq" {
		t.Errorf("wanted %v as last suite in testGroup 1, got %v", "ocean-cq.bvt-tast-cq", group1[len(group1)-1].tsc.GetDisplayName())
	}
}
