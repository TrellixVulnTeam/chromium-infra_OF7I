// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugs

import (
	"errors"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/clustering"
)

type BugToUpdate struct {
	BugName string
	// Cluster details for the given bug.
	Impact *ClusterImpact
}

var ErrCreateSimulated = errors.New("CreateNew did not create a bug as the bug manager is in simulation mode")

// CreateRequest captures key details of a cluster and its impact,
// as needed for filing new bugs.
type CreateRequest struct {
	// Description is a detailed description of the cluster.
	Description *clustering.ClusterDescription
	// Impact describes the impact of cluster.
	Impact *ClusterImpact
}

// ClusterImpact captures details of a cluster's impact, as needed
// to control the priority and verified status of bugs.
type ClusterImpact struct {
	TestResultsFailed   MetricImpact
	TestRunsFailed      MetricImpact
	PresubmitRunsFailed MetricImpact
}

// MetricImpact captures impact measurements for one metric, over
// different timescales.
type MetricImpact struct {
	OneDay   int64
	ThreeDay int64
	SevenDay int64
}

// ExtractResidualPreWeetbixImpact extracts the residual,
// pre-weetbix exoneration, impact from a cluster summary.
func ExtractResidualPreWeetbixImpact(cs *analysis.ClusterSummary) *ClusterImpact {
	return &ClusterImpact{
		TestResultsFailed: MetricImpact{
			OneDay:   cs.Failures1d.ResidualPreWeetbix,
			ThreeDay: cs.Failures3d.ResidualPreWeetbix,
			SevenDay: cs.Failures7d.ResidualPreWeetbix,
		},
		TestRunsFailed: MetricImpact{
			OneDay:   cs.TestRunFails1d.ResidualPreWeetbix,
			ThreeDay: cs.TestRunFails3d.ResidualPreWeetbix,
			SevenDay: cs.TestRunFails7d.ResidualPreWeetbix,
		},
		PresubmitRunsFailed: MetricImpact{
			OneDay:   cs.PresubmitRejects1d.ResidualPreWeetbix,
			ThreeDay: cs.PresubmitRejects3d.ResidualPreWeetbix,
			SevenDay: cs.PresubmitRejects7d.ResidualPreWeetbix,
		},
	}
}

// SetResidualPreWeetbixImpact sets the residual, pre-weetbix exoneration
// impact on a cluster summary.
func SetResidualPreWeetbixImpact(cs *analysis.ClusterSummary, impact *ClusterImpact) {
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
