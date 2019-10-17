// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package swarming

import (
	"infra/libs/skylab/inventory"
)

func init() {
	converters = append(converters, cr50Converter)
	reverters = append(reverters, cr50Reverter)
}

func cr50Converter(dims Dimensions, ls *inventory.SchedulableLabels) {
	if v := ls.GetCr50Phase(); v != inventory.SchedulableLabels_CR50_PHASE_INVALID {
		dims["label-cr50_phase"] = []string{v.String()}
	}
	if v := ls.GetCr50RoKeyid(); v != "" {
		dims["label-cr50_ro_keyid"] = []string{v}
	}
	if v := ls.GetCr50RoVersion(); v != "" {
		dims["label-cr50_ro_version"] = []string{v}
	}
	if v := ls.GetCr50RwKeyid(); v != "" {
		dims["label-cr50_rw_keyid"] = []string{v}
	}
	if v := ls.GetCr50RwVersion(); v != "" {
		dims["label-cr50_rw_version"] = []string{v}
	}
}

func cr50Reverter(ls *inventory.SchedulableLabels, d Dimensions) Dimensions {
	if v, ok := getLastStringValue(d, "label-cr50_phase"); ok {
		if cr50, ok := inventory.SchedulableLabels_CR50_Phase_value[v]; ok {
			*ls.Cr50Phase = inventory.SchedulableLabels_CR50_Phase(cr50)
		}
		delete(d, "label-cr50_phase")
	}
	if v, ok := getLastStringValue(d, "label-cr50_ro_keyid"); ok {
		*ls.Cr50RoKeyid = v
		delete(d, "label-cr50_ro_keyid")
	}
	if v, ok := getLastStringValue(d, "label-cr50_ro_version"); ok {
		*ls.Cr50RoVersion = v
		delete(d, "label-cr50_ro_version")
	}
	if v, ok := getLastStringValue(d, "label-cr50_rw_keyid"); ok {
		*ls.Cr50RwKeyid = v
		delete(d, "label-cr50_rw_keyid")
	}
	if v, ok := getLastStringValue(d, "label-cr50_rw_version"); ok {
		*ls.Cr50RwVersion = v
		delete(d, "label-cr50_rw_version")
	}
	return d
}
