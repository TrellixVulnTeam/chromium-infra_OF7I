// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"fmt"
	"regexp"
	"strings"

	"go.chromium.org/luci/common/errors"

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

// StrToUFSZone refers a map between a string to a UFS defined map.
var StrToUFSZone = map[string]string{
	"atl":        "ZONE_ATLANTA",
	"chromeos1":  "ZONE_CHROMEOS1",
	"chromeos4":  "ZONE_CHROMEOS4",
	"chromeos6":  "ZONE_CHROMEOS6",
	"chromeos2":  "ZONE_CHROMEOS2",
	"chromeos3":  "ZONE_CHROMEOS3",
	"chromeos5":  "ZONE_CHROMEOS5",
	"chromeos7":  "ZONE_CHROMEOS7",
	"chromeos15": "ZONE_CHROMEOS15",
	"atl97":      "ZONE_ATL97",
	"iad97":      "ZONE_IAD97",
	"mtv96":      "ZONE_MTV96",
	"mtv97":      "ZONE_MTV97",
	"lab01":      "ZONE_FUCHSIA",
	"unknown":    "ZONE_UNSPECIFIED",
}

// IsUFSZone checks if a string refers to a valid UFS zone.
func IsUFSZone(zone string) bool {
	_, ok := StrToUFSZone[zone]
	return ok
}

// ValidZoneStr returns a valid str list for zone strings.
func ValidZoneStr() []string {
	ks := make([]string, 0, len(StrToUFSZone))
	for k := range StrToUFSZone {
		ks = append(ks, k)
	}
	return ks
}

// ToUFSZone converts zone string to a UFS zone enum.
func ToUFSZone(zone string) ufspb.Zone {
	v, ok := StrToUFSZone[zone]
	if !ok {
		return ufspb.Zone_ZONE_UNSPECIFIED
	}
	return ufspb.Zone(ufspb.Zone_value[v])
}

// ToUFSRealm returns the realm name based on zone string.
func ToUFSRealm(zone string) string {
	ufsZone := ToUFSZone(zone)
	if ufsUtil.IsInBrowserZone(ufsZone.String()) {
		return ufsUtil.BrowserLabAdminRealm
	}
	if ufsZone == ufspb.Zone_ZONE_CHROMEOS3 || ufsZone == ufspb.Zone_ZONE_CHROMEOS5 ||
		ufsZone == ufspb.Zone_ZONE_CHROMEOS7 || ufsZone == ufspb.Zone_ZONE_CHROMEOS15 {
		return ufsUtil.AcsLabAdminRealm
	}
	return ufsUtil.AtlLabAdminRealm
}

// ReplaceZoneNames replace zone names with Ufs zone names in the filter string
//
// TODO(eshwarn) : Repalce regex and string matching with yacc (https://godoc.org/golang.org/x/tools/cmd/goyacc)
func ReplaceZoneNames(filter string) (string, error) {
	if filter != "" {
		// Remove all the spaces
		filter = fmt.Sprintf(strings.Replace(filter, " ", "", -1))
		// Find the zone filtering condition
		zonefilters := zoneFilterRegex.FindAllString(filter, -1)
		if len(zonefilters) == 0 {
			return filter, nil
		}
		// Aggregate all the zone names
		var zoneNames []string
		for _, lf := range zonefilters {
			keyValue := strings.Split(lf, "=")
			if len(keyValue) < 2 {
				return filter, nil
			}
			zoneNames = append(zoneNames, strings.Split(keyValue[1], ",")...)
		}
		if len(zoneNames) == 0 {
			return filter, nil
		}
		// Repalce all the zone names with UFS zone names matching the enum
		for _, zoneName := range zoneNames {
			ufsZoneName, ok := StrToUFSZone[zoneName]
			if !ok {
				errorMsg := fmt.Sprintf("Invalid zone name %s for filtering.\nValid zone filters: [%s]", zoneName, strings.Join(ValidZoneStr(), ", "))
				return filter, errors.New(errorMsg)
			}
			filter = fmt.Sprintf(strings.Replace(filter, zoneName, ufsZoneName, -1))
		}
	}
	return filter, nil
}
