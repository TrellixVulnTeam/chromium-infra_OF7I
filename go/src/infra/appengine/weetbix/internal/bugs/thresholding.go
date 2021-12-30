// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugs

import (
	"math"

	configpb "infra/appengine/weetbix/internal/config/proto"
)

// MeetsThreshold returns whether the nominal impact of the cluster meets
// or exceeds the specified threshold.
func (c *ClusterImpact) MeetsThreshold(t *configpb.ImpactThreshold) bool {
	return c.MeetsInflatedThreshold(t, 0)
}

// MeetsInflatedThreshold returns whether the impact of the cluster meets or
// exceeds the specified threshold, inflated or deflated by the given factor.
// This method is provided to help implement hysteresis. inflationPercent can
// be positive or negative (or zero), and is interpreted as follows:
// - If inflationPercent is positive, the threshold used is (threshold * (1 + (inflationPercent/100)))
// - If inflationPercent is negative, the threshold used is (threshold / (1 + (-inflationPercent/100))
// i.e. inflationPercent of +100 would result in a threshold that is 200% the
// original threshold being used, inflationPercent of -100 would result in a
// threshold that is 50% of the original.
func (c *ClusterImpact) MeetsInflatedThreshold(t *configpb.ImpactThreshold, inflationPercent int64) bool {
	if c.TestResultsFailed.meetsInflatedThreshold(t.TestResultsFailed, inflationPercent) {
		return true
	}
	if c.TestRunsFailed.meetsInflatedThreshold(t.TestRunsFailed, inflationPercent) {
		return true
	}
	if c.PresubmitRunsFailed.meetsInflatedThreshold(t.PresubmitRunsFailed, inflationPercent) {
		return true
	}
	return false
}

func (m MetricImpact) meetsInflatedThreshold(t *configpb.MetricThreshold, inflationPercent int64) bool {
	if t == nil {
		t = &configpb.MetricThreshold{}
	}
	if meetsInflatedThreshold(m.OneDay, t.OneDay, inflationPercent) {
		return true
	}
	if meetsInflatedThreshold(m.ThreeDay, t.ThreeDay, inflationPercent) {
		return true
	}
	if meetsInflatedThreshold(m.SevenDay, t.SevenDay, inflationPercent) {
		return true
	}
	return false
}

// meetsInflatedThreshold tests whether value exceeds the threshold with given
// hysteresis applied. If threshold is nil, the threshold is considered "not set"
// and the method always returns false.
func meetsInflatedThreshold(value int64, threshold *int64, inflationPercent int64) bool {
	if threshold == nil {
		return false
	}
	thresholdValue := *threshold

	if inflationPercent >= 0 {
		// If close to overflow range, scale values and do approximate thresholding.
		if thresholdValue >= math.MaxInt64/(inflationPercent+100) {
			thresholdValue /= inflationPercent + 100
			value /= inflationPercent + 100
		}

		// I.E. +100% doubles the threshold.
		thresholdValue = (thresholdValue * (100 + inflationPercent)) / 100
	} else {
		// If close to overflow range, scale values and do approximate thresholding.
		if thresholdValue >= math.MaxInt64/100 {
			thresholdValue /= 100
			value /= 100
		}

		// I.E. -100% halves the threshold.
		thresholdValue = (thresholdValue * 100) / (100 + -inflationPercent)
	}

	return value >= thresholdValue
}
