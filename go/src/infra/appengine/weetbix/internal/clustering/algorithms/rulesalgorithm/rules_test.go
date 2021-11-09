// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rulesalgorithm

import (
	"testing"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/clustering/rules/cache"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAlgorithm(t *testing.T) {
	Convey(`Cluster`, t, func() {
		a := &Algorithm{}
		Convey(`Empty Rules`, func() {
			ruleset := &cache.Ruleset{}
			ids := a.Cluster(ruleset, &clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: "Null pointer exception at ip 0x45637271"},
			})
			So(ids, ShouldBeEmpty)
		})
		Convey(`With Rules`, func() {
			rule1, err := cache.NewCachedRule(rules.NewRule(100).WithRuleDefinition(`test = "ninja://test_name_one/"`).Build())
			So(err, ShouldBeNil)
			rule2, err := cache.NewCachedRule(rules.NewRule(101).WithRuleDefinition(`reason LIKE "failed to connect to %.%.%.%"`).Build())
			So(err, ShouldBeNil)

			ruleset := &cache.Ruleset{
				Rules: []*cache.CachedRule{rule1, rule2},
			}

			Convey(`Without failure reason`, func() {
				ids := a.Cluster(ruleset, &clustering.Failure{
					TestID: "ninja://test_name_one/",
				})
				So(ids, ShouldResemble, []*clustering.ClusterID{{Algorithm: AlgorithmName, ID: rule1.RuleID}})

				ids = a.Cluster(ruleset, &clustering.Failure{
					TestID: "ninja://test_name_two/",
				})
				So(ids, ShouldBeEmpty)
			})
			Convey(`Matches one`, func() {
				ids := a.Cluster(ruleset, &clustering.Failure{
					TestID: "ninja://test_name_three/",
					Reason: &pb.FailureReason{PrimaryErrorMessage: "failed to connect to 192.168.0.1"},
				})
				So(ids, ShouldResemble, []*clustering.ClusterID{
					{Algorithm: AlgorithmName, ID: rule2.RuleID},
				})
			})
			Convey(`Matches multiple`, func() {
				ids := a.Cluster(ruleset, &clustering.Failure{
					TestID: "ninja://test_name_one/",
					Reason: &pb.FailureReason{PrimaryErrorMessage: "failed to connect to 192.168.0.1"},
				})
				So(ids, ShouldResemble, []*clustering.ClusterID{
					{Algorithm: AlgorithmName, ID: rule1.RuleID},
					{Algorithm: AlgorithmName, ID: rule2.RuleID},
				})
			})
		})
	})
}
