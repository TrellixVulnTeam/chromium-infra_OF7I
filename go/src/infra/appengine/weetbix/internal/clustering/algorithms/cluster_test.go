// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package algorithms

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms/failurereason"
	"infra/appengine/weetbix/internal/clustering/algorithms/rulesalgorithm"
	"infra/appengine/weetbix/internal/clustering/algorithms/testname"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/clustering/rules/cache"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCluster(t *testing.T) {
	Convey(`Cluster`, t, func() {
		rulesVersion := time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC)

		rule1, err := cache.NewCachedRule(
			rules.NewRule(100).
				WithRuleDefinition(`test = "ninja://test_name_two/"`).
				WithLastUpdated(rulesVersion.Add(-1 * time.Hour)).Build())
		So(err, ShouldBeNil)
		rule2, err := cache.NewCachedRule(
			rules.NewRule(101).
				WithRuleDefinition(`reason LIKE "failed to connect to %.%.%.%"`).
				WithLastUpdated(rulesVersion).Build())
		So(err, ShouldBeNil)

		lastUpdated := time.Now()
		rules := []*cache.CachedRule{rule1, rule2}
		ruleset := cache.NewRuleset("myproject", rules, rulesVersion, lastUpdated)

		failures := []*clustering.Failure{
			{
				TestID: "ninja://test_name_one/",
			},
			{
				TestID: "ninja://test_name_two/",
				Reason: &pb.FailureReason{
					PrimaryErrorMessage: "failed to connect to 192.168.0.1",
				},
			},
		}

		expectedAlgorithms := map[string]struct{}{
			failurereason.AlgorithmName:  {},
			rulesalgorithm.AlgorithmName: {},
			testname.AlgorithmName:       {},
		}
		expectedClusters := [][]*clustering.ClusterID{
			{
				testNameClusterID(failures[0]),
			},
			{
				failureReasonClusterID(failures[1]),
				testNameClusterID(failures[1]),
				ruleClusterID(rule1.RuleID),
				ruleClusterID(rule2.RuleID),
			},
		}

		Convey(`From scratch`, func() {
			existing := NewEmptyClusterResults(len(failures))

			results := Cluster(ruleset, existing, failures)

			So(results.AlgorithmsVersion, ShouldEqual, AlgorithmsVersion)
			So(results.RulesVersion, ShouldEqual, rulesVersion)
			So(results.Algorithms, ShouldResemble, expectedAlgorithms)
			So(diffClusters(results.Clusters, expectedClusters), ShouldBeBlank)
		})
		Convey(`Incrementally`, func() {
			// This is an up-to-date clustering of the test results.
			existing := clustering.ClusterResults{
				AlgorithmsVersion: AlgorithmsVersion,
				RulesVersion:      rulesVersion,
				Algorithms: map[string]struct{}{
					failurereason.AlgorithmName:  {},
					rulesalgorithm.AlgorithmName: {},
					testname.AlgorithmName:       {},
				},
				Clusters: [][]*clustering.ClusterID{
					{
						testNameClusterID(failures[0]),
					},
					{
						failureReasonClusterID(failures[1]),
						testNameClusterID(failures[1]),
						ruleClusterID(rule1.RuleID),
						ruleClusterID(rule2.RuleID),
					},
				},
			}

			Convey(`From already up-to-date clustering`, func() {
				results := Cluster(ruleset, existing, failures)

				// Should not change the clustering.
				So(results.AlgorithmsVersion, ShouldEqual, AlgorithmsVersion)
				So(results.RulesVersion, ShouldEqual, rulesVersion)
				So(results.Algorithms, ShouldResemble, expectedAlgorithms)
				So(diffClusters(results.Clusters, expectedClusters), ShouldBeBlank)
			})

			Convey(`From older suggested clustering algorithm`, func() {
				existing.AlgorithmsVersion--
				delete(existing.Algorithms, failurereason.AlgorithmName)
				existing.Algorithms["failurereason-v1"] = struct{}{}
				existing.Clusters[1][0] = &clustering.ClusterID{
					Algorithm: "failurereason-v1",
					ID:        "old-failure-reason-cluster-id",
				}

				results := Cluster(ruleset, existing, failures)

				// Should produce the same clustering as clustering
				// from scratch.
				So(results.AlgorithmsVersion, ShouldEqual, AlgorithmsVersion)
				So(results.RulesVersion, ShouldEqual, rulesVersion)
				So(results.Algorithms, ShouldResemble, expectedAlgorithms)
				So(diffClusters(results.Clusters, expectedClusters), ShouldBeBlank)
			})
			Convey(`Incrementally from older rule-based clustering`, func() {
				existing.AlgorithmsVersion--
				delete(existing.Algorithms, rulesalgorithm.AlgorithmName)
				existing.Algorithms["rules-v0"] = struct{}{}
				existing.Clusters[1] = []*clustering.ClusterID{
					failureReasonClusterID(failures[1]),
					testNameClusterID(failures[1]),
					{Algorithm: "rules-v0", ID: rule1.RuleID},
					{Algorithm: "rules-v0", ID: "rule-no-longer-matched-with-v1"},
				}

				results := Cluster(ruleset, existing, failures)

				// Should produce the same clustering as clustering
				// from scratch.
				So(results.AlgorithmsVersion, ShouldEqual, AlgorithmsVersion)
				So(results.RulesVersion, ShouldEqual, rulesVersion)
				So(results.Algorithms, ShouldResemble, expectedAlgorithms)
				So(diffClusters(results.Clusters, expectedClusters), ShouldBeBlank)
			})
			Convey(`Incrementally from later clustering algorithms`, func() {
				existing.AlgorithmsVersion = AlgorithmsVersion + 1
				existing.Algorithms = map[string]struct{}{
					"futurealgorithm-v1": {},
				}
				existing.Clusters = [][]*clustering.ClusterID{
					{
						{Algorithm: "futurealgorithm-v1", ID: "aa"},
					},
					{
						{Algorithm: "futurealgorithm-v1", ID: "bb"},
					},
				}

				results := Cluster(ruleset, existing, failures)

				// As the algorithms version is later, the clustering
				// should be left completely untouched.
				So(results, ShouldResemble, existing)
			})
			Convey(`Incrementally from older rules version`, func() {
				existing.RulesVersion = rulesVersion.Add(-1 * time.Hour)
				existing.Clusters[1] = []*clustering.ClusterID{
					failureReasonClusterID(failures[1]),
					testNameClusterID(failures[1]),
					ruleClusterID(rule1.RuleID),
					ruleClusterID("now-deleted-rule-id"),
				}

				results := Cluster(ruleset, existing, failures)

				// Should produce the same clustering as clustering
				// from scratch.
				So(results.AlgorithmsVersion, ShouldEqual, AlgorithmsVersion)
				So(results.RulesVersion, ShouldEqual, rulesVersion)
				So(results.Algorithms, ShouldResemble, expectedAlgorithms)
				So(diffClusters(results.Clusters, expectedClusters), ShouldBeBlank)
			})
			Convey(`Incrementally from newer rules version`, func() {
				existing.RulesVersion = rulesVersion.Add(1 * time.Hour)
				existing.Clusters[1] = []*clustering.ClusterID{
					failureReasonClusterID(failures[1]),
					testNameClusterID(failures[1]),
					ruleClusterID(rule1.RuleID),
					ruleClusterID("later-added-rule-id"),
				}

				results := Cluster(ruleset, existing, failures)

				// Should keep existing rule clusters, as they are newer.
				expectedClusters = existing.Clusters
				So(results.AlgorithmsVersion, ShouldEqual, AlgorithmsVersion)
				So(results.RulesVersion, ShouldEqual, rulesVersion.Add(1*time.Hour))
				So(results.Algorithms, ShouldResemble, expectedAlgorithms)
				So(diffClusters(results.Clusters, expectedClusters), ShouldBeBlank)
			})
		})
	})
}

func testNameClusterID(failure *clustering.Failure) *clustering.ClusterID {
	alg := &testname.Algorithm{}
	return &clustering.ClusterID{
		Algorithm: testname.AlgorithmName,
		ID:        hex.EncodeToString(alg.Cluster(failure)),
	}
}

func failureReasonClusterID(failure *clustering.Failure) *clustering.ClusterID {
	alg := &failurereason.Algorithm{}
	return &clustering.ClusterID{
		Algorithm: failurereason.AlgorithmName,
		ID:        hex.EncodeToString(alg.Cluster(failure)),
	}
}

func ruleClusterID(ruleID string) *clustering.ClusterID {
	return &clustering.ClusterID{
		Algorithm: rulesalgorithm.AlgorithmName,
		ID:        ruleID,
	}
}

// diffClusters checks actual and expected clusters are equivalent (after
// accounting for ordering differences). If not, a message explaining
// the differences is returned.
func diffClusters(actual [][]*clustering.ClusterID, expected [][]*clustering.ClusterID) string {
	if len(actual) != len(expected) {
		return fmt.Sprintf("got clusters for %v test results; want %v", len(actual), len(expected))
	}
	for i, actualClusters := range actual {
		expectedClusters := expected[i]
		expectedClusterSet := make(map[string]struct{})
		for _, e := range expectedClusters {
			expectedClusterSet[e.Key()] = struct{}{}
		}

		actualClusterSet := make(map[string]struct{})
		for _, e := range actualClusters {
			actualClusterSet[e.Key()] = struct{}{}
		}
		for j, a := range actualClusters {
			if _, ok := expectedClusterSet[a.Key()]; ok {
				delete(expectedClusterSet, a.Key())
			} else {
				return fmt.Sprintf("actual clusters for test result %v includes cluster %v at position %v, which is not expected", i, a.Key(), j)
			}
		}
		if len(expectedClusterSet) > 0 {
			var missingClusters []string
			for c := range expectedClusterSet {
				missingClusters = append(missingClusters, c)
			}
			return fmt.Sprintf("actual clusters for test result %v is missing cluster(s): %s", i, strings.Join(missingClusters, ", "))
		}
	}
	return ""
}
