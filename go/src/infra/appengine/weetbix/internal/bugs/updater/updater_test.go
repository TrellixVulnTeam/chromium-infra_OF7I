// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package updater

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

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
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"

	"cloud.google.com/go/bigquery"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/config/validation"
	"go.chromium.org/luci/server/span"
	"google.golang.org/protobuf/proto"
)

func TestRun(t *testing.T) {
	Convey("Run bug updates", t, func() {
		ctx := testutil.SpannerTestContext(t)

		f := &monorail.FakeIssuesStore{
			NextID:            100,
			PriorityFieldName: "projects/chromium/fieldDefs/11",
		}
		user := monorail.AutomationUsers[0]
		mc, err := monorail.NewClient(monorail.UseFakeIssuesClient(ctx, f, user), "myhost")
		So(err, ShouldBeNil)

		suggestedClusters := []*analysis.ClusterSummary{
			makeSuggestedCluster(0),
			makeSuggestedCluster(1),
			makeSuggestedCluster(2),
			makeSuggestedCluster(3),
		}
		ac := &fakeAnalysisClient{
			clusters: suggestedClusters,
		}

		project := "chromium"
		monorailCfg := monorail.ChromiumTestConfig()
		thres := &config.ImpactThreshold{
			// Should be more onerous than the "keep-open" thresholds
			// configured for each individual bug manager.
			UnexpectedFailures_1D: proto.Int64(100),
			UnexpectedFailures_3D: proto.Int64(300),
			UnexpectedFailures_7D: proto.Int64(700),
		}
		projectCfg := &config.ProjectConfig{
			Monorail:           monorailCfg,
			BugFilingThreshold: thres,
		}

		opts := updateOptions{
			project:            project,
			analysisClient:     ac,
			monorailClient:     mc,
			projectConfig:      projectCfg,
			maxBugsFiledPerRun: 1,
		}

		// Unless otherwise specified, assume re-clustering has caught up to
		// the latest version of algorithms.
		err = runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{
			runs.NewRun(0).
				WithProject(project).
				WithAlgorithmsVersion(algorithms.AlgorithmsVersion).
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
			sourceClusterID := reasonClusterID("Failed to connect to 100.1.1.99.")
			suggestedClusters[1].ClusterID = sourceClusterID
			suggestedClusters[1].ExampleFailureReason = bigquery.NullString{StringVal: "Failed to connect to 100.1.1.105.", Valid: true}

			ignoreRuleID := ""
			expectCreate := true

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

				expected := &rules.FailureAssociationRule{
					Project:         "chromium",
					RuleDefinition:  `reason LIKE "Failed to connect to %.%.%.%."`,
					Bug:             bugs.BugID{System: "monorail", ID: "chromium/100"},
					IsActive:        true,
					SourceCluster:   sourceClusterID,
					CreationUser:    rules.WeetbixSystem,
					LastUpdatedUser: rules.WeetbixSystem,
				}

				// Accept whatever bug cluster ID has been generated.
				So(rule.RuleID, ShouldNotBeEmpty)
				expected.RuleID = rule.RuleID

				// Accept creation and last updated times, as set by Spanner.
				So(rule.CreationTime, ShouldNotBeZeroValue)
				expected.CreationTime = rule.CreationTime
				So(rule.LastUpdated, ShouldNotBeZeroValue)
				expected.LastUpdated = rule.LastUpdated

				So(rule, ShouldResemble, expected)
				So(len(f.Issues), ShouldEqual, 1)
				So(f.Issues[0].Issue.Name, ShouldEqual, "projects/chromium/issues/100")
				So(f.Issues[0].Issue.Summary, ShouldContainSubstring, "Failed to connect to 100.1.1.105.")
			}

			Convey("1d unexpected failures", func() {
				suggestedClusters[1].Failures1d.Residual = 100
				test()

				// Further updates do nothing.
				test()
			})
			Convey("3d unexpected failures", func() {
				suggestedClusters[1].Failures3d.Residual = 300
				test()

				// Further updates do nothing.
				test()
			})
			Convey("7d unexpected failures", func() {
				suggestedClusters[1].Failures7d.Residual = 700
				test()

				// Further updates do nothing.
				test()
			})
			Convey("With existing rule filed", func() {
				suggestedClusters[1].Failures1d.Residual = 100

				createTime := time.Date(2021, time.January, 5, 12, 30, 0, 0, time.UTC)
				rule := rules.NewRule(0).
					WithProject(project).
					WithCreationTime(createTime).
					WithLastUpdated(createTime.Add(1 * time.Hour)).
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
						WithRulesVersion(createTime).
						WithCompletedProgress().Build(),
				})
				So(err, ShouldBeNil)

				expectCreate = true
				test()
			})
			Convey("Without re-clustering caught up to latest algorithms", func() {
				err = runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{
					runs.NewRun(0).
						WithProject(project).
						WithAlgorithmsVersion(algorithms.AlgorithmsVersion - 1).
						WithRulesVersion(rules.StartingEpoch).
						WithCompletedProgress().Build(),
				})
				So(err, ShouldBeNil)

				expectCreate = false
				test()
			})
		})
		Convey("With multiple suggested clusters above impact thresold", func() {
			expectBugClusters := func(count int) {
				bugClusters, err := rules.ReadActive(span.Single(ctx), project)
				So(err, ShouldBeNil)
				So(len(bugClusters), ShouldEqual, count)
				So(len(f.Issues), ShouldEqual, count)
			}
			suggestedClusters[1].Failures1d.Residual = 200
			suggestedClusters[2].Failures3d.Residual = 300
			suggestedClusters[3].Failures7d.Residual = 700

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

			expectFinalBugClusters := func() {
				// Check final set of bugs is as expected.
				rs, err := rules.ReadActive(span.Single(ctx), project)
				So(err, ShouldBeNil)
				for _, r := range rs {
					So(r.RuleID, ShouldNotBeEmpty)
					So(r.CreationTime, ShouldNotBeZeroValue)
					So(r.LastUpdated, ShouldNotBeZeroValue)
					// Accept whatever values the implementation has set.
					r.RuleID = ""
					r.CreationTime = time.Time{}
					r.LastUpdated = time.Time{}
				}

				So(rs, ShouldResemble, []*rules.FailureAssociationRule{
					{
						Project:         "chromium",
						RuleDefinition:  `test = "testname-1"`,
						Bug:             bugs.BugID{System: "monorail", ID: "chromium/100"},
						SourceCluster:   testIDClusterID("testname-1"),
						IsActive:        true,
						CreationUser:    rules.WeetbixSystem,
						LastUpdatedUser: rules.WeetbixSystem,
					},
					{
						Project:         "chromium",
						RuleDefinition:  `test = "testname-2"`,
						Bug:             bugs.BugID{System: "monorail", ID: "chromium/101"},
						SourceCluster:   testIDClusterID("testname-2"),
						IsActive:        true,
						CreationUser:    rules.WeetbixSystem,
						LastUpdatedUser: rules.WeetbixSystem,
					},
					{
						Project:         "chromium",
						RuleDefinition:  `test = "testname-3"`,
						Bug:             bugs.BugID{System: "monorail", ID: "chromium/102"},
						SourceCluster:   testIDClusterID("testname-3"),
						IsActive:        true,
						CreationUser:    rules.WeetbixSystem,
						LastUpdatedUser: rules.WeetbixSystem,
					},
				})
				So(len(f.Issues), ShouldEqual, 3)
			}
			expectFinalBugClusters()

			// Further updates do nothing.
			originalIssues := monorail.CopyIssuesStore(f)
			err = updateAnalysisAndBugsForProject(ctx, opts)
			So(err, ShouldBeNil)
			So(f, monorail.ShouldResembleIssuesStore, originalIssues)
			expectFinalBugClusters()

			rs, err := rules.ReadActive(span.Single(ctx), project)
			So(err, ShouldBeNil)

			bugClusters := []*analysis.ClusterSummary{
				makeBugCluster(rs[0].RuleID),
				makeBugCluster(rs[1].RuleID),
				makeBugCluster(rs[2].RuleID),
			}

			Convey("Re-clustering in progress", func() {
				ac.clusters = append(suggestedClusters, bugClusters...)

				Convey("Negligable cluster impact does not affect issue priority or status", func() {
					issue := f.Issues[0].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/100")
					originalPriority := monorail.ChromiumTestIssuePriority(issue)
					originalStatus := issue.Status.Status
					So(originalStatus, ShouldNotEqual, monorail.VerifiedStatus)

					bugs.SetResidualImpact(bugClusters[0], monorail.ChromiumClosureImpact())
					err = updateAnalysisAndBugsForProject(ctx, opts)
					So(err, ShouldBeNil)

					So(len(f.Issues), ShouldEqual, 3)
					issue = f.Issues[0].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/100")
					So(monorail.ChromiumTestIssuePriority(issue), ShouldEqual, originalPriority)
					So(issue.Status.Status, ShouldEqual, originalStatus)

					expectFinalBugClusters()
				})
			})
			Convey("Re-clustering complete", func() {
				ac.clusters = append(suggestedClusters, bugClusters...)

				// Copy impact from suggested clusters to new bug clusters.
				bugClusters[0].Failures1d = suggestedClusters[1].Failures1d
				bugClusters[1].Failures3d = suggestedClusters[2].Failures3d
				bugClusters[2].Failures7d = suggestedClusters[3].Failures7d

				// Clear residual impact on suggested clusters
				suggestedClusters[1].Failures1d.Residual = 0
				suggestedClusters[2].Failures3d.Residual = 0
				suggestedClusters[3].Failures7d.Residual = 0

				// Mark reclustering complete.
				err := runs.SetRunsForTesting(ctx, []*runs.ReclusteringRun{
					runs.NewRun(0).
						WithProject(project).
						WithAlgorithmsVersion(algorithms.AlgorithmsVersion).
						WithRulesVersion(rs[2].LastUpdated).
						WithCompletedProgress().Build(),
				})
				So(err, ShouldBeNil)

				Convey("Increasing cluster impact increases issue priority", func() {
					issue := f.Issues[2].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
					So(monorail.ChromiumTestIssuePriority(issue), ShouldNotEqual, "0")

					bugs.SetResidualImpact(bugClusters[2], monorail.ChromiumP0Impact())
					err = updateAnalysisAndBugsForProject(ctx, opts)
					So(err, ShouldBeNil)

					So(len(f.Issues), ShouldEqual, 3)
					issue = f.Issues[2].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/102")
					So(monorail.ChromiumTestIssuePriority(issue), ShouldEqual, "0")

					expectFinalBugClusters()
				})
				Convey("Decreasing cluster impact decreases issue priority", func() {
					issue := f.Issues[0].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/100")
					So(monorail.ChromiumTestIssuePriority(issue), ShouldNotEqual, "3")

					bugs.SetResidualImpact(bugClusters[0], monorail.ChromiumP3Impact())
					err = updateAnalysisAndBugsForProject(ctx, opts)
					So(err, ShouldBeNil)

					So(len(f.Issues), ShouldEqual, 3)
					issue = f.Issues[0].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/100")
					So(issue.Status.Status, ShouldEqual, monorail.UntriagedStatus)
					So(monorail.ChromiumTestIssuePriority(issue), ShouldEqual, "3")

					expectFinalBugClusters()
				})
				Convey("Deleting cluster closes issue", func() {
					issue := f.Issues[0].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/100")
					So(issue.Status.Status, ShouldEqual, monorail.UntriagedStatus)

					// Drop the bug cluster at index 0.
					bugClusters = bugClusters[1:]
					ac.clusters = append(suggestedClusters, bugClusters...)
					err = updateAnalysisAndBugsForProject(ctx, opts)
					So(err, ShouldBeNil)

					So(len(f.Issues), ShouldEqual, 3)
					issue = f.Issues[0].Issue
					So(issue.Name, ShouldEqual, "projects/chromium/issues/100")
					So(issue.Status.Status, ShouldEqual, monorail.VerifiedStatus)
				})
			})
		})
	})
}

func makeSuggestedCluster(uniqifier int) *analysis.ClusterSummary {
	testID := fmt.Sprintf("testname-%v", uniqifier)
	return &analysis.ClusterSummary{
		ClusterID:     testIDClusterID(testID),
		Failures1d:    analysis.Counts{Residual: 9},
		Failures3d:    analysis.Counts{Residual: 29},
		Failures7d:    analysis.Counts{Residual: 69},
		ExampleTestID: testID,
	}
}

func makeBugCluster(ruleID string) *analysis.ClusterSummary {
	return &analysis.ClusterSummary{
		ClusterID:     bugClusterID(ruleID),
		Failures1d:    analysis.Counts{Residual: 9},
		Failures3d:    analysis.Counts{Residual: 29},
		Failures7d:    analysis.Counts{Residual: 69},
		ExampleTestID: "testname-0",
	}
}

func testIDClusterID(testID string) clustering.ClusterID {
	testAlg, err := algorithms.SuggestingAlgorithm(testname.AlgorithmName)
	So(err, ShouldBeNil)

	return clustering.ClusterID{
		Algorithm: testname.AlgorithmName,
		ID: hex.EncodeToString(testAlg.Cluster(&clustering.Failure{
			TestID: testID,
		})),
	}
}

func reasonClusterID(reason string) clustering.ClusterID {
	reasonAlg, err := algorithms.SuggestingAlgorithm(failurereason.AlgorithmName)
	So(err, ShouldBeNil)

	return clustering.ClusterID{
		Algorithm: failurereason.AlgorithmName,
		ID: hex.EncodeToString(reasonAlg.Cluster(&clustering.Failure{
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
		include := containsValue(opts.AlwaysInclude, c.ClusterID) ||
			(opts.Thresholds.UnexpectedFailures_1D != nil && int64(c.Failures1d.Residual) >= *opts.Thresholds.UnexpectedFailures_1D) ||
			(opts.Thresholds.UnexpectedFailures_3D != nil && int64(c.Failures3d.Residual) >= *opts.Thresholds.UnexpectedFailures_3D) ||
			(opts.Thresholds.UnexpectedFailures_7D != nil && int64(c.Failures7d.Residual) >= *opts.Thresholds.UnexpectedFailures_7D)
		if include {
			results = append(results, c)
		}
	}
	return results, nil
}

func containsValue(values []clustering.ClusterID, value clustering.ClusterID) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
