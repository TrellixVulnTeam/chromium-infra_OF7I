// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package monorail

import (
	"infra/appengine/weetbix/internal/clustering"
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
func ChromiumTestConfig() map[string]*config.MonorailProject {
	projectCfg := map[string]*config.MonorailProject{
		"chromium": {
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
					Threshold: &config.ImpactThreshold{
						UnexpectedFailures_1D: proto.Int64(50),
					},
				},
			},
			PriorityHysteresisPercent: 10,
		},
	}
	return projectCfg
}

// SetChromiumP0Impact sets cluster impact that is consistent with a P0 bug.
func SetChromiumP0Impact(cluster *clustering.Cluster) {
	cluster.UnexpectedFailures1d = 1500
}

// SetChromiumP1Impact sets cluster impact that is consistent with a P1 bug.
func SetChromiumP1Impact(cluster *clustering.Cluster) {
	cluster.UnexpectedFailures1d = 750
}

// SetChromiumLowP0Impact sets cluster impact that is consistent with a P0
// bug, but if hysteresis is applied, could also be compatible with P1.
func SetChromiumLowP0Impact(cluster *clustering.Cluster) {
	// (1000 * 1.1) - 1
	cluster.UnexpectedFailures1d = 1099
}

// SetChromiumP2Impact sets cluster impact that is consistent with a P2 bug.
func SetChromiumP2Impact(cluster *clustering.Cluster) {
	cluster.UnexpectedFailures1d = 300
}

// SetChromiumHighP2Impact sets cluster impact that is consistent with a P2
// bug, but if hysteresis is applied, could also be compatible with P1.
func SetChromiumHighP2Impact(cluster *clustering.Cluster) {
	// (500 / 1.1) + 1
	cluster.UnexpectedFailures1d = 455
}

// SetChromiumP3Impact sets cluster impact that is consistent with a P3 bug.
func SetChromiumP3Impact(cluster *clustering.Cluster) {
	cluster.UnexpectedFailures1d = 75
}

// SetChromiumP3LowImpact sets cluster impact that is consistent with a P3
// bug, but if hysteresis is applied, could also be compatible with a closed
// (verified) bug.
func SetChromiumP3LowImpact(cluster *clustering.Cluster) {
	// (50 * 1.1) - 1
	cluster.UnexpectedFailures1d = 54
}

// SetChromiumClosureHighImpact sets cluster impact that is consistent with a
// closed (verified) bug, but if hysteresis is applied, could also be
// compatible with a P3 bug.
func SetChromiumClosureHighImpact(cluster *clustering.Cluster) {
	// (50 / 1.1) + 1
	cluster.UnexpectedFailures1d = 46
}

// SetChromiumClosureImpact sets cluster impact that is consistent with a
// closed (verified) bug.
func SetChromiumClosureImpact(cluster *clustering.Cluster) {
	cluster.UnexpectedFailures1d = 0
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
