// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"regexp"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsUtil "infra/unifiedfleet/app/util"
)

var locations = []*regexp.Regexp{
	regexp.MustCompile(`ROW[\d]*-RACK[\d]*-HOST[\d]*`),
	regexp.MustCompile(`chromeos[\d]*-row[\d]*-rack[\d]*-host[\d]*`),
	regexp.MustCompile(`chromeos[\d]*-row[\d]*-rack[\d]*-labstation[\d]*`),
	regexp.MustCompile(`chromeos[\d]*-floor`),
	regexp.MustCompile(`chromeos[\d]*-rack[\d]*`),
	regexp.MustCompile(`[\w]*[\d]*-storage[\d]*`),
	regexp.MustCompile(`[\w]*[\d]*-container[\d]*`),
	regexp.MustCompile(`[\w]*[\d]*-desk-[\w]*`),
}

var zoneFilterRegex = regexp.MustCompile(`zone=[a-zA-Z0-9-,_]*`)

// IsLocation determines if a string describes a ChromeOS lab location
func IsLocation(iput string) bool {
	for _, exp := range locations {
		if exp.MatchString(iput) {
			return true
		}
	}
	return false
}

// ToUFSRealm returns the realm name based on zone string.
func ToUFSRealm(zone string) string {
	ufsZone := ufsUtil.ToUFSZone(zone)
	if ufsUtil.IsInBrowserZone(ufsZone.String()) {
		return ufsUtil.BrowserLabAdminRealm
	}
	if ufsZone == ufspb.Zone_ZONE_CHROMEOS3 || ufsZone == ufspb.Zone_ZONE_CHROMEOS5 ||
		ufsZone == ufspb.Zone_ZONE_CHROMEOS7 || ufsZone == ufspb.Zone_ZONE_CHROMEOS15 {
		return ufsUtil.AcsLabAdminRealm
	}
	return ufsUtil.AtlLabAdminRealm
}
