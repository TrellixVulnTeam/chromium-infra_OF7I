// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"fmt"
	"regexp"

	"go.chromium.org/chromiumos/infra/proto/go/lab"
	fleet "infra/libs/fleet/protos/go"
)

// GetHostname returns the hostname of input ChromeOSDevice.
func GetHostname(d *lab.ChromeOSDevice) string {
	switch t := d.GetDevice().(type) {
	case *lab.ChromeOSDevice_Dut:
		return d.GetDut().GetHostname()
	case *lab.ChromeOSDevice_Labstation:
		return d.GetLabstation().GetHostname()
	default:
		panic(fmt.Sprintf("Unknown device type: %v", t))
	}
}

// GetLocation attempts to parse the input string and return a Location object.
// Default location is updated with values from the string. This is done
// because the barcodes do not specify the complete location of the asset
func GetLocation(input string) (loc *fleet.Location) {
	//loc = c.defaultLocation()
	loc = &fleet.Location{}
	// Extract lab if it exists
	for _, exp := range labs {
		labStr := exp.FindString(input)
		if labStr != "" {
			loc.Lab = labStr
		}
	}
	// Extract row if it exists
	for _, exp := range rows {
		rowStr := exp.FindString(input)
		if rowStr != "" {
			loc.Row = num.FindString(rowStr)
			break
		}
	}
	// Extract rack if it exists
	for _, exp := range racks {
		rackStr := exp.FindString(input)
		if rackStr != "" {
			loc.Rack = num.FindString(rackStr)
			break
		}
	}
	// Extract position if it exists
	for _, exp := range hosts {
		positionStr := exp.FindString(input)
		if positionStr != "" {
			loc.Position = num.FindString(positionStr)
			break
		}
	}
	return loc
}

/* Regular expressions to match various parts of the input string - START */

var num = regexp.MustCompile(`[0-9]+`)

var labs = []*regexp.Regexp{
	regexp.MustCompile(`chromeos[\d]*`),
}

var rows = []*regexp.Regexp{
	regexp.MustCompile(`ROW[\d]*`),
	regexp.MustCompile(`row[\d]*`),
}

var racks = []*regexp.Regexp{
	regexp.MustCompile(`RACK[\d]*`),
	regexp.MustCompile(`rack[\d]*`),
}

var hosts = []*regexp.Regexp{
	regexp.MustCompile(`HOST[\d]*`),
	regexp.MustCompile(`host[\d]*`),
}

/* Regular expressions to match various parts of the input string - END */
