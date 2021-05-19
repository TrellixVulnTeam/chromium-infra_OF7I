// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package branch

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	cv "infra/cros/internal/chromeosversion"
	"infra/cros/internal/git"
	"infra/cros/internal/osutils"
)

const (
	// Default location of manifest-internal project.
	manifestInternalProject = "manifest-internal"
	// Default location of chromiumos-overlay project.
	chromiumosOverlayProject = "src/third_party/chromiumos-overlay"
)

var (
	// Regex for extracting build number from branch names.
	branchRegexp = regexp.MustCompile(`(release|stabilize|firmware|factory)-(.*-)?(?P<vinfo>(\d+\.)+)B`)
	// Regex for extracting milestone from branch names.
	milestoneRegexp = regexp.MustCompile(`release-R(\d+)-.*`)
	// Regex for extracting version information from a buildspec path, e.g.
	// 85/13277.0.0.xml.
	buildspecRegexp = regexp.MustCompile(`^(?P<milestone>\d+)\/(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)\.xml$`)
)

// ExtractBuildNum extracts the build number from the branch name, e.g. 13729
// from release-R89-13729.B.
// Returns -1 if the branch name does not contain a build number.
func ExtractBuildNum(branch string) int {
	match := branchRegexp.FindStringSubmatch(branch)
	if match == nil || len(match) < 4 {
		return -1
	}
	result := make(map[string]string)
	for i, name := range branchRegexp.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	if vinfo, ok := result["vinfo"]; !ok {
		return -1
	} else if buildNum, err := strconv.Atoi(strings.Split(vinfo, ".")[0]); err != nil {
		return -1
	} else {
		return buildNum
	}
}

// ParseBuildspec returns version information for a buildspec path of the form
// 85/13277.0.0.xml.
func ParseBuildspec(buildspec string) (*cv.VersionInfo, error) {
	match := buildspecRegexp.FindStringSubmatch(buildspec)
	if match == nil || len(match) < 5 {
		return nil, fmt.Errorf("invalid buildspec")
	}
	result := make(map[string]string)
	for i, name := range buildspecRegexp.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	cmps := []int{}
	for _, component := range []string{"milestone", "major", "minor", "patch"} {
		cmp, ok := result[component]
		if !ok {
			return nil, fmt.Errorf("buildspec missing component %s", component)
		}
		num, err := strconv.Atoi(strings.Split(cmp, ".")[0])
		if err != nil {
			return nil, fmt.Errorf("bad component %s: %s", component, cmp)
		}
		cmps = append(cmps, num)
	}

	return &cv.VersionInfo{
		ChromeBranch:      cmps[0],
		BuildNumber:       cmps[1],
		BranchBuildNumber: cmps[2],
		PatchNumber:       cmps[3],
	}, nil
}

func releaseBranches(branchList []string, minMilestone int) ([]string, error) {
	branches := []string{}
	for _, branch := range branchList {
		if strings.HasPrefix(branch, fmt.Sprintf("release-R")) {
			match := milestoneRegexp.FindStringSubmatch(branch)
			if match == nil || len(match) < 2 {
				return nil, fmt.Errorf("malformatted release branch: %s", branch)
			}
			if milestone, err := strconv.Atoi(match[1]); err != nil {
				return nil, fmt.Errorf("malformatted release branch: %s", branch)
			} else if milestone >= minMilestone {
				branches = append(branches, branch)
			}
		}
	}
	return branches, nil
}

func nonReleaseBranches(branchList []string, minBuildNumber int) []string {
	branches := []string{}
	for _, branch := range branchList {
		if strings.HasPrefix(branch, "stabilize-") ||
			strings.HasPrefix(branch, "factory-") ||
			strings.HasPrefix(branch, "firmware-") {
			if ExtractBuildNum(branch) >= minBuildNumber {
				branches = append(branches, branch)
			}
		}
	}
	return branches
}

// BranchesFromMilestone returns a list of branch names in the sentinel
// manifest-internal repository associated with a milestone >= the specified
// minimum milestone.
func BranchesFromMilestone(chromeosCheckout string, minMilestone int) ([]string, error) {
	manifestInternalPath := fmt.Sprintf("%s/%s", chromeosCheckout, manifestInternalProject)
	if !osutils.PathExists(manifestInternalPath) {
		return nil, fmt.Errorf("manifest-internal checkout not found at %s", manifestInternalPath)
	}

	// Fetch all to make sure we have all remote branches.
	err := git.RunGitIgnoreOutput(manifestInternalPath, []string{"fetch", "--all"})
	if err != nil {
		return nil, err
	}

	output, err := git.RunGit(manifestInternalPath, []string{"branch", "-r"})
	if err != nil {
		return nil, err
	}
	branchList := strings.Split(strings.TrimSpace(output.Stdout), "\n")
	for i, branch := range branchList {
		branchList[i] = strings.Split(branch, "/")[1]
	}

	// Find release branches.
	releaseBranches, err := releaseBranches(branchList, minMilestone)
	if err != nil {
		return nil, err
	}

	minBuildNum := -1
	branches := []string{}
	for _, branch := range releaseBranches {
		buildNum := ExtractBuildNum(branch)
		if minBuildNum == -1 || buildNum < minBuildNum {
			minBuildNum = buildNum
		}
		branches = append(branches, branch)
	}
	// Use build number to find other (factory/firmware/stabilize) branches.
	branches = append(branches, nonReleaseBranches(branchList, minBuildNum)...)

	return branches, nil
}
