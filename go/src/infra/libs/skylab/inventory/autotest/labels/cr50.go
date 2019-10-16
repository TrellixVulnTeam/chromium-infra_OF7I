// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package labels

import (
	"strings"

	"fmt"
	"infra/libs/skylab/inventory"
)

func init() {
	converters = append(converters, cr50Converter)
	reverters = append(reverters, cr50Reverter)
}

func cr50Converter(ls *inventory.SchedulableLabels) []string {
	var labels []string
	if v := ls.GetCr50Phase(); v != inventory.SchedulableLabels_CR50_PHASE_INVALID {
		const plen = 11 // len("CR50_PHASE_")
		lv := "cr50:" + strings.ToLower(v.String()[plen:])
		labels = append(labels, lv)
	}
	if v := ls.GetCr50RoKeyid(); v != "" {
		lv := fmt.Sprintf("cr50-ro-keyid:%s", v)
		labels = append(labels, lv)
	}
	{
		major := ls.GetCr50RoVersionMajor()
		minor := ls.GetCr50RoVersionMinor()
		rev := ls.GetCr50RoVersionRev()
		if major != 0 || minor != 0 || rev != 0 {
			lv := fmt.Sprintf("cr50-ro-version:%d.%d.%d", major, minor, rev)
			labels = append(labels, lv)
		}
	}
	if v := ls.GetCr50RwKeyid(); v != "" {
		lv := fmt.Sprintf("cr50-rw-keyid:%s", v)
		labels = append(labels, lv)
	}
	{
		major := ls.GetCr50RwVersionMajor()
		minor := ls.GetCr50RwVersionMinor()
		rev := ls.GetCr50RwVersionRev()
		if major != 0 || minor != 0 || rev != 0 {
			lv := fmt.Sprintf("cr50-rw-version:%d.%d.%d", major, minor, rev)
			labels = append(labels, lv)
		}
	}
	{
		major := ls.GetCr50VersionMajor()
		minor := ls.GetCr50VersionMinor()
		rev := ls.GetCr50VersionRev()
		if major != 0 || minor != 0 || rev != 0 {
			lv := fmt.Sprintf("cr50:%d.%d.%d", major, minor, rev)
			labels = append(labels, lv)
		}
	}
	return labels
}

func cr50Reverter(ls *inventory.SchedulableLabels, labels []string) []string {
	for i := 0; i < len(labels); i++ {
		k, v := splitLabel(labels[i])
		switch k {
		case "cr50":
			vn := "CR50_PHASE_" + strings.ToUpper(v)
			type t = inventory.SchedulableLabels_CR50_Phase
			vals := inventory.SchedulableLabels_CR50_Phase_value
			if _, ok := vals[vn]; ok {
				*ls.Cr50Phase = t(vals[vn])
			} else {
				var major, minor, rev int32
				if n, err := fmt.Sscanf(v, "%d.%d.%d", &major, &minor, &rev); n != 3 || err != nil {
					continue
				}
				ls.Cr50VersionMajor = &major
				ls.Cr50VersionMinor = &minor
				ls.Cr50VersionRev = &rev
			}
		case "cr50-ro-keyid":
			ls.Cr50RoKeyid = &v
		case "cr50-ro-version":
			var major, minor, rev int32
			if n, err := fmt.Sscanf(v, "%d.%d.%d", &major, &minor, &rev); n != 3 || err != nil {
				continue
			}
			ls.Cr50RoVersionMajor = &major
			ls.Cr50RoVersionMinor = &minor
			ls.Cr50RoVersionRev = &rev
		case "cr50-rw-keyid":
			ls.Cr50RwKeyid = &v
		case "cr50-rw-version":
			var major, minor, rev int32
			if n, err := fmt.Sscanf(v, "%d.%d.%d", &major, &minor, &rev); n != 3 || err != nil {
				continue
			}
			ls.Cr50RwVersionMajor = &major
			ls.Cr50RwVersionMinor = &minor
			ls.Cr50RwVersionRev = &rev
		default:
			continue
		}
		labels = removeLabel(labels, i)
		i--
	}
	return labels
}
