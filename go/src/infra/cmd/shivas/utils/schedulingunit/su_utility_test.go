// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package schedulingunit

import (
	"testing"
	"time"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

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

	Convey("Test when where one child DUT in reserved.", t, func() {
		s := []string{"ready", "reserved", "needs_deploy", "needs_repair", "repair_failed"}
		So(schedulingUnitDutState(s), ShouldEqual, "reserved")
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
		So(joinSingleValueLabel(l), ShouldResemble, []string{"nami", "coral", "nami_2", "nami_3"})
	})
}

func TestDutLabelValues(t *testing.T) {
	Convey("Test get DUT's label values.", t, func() {
		dims := []swarming.Dimensions{
			{
				"dut_name":    {"host1"},
				"label-board": {"coral"},
				"label-model": {"babytiger"},
				"dut_state":   {"ready"},
			},
			{
				"dut_name":    {"host2"},
				"label-board": {"nami"},
				"label-model": {"bard"},
				"dut_state":   {"repair_failed"},
			},
			{
				"dut_name":    {"host3"},
				"label-board": {"eve"},
				"label-model": {"eve"},
				"dut_state":   {"ready"},
			},
		}
		So(dutLabelValues("dut_name", dims), ShouldResemble, []string{"host1", "host2", "host3"})
		So(dutLabelValues("label-board", dims), ShouldResemble, []string{"coral", "nami", "eve"})
		So(dutLabelValues("label-model", dims), ShouldResemble, []string{"babytiger", "bard", "eve"})
		So(dutLabelValues("dut_state", dims), ShouldResemble, []string{"ready", "repair_failed", "ready"})
		So(dutLabelValues("IM_NOT_EXIST", dims), ShouldResemble, []string(nil))
	})
}

func TestLabelIntersection(t *testing.T) {
	Convey("Test find intersection from a given label name.", t, func() {
		dims := []swarming.Dimensions{
			{
				"label-device-stable": {"True"},
				"label-foo":           {"common_value1", "common_value2", "common_value3", "special_value1"},
				"label-foo2":          {"value"},
			},
			{
				"label-device-stable": {"True"},
				"label-foo":           {"common_value1", "common_value2", "common_value3", "special_value2"},
				"label-foo2":          {"value"},
			},
			{
				"label-device-stable": {"True"},
				"label-foo":           {"common_value1", "common_value2", "common_value3", "special_value3"},
			},
		}
		So(labelIntersection("label-device-stable", dims), ShouldResemble, []string{"True"})
		So(labelIntersection("label-foo", dims), ShouldResemble, []string{"common_value1", "common_value2", "common_value3"})
		So(labelIntersection("label-foo2", dims), ShouldResemble, []string(nil))
	})
}

func TestSchedulingUnitDimensions(t *testing.T) {
	Convey("Test with a non-empty scheduling unit with all devices are stable.", t, func() {
		su := &ufspb.SchedulingUnit{
			Name:  "schedulingunit/test-unit1",
			Pools: []string{"nearby_sharing"},
		}
		dims := []swarming.Dimensions{
			{
				"dut_name":            {"host1"},
				"label-board":         {"coral"},
				"label-model":         {"babytiger"},
				"dut_state":           {"ready"},
				"random-label1":       {"123"},
				"label-device-stable": {"True"},
			},
			{
				"dut_name":            {"host2"},
				"label-board":         {"nami"},
				"label-model":         {"bard"},
				"dut_state":           {"repair_failed"},
				"random-label2":       {"abc"},
				"label-device-stable": {"True"},
			},
			{
				"dut_name":            {"host3"},
				"label-board":         {"eve"},
				"label-model":         {"eve"},
				"dut_state":           {"ready"},
				"random-label2":       {"!@#"},
				"label-device-stable": {"True"},
			},
		}
		expectedResult := map[string][]string{
			"dut_name":            {"test-unit1"},
			"dut_id":              {"test-unit1"},
			"label-pool":          {"nearby_sharing"},
			"label-dut_count":     {"3"},
			"label-multiduts":     {"True"},
			"label-managed_dut":   {"host1", "host2", "host3"},
			"dut_state":           {"repair_failed"},
			"label-board":         {"coral", "nami", "eve"},
			"label-model":         {"babytiger", "bard", "eve"},
			"label-device-stable": {"True"},
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

	Convey("Test with an scheduling unit that include non-stable device.", t, func() {
		su := &ufspb.SchedulingUnit{
			Name:  "schedulingunit/test-unit1",
			Pools: []string{"nearby_sharing"},
		}
		dims := []swarming.Dimensions{
			{
				"dut_name":            {"host1"},
				"label-board":         {"coral"},
				"label-model":         {"babytiger"},
				"dut_state":           {"ready"},
				"random-label1":       {"123"},
				"label-device-stable": {"True"},
			},
			{
				"dut_name":            {"host2"},
				"label-board":         {"nami"},
				"label-model":         {"bard"},
				"dut_state":           {"repair_failed"},
				"random-label2":       {"abc"},
				"label-device-stable": {"True"},
			},
			{
				"dut_name":      {"host3"},
				"label-board":   {"eve"},
				"label-model":   {"eve"},
				"dut_state":     {"ready"},
				"random-label2": {"!@#"},
			},
		}
		expectedResult := map[string][]string{
			"dut_name":          {"test-unit1"},
			"dut_id":            {"test-unit1"},
			"label-pool":        {"nearby_sharing"},
			"label-dut_count":   {"3"},
			"label-multiduts":   {"True"},
			"label-managed_dut": {"host1", "host2", "host3"},
			"dut_state":         {"repair_failed"},
			"label-board":       {"coral", "nami", "eve"},
			"label-model":       {"babytiger", "bard", "eve"},
		}
		So(SchedulingUnitDimensions(su, dims), ShouldResemble, expectedResult)
	})
}

func TestSchedulingUnitBotState(t *testing.T) {
	Convey("Test scheduling unit bot state.", t, func() {
		t, _ := time.Parse(time.RFC3339, "2021-05-07T11:54:36.225Z")
		su := &ufspb.SchedulingUnit{
			Name:       "schedulingunit/test-unit1",
			UpdateTime: timestamppb.New(t),
		}
		expectedResult := map[string][]string{
			"scheduling_unit_version_index": {"2021-05-07 11:54:36.225 UTC"},
		}
		So(SchedulingUnitBotState(su), ShouldResemble, expectedResult)
	})
}
