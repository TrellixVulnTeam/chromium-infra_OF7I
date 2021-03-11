// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package generator

import (
	"errors"
	"fmt"
	"log"

	"infra/cros/internal/gerrit"

	"github.com/golang/protobuf/ptypes/wrappers"
	"go.chromium.org/chromiumos/infra/proto/go/chromiumos"
	"go.chromium.org/chromiumos/infra/proto/go/testplans"
	bbproto "go.chromium.org/luci/buildbucket/proto"
)

type buildID struct {
	buildTarget string
	builderName string
}

// buildResult is a conglomeration of data about a build and how to test it.
type buildResult struct {
	buildID           buildID
	build             *bbproto.Build
	perTargetTestReqs *testplans.PerTargetTestRequirements
}

type suitesForGroups struct {
	// Keys are names of suites that must be included in the results after
	// applying subtractive rules. Value will always be true.
	onlyKeepSuites map[string]bool
	// Keys are names of suites that must be included in the results as a result
	// of additive rules. Value will always be true.
	additionalSuites map[string]bool
}

// CreateTestPlan generates the test plan that must be run as part of a Chrome OS build.
func CreateTestPlan(
	targetTestReqs *testplans.TargetTestRequirementsCfg,
	sourceTreeCfg *testplans.SourceTreeTestCfg,
	boardPriorityList *testplans.BoardPriorityList,
	unfilteredBbBuilds []*bbproto.Build,
	gerritChanges []*bbproto.GerritChange,
	changeRevs *gerrit.ChangeRevData,
	repoToBranchToSrcRoot map[string]map[string]string) (*testplans.GenerateTestPlanResponse, error) {
	testPlan := &testplans.GenerateTestPlanResponse{}

	// Match up the builds from the input to test requirements in the config.
	targetBuildResults := make([]buildResult, 0)
	buildResults := eligibleTestBuilds(unfilteredBbBuilds)
perTargetTestReq:
	for _, pttr := range targetTestReqs.PerTargetTestRequirements {
		tbr, err := selectBuildForRequirements(pttr, buildResults)
		if err != nil {
			return testPlan, err
		}
		if tbr == nil {
			// Occurs when there are no affected builds for this TargetTestRequirement.
			continue perTargetTestReq
		}
		targetBuildResults = append(targetBuildResults, *tbr)
	}

	// Get the source paths of files in the CL(s), and figure out any test pruning
	// possibilities based on those files.
	srcPaths, err := srcPaths(gerritChanges, changeRevs, repoToBranchToSrcRoot)
	if err != nil {
		return testPlan, err
	}
	pruneResult, err := extractPruneResult(sourceTreeCfg, srcPaths)
	if err != nil {
		return testPlan, err
	}

	// If oneof or only rules were found for the provided source paths, figure
	// out the right suites to test for those rules.
	onlyKeepSuites := make(map[string]bool)
	if pruneResult.hasOnlyKeepSuiteRules() {
		onlyKeepSuites, err = getOnlyKeepAllSuitesAndOneofSuites(targetBuildResults, pruneResult, boardPriorityList)
		if err != nil {
			return nil, err
		}
	}
	additionalSuites := make(map[string]bool)
	if pruneResult.hasAddAllOrOneTestRules() {
		additionalSuites, err = getAddAllSuitesAndOneofSuites(targetBuildResults, pruneResult, boardPriorityList)
		if err != nil {
			return nil, err
		}
	}
	sfg := suitesForGroups{
		onlyKeepSuites:   onlyKeepSuites,
		additionalSuites: additionalSuites,
	}
	return createResponse(targetBuildResults, pruneResult, sfg)
}

func getOnlyKeepAllSuitesAndOneofSuites(targetBuildResults []buildResult, pruneResult *testPruneResult, boardPriorityList *testplans.BoardPriorityList) (map[string]bool, error) {
	// Test group --> test suites, sorted in descending order of preference that the
	// planner should use to pick from the group.
	groupsToSortedSuites, err := groupAndSort(targetBuildResults, boardPriorityList)
	if err != nil {
		return nil, err
	}

	suitesForOneofAndOnly := make(map[string]bool)
	if pruneResult.hasOnlyKeepSuiteRules() {
		for onlyGroup := range pruneResult.onlyKeepAllSuitesInGroups {
			sorted := groupsToSortedSuites[onlyGroup]
			for _, s := range sorted {
				suitesForOneofAndOnly[s.tsc.GetDisplayName()] = true
				log.Printf("Using OnlyTest rule for testGroup %v, adding %v", onlyGroup, s.tsc.GetDisplayName())
			}
		}
		for oneofGroup := range pruneResult.onlyKeepOneSuiteFromEachGroup {
			sorted := groupsToSortedSuites[oneofGroup]

			if len(sorted) > 0 {
				suitesForOneofAndOnly[sorted[0].tsc.GetDisplayName()] = true
				log.Printf("Using OneOfTest rule for testGroup %v, adding %v", oneofGroup, sorted[0].tsc.GetDisplayName())
			}
		}
	}
	log.Printf("OnlyKeepAllSuitesAndOneofSuites: %v", suitesForOneofAndOnly)
	return suitesForOneofAndOnly, nil
}

func getAddAllSuitesAndOneofSuites(targetBuildResults []buildResult, pruneResult *testPruneResult, boardPriorityList *testplans.BoardPriorityList) (map[string]bool, error) {
	// Test group --> test suites, sorted in descending order of preference that the
	// planner should use to pick from the group.
	groupsToSortedSuites, err := groupAndSort(targetBuildResults, boardPriorityList)
	if err != nil {
		return nil, err
	}

	suitesForAddAllAndOneof := make(map[string]bool)
	if pruneResult.hasAddAllOrOneTestRules() {
		for g := range pruneResult.addAllSuitesInGroups {
			sorted := groupsToSortedSuites[g]
			for _, s := range sorted {
				suitesForAddAllAndOneof[s.tsc.GetDisplayName()] = true
				log.Printf("Using AddAllSuitesInGroups rule for testGroup %v, adding %v", g, s.tsc.GetDisplayName())
			}
		}
		for g := range pruneResult.addOneSuiteFromEachGroup {
			sorted := groupsToSortedSuites[g]
			if len(sorted) > 0 {
				suitesForAddAllAndOneof[sorted[0].tsc.GetDisplayName()] = true
				log.Printf("Using AddOneSuiteFromEachGroup rule for testGroup %v, adding %v", g, sorted[0].tsc.GetDisplayName())
			}
		}
	}
	log.Printf("AddAllSuitesAndOneofSuites: %v", suitesForAddAllAndOneof)
	return suitesForAddAllAndOneof, nil
}

func eligibleTestBuilds(unfilteredBbBuilds []*bbproto.Build) map[buildID]*bbproto.Build {
	buildIDToBuild := make(map[buildID]*bbproto.Build)
	for _, bb := range unfilteredBbBuilds {
		bt := getBuildTarget(bb)
		if len(bt) == 0 {
			log.Printf("filtering out build without a build target: %s", bb.GetBuilder().GetBuilder())
		} else if isPointlessBuild(bb) {
			log.Printf("filtering out because marked as pointless: %s", bb.GetBuilder().GetBuilder())
		} else if !hasTestArtifacts(bb) {
			log.Printf("filtering out with missing test artifacts: %s", bb.GetBuilder().GetBuilder())
		} else {
			buildIDToBuild[buildID{buildTarget: bt, builderName: bb.GetBuilder().GetBuilder()}] = bb
		}
	}
	return buildIDToBuild
}

func isPointlessBuild(bb *bbproto.Build) bool {
	pointlessBuild, ok := bb.GetOutput().GetProperties().GetFields()["pointless_build"]
	return ok && pointlessBuild.GetBoolValue()
}

func hasTestArtifacts(b *bbproto.Build) bool {
	art, ok := b.GetOutput().GetProperties().GetFields()["artifacts"]
	if !ok {
		return false
	}
	fba, ok := art.GetStructValue().GetFields()["files_by_artifact"]
	if !ok {
		return false
	}

	// The presence of any one of these artifacts is enough to tell us that this
	// build should be considered for testing.
	testArtifacts := []string{
		"AUTOTEST_FILES",
		"IMAGE_ZIP",
		"PINNED_GUEST_IMAGES",
		"TAST_FILES",
		"TEST_UPDATE_PAYLOAD",
	}
	fileToArtifact := fba.GetStructValue().GetFields()
	for _, ta := range testArtifacts {
		if _, ok := fileToArtifact[ta]; ok {
			return true
		}
	}
	return false
}

// getBuildTarget returns the build target from the given build, or empty string if none is found.
func getBuildTarget(bb *bbproto.Build) string {
	btStruct, ok := bb.Output.Properties.Fields["build_target"]
	if !ok {
		return ""
	}
	bt, ok := btStruct.GetStructValue().Fields["name"]
	if !ok {
		return ""
	}
	return bt.GetStringValue()
}

// createResponse creates the final GenerateTestPlanResponse.
func createResponse(targetBuildResults []buildResult, pruneResult *testPruneResult, sfg suitesForGroups) (*testplans.GenerateTestPlanResponse, error) {

	resp := &testplans.GenerateTestPlanResponse{}
	// loop over the merged (Buildbucket build, TargetTestRequirements).
	for _, tbr := range targetBuildResults {
		art, ok := tbr.build.Output.Properties.Fields["artifacts"]
		if !ok {
			return nil, fmt.Errorf("found no artifacts output property for builder %s", tbr.buildID.builderName)
		}
		gsBucket, ok := art.GetStructValue().Fields["gs_bucket"]
		if !ok {
			return nil, fmt.Errorf("found no artifacts.gs_bucket property for builder %s", tbr.buildID.builderName)
		}
		gsPath, ok := art.GetStructValue().Fields["gs_path"]
		if !ok {
			return nil, fmt.Errorf("found no artifacts.gs_path property for builder %s", tbr.buildID.builderName)
		}
		filesByArtifact, ok := art.GetStructValue().Fields["files_by_artifact"]
		if !ok {
			return nil, fmt.Errorf("found no artifacts.files_by_artifact property for builder %s", tbr.buildID.builderName)
		}
		bp := &testplans.BuildPayload{
			ArtifactsGsBucket: gsBucket.GetStringValue(),
			ArtifactsGsPath:   gsPath.GetStringValue(),
			FilesByArtifact:   filesByArtifact.GetStructValue(),
		}
		pttr := tbr.perTargetTestReqs
		bt := chromiumos.BuildTarget{Name: tbr.buildID.buildTarget}
		tuc := &testplans.TestUnitCommon{BuildTarget: &bt, BuildPayload: bp, BuilderName: tbr.buildID.builderName}
		criticalBuild := tbr.build.Critical != bbproto.Trinary_NO
		if !criticalBuild {
			// We formerly didn't test noncritical builders, but now we do.
			// See https://crbug.com/1040602.
			log.Printf("Builder %s is not critical, but we can still test it.", tbr.buildID.builderName)
		}

		if pttr.HwTestCfg != nil {
			hwTestUnit := getHwTestUnit(tuc, pttr.HwTestCfg.HwTest, pruneResult, sfg, criticalBuild)
			if hwTestUnit != nil {
				resp.HwTestUnits = append(resp.HwTestUnits, hwTestUnit)
			}
		}
		if pttr.DirectTastVmTestCfg != nil {
			directTastVMTestUnit := getTastVMTestUnit(tuc, pttr.DirectTastVmTestCfg.TastVmTest, pruneResult, sfg, criticalBuild)
			if directTastVMTestUnit != nil {
				resp.DirectTastVmTestUnits = append(resp.DirectTastVmTestUnits, directTastVMTestUnit)
			}
		}
		if pttr.VmTestCfg != nil {
			vmTestUnit := getVMTestUnit(tuc, pttr.VmTestCfg.VmTest, pruneResult, sfg, criticalBuild)
			if vmTestUnit != nil {
				resp.VmTestUnits = append(resp.VmTestUnits, vmTestUnit)
			}
		}
	}
	return resp, nil
}

func getHwTestUnit(tuc *testplans.TestUnitCommon, tests []*testplans.HwTestCfg_HwTest, pruneResult *testPruneResult, sfg suitesForGroups, criticalBuild bool) *testplans.HwTestUnit {
	if tests == nil {
		return nil
	}
	tu := &testplans.HwTestUnit{
		Common:    tuc,
		HwTestCfg: &testplans.HwTestCfg{},
	}
testLoop:
	for _, t := range tests {
		if pruneResult.disableHWTests {
			log.Printf("no HW testing needed for %v", t.Common.DisplayName)
			continue testLoop
		}
		// Always test if there's an alsoTest rule.
		mustAlsoTest := sfg.additionalSuites[t.GetCommon().GetDisplayName()]
		if mustAlsoTest {
			log.Printf("Including %v due to additive test rule", t.GetCommon().GetDisplayName())
		} else {
			inOnlyTestMode := len(sfg.onlyKeepSuites) > 0
			if inOnlyTestMode {
				// If there are only/oneof rules in effect, we keep the suite if that
				// suite is in the `only` map, but not otherwise.
				testNotNeeded := !sfg.onlyKeepSuites[t.Common.GetDisplayName()]
				if testNotNeeded {
					log.Printf("using OnlyTest rule to skip HW testing for %v", t.Common.DisplayName)
					continue testLoop
				}
			} else {
				// If we have no only/oneof rules in effect, we keep the suite unless
				// there's a disableByDefault rule in effect.
				if t.Common.DisableByDefault {
					log.Printf("%v is disabled by default, and it was not triggered to be enabled", t.Common.DisplayName)
					continue testLoop
				}
			}
		}
		log.Printf("adding testing for %v", t.Common.DisplayName)
		t.Common = withCritical(t.Common, criticalBuild)
		tu.HwTestCfg.HwTest = append(tu.HwTestCfg.HwTest, t)
	}
	if len(tu.HwTestCfg.HwTest) > 0 {
		return tu
	}
	return nil
}

func getTastVMTestUnit(tuc *testplans.TestUnitCommon, tests []*testplans.TastVmTestCfg_TastVmTest, pruneResult *testPruneResult, sfg suitesForGroups, criticalBuild bool) *testplans.TastVmTestUnit {
	if tests == nil {
		return nil
	}
	tu := &testplans.TastVmTestUnit{
		Common:        tuc,
		TastVmTestCfg: &testplans.TastVmTestCfg{},
	}
testLoop:
	for _, t := range tests {
		if pruneResult.disableVMTests {
			log.Printf("no Tast VM testing needed for %v", t.Common.DisplayName)
			continue testLoop
		}
		// Always test if there's an alsoTest rule.
		mustAlsoTest := sfg.additionalSuites[t.GetCommon().GetDisplayName()]
		if mustAlsoTest {
			log.Printf("Including %v due to additive test rule", t.GetCommon().GetDisplayName())
		} else {
			inOnlyTestMode := len(sfg.onlyKeepSuites) > 0
			if inOnlyTestMode {
				// If there are only/oneof rules in effect, we keep the suite if that
				// suite is in the `only` map, but not otherwise.
				testNotNeeded := !sfg.onlyKeepSuites[t.Common.GetDisplayName()]
				if testNotNeeded {
					log.Printf("using OnlyTest rule to skip HW testing for %v", t.Common.DisplayName)
					continue testLoop
				}
			} else {
				// If we have no only/oneof rules in effect, we keep the suite unless
				// there's a disableByDefault rule in effect.
				if t.Common.DisableByDefault {
					log.Printf("%v is disabled by default, and it was not triggered to be enabled", t.Common.DisplayName)
					continue testLoop
				}
			}
		}
		log.Printf("adding testing for %v", t.Common.DisplayName)
		t.Common = withCritical(t.Common, criticalBuild)
		tu.TastVmTestCfg.TastVmTest = append(tu.TastVmTestCfg.TastVmTest, t)
	}
	if len(tu.TastVmTestCfg.TastVmTest) > 0 {
		return tu
	}
	return nil
}

func getVMTestUnit(tuc *testplans.TestUnitCommon, tests []*testplans.VmTestCfg_VmTest, pruneResult *testPruneResult, sfg suitesForGroups, criticalBuild bool) *testplans.VmTestUnit {
	if tests == nil {
		return nil
	}
	tu := &testplans.VmTestUnit{
		Common:    tuc,
		VmTestCfg: &testplans.VmTestCfg{},
	}
testLoop:
	for _, t := range tests {
		if pruneResult.disableVMTests {
			log.Printf("no VM testing needed for %v", t.Common.DisplayName)
			continue testLoop
		}
		// Always test if there's an alsoTest rule.
		mustAlsoTest := sfg.additionalSuites[t.GetCommon().GetDisplayName()]
		if mustAlsoTest {
			log.Printf("Including %v due to additive test rule", t.GetCommon().GetDisplayName())
		} else {
			inOnlyTestMode := len(sfg.onlyKeepSuites) > 0
			if inOnlyTestMode {
				// If there are only/oneof rules in effect, we keep the suite if that
				// suite is in the `only` map, but not otherwise.
				testNotNeeded := !sfg.onlyKeepSuites[t.Common.GetDisplayName()]
				if testNotNeeded {
					log.Printf("using OnlyTest rule to skip HW testing for %v", t.Common.DisplayName)
					continue testLoop
				}
			} else {
				// If we have no only/oneof rules in effect, we keep the suite unless
				// there's a disableByDefault rule in effect.
				if t.Common.DisableByDefault {
					log.Printf("%v is disabled by default, and it was not triggered to be enabled", t.Common.DisplayName)
					continue testLoop
				}
			}
		}
		log.Printf("adding testing for %v", t.Common.DisplayName)
		t.Common = withCritical(t.Common, criticalBuild)
		tu.VmTestCfg.VmTest = append(tu.VmTestCfg.VmTest, t)
	}
	if len(tu.VmTestCfg.VmTest) > 0 {
		return tu
	}
	return nil
}

func withCritical(tsc *testplans.TestSuiteCommon, buildCritical bool) *testplans.TestSuiteCommon {
	if tsc == nil {
		tsc = &testplans.TestSuiteCommon{}
	}
	suiteCritical := true
	if tsc.Critical != nil {
		suiteCritical = tsc.Critical.Value
	}
	// If either the build was noncritical or the suite is configured to be
	// noncritical, then make the suite noncritical. As of now we don't even
	// schedule suites for noncritical builders, but if we ever change that logic,
	// this seems like the right way to set suite criticality.
	tsc.Critical = &wrappers.BoolValue{Value: buildCritical && suiteCritical}
	if !tsc.Critical.Value {
		log.Printf("Marking %s as not critical", tsc.DisplayName)
	}
	return tsc
}

// selectBuildForRequirements finds a build that best matches the provided PerTargetTestRequirements.
// e.g. if the requirements want a build for a reef build target, this method will find a successful,
// non-early-terminated build.
func selectBuildForRequirements(
	pttr *testplans.PerTargetTestRequirements,
	buildIDToBuild map[buildID]*bbproto.Build) (*buildResult, error) {

	log.Printf("Considering testing for TargetCritera %v", pttr.TargetCriteria)
	if pttr.TargetCriteria.GetBuildTarget() == "" {
		return nil, errors.New("found a PerTargetTestRequirement without a build target")
	}
	eligibleBuildIds := []buildID{{pttr.TargetCriteria.GetBuildTarget(), pttr.TargetCriteria.GetBuilderName()}}
	bt, err := pickBuilderToTest(eligibleBuildIds, buildIDToBuild)
	if err != nil {
		// Expected when a necessary builder failed, and thus we cannot continue with testing.
		return nil, err
	}
	if bt == nil {
		// There are no builds for these test criteria, so this PerTargetTestRequirement is
		// irrelevant. Continue on to the next one.
		// This happens when no build was relevant due to an EarlyTerminationStatus.
		return nil, nil
	}
	br := buildIDToBuild[*bt]
	return &buildResult{
			build:             br,
			buildID:           buildID{buildTarget: getBuildTarget(br), builderName: br.GetBuilder().GetBuilder()},
			perTargetTestReqs: pttr},
		nil
}

// pickBuilderToTest returns up to one buildID that should be tested, out of the provided slice
// of buildIDs. The returned buildID, if present, is guaranteed to be one with a BuildResult.
func pickBuilderToTest(buildIDs []buildID, buildIDToBuild map[buildID]*bbproto.Build) (*buildID, error) {
	// Relevant results are those builds that weren't terminated early.
	// Early termination is a good thing. It just means that the build wasn't affected by the relevant commits.
	relevantReports := make(map[buildID]*bbproto.Build)
	for _, bt := range buildIDs {
		br, found := buildIDToBuild[bt]
		if !found {
			log.Printf("No build found for buildID %s", bt)
			continue
		}
		relevantReports[bt] = br
	}
	if len(relevantReports) == 0 {
		// None of the builds were relevant, so none of these builds needs testing.
		return nil, nil
	}
	for _, bt := range buildIDs {
		// Find and return the first relevant, successful build.
		result, found := relevantReports[bt]
		if found && result.Status == bbproto.Status_SUCCESS {
			return &bt, nil
		}
	}
	log.Printf("can't test for builders %v because all builders failed\n", buildIDs)
	return nil, nil
}

// srcPaths extracts the source paths from each of the provided Gerrit changes.
func srcPaths(
	changes []*bbproto.GerritChange,
	changeRevs *gerrit.ChangeRevData,
	repoToBranchToSrcRoot map[string]map[string]string) ([]string, error) {
	srcPaths := make([]string, 0)
changeLoop:
	for _, commit := range changes {
		chRev, err := changeRevs.GetChangeRev(commit.Host, commit.Change, int32(commit.Patchset))
		if err != nil {
			return srcPaths, err
		}
		for _, file := range chRev.Files {
			branchMapping, found := repoToBranchToSrcRoot[chRev.Project]
			if !found {
				return srcPaths, fmt.Errorf("Found no branch mapping for project %s", chRev.Project)
			}
			srcRootMapping, found := branchMapping[chRev.Branch]
			if !found {
				log.Printf("Found no source mapping for project %s and branch %s", chRev.Project, chRev.Branch)
				continue changeLoop
			}
			srcPaths = append(srcPaths, fmt.Sprintf("%s/%s", srcRootMapping, file))
		}
	}
	return srcPaths, nil
}
