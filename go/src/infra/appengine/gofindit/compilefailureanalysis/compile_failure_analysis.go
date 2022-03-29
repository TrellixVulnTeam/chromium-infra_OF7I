// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Package compilefailureanalysis is the component for analyzing
// compile failures.
// It has 2 main components: heuristic analysis and nth_section analysis
package compilefailureanalysis

import (
	"context"
	"fmt"
	"infra/appengine/gofindit/compilefailureanalysis/heuristic"
	"infra/appengine/gofindit/compilefailureanalysis/nthsection"
	"infra/appengine/gofindit/internal/buildbucket"
	gfim "infra/appengine/gofindit/model"
	gfipb "infra/appengine/gofindit/proto"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
)

// AnalyzeFailure receives failure information and perform analysis.
// Note that this assumes that the failure is new (i.e. the client of this
// function should make sure this is not a duplicate analysis)
func AnalyzeFailure(
	c context.Context,
	cf *gfim.CompileFailure,
	first_failed_build_id int64,
	last_passed_build_id int64,
) (*gfim.CompileFailureAnalysis, error) {
	regression_range, e := findRegressionRange(c, first_failed_build_id, last_passed_build_id)
	if e != nil {
		return nil, e
	}

	logging.Infof(c, "Regression range: %v", regression_range)
	// Creates a new CompileFailureAnalysis entity in datastore
	analysis := &gfim.CompileFailureAnalysis{
		CompileFailure:         datastore.KeyForObj(c, cf),
		CreateTime:             clock.Now(c),
		Status:                 gfipb.AnalysisStatus_CREATED,
		FirstFailedBuildId:     first_failed_build_id,
		LastPassedBuildId:      last_passed_build_id,
		InitialRegressionRange: regression_range,
	}
	e = datastore.Put(c, analysis)
	if e != nil {
		return nil, e
	}

	// TODO (nqmtuan): run heuristic analysis and nth-section analysis in parallel
	// Heuristic analysis
	heuristicResult, e := heuristic.Analyze(c, analysis, regression_range)
	if e != nil {
		logging.Errorf(c, "Error during heuristic analysis: %v", e)
	}

	// Nth-section analysis
	_, e = nthsection.Analyze(c, analysis, regression_range)
	if e != nil {
		logging.Errorf(c, "Error during nthsection analysis: %v", e)
	}

	// TODO: For now, just check heuristic analysis status
	// We need to implement nth-section analysis as well
	analysis.Status = heuristicResult.Status
	analysis.EndTime = heuristicResult.EndTime

	e = datastore.Put(c, analysis)
	if e != nil {
		return nil, fmt.Errorf("Failed saving analysis: %w", e)
	}
	return analysis, nil
}

// findRegressionRange takes in the first failed and last passed buildID
// and returns the regression range based on GitilesCommit.
func findRegressionRange(
	c context.Context,
	first_failed_build_id int64,
	last_passed_build_id int64,
) (*gfipb.RegressionRange, error) {
	first_failed_build, err := buildbucket.GetBuild(c, first_failed_build_id, nil)
	if err != nil {
		return nil, fmt.Errorf("Error getting build %d: %w", first_failed_build_id, err)
	}

	last_passed_build, err := buildbucket.GetBuild(c, last_passed_build_id, nil)
	if err != nil {
		return nil, fmt.Errorf("Error getting build %d: %w", last_passed_build_id, err)
	}

	return &gfipb.RegressionRange{
		FirstFailed: first_failed_build.GetInput().GitilesCommit,
		LastPassed:  last_passed_build.GetInput().GitilesCommit,
	}, nil
}
