// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package schedulingunit

import (
	"context"
	"fmt"
	"sort"

	"infra/libs/skylab/inventory/swarming"
	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsUtil "infra/unifiedfleet/app/util"
)

var dutStateWeights = map[string]int{
	"ready":               1,
	"needs_repair":        2,
	"repair_failed":       3,
	"needs_manual_repair": 4,
	"needs_replacement":   4,
	"needs_deploy":        4,
	"reserved":            5,
}

var suStateMap = map[int]string{
	0: "unknown",
	1: "ready",
	2: "needs_repair",
	3: "repair_failed",
	4: "needs_manual_attention",
	5: "reserved",
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
			suffix = fmt.Sprintf("_%d", occurrences[l])
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

// labelIntersection takes a label name and a slice of device dimensions, and
// return only values that are common in all devices.
func labelIntersection(label string, dims []swarming.Dimensions) []string {
	valueCount := make(map[string]int)
	for _, dim := range dims {
		if values, ok := dim[label]; ok {
			for _, v := range values {
				valueCount[v] += 1
			}
		}
	}
	// Iterate over the keys of valueCount in lexicographic order.
	var keys []string
	var labels []string
	for label := range valueCount {
		keys = append(keys, label)
	}
	sort.Strings(keys)
	for _, label := range keys {
		count := valueCount[label]
		if count == len(dims) {
			labels = append(labels, label)
		}
	}
	return labels
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
	// conjunctionLabels define labels we want present it in SU only if all
	// their devices has the given label, and SU will only inherit values that
	// are common among all devices under the SU.
	conjunctionLabels := []string{"label-device-stable"}
	for _, label := range conjunctionLabels {
		values := labelIntersection(label, dutsDims)
		if len(values) > 0 {
			suDims[label] = values
		}
	}
	return suDims
}

func SchedulingUnitBotState(su *ufspb.SchedulingUnit) map[string][]string {
	return map[string][]string{
		"scheduling_unit_version_index": {su.GetUpdateTime().AsTime().Format(ufsUtil.TimestampBasedVersionKeyFormat)},
	}
}

// CheckIfLSEBelongsToSU checks if the DUT/Labstation belongs to a SchedulingUnit.
//
// User is not allowed to udpate a DUT/Labstation which belongs to a SU.
// The DUT/Labstation needs to be removed from the SU and then updated.
func CheckIfLSEBelongsToSU(ctx context.Context, ic ufsAPI.FleetClient, lseName string) error {
	req := &ufsAPI.ListSchedulingUnitsRequest{
		Filter:   fmt.Sprintf("duts=%s", lseName),
		KeysOnly: true,
	}
	res, err := ic.ListSchedulingUnits(ctx, req)
	if err != nil {
		return err
	}
	if len(res.GetSchedulingUnits()) > 0 {
		return fmt.Errorf("DUT/Labstation is associated with SchedulingUnit. Run `shivas update schedulingunit -name %s -removeduts %s` to remove association before updating the DUT/Labstation.", ufsUtil.RemovePrefix(res.GetSchedulingUnits()[0].GetName()), lseName)
	}
	return nil
}
