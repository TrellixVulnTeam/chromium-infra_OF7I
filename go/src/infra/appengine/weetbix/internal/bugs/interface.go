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
	// The ID of the Failure Association Rule being created.
	RuleID string
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

// ExtractResidualImpact extracts the residual impact from a cluster summary.
func ExtractResidualImpact(cs *analysis.ClusterSummary) *ClusterImpact {
	return &ClusterImpact{
		TestResultsFailed: MetricImpact{
			OneDay:   cs.Failures1d.Residual,
			ThreeDay: cs.Failures3d.Residual,
			SevenDay: cs.Failures7d.Residual,
		},
		TestRunsFailed: MetricImpact{
			OneDay:   cs.TestRunFails1d.Residual,
			ThreeDay: cs.TestRunFails3d.Residual,
			SevenDay: cs.TestRunFails7d.Residual,
		},
		PresubmitRunsFailed: MetricImpact{
			OneDay:   cs.PresubmitRejects1d.Residual,
			ThreeDay: cs.PresubmitRejects3d.Residual,
			SevenDay: cs.PresubmitRejects7d.Residual,
		},
	}
}

// SetResidualImpact sets the residual impact on a cluster summary.
func SetResidualImpact(cs *analysis.ClusterSummary, impact *ClusterImpact) {
	cs.Failures1d.Residual = impact.TestResultsFailed.OneDay
	cs.Failures3d.Residual = impact.TestResultsFailed.ThreeDay
	cs.Failures7d.Residual = impact.TestResultsFailed.SevenDay

	cs.TestRunFails1d.Residual = impact.TestRunsFailed.OneDay
	cs.TestRunFails3d.Residual = impact.TestRunsFailed.ThreeDay
	cs.TestRunFails7d.Residual = impact.TestRunsFailed.SevenDay

	cs.PresubmitRejects1d.Residual = impact.PresubmitRunsFailed.OneDay
	cs.PresubmitRejects3d.Residual = impact.PresubmitRunsFailed.ThreeDay
	cs.PresubmitRejects7d.Residual = impact.PresubmitRunsFailed.SevenDay
}
