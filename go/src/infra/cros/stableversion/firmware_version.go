// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package stableversion

import (
	"fmt"
	"regexp"
)

// capture groups:
// company, platform, tip, branch, branchbranch
var firmwareVersionPattern *regexp.Regexp = regexp.MustCompile(`\A(?P<company>[A-Za-z0-9\-]+)_(?P<platform>[A-Za-z0-9_\-]+)\.(?P<tip>[0-9]+)\.(?P<branch>[0-9]+)\.(?P<branchbranch>[0-9]+)\z`)

// ParseFirmwareVersion takes a read-write firmware version and extracts
// semantically meaningful elements.
func ParseFirmwareVersion(s string) (string, string, int32, int32, int32, error) {
	if s == "" {
		return "", "", 0, 0, 0, fmt.Errorf("rw firmware version cannot be empty")
	}
	if firmwareVersionPattern.FindString(s) == "" {
		return "", "", 0, 0, 0, fmt.Errorf("rw firmware version is not valid")
	}
	m, err := findMatchMap(firmwareVersionPattern, s)
	if err != nil {
		return "", "", 0, 0, 0, err
	}
	company, err := extractString(m, "company")
	if err != nil {
		return "", "", 0, 0, 0, err
	}
	platform, err := extractString(m, "platform")
	if err != nil {
		return "", "", 0, 0, 0, err
	}
	tip, err := extractInt32(m, "tip")
	if err != nil {
		return "", "", 0, 0, 0, err
	}
	branch, err := extractInt32(m, "branch")
	if err != nil {
		return "", "", 0, 0, 0, err
	}
	branchBranch, err := extractInt32(m, "branchbranch")
	if err != nil {
		return "", "", 0, 0, 0, err
	}
	return company, platform, tip, branch, branchBranch, nil
}

// ValidateFirmwareVersion checks whether a string is a valid read-write
// firmware version. e.g. Google_Rammus.11275.41.0
func ValidateFirmwareVersion(r string) error {
	_, _, _, _, _, err := ParseFirmwareVersion(r)
	return err
}

// SerializeFirmwareVersion takes a list of components of a RWFirmwareVersion
func SerializeFirmwareVersion(company string, platform string, tip int32, branch int32, branchBranch int32) string {
	return fmt.Sprintf("%s_%s.%d.%d.%d", company, platform, tip, branch, branchBranch)
}
