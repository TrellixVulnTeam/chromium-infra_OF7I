// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rulesalgorithm

import (
	"testing"
	"time"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/clustering/rules/cache"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAlgorithm(t *testing.T) {
	Convey(`Cluster from scratch`, t, func() {
		a := &Algorithm{}
		existingRulesVersion := rules.StartingEpoch
		existingIDs := make(map[string]struct{})
		Convey(`Empty Rules`, func() {
			ruleset := &cache.Ruleset{}
			ids := a.Cluster(ruleset, existingRulesVersion, existingIDs, &clustering.Failure{
				Reason: &pb.FailureReason{PrimaryErrorMessage: "Null pointer exception at ip 0x45637271"},
			})
			So(ids, ShouldBeEmpty)
		})
		Convey(`With Rules`, func() {
			rule1, err := cache.NewCachedRule(
				rules.NewRule(100).
					WithRuleDefinition(`test = "ninja://test_name_one/"`).
					Build())
			So(err, ShouldBeNil)
			rule2, err := cache.NewCachedRule(
				rules.NewRule(101).
					WithRuleDefinition(`reason LIKE "failed to connect to %.%.%.%"`).
					Build())
			So(err, ShouldBeNil)

			rulesVersion := time.Now()
			lastUpdated := time.Now()
			rules := []*cache.CachedRule{rule1, rule2}
			ruleset := cache.NewRuleset("myproject", rules, rulesVersion, lastUpdated)

			Convey(`Without failure reason`, func() {
				ids := a.Cluster(ruleset, existingRulesVersion, existingIDs, &clustering.Failure{
					TestID: "ninja://test_name_one/",
				})
				So(ids, ShouldResemble, map[string]struct{}{
					rule1.RuleID: {},
				})

				ids = a.Cluster(ruleset, existingRulesVersion, existingIDs, &clustering.Failure{
					TestID: "ninja://test_name_two/",
				})
				So(ids, ShouldBeEmpty)
			})
			Convey(`Matches one`, func() {
				ids := a.Cluster(ruleset, existingRulesVersion, existingIDs, &clustering.Failure{
					TestID: "ninja://test_name_three/",
					Reason: &pb.FailureReason{
						PrimaryErrorMessage: "failed to connect to 192.168.0.1",
					},
				})
				So(ids, ShouldResemble, map[string]struct{}{
					rule2.RuleID: {},
				})
			})
			Convey(`Matches multiple`, func() {
				ids := a.Cluster(ruleset, existingRulesVersion, existingIDs, &clustering.Failure{
					TestID: "ninja://test_name_one/",
					Reason: &pb.FailureReason{
						PrimaryErrorMessage: "failed to connect to 192.168.0.1",
					},
				})
				So(ids, ShouldResemble, map[string]struct{}{
					rule1.RuleID: {},
					rule2.RuleID: {},
				})
			})
		})
	})
	Convey(`Cluster incrementally`, t, func() {
		a := &Algorithm{}
		originalRulesVersion := time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC)
		testFailure := &clustering.Failure{
			TestID: "ninja://test_name_one/",
			Reason: &pb.FailureReason{
				PrimaryErrorMessage: "failed to connect to 192.168.0.1",
			},
		}

		// The ruleset we are incrementally clustering with has a new rule
		// (rule 3) and no longer has rule 2. We silently set the definition
		// of rule1 to FALSE without changing its last updated time (this
		// should never happen in reality) to check it is never evaluated.
		rule1, err := cache.NewCachedRule(
			rules.NewRule(100).WithRuleDefinition(`FALSE`).
				WithLastUpdated(originalRulesVersion).Build())
		So(err, ShouldBeNil)
		rule3, err := cache.NewCachedRule(
			rules.NewRule(102).
				WithRuleDefinition(`reason LIKE "failed to connect to %"`).
				WithLastUpdated(originalRulesVersion.Add(time.Hour)).Build())
		So(err, ShouldBeNil)

		rules := []*cache.CachedRule{rule1, rule3}
		newRulesVersion := originalRulesVersion.Add(time.Hour)
		lastUpdated := time.Now()
		secondRuleset := cache.NewRuleset("myproject", rules, newRulesVersion, lastUpdated)

		firstIDs := map[string]struct{}{
			rule1.RuleID: {},
			"rule2-id":   {},
		}

		// Test incrementally clustering leads to the correct outcome,
		// matching rule 3 and unmatching rule 2.
		secondIDs := a.Cluster(secondRuleset, originalRulesVersion, firstIDs, testFailure)
		So(secondIDs, ShouldResemble, map[string]struct{}{
			rule1.RuleID: {},
			rule3.RuleID: {},
		})
	})
}
