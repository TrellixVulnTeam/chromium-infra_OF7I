// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package generator

import (
	"fmt"
	"log"

	"infra/cros/internal/match"

	"go.chromium.org/chromiumos/infra/proto/go/testplans"
)

type testType int

type testGroup string

const (
	hw testType = iota
	vm
)

var (
	testTypeFilter = map[testType]func(testReqs *testplans.SourceTestRules) bool{
		hw: func(testReqs *testplans.SourceTestRules) bool {
			return testReqs.GetSubtractiveRule().GetDisableHwTests()
		},
		vm: func(testReqs *testplans.SourceTestRules) bool {
			return testReqs.GetSubtractiveRule().GetDisableVmTests()
		},
	}
)

func (tt testType) String() string {
	return [...]string{"Hw", "Vm"}[tt]
}

type testPruneResult struct {
	disableHWTests                bool
	disableVMTests                bool
	onlyKeepAllSuitesInGroups     map[testGroup]bool
	onlyKeepOneSuiteFromEachGroup map[testGroup]bool
	addAllSuitesInGroups          map[testGroup]bool
	addOneSuiteFromEachGroup      map[testGroup]bool
}

func (tpr testPruneResult) hasOnlyKeepSuiteRules() bool {
	return len(tpr.onlyKeepOneSuiteFromEachGroup) > 0 || len(tpr.onlyKeepAllSuitesInGroups) > 0
}

func (tpr testPruneResult) hasAddAllOrOneTestRules() bool {
	return len(tpr.addOneSuiteFromEachGroup) > 0 || len(tpr.addAllSuitesInGroups) > 0
}

func (tpr testPruneResult) canSkipForOnlyTestRule(groups []*testplans.TestSuiteCommon_TestSuiteGroup) bool {
	// If the source config didn't specify any onlyKeepAllSuitesInGroups, we can't skip testing for the groups in the params.
	if len(tpr.onlyKeepAllSuitesInGroups) == 0 {
		return false
	}
	for _, g := range groups {
		if tpr.onlyKeepAllSuitesInGroups[testGroup(g.TestSuiteGroup)] {
			return false
		}
	}
	return true
}

func (tpr testPruneResult) mustAddForAlsoTestRule(groups []*testplans.TestSuiteCommon_TestSuiteGroup) bool {
	for _, g := range groups {
		if tpr.addAllSuitesInGroups[testGroup(g.TestSuiteGroup)] {
			return true
		}
	}
	return false
}

func extractPruneResult(
	sourceTreeCfg *testplans.SourceTreeTestCfg,
	srcPaths []string) (*testPruneResult, error) {

	result := &testPruneResult{}

	if len(srcPaths) == 0 {
		// This happens during postsubmit runs, for example.
		log.Print("no gerrit_changes, so no tests will be skipped")
		return result, nil
	}

	disableHW := true
disableHWLoop:
	for _, fileSrcPath := range srcPaths {
		disableHWForPath, err := canDisableTestingForPath(fileSrcPath, sourceTreeCfg, hw)
		if err != nil {
			return result, err
		}
		if !disableHWForPath {
			log.Printf("cannot disable HW testing due to file %s", fileSrcPath)
			disableHW = false
			break disableHWLoop
		}
	}
	disableVM := true
disableVMLoop:
	for _, fileSrcPath := range srcPaths {
		disableVMForPath, err := canDisableTestingForPath(fileSrcPath, sourceTreeCfg, vm)
		if err != nil {
			return result, err
		}
		if !disableVMForPath {
			log.Printf("cannot disable VM testing due to file %s", fileSrcPath)
			disableVM = false
			break disableVMLoop
		}
	}

	onlyKeepAllSuitesInGroups := make(map[testGroup]bool)
	onlyKeepOneSuiteFromEachGroup := make(map[testGroup]bool)
fileLoop:
	for _, fileSrcPath := range srcPaths {
		fileOnlyTestGroups, err := getOnlyTestGroups(fileSrcPath, sourceTreeCfg)
		if err != nil {
			return result, err
		}
		fileOneofTestGroups, err := getOneofTestGroups(fileSrcPath, sourceTreeCfg)
		if err != nil {
			return result, err
		}
		if len(fileOnlyTestGroups) == 0 && len(fileOneofTestGroups) == 0 {
			log.Printf("cannot use subtractive test group rules on set of builders for testing due to %s", fileSrcPath)
			onlyKeepAllSuitesInGroups = make(map[testGroup]bool)
			onlyKeepOneSuiteFromEachGroup = make(map[testGroup]bool)
			break fileLoop
		} else {
			for g, include := range fileOnlyTestGroups {
				onlyKeepAllSuitesInGroups[g] = include
			}
			for g, include := range fileOneofTestGroups {
				onlyKeepOneSuiteFromEachGroup[g] = include
			}
		}

	}

	addAllSuitesInGroups := make(map[testGroup]bool)
	addOneSuiteFromEachGroup := make(map[testGroup]bool)
	for _, fileSrcPath := range srcPaths {
		fileAlsoTestGroups, err := getAddAllSuitesInGroups(fileSrcPath, sourceTreeCfg)
		if err != nil {
			return result, err
		}
		for k, v := range fileAlsoTestGroups {
			addAllSuitesInGroups[k] = v
			log.Printf("Will also test testGroup %v due to file %v", k, fileSrcPath)
		}

		fileAddOneSuiteFromEachGroup, err := getAddOneSuiteFromEachGroup(fileSrcPath, sourceTreeCfg)
		if err != nil {
			return result, err
		}
		for k, v := range fileAddOneSuiteFromEachGroup {
			addOneSuiteFromEachGroup[k] = v
			log.Printf("Will also test one suite from testGroup %v due to file %v", k, fileSrcPath)
		}
	}

	return &testPruneResult{
			disableHWTests:                disableHW,
			disableVMTests:                disableVM,
			onlyKeepAllSuitesInGroups:     onlyKeepAllSuitesInGroups,
			onlyKeepOneSuiteFromEachGroup: onlyKeepOneSuiteFromEachGroup,
			addAllSuitesInGroups:          addAllSuitesInGroups,
			addOneSuiteFromEachGroup:      addOneSuiteFromEachGroup},
		nil
}

func getOnlyTestGroups(
	sourcePath string,
	sourceTreeCfg *testplans.SourceTreeTestCfg) (map[testGroup]bool, error) {
	onlyTestGroups := make(map[testGroup]bool)
	for _, str := range sourceTreeCfg.GetSourceTestRules() {
		match, err := match.FilePatternMatches(str.GetFilePattern(), sourcePath)
		if err != nil {
			return onlyTestGroups, err
		}
		if match {
			okasig := str.GetSubtractiveRule().GetOnlyKeepAllSuitesInGroups()
			for _, o := range okasig.GetName() {
				onlyTestGroups[testGroup(o)] = true
			}
		}
	}
	return onlyTestGroups, nil
}

// getOneofTestGroups extracts rules from config about any type of oneof testing
// that can be done for the provided path. For each of the keys in the returned
// map, at least one test suite must be tested.
func getOneofTestGroups(
	sourcePath string,
	sourceTreeCfg *testplans.SourceTreeTestCfg) (map[testGroup]bool, error) {
	oneofTestGroups := make(map[testGroup]bool)
	for _, str := range sourceTreeCfg.GetSourceTestRules() {
		match, err := match.FilePatternMatches(str.GetFilePattern(), sourcePath)
		if err != nil {
			return oneofTestGroups, err
		}
		okosfeg := str.GetSubtractiveRule().GetOnlyKeepOneSuiteFromEachGroup()
		if match {
			for _, g := range okosfeg.GetName() {
				oneofTestGroups[testGroup(g)] = true
			}
		}
	}
	return oneofTestGroups, nil
}

func getAddAllSuitesInGroups(
	sourcePath string,
	sourceTreeCfg *testplans.SourceTreeTestCfg) (map[testGroup]bool, error) {
	alsoTestGroups := make(map[testGroup]bool)
	for _, str := range sourceTreeCfg.GetSourceTestRules() {
		match, err := match.FilePatternMatches(str.GetFilePattern(), sourcePath)
		if err != nil {
			return alsoTestGroups, err
		}
		if match {
			aasig := str.GetAdditiveRule().GetAddAllSuitesInGroups()
			for _, g := range aasig.GetName() {
				alsoTestGroups[testGroup(g)] = true
			}
		}
	}
	return alsoTestGroups, nil
}

func getAddOneSuiteFromEachGroup(
	sourcePath string,
	sourceTreeCfg *testplans.SourceTreeTestCfg) (map[testGroup]bool, error) {
	alsoTestOneofEachGroup := make(map[testGroup]bool)
	for _, str := range sourceTreeCfg.GetSourceTestRules() {
		match, err := match.FilePatternMatches(str.GetFilePattern(), sourcePath)
		if err != nil {
			return alsoTestOneofEachGroup, err
		}
		if match {
			aosfeg := str.GetAdditiveRule().GetAddOneSuiteFromEachGroup()
			for _, g := range aosfeg.GetName() {
				alsoTestOneofEachGroup[testGroup(g)] = true
			}
		}
	}
	return alsoTestOneofEachGroup, nil
}

// canDisableTestingForPath determines whether a particular testing type is unnecessary for
// a given file, based on source tree test restrictions.
func canDisableTestingForPath(sourcePath string, sourceTreeCfg *testplans.SourceTreeTestCfg, tt testType) (bool, error) {
	for _, str := range sourceTreeCfg.GetSourceTestRules() {
		testFilter, ok := testTypeFilter[tt]
		if !ok {
			return false, fmt.Errorf("Missing test filter for %v", tt)
		}
		if testFilter(str) {
			match, err := match.FilePatternMatches(str.GetFilePattern(), sourcePath)
			if err != nil {
				return false, err
			}
			if match {
				return true, nil
			}
		}
	}
	return false, nil
}
