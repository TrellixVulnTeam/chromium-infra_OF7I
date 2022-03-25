// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package updater

import (
	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/bugs"
)

// ExtractResidualPreWeetbixImpact extracts the residual,
// pre-weetbix exoneration, impact from a cluster summary.
func ExtractResidualPreWeetbixImpact(cs *analysis.ClusterSummary) *bugs.ClusterImpact {
	return &bugs.ClusterImpact{
		TestResultsFailed: bugs.MetricImpact{
			OneDay:   cs.Failures1d.ResidualPreWeetbix,
			ThreeDay: cs.Failures3d.ResidualPreWeetbix,
			SevenDay: cs.Failures7d.ResidualPreWeetbix,
		},
		TestRunsFailed: bugs.MetricImpact{
			OneDay:   cs.TestRunFails1d.ResidualPreWeetbix,
			ThreeDay: cs.TestRunFails3d.ResidualPreWeetbix,
			SevenDay: cs.TestRunFails7d.ResidualPreWeetbix,
		},
		PresubmitRunsFailed: bugs.MetricImpact{
			OneDay:   cs.PresubmitRejects1d.ResidualPreWeetbix,
			ThreeDay: cs.PresubmitRejects3d.ResidualPreWeetbix,
			SevenDay: cs.PresubmitRejects7d.ResidualPreWeetbix,
		},
	}
}

// SetResidualPreWeetbixImpact sets the residual, pre-weetbix exoneration
// impact on a cluster summary.
func SetResidualPreWeetbixImpact(cs *analysis.ClusterSummary, impact *bugs.ClusterImpact) {
	cs.Failures1d.ResidualPreWeetbix = impact.TestResultsFailed.OneDay
	cs.Failures3d.ResidualPreWeetbix = impact.TestResultsFailed.ThreeDay
	cs.Failures7d.ResidualPreWeetbix = impact.TestResultsFailed.SevenDay

	cs.TestRunFails1d.ResidualPreWeetbix = impact.TestRunsFailed.OneDay
	cs.TestRunFails3d.ResidualPreWeetbix = impact.TestRunsFailed.ThreeDay
	cs.TestRunFails7d.ResidualPreWeetbix = impact.TestRunsFailed.SevenDay

	cs.PresubmitRejects1d.ResidualPreWeetbix = impact.PresubmitRunsFailed.OneDay
	cs.PresubmitRejects3d.ResidualPreWeetbix = impact.PresubmitRunsFailed.ThreeDay
	cs.PresubmitRejects7d.ResidualPreWeetbix = impact.PresubmitRunsFailed.SevenDay
}
