// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms"
	"infra/appengine/weetbix/internal/clustering/reclustering"
	"infra/appengine/weetbix/internal/clustering/rules"
	pb "infra/appengine/weetbix/proto/v1"
)

// MaxTestResults is the maximum number of test results to cluster in one
// call to Cluster(...).
const MaxTestResults = 1000

type clustersServer struct{}

func NewClustersServer() *pb.DecoratedClusters {
	return &pb.DecoratedClusters{
		Prelude:  checkAllowedPrelude,
		Service:  &clustersServer{},
		Postlude: gRPCifyAndLogPostlude,
	}
}

// Cluster clusters a list of test failures. See proto definition for more.
func (*clustersServer) Cluster(ctx context.Context, req *pb.ClusterRequest) (*pb.ClusterResponse, error) {
	if len(req.TestResults) > MaxTestResults {
		return nil, invalidArgumentError(fmt.Errorf(
			"too many test results: at most %v test results can be clustered in one request", MaxTestResults))
	}

	failures := make([]*clustering.Failure, 0, len(req.TestResults))
	for i, tr := range req.TestResults {
		if err := validateTestResult(i, tr); err != nil {
			return nil, err
		}
		failures = append(failures, &clustering.Failure{
			TestID: tr.TestId,
			Reason: tr.FailureReason,
		})
	}

	// Fetch a recent project configuration.
	// (May be a recent value that was cached.)
	cfg, err := readProjectConfig(ctx, req.Project)
	if err != nil {
		return nil, err
	}

	// Fetch a recent ruleset. (May be a recent value that was cached.)
	ruleset, err := reclustering.Ruleset(ctx, req.Project, rules.StartingEpoch)
	if err != nil {
		return nil, err
	}

	// Perform clustering from scratch. (Incremental clustering does not make
	// sense for this RPC.)
	existing := algorithms.NewEmptyClusterResults(len(req.TestResults))

	results := algorithms.Cluster(cfg, ruleset, existing, failures)

	// Construct the response proto.
	clusteredTRs := make([]*pb.ClusterResponse_ClusteredTestResult, 0, len(results.Clusters))
	for i, r := range results.Clusters {
		request := req.TestResults[i]

		entries := make([]*pb.ClusterResponse_ClusteredTestResult_ClusterEntry, 0, len(r))
		for _, clusterID := range r {
			entry := &pb.ClusterResponse_ClusteredTestResult_ClusterEntry{
				ClusterId: createClusterIdPB(clusterID),
			}
			if clusterID.IsBugCluster() {
				// For bug clusters, the ID of the cluster is also the ID of
				// the rule that defines it. Use this property to lookup the
				// associated rule.
				ruleID := clusterID.ID
				rule := ruleset.ActiveRulesByID[ruleID]
				entry.Bug = createAssociatedBugPB(rule.Rule.BugID, cfg.Config)
			}
			entries = append(entries, entry)
		}
		clusteredTR := &pb.ClusterResponse_ClusteredTestResult{
			RequestTag: request.RequestTag,
			Clusters:   entries,
		}
		clusteredTRs = append(clusteredTRs, clusteredTR)
	}

	version := &pb.ClusteringVersion{
		AlgorithmsVersion: results.AlgorithmsVersion,
		RulesVersion:      timestamppb.New(results.RulesVersion),
		ConfigVersion:     timestamppb.New(results.ConfigVersion),
	}

	return &pb.ClusterResponse{
		ClusteredTestResults: clusteredTRs,
		ClusteringVersion:    version,
	}, nil
}

func validateTestResult(i int, tr *pb.ClusterRequest_TestResult) error {
	if tr.TestId == "" {
		return invalidArgumentError(fmt.Errorf("test result %v: test ID must not be empty", i))
	}
	return nil
}
