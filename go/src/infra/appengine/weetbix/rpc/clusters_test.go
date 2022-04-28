// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"context"
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
	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/bugs"
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

func TestClusters(t *testing.T) {
	Convey("With a clusters server", t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx = caching.WithEmptyProcessCache(ctx)

		// For user identification.
		ctx = authtest.MockAuthConfig(ctx)
		ctx = auth.WithState(ctx, &authtest.FakeState{
			Identity:       "user:someone@example.com",
			IdentityGroups: []string{"weetbix-access"},
		})
		ctx = secrets.Use(ctx, &testsecrets.Store{})

		// Provides datastore implementation needed for project config.
		ctx = memory.Use(ctx)
		analysisClient := newFakeAnalysisClient()
		server := NewClustersServer(analysisClient)

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

		Convey("Unauthorised requests are rejected", func() {
			// Ensure no access to weetbix-access.
			ctx = auth.WithState(ctx, &authtest.FakeState{
				Identity: "user:someone@example.com",
				// Not a member of weetbix-access.
				IdentityGroups: []string{"other-group"},
			})

			// Make some request (the request should not matter, as
			// a common decorator is used for all requests.)
			request := &pb.ClusterRequest{
				Project: "chromium",
			}

			rule, err := server.Cluster(ctx, request)
			st, _ := grpcStatus.FromError(err)
			So(st.Code(), ShouldEqual, codes.PermissionDenied)
			So(st.Message(), ShouldEqual, "not a member of weetbix-access")
			So(rule, ShouldBeNil)
		})
		Convey("Cluster", func() {
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
				st, _ := grpcStatus.FromError(err)
				So(st.Code(), ShouldEqual, codes.InvalidArgument)
				So(st.Message(), ShouldEqual, "test result 1: test ID must not be empty")
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
				st, _ := grpcStatus.FromError(err)
				So(st.Code(), ShouldEqual, codes.InvalidArgument)
				So(st.Message(), ShouldEqual, "too many test results: at most 1000 test results can be clustered in one request")
				So(response, ShouldBeNil)
			})
			Convey("With project not configured", func() {
				request.Project = "not-exists"

				// Run
				response, err := server.Cluster(ctx, request)

				// Verify
				st, _ := grpcStatus.FromError(err)
				So(st.Code(), ShouldEqual, codes.FailedPrecondition)
				So(st.Message(), ShouldEqual, "project does not exist in Weetbix")
				So(response, ShouldBeNil)
			})
		})
		Convey("BatchGetPresubmitImpact", func() {
			// The minimum rules version incorporated in all reclustering.
			rulesVersionCompleted := time.Date(2020, time.April, 1, 2, 3, 4, 5, time.UTC)

			rs := []*rules.FailureAssociationRule{
				rules.NewRule(0).
					WithProject("testproject").
					WithRuleID("11111100000000000000000000000000").
					WithSourceCluster(clustering.ClusterID{
						Algorithm: "reason-v1",
						ID:        "cccccc00000000000000000000000000",
					}).
					WithCreationTime(rulesVersionCompleted).
					WithPredicateLastUpdated(rulesVersionCompleted.Add(time.Hour)).
					Build(),
				rules.NewRule(1).
					WithProject("testproject").
					WithRuleID("11111100000000000000000000000001").
					WithSourceCluster(clustering.ClusterID{
						Algorithm: "reason-v1",
						ID:        "cccccc00000000000000000000000001",
					}).
					WithCreationTime(rulesVersionCompleted.Add(time.Second)).
					WithPredicateLastUpdated(rulesVersionCompleted.Add(time.Second)).
					Build(),
				rules.NewRule(2).
					WithProject("testproject").
					WithRuleID("11111100000000000000000000000002").
					WithSourceCluster(clustering.ClusterID{}).
					WithCreationTime(rulesVersionCompleted.Add(time.Hour)).
					WithPredicateLastUpdated(rulesVersionCompleted.Add(time.Hour)).
					Build(),
				rules.NewRule(3).
					WithProject("testproject").
					WithRuleID("11111100000000000000000000000003").
					WithCreationTime(rulesVersionCompleted.Add(-1 * time.Hour)).
					WithActive(false).
					Build(),
			}
			err := rules.SetRulesForTesting(ctx, rs)
			So(err, ShouldBeNil)

			rns := []*runs.ReclusteringRun{
				runs.NewRun(0).
					WithProject("testproject").
					WithRulesVersion(rulesVersionCompleted).
					WithCompletedProgress().
					Build(),
			}
			err = runs.SetRunsForTesting(ctx, rns)
			So(err, ShouldBeNil)

			analysisClient.clustersByProject["testproject"] = []analysis.ClusterPresubmitImpact{
				{
					ClusterID: clustering.ClusterID{
						Algorithm: rulesalgorithm.AlgorithmName,
						ID:        "11111100000000000000000000000000",
					},
					DistinctUserClTestRunsFailed12h: 1,
					DistinctUserClTestRunsFailed1d:  2,
				},
				{
					ClusterID: clustering.ClusterID{
						Algorithm: "reason-v1",
						ID:        "cccccc00000000000000000000000001",
					},
					DistinctUserClTestRunsFailed12h: 3,
					DistinctUserClTestRunsFailed1d:  4,
				},
				{
					ClusterID: clustering.ClusterID{
						Algorithm: rulesalgorithm.AlgorithmName,
						ID:        "11111100000000000000000000000002",
					},
					DistinctUserClTestRunsFailed12h: 5,
					DistinctUserClTestRunsFailed1d:  6,
				},
				{
					ClusterID: clustering.ClusterID{
						Algorithm: rulesalgorithm.AlgorithmName,
						ID:        "11111100000000000000000000000003",
					},
					DistinctUserClTestRunsFailed12h: 7,
					DistinctUserClTestRunsFailed1d:  8,
				},
			}

			request := &pb.BatchGetClusterPresubmitImpactRequest{
				Parent: "projects/testproject",
				Names: []string{
					// One bug cluster, initial (re-)clustering complete.
					// Should use impact calculated for bug cluster.
					"projects/testproject/clusters/rules/11111100000000000000000000000000/presubmitImpact",

					// One bug cluster, initial (re-)clustering not yet complete.
					// Should use impact calculated for source cluster.
					"projects/testproject/clusters/rules/11111100000000000000000000000001/presubmitImpact",

					// Suggested cluster.
					"projects/testproject/clusters/reason-v1/cccccc00000000000000000000000001/presubmitImpact",

					// One bug cluster, initial (re-)clustering complete,
					// but no source cluster. Should use impact for bug
					// cluster.
					"projects/testproject/clusters/rules/11111100000000000000000000000002/presubmitImpact",

					// One bug cluster, initial (re-)clustering complete,
					// but now inactive. Should have zero impact.
					"projects/testproject/clusters/rules/11111100000000000000000000000003/presubmitImpact",

					// Cluster for which no impact data exists.
					"projects/testproject/clusters/reason-v1/cccccc0000000000000000000000ffff/presubmitImpact",
				},
			}

			expectedResponse := &pb.BatchGetClusterPresubmitImpactResponse{
				PresubmitImpact: []*pb.ClusterPresubmitImpact{
					{
						Name:                         "projects/testproject/clusters/rules/11111100000000000000000000000000/presubmitImpact",
						DistinctClTestRunsFailed_12H: 1,
						DistinctClTestRunsFailed_24H: 2,
					},
					{
						Name:                         "projects/testproject/clusters/rules/11111100000000000000000000000001/presubmitImpact",
						DistinctClTestRunsFailed_12H: 3,
						DistinctClTestRunsFailed_24H: 4,
					},
					{
						Name:                         "projects/testproject/clusters/reason-v1/cccccc00000000000000000000000001/presubmitImpact",
						DistinctClTestRunsFailed_12H: 3,
						DistinctClTestRunsFailed_24H: 4,
					},
					{
						Name:                         "projects/testproject/clusters/rules/11111100000000000000000000000002/presubmitImpact",
						DistinctClTestRunsFailed_12H: 5,
						DistinctClTestRunsFailed_24H: 6,
					},
					{
						Name:                         "projects/testproject/clusters/rules/11111100000000000000000000000003/presubmitImpact",
						DistinctClTestRunsFailed_12H: 0,
						DistinctClTestRunsFailed_24H: 0,
					},
					{
						Name:                         "projects/testproject/clusters/reason-v1/cccccc0000000000000000000000ffff/presubmitImpact",
						DistinctClTestRunsFailed_12H: 0,
						DistinctClTestRunsFailed_24H: 0,
					},
				},
			}

			Convey("With a valid request", func() {
				Convey("No duplciate requests", func() {
					// Run
					response, err := server.BatchGetPresubmitImpact(ctx, request)

					// Verify
					So(err, ShouldBeNil)
					So(response, ShouldResembleProto, expectedResponse)
				})
				Convey("Duplicate requests", func() {
					// Even if request items are duplicated, the request
					// should still succeed and return correct results.
					request.Names = append(request.Names, request.Names...)
					expectedResponse.PresubmitImpact = append(expectedResponse.PresubmitImpact, expectedResponse.PresubmitImpact...)

					// Run
					response, err := server.BatchGetPresubmitImpact(ctx, request)

					// Verify
					So(err, ShouldBeNil)
					So(response, ShouldResembleProto, expectedResponse)
				})
			})
			Convey("With invalid request", func() {
				Convey("Invalid parent", func() {
					request.Parent = "blah"

					// Run
					response, err := server.BatchGetPresubmitImpact(ctx, request)

					// Verify
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual, "parent: invalid project name, expected format: projects/{project}")
					So(response, ShouldBeNil)
				})
				Convey("No names specified", func() {
					request.Names = []string{}

					// Run
					response, err := server.BatchGetPresubmitImpact(ctx, request)

					// Verify
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual, "names must be specified")
					So(response, ShouldBeNil)
				})
				Convey("Parent does not match request items", func() {
					// Request asks for project "blah" but parent asks for
					// project "testproject".
					So(request.Parent, ShouldEqual, "projects/testproject")
					request.Names[1] = "projects/blah/clusters/reason-v1/cccccc00000000000000000000000001/presubmitImpact"

					// Run
					response, err := server.BatchGetPresubmitImpact(ctx, request)

					// Verify
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual, `name 1: project must match parent project ("testproject")`)
					So(response, ShouldBeNil)
				})
				Convey("Invalid name", func() {
					request.Names[1] = "invalid"

					// Run
					response, err := server.BatchGetPresubmitImpact(ctx, request)

					// Verify
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual, "name 1: invalid cluster presubmit impact name, expected format: projects/{project}/clusters/{cluster_alg}/{cluster_id}/presubmitImpact")
					So(response, ShouldBeNil)
				})
				Convey("Invalid cluster algorithm in name", func() {
					request.Names[1] = "projects/blah/clusters/reason/cccccc00000000000000000000000001/presubmitImpact"

					// Run
					response, err := server.BatchGetPresubmitImpact(ctx, request)

					// Verify
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual, "name 1: invalid cluster presubmit impact name: algorithm not valid")
					So(response, ShouldBeNil)
				})
				Convey("Invalid cluster ID in name", func() {
					request.Names[1] = "projects/blah/clusters/reason-v1/123/presubmitImpact"

					// Run
					response, err := server.BatchGetPresubmitImpact(ctx, request)

					// Verify
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual, "name 1: invalid cluster presubmit impact name: ID is not valid lowercase hexadecimal bytes")
					So(response, ShouldBeNil)
				})
				Convey("Too many request items", func() {
					var names []string
					for i := 0; i < 1001; i++ {
						names = append(names, "projects/testproject/clusters/rules/11111100000000000000000000000000/presubmitImpact")
					}
					request.Names = names

					// Run
					response, err := server.BatchGetPresubmitImpact(ctx, request)

					// Verify
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.InvalidArgument)
					So(st.Message(), ShouldEqual, "too many names: presubmit impact for at most 1000 clusters can be retrieved in one request")
					So(response, ShouldBeNil)
				})
				Convey("Dataset does not exist", func() {
					delete(analysisClient.clustersByProject, "testproject")

					// Run
					response, err := server.BatchGetPresubmitImpact(ctx, request)

					// Verify
					st, _ := grpcStatus.FromError(err)
					So(st.Code(), ShouldEqual, codes.NotFound)
					So(st.Message(), ShouldEqual, "project does not exist in Weetbix or cluster analysis is not yet available")
					So(response, ShouldBeNil)
				})
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

type fakeAnalysisClient struct {
	clustersByProject map[string][]analysis.ClusterPresubmitImpact
}

func newFakeAnalysisClient() *fakeAnalysisClient {
	return &fakeAnalysisClient{
		clustersByProject: make(map[string][]analysis.ClusterPresubmitImpact),
	}
}

func (f *fakeAnalysisClient) ReadClusterPresubmitImpact(ctx context.Context, project string, clusterIDs []clustering.ClusterID) ([]analysis.ClusterPresubmitImpact, error) {
	clusters, ok := f.clustersByProject[project]
	if !ok {
		return nil, analysis.ProjectNotExistsErr
	}

	var results []analysis.ClusterPresubmitImpact
	for _, c := range clusters {
		include := false
		for _, ci := range clusterIDs {
			if ci == c.ClusterID {
				include = true
			}
		}
		if include {
			results = append(results, c)
		}
	}
	return results, nil
}
