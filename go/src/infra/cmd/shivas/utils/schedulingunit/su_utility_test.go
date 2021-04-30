// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package schedulingunit

import (
	"testing"

	"infra/libs/skylab/inventory/swarming"

	ufspb "infra/unifiedfleet/api/v1/models"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSchedulingUnitDutState(t *testing.T) {
	Convey("Test when all child DUTs in ready.", t, func() {
		s := []string{"ready", "ready", "ready", "ready", "ready"}
		So(schedulingUnitDutState(s), ShouldEqual, "ready")
	})

	Convey("Test when where one child DUT in needs_repair.", t, func() {
		s := []string{"ready", "ready", "ready", "ready", "needs_repair"}
		So(schedulingUnitDutState(s), ShouldEqual, "needs_repair")
	})

	Convey("Test when where one child DUT in repair_failed.", t, func() {
		s := []string{"ready", "ready", "ready", "needs_repair", "repair_failed"}
		So(schedulingUnitDutState(s), ShouldEqual, "repair_failed")
	})

	Convey("Test when where one child DUT in needs_manual_repair.", t, func() {
		s := []string{"ready", "ready", "needs_manual_repair", "needs_repair", "repair_failed"}
		So(schedulingUnitDutState(s), ShouldEqual, "needs_manual_attention")
	})

	Convey("Test when where one child DUT in needs_replacement.", t, func() {
		s := []string{"ready", "ready", "needs_replacement", "needs_repair", "repair_failed"}
		So(schedulingUnitDutState(s), ShouldEqual, "needs_manual_attention")
	})

	Convey("Test when where one child DUT in needs_deploy.", t, func() {
		s := []string{"ready", "ready", "needs_deploy", "needs_repair", "repair_failed"}
		So(schedulingUnitDutState(s), ShouldEqual, "needs_manual_attention")
	})

	Convey("Test when input is an empty slice", t, func() {
		var s []string
		So(schedulingUnitDutState(s), ShouldEqual, "unknown")
	})
}

func TestJoinSingleValueLabel(t *testing.T) {
	Convey("Test with no repeat labels", t, func() {
		l := []string{"eve", "nami", "coral"}
		So(joinSingleValueLabel(l), ShouldResemble, []string{"eve", "nami", "coral"})
	})

	Convey("Test with repeat labels", t, func() {
		l := []string{"nami", "coral", "nami", "nami"}
		So(joinSingleValueLabel(l), ShouldResemble, []string{"nami", "coral", "nami2", "nami3"})
	})
}

func TestDutLabelValues(t *testing.T) {
	Convey("Test get DUT's label values.", t, func() {
		dims := []swarming.Dimensions{
			{
				"label-board": {"coral"},
				"label-model": {"babytiger"},
				"dut_state":   {"ready"},
			},
			{
				"label-board": {"nami"},
				"label-model": {"bard"},
				"dut_state":   {"repair_failed"},
			},
			{
				"label-board": {"eve"},
				"label-model": {"eve"},
				"dut_state":   {"ready"},
			},
		}
		So(dutLabelValues("label-board", dims), ShouldResemble, []string{"coral", "nami", "eve"})
		So(dutLabelValues("label-model", dims), ShouldResemble, []string{"babytiger", "bard", "eve"})
		So(dutLabelValues("dut_state", dims), ShouldResemble, []string{"ready", "repair_failed", "ready"})
		So(dutLabelValues("IM_NOT_EXIST", dims), ShouldResemble, []string(nil))
	})
}

func TestSchedulingUnitDimensions(t *testing.T) {
	Convey("Test with a non-empty scheduling unit.", t, func() {
		su := &ufspb.SchedulingUnit{
			Name:  "schedulingunit/test-unit1",
			Pools: []string{"nearby_sharing"},
		}
		dims := []swarming.Dimensions{
			{
				"label-board":   {"coral"},
				"label-model":   {"babytiger"},
				"dut_state":     {"ready"},
				"random-label1": {"123"},
			},
			{
				"label-board":   {"nami"},
				"label-model":   {"bard"},
				"dut_state":     {"repair_failed"},
				"random-label2": {"abc"},
			},
			{
				"label-board":   {"eve"},
				"label-model":   {"eve"},
				"dut_state":     {"ready"},
				"random-label2": {"!@#"},
			},
		}
		expectedResult := map[string][]string{
			"dut_name":        {"test-unit1"},
			"dut_id":          {"test-unit1"},
			"label-pool":      {"nearby_sharing"},
			"label-dut_count": {"3"},
			"label-multiduts": {"True"},
			"dut_state":       {"repair_failed"},
			"label-board":     {"coral", "nami", "eve"},
			"label-model":     {"babytiger", "bard", "eve"},
		}
		So(SchedulingUnitDimensions(su, dims), ShouldResemble, expectedResult)
	})

	Convey("Test with an empty scheduling unit.", t, func() {
		su := &ufspb.SchedulingUnit{
			Name:  "schedulingunit/test-unit1",
			Pools: []string{"nearby_sharing"},
		}
		var dims []swarming.Dimensions
		expectedResult := map[string][]string{
			"dut_name":        {"test-unit1"},
			"dut_id":          {"test-unit1"},
			"label-pool":      {"nearby_sharing"},
			"label-dut_count": {"0"},
			"label-multiduts": {"True"},
			"dut_state":       {"unknown"},
		}
		So(SchedulingUnitDimensions(su, dims), ShouldResemble, expectedResult)
	})
}
