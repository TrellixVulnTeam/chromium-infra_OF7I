// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package schedulingunit

import (
	"fmt"

	"infra/libs/skylab/inventory/swarming"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsUtil "infra/unifiedfleet/app/util"
)

var dutStateWeights = map[string]int{
	"ready":               1,
	"needs_repair":        2,
	"repair_failed":       3,
	"needs_manual_repair": 4,
	"needs_replacement":   4,
	"needs_deploy":        4,
}

var suStateMap = map[int]string{
	0: "unknown",
	1: "ready",
	2: "needs_repair",
	3: "repair_failed",
	4: "needs_manual_attention",
}

func schedulingUnitDutState(states []string) string {
	record := 0
	for _, s := range states {
		if dutStateWeights[s] > record {
			record = dutStateWeights[s]
		}
	}
	return suStateMap[record]
}

func joinSingleValueLabel(labels []string) []string {
	var res []string
	occurrences := make(map[string]int)
	for _, l := range labels {
		occurrences[l] += 1
		// Swarming doesn't allow repeat value of a dimension, so we give
		// them a suffix. E.g. A scheduling unit contains two eve board DUTs
		// will have label-board: [eve, eve2]
		suffix := ""
		if occurrences[l] > 1 {
			suffix = fmt.Sprintf("%d", occurrences[l])
		}
		res = append(res, l+suffix)
	}
	return res
}

func dutLabelValues(label string, dims []swarming.Dimensions) []string {
	var res []string
	for _, dim := range dims {
		if v, ok := dim[label]; ok {
			if len(v) > 0 {
				res = append(res, v[0])
			}
		}
	}
	return res
}

func SchedulingUnitDimensions(su *ufspb.SchedulingUnit, dutsDims []swarming.Dimensions) map[string][]string {
	suDims := map[string][]string{
		"dut_name":        {ufsUtil.RemovePrefix(su.GetName())},
		"dut_id":          {ufsUtil.RemovePrefix(su.GetName())},
		"label-pool":      su.GetPools(),
		"label-dut_count": {fmt.Sprintf("%d", len(dutsDims))},
		"label-multiduts": {"True"},
		"dut_state":       {schedulingUnitDutState(dutLabelValues("dut_state", dutsDims))},
	}
	singleValueLabels := map[string]string{
		"label-board": "label-board",
		"label-model": "label-model",
		"dut_name":    "label-managed_dut",
	}
	for dutLabelName, suLabelName := range singleValueLabels {
		joinedLabels := joinSingleValueLabel(dutLabelValues(dutLabelName, dutsDims))
		if len(joinedLabels) > 0 {
			suDims[suLabelName] = joinedLabels
		}
	}
	return suDims
}

func SchedulingUnitBotState(su *ufspb.SchedulingUnit) map[string][]string {
	return map[string][]string{
		"scheduling_unit_version_index": {su.GetUpdateTime().AsTime().Format(ufsUtil.TimestampBasedVersionKeyFormat)},
	}
}
