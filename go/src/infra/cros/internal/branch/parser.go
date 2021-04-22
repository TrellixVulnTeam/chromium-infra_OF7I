// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package branch

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

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
)

// extractBuildNum extracts the build number from the branch name, e.g. 13729
// from release-R89-13729.B.
// Returns -1 if the branch name does not contain a build number.
func extractBuildNum(branch string) int {
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
			if extractBuildNum(branch) >= minBuildNumber {
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
		buildNum := extractBuildNum(branch)
		if minBuildNum == -1 || buildNum < minBuildNum {
			minBuildNum = buildNum
		}
		branches = append(branches, branch)
	}
	// Use build number to find other (factory/firmware/stabilize) branches.
	branches = append(branches, nonReleaseBranches(branchList, minBuildNum)...)

	return branches, nil
}
