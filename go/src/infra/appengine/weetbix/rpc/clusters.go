// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package rpc

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/grpc/appstatus"
	"go.chromium.org/luci/server/span"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/analysis"
	"infra/appengine/weetbix/internal/clustering"
	"infra/appengine/weetbix/internal/clustering/algorithms"
	"infra/appengine/weetbix/internal/clustering/reclustering"
	"infra/appengine/weetbix/internal/clustering/rules/cache"
	"infra/appengine/weetbix/internal/clustering/runs"
	pb "infra/appengine/weetbix/proto/v1"
)

// MaxClusterRequestSize is the maximum number of test results to cluster in
// one call to Cluster(...).
const MaxClusterRequestSize = 1000

// MaxPresubmitImpactRequestSize is the maximum number of clusters to obtain
// impact for in one call to BatchGetPresubmitImpact().
const MaxPresubmitImpactRequestSize = 1000

type AnalysisClient interface {
	ReadClusterPresubmitImpact(ctx context.Context, project string, clusterIDs []clustering.ClusterID) ([]analysis.ClusterPresubmitImpact, error)
}

type clustersServer struct {
	analysisClient AnalysisClient
}

func NewClustersServer(analysisClient AnalysisClient) *pb.DecoratedClusters {
	return &pb.DecoratedClusters{
		Prelude:  checkAllowedPrelude,
		Service:  &clustersServer{analysisClient: analysisClient},
		Postlude: gRPCifyAndLogPostlude,
	}
}

// Cluster clusters a list of test failures. See proto definition for more.
func (*clustersServer) Cluster(ctx context.Context, req *pb.ClusterRequest) (*pb.ClusterResponse, error) {
	if len(req.TestResults) > MaxClusterRequestSize {
		return nil, invalidArgumentError(fmt.Errorf(
			"too many test results: at most %v test results can be clustered in one request", MaxClusterRequestSize))
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

	// Fetch a recent ruleset.
	ruleset, err := reclustering.Ruleset(ctx, req.Project, cache.StrongRead)
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

func (c *clustersServer) BatchGetPresubmitImpact(ctx context.Context, req *pb.BatchGetClusterPresubmitImpactRequest) (*pb.BatchGetClusterPresubmitImpactResponse, error) {
	project, err := parseProjectName(req.Parent)
	if err != nil {
		return nil, invalidArgumentError(errors.Annotate(err, "parent").Err())
	}

	if len(req.Names) > MaxPresubmitImpactRequestSize {
		return nil, invalidArgumentError(fmt.Errorf(
			"too many names: presubmit impact for at most %v clusters can be retrieved in one request", MaxPresubmitImpactRequestSize))
	}
	if len(req.Names) == 0 {
		// Return INVALID_ARGUMENT if no names specified, as per google.aip.dev/231.
		return nil, invalidArgumentError(errors.New("names must be specified"))
	}

	// Fetch the latest ruleset.
	ruleset, err := reclustering.Ruleset(ctx, project, cache.StrongRead)
	if err != nil {
		return nil, err
	}

	run, err := runs.ReadLastComplete(span.Single(ctx), project)
	if err != nil {
		return nil, err
	}

	// The cluster ID to read for each request item. If no cluster ID
	// should be read, the entry is an empty cluster ID.
	clusterIDs := make([]clustering.ClusterID, 0, len(req.Names))

	for i, name := range req.Names {
		clusterProject, clusterID, err := parseClusterPresubmitImpactName(name)
		if err != nil {
			return nil, invalidArgumentError(errors.Annotate(err, "name %v", i).Err())
		}
		if clusterProject != project {
			return nil, invalidArgumentError(fmt.Errorf("name %v: project must match parent project (%q)", i, project))
		}
		if clusterID.IsBugCluster() {
			rule, ok := ruleset.ActiveRulesByID[clusterID.ID]
			if ok {
				if run.RulesVersion.Before(rule.Rule.CreationTime) &&
					!rule.Rule.SourceCluster.IsEmpty() {
					// Re-clustering has not yet caught up to this new rule, and
					// there is a source cluster.
					// Use impact from the source cluster instead.
					clusterID = rule.Rule.SourceCluster
				}
				// Else: read the bug cluster.
			} else {
				// Rule never existed, or is inactive.
				// Read no cluster.
				clusterID = clustering.ClusterID{}
			}
		}
		clusterIDs = append(clusterIDs, clusterID)
	}

	clusterIDsToRead := make([]clustering.ClusterID, 0, len(clusterIDs))
	for _, clusterID := range clusterIDs {
		if !clusterID.IsEmpty() {
			clusterIDsToRead = append(clusterIDsToRead, clusterID)
		}
	}

	presubmitImpact, err := c.analysisClient.ReadClusterPresubmitImpact(ctx, project, clusterIDsToRead)
	if err != nil {
		if err == analysis.ProjectNotExistsErr {
			return nil, appstatus.Error(codes.NotFound,
				"project does not exist in Weetbix or cluster analysis is not yet available")
		}
		return nil, err
	}

	presubmitImpactByCluster := make(map[string]analysis.ClusterPresubmitImpact)
	for _, pi := range presubmitImpact {
		presubmitImpactByCluster[pi.ClusterID.Key()] = pi
	}

	// As per google.aip.dev/231, the order of responses must be the
	// same as the names in the request.
	results := make([]*pb.ClusterPresubmitImpact, 0, len(clusterIDs))
	for i, clusterID := range clusterIDs {
		name := req.Names[i]

		impact, ok := presubmitImpactByCluster[clusterID.Key()]
		if !ok {
			impact = analysis.ClusterPresubmitImpact{
				// No impact available for cluster (e.g. because no examples
				// in BigQuery). Use suitable default values.
				DistinctUserClTestRunsFailed12h: 0,
				DistinctUserClTestRunsFailed1d:  0,
			}
		}

		results = append(results, &pb.ClusterPresubmitImpact{
			Name:                         name,
			DistinctClTestRunsFailed_12H: impact.DistinctUserClTestRunsFailed12h,
			DistinctClTestRunsFailed_24H: impact.DistinctUserClTestRunsFailed1d,
		})
	}
	return &pb.BatchGetClusterPresubmitImpactResponse{
		PresubmitImpact: results,
	}, nil
}
