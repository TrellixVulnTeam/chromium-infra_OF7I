// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// package server implements the server to handle pRPC requests.
package server

import (
	"context"
	gfim "infra/appengine/gofindit/model"
	gfipb "infra/appengine/gofindit/proto"

	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GoFinditServer implements the proto service GoFinditService.
type GoFinditServer struct{}

// GetAnalysis returns the analysis given the analysis id
func (server *GoFinditServer) GetAnalysis(c context.Context, req *gfipb.GetAnalysisRequest) (*gfipb.Analysis, error) {
	analysis := &gfim.CompileFailureAnalysis{
		Id: req.AnalysisId,
	}
	switch err := datastore.Get(c, analysis); err {
	case nil:
		//continue
	case datastore.ErrNoSuchEntity:
		return nil, status.Errorf(codes.NotFound, "Analysis %d not found: %v", req.AnalysisId, err)
	default:
		return nil, status.Errorf(codes.Internal, "Error in retrieving analysis: %s", err)
	}
	result, err := GetAnalysisResult(c, analysis)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error getting analysis result: %s", err)
	}
	return result, nil
}

// QueryAnalysis returns the analysis given a query
func (server *GoFinditServer) QueryAnalysis(c context.Context, req *gfipb.QueryAnalysisRequest) (*gfipb.QueryAnalysisResponse, error) {
	// TODO(nqmtuan): Implement this
	return nil, nil
}

// TriggerAnalysis triggers an analysis for a failure
func (server *GoFinditServer) TriggerAnalysis(c context.Context, req *gfipb.TriggerAnalysisRequest) (*gfipb.TriggerAnalysisResponse, error) {
	// TODO(nqmtuan): Implement this
	return nil, nil
}

// UpdateAnalysis updates the information of an analysis.
// At the mean time, it is only used for update the bugs associated with an
// analysis.
func (server *GoFinditServer) UpdateAnalysis(c context.Context, req *gfipb.UpdateAnalysisRequest) (*gfipb.Analysis, error) {
	// TODO(nqmtuan): Implement this
	return nil, nil
}

// GetAnalysisResult returns an analysis for pRPC from CompileFailureAnalysis
func GetAnalysisResult(c context.Context, analysis *gfim.CompileFailureAnalysis) (*gfipb.Analysis, error) {
	result := &gfipb.Analysis{
		AnalysisId:      analysis.Id,
		Status:          analysis.Status,
		CreatedTime:     timestamppb.New(analysis.CreateTime),
		EndTime:         timestamppb.New(analysis.EndTime),
		FirstFailedBbid: analysis.FirstFailedBuildId,
		LastPassedBbid:  analysis.LastPassedBuildId,
	}

	// Gets heuristic analysis results.
	q := datastore.NewQuery("CompileHeuristicAnalysis").Eq("parent", datastore.KeyForObj(c, analysis))
	heuristicAnalyses := []*gfim.CompileHeuristicAnalysis{}
	err := datastore.GetAll(c, q, &heuristicAnalyses)

	if err != nil {
		return nil, err
	}

	if len(heuristicAnalyses) == 0 {
		// No heuristic analysis, just return
		return result, nil
	}

	if len(heuristicAnalyses) > 1 {
		logging.Warningf(c, "Found multiple heuristic analysis for analysis %d", analysis.Id)
	}
	heuristicAnalysis := heuristicAnalyses[0]

	// Getting the suspects for heuristic analysis
	suspects := []*gfim.Suspect{}
	q = datastore.NewQuery("Suspect").Eq("parent", datastore.KeyForObj(c, heuristicAnalysis)).Order("-score")
	err = datastore.GetAll(c, q, &suspects)
	if err != nil {
		return nil, err
	}
	pbSuspects := make([]*gfipb.HeuristicSuspect, len(suspects))
	for i, suspect := range suspects {
		pbSuspects[i] = &gfipb.HeuristicSuspect{
			GitilesCommit: &suspect.GitilesCommit,
			ReviewUrl:     suspect.ReviewUrl,
			Score:         int32(suspect.Score),
			Justification: suspect.Justification,
		}
	}
	heuristicResult := &gfipb.HeuristicAnalysisResult{
		Status:    heuristicAnalysis.Status,
		StartTime: timestamppb.New(heuristicAnalysis.StartTime),
		EndTime:   timestamppb.New(heuristicAnalysis.EndTime),
		Suspects:  pbSuspects,
	}

	result.HeuristicResult = heuristicResult

	// TODO (nqmtuan): query for nth-section result
	return result, nil
}
