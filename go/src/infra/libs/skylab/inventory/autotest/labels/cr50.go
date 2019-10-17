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
	if v := ls.GetCr50RoVersion(); v != "" {
		lv := fmt.Sprintf("cr50-ro-version:%s", v)
		labels = append(labels, lv)
	}
	if v := ls.GetCr50RwKeyid(); v != "" {
		lv := fmt.Sprintf("cr50-rw-keyid:%s", v)
		labels = append(labels, lv)
	}
	if v := ls.GetCr50RwVersion(); v != "" {
		lv := fmt.Sprintf("cr50-rw-version:%s", v)
		labels = append(labels, lv)
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
			*ls.Cr50Phase = t(vals[vn])
		case "cr50-ro-keyid":
			ls.Cr50RoKeyid = &v
		case "cr50-ro-version":
			ls.Cr50RoVersion = &v
		case "cr50-rw-keyid":
			ls.Cr50RwKeyid = &v
		case "cr50-rw-version":
			ls.Cr50RwVersion = &v
		default:
			continue
		}
		labels = removeLabel(labels, i)
		i--
	}
	return labels
}
