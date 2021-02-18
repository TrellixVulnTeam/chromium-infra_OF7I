// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"regexp"
	"strings"

	"go.chromium.org/luci/common/errors"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/util"
	ufsUtil "infra/unifiedfleet/app/util"
)

var numRegex = regexp.MustCompile(`[0-9]+`)

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

var labs = []*regexp.Regexp{
	regexp.MustCompile(`chromeos[\d]*`),
	regexp.MustCompile(`CHROMEOS[\d]*`),
}

var rows = []*regexp.Regexp{
	regexp.MustCompile(`ROW[\d]*`),
	regexp.MustCompile(`row[\d]*`),
}

var racks = []*regexp.Regexp{
	regexp.MustCompile(`CHROMEOS[\d]*-ROW[\d]*-RACK[\d]*`),
	regexp.MustCompile(`chromeos[\d]*-row[\d]*-rack[\d]*`),
}

var rackNumbers = []*regexp.Regexp{
	regexp.MustCompile(`RACK[\d]*`),
	regexp.MustCompile(`rack[\d]*`),
}

var hosts = []*regexp.Regexp{
	regexp.MustCompile(`HOST[\d]*`),
	regexp.MustCompile(`host[\d]*`),
	regexp.MustCompile(`labstation[\d]*`),
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

// GetLocation returns Location proto from barcode name
func GetLocation(input string) (*ufspb.Location, error) {
	if input == "" {
		return nil, errors.Reason("Invalid input").Err()
	}
	loc := &ufspb.Location{}
	// Extract lab if it exists
	for _, exp := range labs {
		labStr := exp.FindString(input)
		if labStr != "" && util.IsUFSZone(labStr) {
			labStr = strings.ToLower(labStr)
			loc.Zone = ufsUtil.ToUFSZone(labStr)
			break
		}
	}
	// Extract row if it exists
	for _, exp := range rows {
		rowStr := exp.FindString(input)
		if rowStr != "" {
			loc.Row = numRegex.FindString(rowStr)
			break
		}
	}
	// Extract rack if it exists
	for _, exp := range racks {
		rackStr := exp.FindString(input)
		if rackStr != "" {
			loc.Rack = rackStr
			break
		}
	}
	// Extract rack number if it exists
	for _, exp := range rackNumbers {
		rackNumberStr := exp.FindString(input)
		if rackNumberStr != "" {
			loc.RackNumber = numRegex.FindString(rackNumberStr)
			break
		}
	}
	// Extract position if it exists
	for _, exp := range hosts {
		positionStr := exp.FindString(input)
		if positionStr != "" {
			loc.Position = numRegex.FindString(positionStr)
			break
		}
	}
	loc.BarcodeName = input
	return loc, nil
}
