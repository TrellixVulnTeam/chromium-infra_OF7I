// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package updater

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/config/validation"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/span"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/bugs/monorail"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms"
	"infra/appengine/weetbix/internal/clustering/algorithms/failurereason"
	"infra/appengine/weetbix/internal/clustering/algorithms/rulesalgorithm"
	"infra/appengine/weetbix/internal/clustering/algorithms/testname"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/clustering/runs"
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/config/compiledcfg"
	configpb "infra/appengine/weetbix/internal/config/proto"
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"
)

func TestRun(t *testing.T) {
	Convey("Run bug updates", t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx = memory.Use(ctx)

		f := &monorail.FakeIssuesStore{
			NextID:            100,
			PriorityFieldName: "projects/chromium/fieldDefs/11",
		}
		user := monorail.AutomationUsers[0]
		mc, err := monorail.NewClient(monorail.UseFakeIssuesClient(ctx, f, user), "myhost")
		So(err, ShouldBeNil)

		project := "chromium"
		monorailCfg := monorail.ChromiumTestConfig()
		thres := &configpb.ImpactThreshold{
			// Should be more onerous than the "keep-open" thresholds
			// configured for each individual bug manager.
			TestResultsFailed: &configpb.MetricThreshold{
				OneDay:   proto.Int64(100),
				ThreeDay: proto.Int64(300),
				SevenDay: proto.Int64(700),
			},
		}
		projectCfg := &configpb.ProjectConfig{
			Monorail:           monorailCfg,
			BugFilingThreshold: thres,
			LastUpdated:        timestamppb.New(time.Date(2030, time.July, 1, 0, 0, 0, 0, time.UTC)),
		}
		projectsCfg := map[string]*configpb.ProjectConfig{
			project: projectCfg,
		}
		err = config.SetTestProjectConfig(ctx, projectsCfg)
		So(err, ShouldBeNil)

		compiledCfg, err := compiledcfg.NewConfig(projectCfg)
		So(err, ShouldBeNil)

		suggestedClusters := []*analysis.ClusterSummary{
			makeReasonCluster(compiledCfg, 0),
			makeReasonCluster(compiledCfg, 1),
			makeReasonCluster(compiledCfg, 2),
			makeReasonCluster(compiledCfg, 3),
		}
		ac := &fakeAnalysisClient{
			clusters: suggestedClusters,
		}

		opts := updateOptions{
			appID:              "chops-weetbix-test",
			project:            project,
			analysisClient:     ac,
			monorailClient:     mc,
			enableBugUpdates:   true,
			maxBugsFiledPerRun: 1,
		}

		// Unless otherwise specified, assume re-clustering has caught up to
		// the latest version of algorithms and config.
		err = runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{
			runs.NewRun(0).
				WithProject(project).
				WithAlgorithmsVersion(algorithms.AlgorithmsVersion).
				WithConfigVersion(projectCfg.LastUpdated.AsTime()).
				WithRulesVersion(rules.StartingEpoch).
				WithCompletedProgress().Build(),
		})
		So(err, ShouldBeNil)

		Convey("Configuration used for testing is valid", func() {
			c := validation.Context{Context: context.Background()}

			config.ValidateProjectConfig(&c, projectCfg)
			So(c.Finalize(), ShouldBeNil)
		})
		Convey("With no impactful clusters", func() {
			err = updateAnalysisAndBugsForProject(ctx, opts)
			So(err, ShouldBeNil)

			// No failure association rules.
			rs, err := rules.ReadActive(span.Single(ctx), project)
			So(err, ShouldBeNil)
			So(rs, ShouldResemble, []*rules.FailureAssociationRule{})

			// No monorail issues.
			So(f.Issues, ShouldBeNil)
		})
		Convey("With a suggested cluster above impact thresold", func() {
			sourceClusterID := reasonClusterID(compiledCfg, "Failed to connect to 100.1.1.99.")
			suggestedClusters[1].ClusterID = sourceClusterID
			suggestedClusters[1].ExampleFailureReason = bigquery.NullString{StringVal: "Failed to connect to 100.1.1.105.", Valid: true}
			suggestedClusters[1].TopTestIDs = []analysis.TopCount{
				{Value: "network-test-1", Count: 10},
				{Value: "network-test-2", Count: 10},
			}

			ignoreRuleID := ""
			expectCreate := true

			expectedRule := &rules.FailureAssociationRule{
				Project:         "chromium",
				RuleDefinition:  `reason LIKE "Failed to connect to %.%.%.%."`,
				BugID:           bugs.BugID{System: "monorail", ID: "chromium/100"},
				IsActive:        true,
				IsManagingBug:   true,
				SourceCluster:   sourceClusterID,
				CreationUser:    rules.WeetbixSystem,
				LastUpdatedUser: rules.WeetbixSystem,
			}

			expectedBugSummary := "Failed to connect to 100.1.1.105."

			// Expect the bug description to contain the top tests.
			expectedBugContents := []string{
				"network-test-1",
				"network-test-2",
			}

			test := func() {
				err = updateAnalysisAndBugsForProject(ctx, opts)
				So(err, ShouldBeNil)

				rs, err := rules.ReadActive(span.Single(ctx), project)
				So(err, ShouldBeNil)

				var cleanedRules []*rules.FailureAssociationRule
				for _, r := range rs {
					if r.RuleID != ignoreRuleID {
						cleanedRules = append(cleanedRules, r)
					}
				}

				if !expectCreate {
					So(len(cleanedRules), ShouldEqual, 0)
					return
				}

				So(len(cleanedRules), ShouldEqual, 1)
				rule := cleanedRules[0]

				// Accept whatever bug cluster ID has been generated.
				So(rule.RuleID, ShouldNotBeEmpty)
				expectedRule.RuleID = rule.RuleID

				// Accept creation and last updated times, as set by Spanner.
				So(rule.CreationTime, ShouldNotBeZeroValue)
				expectedRule.CreationTime = rule.CreationTime
				So(rule.LastUpdated, ShouldNotBeZeroValue)
				expectedRule.LastUpdated = rule.LastUpdated
				So(rule.PredicateLastUpdated, ShouldNotBeZeroValue)
				expectedRule.PredicateLastUpdated = rule.PredicateLastUpdated
				So(rule, ShouldResemble, expectedRule)

				So(len(f.Issues), ShouldEqual, 1)
				So(f.Issues[0].Issue.Name, ShouldEqual, "projects/chromium/issues/100")
				So(f.Issues[0].Issue.Summary, ShouldContainSubstring, expectedBugSummary)
				So(len(f.Issues[0].Comments), ShouldEqual, 2)
				for _, expectedContent := range expectedBugContents {
					So(f.Issues[0].Comments[0].Content, ShouldContainSubstring, expectedContent)
				}
				// Expect a link to the bug and the rule.
				So(f.Issues[0].Comments[1].Content, ShouldContainSubstring, "https://chops-weetbix-test.appspot.com/b/chromium/100")
			}

			Convey("1d unexpected failures", func() {
				Convey("Reason cluster", func() {
					Convey("Above thresold", func() {
						suggestedClusters[1].Failures1d.ResidualPreWeetbix = 100
						test()

						// Further updates do nothing.
						test()
					})
					Convey("Below threshold", func() {
						suggestedClusters[1].Failures1d.ResidualPreWeetbix = 99
						expectCreate = false
						test()
					})
				})
				Convey("Test name cluster", func() {
					suggestedClusters[1].ClusterID = testIDClusterID(compiledCfg, "ui-test-1")
					suggestedClusters[1].TopTestIDs = []analysis.TopCount{
						{Value: "ui-test-1", Count: 10},
					}
					expectedRule.RuleDefinition = `test = "ui-test-1"`
					expectedRule.SourceCluster = suggestedClusters[1].ClusterID
					expectedBugSummary = "ui-test-1"
					expectedBugContents = []string{"ui-test-1"}

					// 34% more impact is required for a test name cluster to
					// be filed, compared to a failure reason cluster.
					Convey("Above thresold", func() {
						suggestedClusters[1].Failures1d.ResidualPreWeetbix = 134
						test()

						// Further updates do nothing.
						test()
					})
					Convey("Below threshold", func() {
						suggestedClusters[1].Failures1d.ResidualPreWeetbix = 133
						expectCreate = false
						test()
					})
				})
			})
			Convey("3d unexpected failures", func() {
				suggestedClusters[1].Failures3d.ResidualPreWeetbix = 300
				test()

				// Further updates do nothing.
				test()
			})
			Convey("7d unexpected failures", func() {
				suggestedClusters[1].Failures7d.ResidualPreWeetbix = 700
				test()

				// Further updates do nothing.
				test()
			})
			Convey("With existing rule filed", func() {
				suggestedClusters[1].Failures1d.ResidualPreWeetbix = 100

				createTime := time.Date(2021, time.January, 5, 12, 30, 0, 0, time.UTC)
				rule := rules.NewRule(0).
					WithProject(project).
					WithCreationTime(createTime).
					WithPredicateLastUpdated(createTime.Add(1 * time.Hour)).
					WithLastUpdated(createTime.Add(2 * time.Hour)).
					WithSourceCluster(sourceClusterID).Build()
				err := rules.SetRulesForTesting(ctx, []*rules.FailureAssociationRule{
					rule,
				})
				So(err, ShouldBeNil)
				ignoreRuleID = rule.RuleID

				// Initially do not expect a new bug to be filed.
				expectCreate = false
				test()

				// Once re-clustering has incorporated the version of rules
				// that included this new rule, it is OK to file another bug
				// for the suggested cluster if sufficient impact remains.
				// This should only happen when the rule definition has been
				// manually narrowed in some way from the originally filed bug.
				err = runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{
					runs.NewRun(0).
						WithProject(project).
						WithAlgorithmsVersion(algorithms.AlgorithmsVersion).
						WithConfigVersion(projectCfg.LastUpdated.AsTime()).
						WithRulesVersion(createTime).
						WithCompletedProgress().Build(),
				})
				So(err, ShouldBeNil)

				expectCreate = true
				test()
			})
			Convey("With bug updates disabled", func() {
				suggestedClusters[1].Failures1d.ResidualPreWeetbix = 100

				opts.enableBugUpdates = false

				expectCreate = false
				test()
			})
			Convey("Without re-clustering caught up to latest algorithms", func() {
				suggestedClusters[1].Failures1d.ResidualPreWeetbix = 100

				err = runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{
					runs.NewRun(0).
						WithProject(project).
						WithAlgorithmsVersion(algorithms.AlgorithmsVersion - 1).
						WithConfigVersion(projectCfg.LastUpdated.AsTime()).
						WithRulesVersion(rules.StartingEpoch).
						WithCompletedProgress().Build(),
				})
				So(err, ShouldBeNil)

				expectCreate = false
				test()
			})
			Convey("Without re-clustering caught up to latest config", func() {
				suggestedClusters[1].Failures1d.ResidualPreWeetbix = 100

				err = runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{
					runs.NewRun(0).
						WithProject(project).
						WithAlgorithmsVersion(algorithms.AlgorithmsVersion).
						WithConfigVersion(projectCfg.LastUpdated.AsTime().Add(-1 * time.Hour)).
						WithRulesVersion(rules.StartingEpoch).
						WithCompletedProgress().Build(),
				})
				So(err, ShouldBeNil)

				expectCreate = false
				test()
			})
		})
		Convey("With both failure reason and test name clusters above bug-filing threshold", func() {
			// Reason cluster above the 3-day failure threshold.
			suggestedClusters[2] = makeReasonCluster(compiledCfg, 2)
			suggestedClusters[2].Failures3d.ResidualPreWeetbix = 400
			suggestedClusters[2].Failures7d.ResidualPreWeetbix = 400

			// Test name cluster with 33% more impact.
			suggestedClusters[1] = makeTestNameCluster(compiledCfg, 3)
			suggestedClusters[1].Failures3d.ResidualPreWeetbix = 532
			suggestedClusters[1].Failures7d.ResidualPreWeetbix = 532

			// Limit to one bug filed each time, so that
			// we test change throttling.
			opts.maxBugsFiledPerRun = 1

			Convey("Reason clusters preferred over test name clusters", func() {
				// Test name cluster has <34% more impact than the reason
				// cluster.
				err = updateAnalysisAndBugsForProject(ctx, opts)
				So(err, ShouldBeNil)

				// Reason cluster filed.
				bugClusters, err := rules.ReadActive(span.Single(ctx), project)
				So(err, ShouldBeNil)
				So(len(bugClusters), ShouldEqual, 1)
				So(bugClusters[0].SourceCluster, ShouldResemble, suggestedClusters[2].ClusterID)
				So(bugClusters[0].SourceCluster.IsFailureReasonCluster(), ShouldBeTrue)
			})
			Convey("Test name clusters can be filed if significantly more impact", func() {
				// Reduce impact of the reason-based cluster so that the
				// test name cluster has >34% more impact than the reason
				// cluster.
				suggestedClusters[2].Failures3d.ResidualPreWeetbix = 390
				suggestedClusters[2].Failures7d.ResidualPreWeetbix = 390

				err = updateAnalysisAndBugsForProject(ctx, opts)
				So(err, ShouldBeNil)

				// Test name cluster filed first.
				bugClusters, err := rules.ReadActive(span.Single(ctx), project)
				So(err, ShouldBeNil)
				So(len(bugClusters), ShouldEqual, 1)
				So(bugClusters[0].SourceCluster, ShouldResemble, suggestedClusters[1].ClusterID)
				So(bugClusters[0].SourceCluster.IsTestNameCluster(), ShouldBeTrue)
			})
		})
		Convey("With multiple suggested clusters above impact thresold", func() {
			expectBugClusters := func(count int) {
				bugClusters, err := rules.ReadActive(span.Single(ctx), project)
				So(err, ShouldBeNil)
				So(len(bugClusters), ShouldEqual, count)
				So(len(f.Issues), ShouldEqual, count)
			}
			// Use a mix of test name and failure reason clusters for
			// code path coverage.
			suggestedClusters[0] = makeTestNameCluster(compiledCfg, 0)
			suggestedClusters[0].Failures7d.ResidualPreWeetbix = 940
			suggestedClusters[1] = makeReasonCluster(compiledCfg, 1)
			suggestedClusters[1].Failures3d.ResidualPreWeetbix = 300
			suggestedClusters[1].Failures7d.ResidualPreWeetbix = 300
			suggestedClusters[2] = makeReasonCluster(compiledCfg, 2)
			suggestedClusters[2].Failures1d.ResidualPreWeetbix = 200
			suggestedClusters[2].Failures3d.ResidualPreWeetbix = 200
			suggestedClusters[2].Failures7d.ResidualPreWeetbix = 200

			// Limit to one bug filed each time, so that
			// we test change throttling.
			opts.maxBugsFiledPerRun = 1

			err = updateAnalysisAndBugsForProject(ctx, opts)
			So(err, ShouldBeNil)
			expectBugClusters(1)

			err = updateAnalysisAndBugsForProject(ctx, opts)
			So(err, ShouldBeNil)
			expectBugClusters(2)

			err = updateAnalysisAndBugsForProject(ctx, opts)
			So(err, ShouldBeNil)

			expectFinalRules := func() {
				// Check final set of rules is as expected.
				rs, err := rules.ReadActive(span.Single(ctx), project)
				So(err, ShouldBeNil)
				for _, r := range rs {
					So(r.RuleID, ShouldNotBeEmpty)
					So(r.CreationTime, ShouldNotBeZeroValue)
					So(r.LastUpdated, ShouldNotBeZeroValue)
					So(r.PredicateLastUpdated, ShouldNotBeZeroValue)
					// Accept whatever values the implementation has set.
					r.RuleID = ""
					r.CreationTime = time.Time{}
					r.LastUpdated = time.Time{}
					r.PredicateLastUpdated = time.Time{}
				}

				So(rs, ShouldResemble, []*rules.FailureAssociationRule{
					{
						Project:         "chromium",
						RuleDefinition:  `test = "testname-0"`,
						BugID:           bugs.BugID{System: "monorail", ID: "chromium/100"},
						SourceCluster:   suggestedClusters[0].ClusterID,
						IsActive:        true,
						IsManagingBug:   true,
						CreationUser:    rules.WeetbixSystem,
						LastUpdatedUser: rules.WeetbixSystem,
					},
					{
						Project:         "chromium",
						RuleDefinition:  `reason LIKE "want foo, got bar"`,
						BugID:           bugs.BugID{System: "monorail", ID: "chromium/101"},
						SourceCluster:   suggestedClusters[1].ClusterID,
						IsActive:        true,
						IsManagingBug:   true,
						CreationUser:    rules.WeetbixSystem,
						LastUpdatedUser: rules.WeetbixSystem,
					},
					{
						Project:         "chromium",
						RuleDefinition:  `reason LIKE "want foofoo, got bar"`,
						BugID:           bugs.BugID{System: "monorail", ID: "chromium/102"},
						SourceCluster:   suggestedClusters[2].ClusterID,
						IsActive:        true,
						IsManagingBug:   true,
						CreationUser:    rules.WeetbixSystem,
						LastUpdatedUser: rules.WeetbixSystem,
					},
				})
				So(len(f.Issues), ShouldEqual, 3)
			}
			expectFinalRules()

			// Further updates do nothing.
			originalIssues := monorail.CopyIssuesStore(f)
			err = updateAnalysisAndBugsForProject(ctx, opts)
			So(err, ShouldBeNil)
			So(f, monorail.ShouldResembleIssuesStore, originalIssues)
			expectFinalRules()

			rs, err := rules.ReadActive(span.Single(ctx), project)
			So(err, ShouldBeNil)

			bugClusters := []*analysis.ClusterSummary{
				makeBugCluster(rs[0].RuleID),
				makeBugCluster(rs[1].RuleID),
				makeBugCluster(rs[2].RuleID),
			}

			Convey("Re-clustering in progress", func() {
				ac.clusters = append(suggestedClusters, bugClusters[1:]...)

				Convey("Negligable cluster impact does not affect issue priority or status", func() {
					issue := f.Issues[0].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/100")
					originalPriority := monorail.ChromiumTestIssuePriority(issue)
					originalStatus := issue.Status.Status
					So(originalStatus, ShouldNotEqual, monorail.VerifiedStatus)

					SetResidualPreWeetbixImpact(
						bugClusters[1], monorail.ChromiumClosureImpact())
					err = updateAnalysisAndBugsForProject(ctx, opts)
					So(err, ShouldBeNil)

					So(len(f.Issues), ShouldEqual, 3)
					issue = f.Issues[0].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/100")
					So(monorail.ChromiumTestIssuePriority(issue), ShouldEqual, originalPriority)
					So(issue.Status.Status, ShouldEqual, originalStatus)

					expectFinalRules()
				})
			})
			Convey("Re-clustering complete", func() {
				ac.clusters = append(suggestedClusters, bugClusters[1:]...)

				// Copy impact from suggested clusters to new bug clusters.
				bugClusters[0].Failures7d = suggestedClusters[0].Failures7d
				bugClusters[1].Failures3d = suggestedClusters[1].Failures3d
				bugClusters[1].Failures7d = suggestedClusters[1].Failures7d
				bugClusters[2].Failures1d = suggestedClusters[2].Failures1d
				bugClusters[2].Failures3d = suggestedClusters[2].Failures3d
				bugClusters[2].Failures7d = suggestedClusters[2].Failures7d

				// Clear residual impact on suggested clusters
				suggestedClusters[0].Failures7d.ResidualPreWeetbix = 0
				suggestedClusters[1].Failures3d.ResidualPreWeetbix = 0
				suggestedClusters[1].Failures7d.ResidualPreWeetbix = 0
				suggestedClusters[2].Failures1d.ResidualPreWeetbix = 0
				suggestedClusters[2].Failures3d.ResidualPreWeetbix = 0
				suggestedClusters[2].Failures7d.ResidualPreWeetbix = 0

				// Mark reclustering complete.
				err := runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{
					runs.NewRun(0).
						WithProject(project).
						WithAlgorithmsVersion(algorithms.AlgorithmsVersion).
						WithConfigVersion(projectCfg.LastUpdated.AsTime()).
						WithRulesVersion(rs[2].PredicateLastUpdated).
						WithCompletedProgress().Build(),
				})
				So(err, ShouldBeNil)

				Convey("Cluster impact does not change if bug not managed by rule", func() {
					// Set IsManagingBug to false on one rule.
					rs[2].IsManagingBug = false
					rules.SetRulesForTesting(ctx, rs)

					issue := f.Issues[2].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
					originalPriority := monorail.ChromiumTestIssuePriority(issue)
					originalStatus := issue.Status.Status
					So(originalPriority, ShouldNotEqual, "0")

					// Set P0 impact on the cluster.
					SetResidualPreWeetbixImpact(
						bugClusters[2], monorail.ChromiumP0Impact())
					err = updateAnalysisAndBugsForProject(ctx, opts)
					So(err, ShouldBeNil)

					// Check that the rule priority and status has not changed.
					So(len(f.Issues), ShouldEqual, 3)
					issue = f.Issues[2].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
					So(issue.Status.Status, ShouldEqual, originalStatus)
					So(monorail.ChromiumTestIssuePriority(issue), ShouldEqual, originalPriority)
				})
				Convey("Increasing cluster impact increases issue priority", func() {
					issue := f.Issues[2].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
					So(monorail.ChromiumTestIssuePriority(issue), ShouldNotEqual, "0")

					SetResidualPreWeetbixImpact(
						bugClusters[2], monorail.ChromiumP0Impact())
					err = updateAnalysisAndBugsForProject(ctx, opts)
					So(err, ShouldBeNil)

					So(len(f.Issues), ShouldEqual, 3)
					issue = f.Issues[2].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
					So(monorail.ChromiumTestIssuePriority(issue), ShouldEqual, "0")

					expectFinalRules()
				})
				Convey("Decreasing cluster impact decreases issue priority", func() {
					issue := f.Issues[2].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
					So(monorail.ChromiumTestIssuePriority(issue), ShouldNotEqual, "3")

					SetResidualPreWeetbixImpact(
						bugClusters[2], monorail.ChromiumP3Impact())
					err = updateAnalysisAndBugsForProject(ctx, opts)
					So(err, ShouldBeNil)

					So(len(f.Issues), ShouldEqual, 3)
					issue = f.Issues[2].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
					So(issue.Status.Status, ShouldEqual, monorail.UntriagedStatus)
					So(monorail.ChromiumTestIssuePriority(issue), ShouldEqual, "3")

					expectFinalRules()
				})
				Convey("Deleting cluster closes issue", func() {
					issue := f.Issues[2].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
					So(issue.Status.Status, ShouldEqual, monorail.UntriagedStatus)

					// Drop the bug cluster at index 2.
					bugClusters = bugClusters[:2]
					ac.clusters = append(suggestedClusters, bugClusters...)
					err = updateAnalysisAndBugsForProject(ctx, opts)
					So(err, ShouldBeNil)

					So(len(f.Issues), ShouldEqual, 3)
					issue = f.Issues[2].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
					So(issue.Status.Status, ShouldEqual, monorail.VerifiedStatus)
				})
			})
		})
	})
}

func makeTestNameCluster(config *compiledcfg.ProjectConfig, uniqifier int) *analysis.ClusterSummary {
	testID := fmt.Sprintf("testname-%v", uniqifier)
	return &analysis.ClusterSummary{
		ClusterID:  testIDClusterID(config, testID),
		Failures1d: analysis.Counts{ResidualPreWeetbix: 9},
		Failures3d: analysis.Counts{ResidualPreWeetbix: 29},
		Failures7d: analysis.Counts{ResidualPreWeetbix: 69},
		TopTestIDs: []analysis.TopCount{{Value: testID, Count: 1}},
	}
}

func makeReasonCluster(config *compiledcfg.ProjectConfig, uniqifier int) *analysis.ClusterSummary {
	// Because the failure reason clustering algorithm removes numbers
	// when clustering failure reasons, it is better not to use the
	// uniqifier directly in the reason, to avoid cluster ID collisions.
	var foo strings.Builder
	for i := 0; i < uniqifier; i++ {
		foo.WriteString("foo")
	}
	reason := fmt.Sprintf("want %s, got bar", foo.String())

	return &analysis.ClusterSummary{
		ClusterID:  reasonClusterID(config, reason),
		Failures1d: analysis.Counts{ResidualPreWeetbix: 9},
		Failures3d: analysis.Counts{ResidualPreWeetbix: 29},
		Failures7d: analysis.Counts{ResidualPreWeetbix: 69},
		TopTestIDs: []analysis.TopCount{
			{Value: fmt.Sprintf("testname-a-%v", uniqifier), Count: 1},
			{Value: fmt.Sprintf("testname-b-%v", uniqifier), Count: 1},
		},
		ExampleFailureReason: bigquery.NullString{Valid: true, StringVal: reason},
	}
}

func makeBugCluster(ruleID string) *analysis.ClusterSummary {
	return &analysis.ClusterSummary{
		ClusterID:  bugClusterID(ruleID),
		Failures1d: analysis.Counts{ResidualPreWeetbix: 9},
		Failures3d: analysis.Counts{ResidualPreWeetbix: 29},
		Failures7d: analysis.Counts{ResidualPreWeetbix: 69},
		TopTestIDs: []analysis.TopCount{{Value: "testname-0", Count: 1}},
	}
}

func testIDClusterID(config *compiledcfg.ProjectConfig, testID string) clustering.ClusterID {
	testAlg, err := algorithms.SuggestingAlgorithm(testname.AlgorithmName)
	So(err, ShouldBeNil)

	return clustering.ClusterID{
		Algorithm: testname.AlgorithmName,
		ID: hex.EncodeToString(testAlg.Cluster(config, &clustering.Failure{
			TestID: testID,
		})),
	}
}

func reasonClusterID(config *compiledcfg.ProjectConfig, reason string) clustering.ClusterID {
	reasonAlg, err := algorithms.SuggestingAlgorithm(failurereason.AlgorithmName)
	So(err, ShouldBeNil)

	return clustering.ClusterID{
		Algorithm: failurereason.AlgorithmName,
		ID: hex.EncodeToString(reasonAlg.Cluster(config, &clustering.Failure{
			Reason: &pb.FailureReason{PrimaryErrorMessage: reason},
		})),
	}
}

func bugClusterID(ruleID string) clustering.ClusterID {
	return clustering.ClusterID{
		Algorithm: rulesalgorithm.AlgorithmName,
		ID:        ruleID,
	}
}

type fakeAnalysisClient struct {
	analysisBuilt bool
	clusters      []*analysis.ClusterSummary
}

func (f *fakeAnalysisClient) RebuildAnalysis(ctx context.Context, project string) error {
	f.analysisBuilt = true
	return nil
}

func (f *fakeAnalysisClient) ReadImpactfulClusters(ctx context.Context, opts analysis.ImpactfulClusterReadOptions) ([]*analysis.ClusterSummary, error) {
	if !f.analysisBuilt {
		return nil, errors.New("cluster_summaries does not exist")
	}
	var results []*analysis.ClusterSummary
	for _, c := range f.clusters {
		include := opts.AlwaysIncludeBugClusters && c.ClusterID.IsBugCluster()
		if opts.Thresholds.TestResultsFailed != nil {
			include = include ||
				exceedsThreshold(c.Failures1d.ResidualPreWeetbix, opts.Thresholds.TestResultsFailed.OneDay) ||
				exceedsThreshold(c.Failures3d.ResidualPreWeetbix, opts.Thresholds.TestResultsFailed.ThreeDay) ||
				exceedsThreshold(c.Failures7d.ResidualPreWeetbix, opts.Thresholds.TestResultsFailed.SevenDay)
		}
		if opts.Thresholds.TestRunsFailed != nil {
			include = include ||
				exceedsThreshold(c.TestRunFails1d.ResidualPreWeetbix, opts.Thresholds.TestRunsFailed.OneDay) ||
				exceedsThreshold(c.TestRunFails3d.ResidualPreWeetbix, opts.Thresholds.TestRunsFailed.ThreeDay) ||
				exceedsThreshold(c.TestRunFails7d.ResidualPreWeetbix, opts.Thresholds.TestRunsFailed.SevenDay)
		}
		if opts.Thresholds.PresubmitRunsFailed != nil {
			include = include ||
				exceedsThreshold(c.PresubmitRejects1d.ResidualPreWeetbix, opts.Thresholds.PresubmitRunsFailed.OneDay) ||
				exceedsThreshold(c.PresubmitRejects3d.ResidualPreWeetbix, opts.Thresholds.PresubmitRunsFailed.ThreeDay) ||
				exceedsThreshold(c.PresubmitRejects7d.ResidualPreWeetbix, opts.Thresholds.PresubmitRunsFailed.SevenDay)
		}
		if include {
			results = append(results, c)
		}
	}
	return results, nil
}

func exceedsThreshold(value int64, threshold *int64) bool {
	// threshold == nil is treated as an unsatisfiable threshold.
	return threshold != nil && value >= *threshold
}
