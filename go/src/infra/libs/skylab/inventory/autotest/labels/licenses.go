// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package labels

import (
	"strings"

	"infra/libs/skylab/inventory"
)

func init() {
	converters = append(converters, licensesConverter)

	reverters = append(reverters, licensesReverter)
}

func licensesConverter(ls *inventory.SchedulableLabels) []string {
	var labels []string
	for _, v := range ls.GetLicenses() {
		const plen = 13 // len("LICENSE_TYPE_")
		lv := "license_" + strings.ToLower(v.Type.String()[plen:])
		labels = append(labels, lv)
	}
	return labels
}

func licensesReverter(ls *inventory.SchedulableLabels, labels []string) []string {
	ls.Licenses = nil
	for i := 0; i < len(labels); i++ {
		k, _ := splitLabel(labels[i])
		if strings.HasPrefix(k, "license_") {
			const plen = 8 // len("license_")
			ln := "LICENSE_TYPE_" + strings.ToUpper(k[plen:])
			vals := inventory.LicenseType_value
			l := inventory.NewLicense()
			*l.Type = inventory.LicenseType(vals[ln])
			// Identifier is not tracked in autotest labels, so is lost upon
			// reversion.
			*l.Identifier = ""
			ls.Licenses = append(ls.Licenses, l)

			labels = removeLabel(labels, i)
			i--
		}
	}
	return labels
}
