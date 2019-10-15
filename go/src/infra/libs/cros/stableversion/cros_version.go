// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"fmt"
	"regexp"
)

// capture groups:
// release, tip, branch, branchbranch
var crosVersionPattern *regexp.Regexp = regexp.MustCompile(`\AR(?P<release>[0-9]+)-(?P<tip>[0-9]+)\.(?P<branch>[0-9]+)\.(?P<branchbranch>[0-9]+)\z`)

// ParseCrOSVersion takes a version string and extracts version info
func ParseCrOSVersion(v string) (int32, int32, int32, int32, error) {
	if v == "" {
		return 0, 0, 0, 0, fmt.Errorf("empty version string is invalid")
	}
	if crosVersionPattern.FindString(v) == "" {
		return 0, 0, 0, 0, fmt.Errorf("version string is not valid")
	}
	m, err := findMatchMap(crosVersionPattern, v)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	release, err := extractInt32(m, "release")
	if err != nil {
		return 0, 0, 0, 0, err
	}
	tip, err := extractInt32(m, "tip")
	if err != nil {
		return 0, 0, 0, 0, err
	}
	branch, err := extractInt32(m, "branch")
	if err != nil {
		return 0, 0, 0, 0, err
	}
	branchBranch, err := extractInt32(m, "branchbranch")
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return release, tip, branch, branchBranch, nil
}

// ValidateCrOSVersion checks that a CrOSVersion describes
// a sensible version such as "R76-12239.46.5"
func ValidateCrOSVersion(v string) error {
	_, _, _, _, err := ParseCrOSVersion(v)
	return err
}

// SerializeCrOSVersion takes a CrOSVersion specification
// and produces a string in the canonical format.
func SerializeCrOSVersion(release, tip, branch, branchBranch int32) string {
	return fmt.Sprintf("R%d-%d.%d.%d", release, tip, branch, branchBranch)
}
