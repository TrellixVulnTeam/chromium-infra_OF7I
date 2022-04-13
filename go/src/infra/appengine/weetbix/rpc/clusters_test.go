// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"encoding/hex"
	"sort"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/auth"
	"go.chromium.org/luci/server/auth/authtest"
	"go.chromium.org/luci/server/caching"
	"go.chromium.org/luci/server/secrets"
	"go.chromium.org/luci/server/secrets/testsecrets"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/bugs"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms"
	"infra/appengine/weetbix/internal/clustering/algorithms/failurereason"
	"infra/appengine/weetbix/internal/clustering/algorithms/testname"
	"infra/appengine/weetbix/internal/clustering/rules"
	"infra/appengine/weetbix/internal/config"
	"infra/appengine/weetbix/internal/config/compiledcfg"
	configpb "infra/appengine/weetbix/internal/config/proto"
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"
)

func TestClusters(t *testing.T) {
	Convey("With a clusters server", t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx = caching.WithEmptyProcessCache(ctx)

		// For user identification.
		ctx = authtest.MockAuthConfig(ctx)
		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity: "user:someone@example.com",
		})
		ctx = secrets.Use(ctx, &testsecrets.Store{})

		// Provides datastore implementation needed for project config.
		ctx = memory.Use(ctx)
		server := &clustersServer{}

		configVersion := time.Date(2025, time.August, 12, 0, 1, 2, 3, time.UTC)
		projectChromium := config.CreatePlaceholderProjectConfig()
		projectChromium.LastUpdated = timestamppb.New(configVersion)
		projectChromium.Monorail.DisplayPrefix = "crbug.com"
		projectChromium.Monorail.MonorailHostname = "bugs.chromium.org"

		configs := make(map[string]*configpb.ProjectConfig)
		configs["chromium"] = projectChromium
		err := config.SetTestProjectConfig(ctx, configs)
		So(err, ShouldBeNil)

		compiledChromiumCfg, err := compiledcfg.NewConfig(projectChromium)
		So(err, ShouldBeNil)

		// Rules version is in microsecond granularity, consistent with
		// the granularity of Spanner commit timestamps.
		rulesVersion := time.Date(2021, time.February, 12, 1, 2, 4, 5000, time.UTC)
		rs := []*rules.FailureAssociationRule{
			rules.NewRule(0).
				WithProject("chromium").
				WithRuleDefinition(`test LIKE "%TestSuite.TestName%"`).
				WithPredicateLastUpdated(rulesVersion.Add(-1 * time.Hour)).
				WithBug(bugs.BugID{
					System: "monorail",
					ID:     "chromium/7654321",
				}).Build(),
			rules.NewRule(1).
				WithProject("chromium").
				WithRuleDefinition(`reason LIKE "my_file.cc(%): Check failed: false."`).
				WithPredicateLastUpdated(rulesVersion).
				WithBug(bugs.BugID{
					System: "buganizer",
					ID:     "82828282",
				}).Build(),
			rules.NewRule(2).
				WithProject("chromium").
				WithRuleDefinition(`test LIKE "%Other%"`).
				WithPredicateLastUpdated(rulesVersion.Add(-2 * time.Hour)).
				WithBug(bugs.BugID{
					System: "monorail",
					ID:     "chromium/912345",
				}).Build(),
		}
		err = rules.SetRulesForTesting(ctx, rs)
		So(err, ShouldBeNil)

		Convey("Call Cluster", func() {
			request := &pb.ClusterRequest{
				Project: "chromium",
				TestResults: []*pb.ClusterRequest_TestResult{
					{
						RequestTag: "my tag 1",
						TestId:     "ninja://chrome/test:interactive_ui_tests/TestSuite.TestName",
						FailureReason: &pb.FailureReason{
							PrimaryErrorMessage: "my_file.cc(123): Check failed: false.",
						},
					},
					{
						RequestTag: "my tag 2",
						TestId:     "Other_test",
					},
				},
			}

			Convey("With a valid request", func() {
				// Run
				response, err := server.Cluster(ctx, request)

				// Verify
				So(err, ShouldBeNil)
				So(response, ShouldResembleProto, &pb.ClusterResponse{
					ClusteredTestResults: []*pb.ClusterResponse_ClusteredTestResult{
						{
							RequestTag: "my tag 1",
							Clusters: sortClusterEntries([]*pb.ClusterResponse_ClusteredTestResult_ClusterEntry{
								{
									ClusterId: &pb.ClusterId{
										Algorithm: "rules",
										Id:        rs[0].RuleID,
									},
									Bug: &pb.AssociatedBug{
										System:   "monorail",
										Id:       "chromium/7654321",
										LinkText: "crbug.com/7654321",
										Url:      "https://bugs.chromium.org/p/chromium/issues/detail?id=7654321",
									},
								}, {
									ClusterId: &pb.ClusterId{
										Algorithm: "rules",
										Id:        rs[1].RuleID,
									},
									Bug: &pb.AssociatedBug{
										System:   "buganizer",
										Id:       "82828282",
										LinkText: "b/82828282",
										Url:      "https://issuetracker.google.com/issues/82828282",
									},
								},
								failureReasonClusterEntry(compiledChromiumCfg, "my_file.cc(123): Check failed: false."),
								testNameClusterEntry(compiledChromiumCfg, "ninja://chrome/test:interactive_ui_tests/TestSuite.TestName"),
							}),
						},
						{
							RequestTag: "my tag 2",
							Clusters: sortClusterEntries([]*pb.ClusterResponse_ClusteredTestResult_ClusterEntry{
								{
									ClusterId: &pb.ClusterId{
										Algorithm: "rules",
										Id:        rs[2].RuleID,
									},
									Bug: &pb.AssociatedBug{
										System:   "monorail",
										Id:       "chromium/912345",
										LinkText: "crbug.com/912345",
										Url:      "https://bugs.chromium.org/p/chromium/issues/detail?id=912345",
									},
								},
								testNameClusterEntry(compiledChromiumCfg, "Other_test"),
							}),
						},
					},
					ClusteringVersion: &pb.ClusteringVersion{
						AlgorithmsVersion: algorithms.AlgorithmsVersion,
						RulesVersion:      timestamppb.New(rulesVersion),
						ConfigVersion:     timestamppb.New(configVersion),
					},
				})
			})
			Convey("With missing test ID", func() {
				request.TestResults[1].TestId = ""

				// Run
				response, err := server.Cluster(ctx, request)

				// Verify
				So(err, ShouldErrLike, "test result 1: test ID must not be empty")
				So(response, ShouldBeNil)
			})
			Convey("With too many test results", func() {
				var testResults []*pb.ClusterRequest_TestResult
				for i := 0; i < 1001; i++ {
					testResults = append(testResults, &pb.ClusterRequest_TestResult{
						TestId: "AnotherTest",
					})
				}
				request.TestResults = testResults

				// Run
				response, err := server.Cluster(ctx, request)

				// Verify
				So(err, ShouldErrLike, "too many test results: at most 1000 test results can be clustered in one request")
				So(response, ShouldBeNil)
			})
		})
	})
}

func failureReasonClusterEntry(projectcfg *compiledcfg.ProjectConfig, primaryErrorMessage string) *pb.ClusterResponse_ClusteredTestResult_ClusterEntry {
	alg := &failurereason.Algorithm{}
	clusterID := alg.Cluster(projectcfg, &clustering.Failure{
		Reason: &pb.FailureReason{
			PrimaryErrorMessage: primaryErrorMessage,
		},
	})
	return &pb.ClusterResponse_ClusteredTestResult_ClusterEntry{
		ClusterId: &pb.ClusterId{
			Algorithm: failurereason.AlgorithmName,
			Id:        hex.EncodeToString(clusterID),
		},
	}
}

func testNameClusterEntry(projectcfg *compiledcfg.ProjectConfig, testID string) *pb.ClusterResponse_ClusteredTestResult_ClusterEntry {
	alg := &testname.Algorithm{}
	clusterID := alg.Cluster(projectcfg, &clustering.Failure{
		TestID: testID,
	})
	return &pb.ClusterResponse_ClusteredTestResult_ClusterEntry{
		ClusterId: &pb.ClusterId{
			Algorithm: testname.AlgorithmName,
			Id:        hex.EncodeToString(clusterID),
		},
	}
}

// sortClusterEntries sorts clusters by ascending Cluster ID.
func sortClusterEntries(entries []*pb.ClusterResponse_ClusteredTestResult_ClusterEntry) []*pb.ClusterResponse_ClusteredTestResult_ClusterEntry {
	result := make([]*pb.ClusterResponse_ClusteredTestResult_ClusterEntry, len(entries))
	copy(result, entries)
	sort.Slice(result, func(i, j int) bool {
		if result[i].ClusterId.Algorithm != result[j].ClusterId.Algorithm {
			return result[i].ClusterId.Algorithm < result[j].ClusterId.Algorithm
		}
		return result[i].ClusterId.Id < result[j].ClusterId.Id
	})
	return result
}
