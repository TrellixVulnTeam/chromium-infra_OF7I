// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"infra/libs/skylab/inventory"
)

func init() {
	converters = append(converters, licensesConverter)
	reverters = append(reverters, licensesReverter)

}

func licensesConverter(dims Dimensions, ls *inventory.SchedulableLabels) {
	for _, v := range ls.GetLicenses() {
		appendDim(dims, "label-license", v.Type.String())
	}
}

func licensesReverter(ls *inventory.SchedulableLabels, d Dimensions) Dimensions {
	ls.Licenses = make([]*inventory.License, len(d["label-license"]))
	for i, v := range d["label-license"] {
		l := inventory.NewLicense()
		if p, ok := inventory.LicenseType_value[v]; ok {
			*l.Type = inventory.LicenseType(p)
			// Identifier is not tracked in swarming tags, so is lost upon
			// reversion.
			*l.Identifier = ""
		}
		ls.Licenses[i] = l
	}
	delete(d, "label-license")
	return d
}
