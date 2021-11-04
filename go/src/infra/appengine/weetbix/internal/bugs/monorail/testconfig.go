// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/config"
	mpb "infra/monorailv2/api/v3/api_proto"

	"github.com/golang/protobuf/proto"
)

// ChromiumTestPriorityField is the resource name of the priority field
// that is consistent with ChromiumTestConfig.
const ChromiumTestPriorityField = "projects/chromium/fieldDefs/11"

// ChromiumTestTypeField is the resource name of the type field
// that is consistent with ChromiumTestConfig.
const ChromiumTestTypeField = "projects/chromium/fieldDefs/10"

// ChromiumTestConfig provides chromium-like configuration for tests
// to use.
func ChromiumTestConfig() *config.MonorailProject {
	projectCfg := &config.MonorailProject{
		Project: "chromium",
		DefaultFieldValues: []*config.MonorailFieldValue{
			{
				FieldId: 10,
				Value:   "Bug",
			},
		},
		PriorityFieldId: 11,
		Priorities: []*config.MonorailPriority{
			{
				Priority: "0",
				Threshold: &config.ImpactThreshold{
					UnexpectedFailures_1D: proto.Int64(1000),
				},
			},
			{
				Priority: "1",
				Threshold: &config.ImpactThreshold{
					UnexpectedFailures_1D: proto.Int64(500),
				},
			},
			{
				Priority: "2",
				Threshold: &config.ImpactThreshold{
					UnexpectedFailures_1D: proto.Int64(100),
				},
			},
			{
				Priority: "3",
				// Should be less onerous than the bug-filing thresholds
				// used in BugUpdater tests, to avoid bugs that were filed
				// from being immediately closed.
				Threshold: &config.ImpactThreshold{
					UnexpectedFailures_1D: proto.Int64(50),
					UnexpectedFailures_3D: proto.Int64(300),
					UnexpectedFailures_7D: proto.Int64(700),
				},
			},
		},
		PriorityHysteresisPercent: 10,
	}
	return projectCfg
}

// ChromiumP0Impact returns cluster impact that is consistent with a P0 bug.
func ChromiumP0Impact() *bugs.ClusterImpact {
	return &bugs.ClusterImpact{
		Failures1d: 1500,
	}
}

// ChromiumP1Impact returns cluster impact that is consistent with a P1 bug.
func ChromiumP1Impact() *bugs.ClusterImpact {
	return &bugs.ClusterImpact{
		Failures1d: 750,
	}
}

// ChromiumLowP0Impact returns cluster impact that is consistent with a P0
// bug, but if hysteresis is applied, could also be compatible with P1.
func ChromiumLowP0Impact() *bugs.ClusterImpact {
	return &bugs.ClusterImpact{
		// (1000 * (1.0 + PriorityHysteresisPercent / 100.0)) - 1
		Failures1d: 1099,
	}
}

// ChromiumP2Impact returns cluster impact that is consistent with a P2 bug.
func ChromiumP2Impact() *bugs.ClusterImpact {
	return &bugs.ClusterImpact{
		Failures1d: 300,
	}
}

// ChromiumHighP2Impact returns cluster impact that is consistent with a P2
// bug, but if hysteresis is applied, could also be compatible with P1.
func ChromiumHighP2Impact() *bugs.ClusterImpact {
	return &bugs.ClusterImpact{
		// (500 / (1.0 + PriorityHysteresisPercent / 100.0)) + 1
		Failures1d: 455,
	}
}

// ChromiumP3Impact returns cluster impact that is consistent with a P3 bug.
func ChromiumP3Impact() *bugs.ClusterImpact {
	return &bugs.ClusterImpact{
		Failures1d: 75,
	}
}

// ChromiumP3LowImpact returns cluster impact that is consistent with a P3
// bug, but if hysteresis is applied, could also be compatible with a closed
// (verified) bug.
func ChromiumP3LowImpact() *bugs.ClusterImpact {
	return &bugs.ClusterImpact{
		// (50 * (1.0 + PriorityHysteresisPercent / 100.0)) - 1
		Failures1d: 54,
	}
}

// ChromiumClosureHighImpact returns cluster impact that is consistent with a
// closed (verified) bug, but if hysteresis is applied, could also be
// compatible with a P3 bug.
func ChromiumClosureHighImpact() *bugs.ClusterImpact {
	return &bugs.ClusterImpact{
		// (50 / (1.0 + PriorityHysteresisPercent / 100.0)) + 1
		Failures1d: 46,
	}
}

// ChromiumClosureImpact returns cluster impact that is consistent with a
// closed (verified) bug.
func ChromiumClosureImpact() *bugs.ClusterImpact {
	return &bugs.ClusterImpact{}
}

// ChromiumTestIssuePriority returns the priority of an issue, assuming
// it has been created consistent with ChromiumTestConfig.
func ChromiumTestIssuePriority(issue *mpb.Issue) string {
	for _, fv := range issue.FieldValues {
		if fv.Field == ChromiumTestPriorityField {
			return fv.Value
		}
	}
	return ""
}
