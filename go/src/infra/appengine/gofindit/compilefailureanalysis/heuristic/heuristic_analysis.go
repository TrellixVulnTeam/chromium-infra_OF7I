// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristic

import (
	"context"
	"fmt"
	"infra/appengine/gofindit/internal/gitiles"
	"infra/appengine/gofindit/model"
	gfim "infra/appengine/gofindit/model"
	gfipb "infra/appengine/gofindit/proto"

	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
)

func Analyze(
	c context.Context,
	cfa *gfim.CompileFailureAnalysis,
	rr *gfipb.RegressionRange) (*gfim.CompileHeuristicAnalysis, error) {
	// Create a new HeuristicAnalysis Entity
	heuristic_analysis := &gfim.CompileHeuristicAnalysis{
		ParentAnalysis: datastore.KeyForObj(c, cfa),
		StartTime:      clock.Now(c),
		Status:         gfipb.AnalysisStatus_CREATED,
	}

	if err := datastore.Put(c, heuristic_analysis); err != nil {
		return nil, err
	}

	// Get changelogs for heuristic analysis
	changelogs, err := getChangeLogs(c, rr)
	if err != nil {
		return nil, fmt.Errorf("Failed getting changelogs %w", err)
	}
	logging.Infof(c, "Changelogs has %d logs", len(changelogs))
	// TODO (nqmtuan) implement heuristic analysis
	return heuristic_analysis, nil
}

// getChangeLogs queries Gitiles for changelogs in the regression range
func getChangeLogs(c context.Context, rr *gfipb.RegressionRange) ([]*model.ChangeLog, error) {
	if rr.LastPassed.Host != rr.FirstFailed.Host || rr.LastPassed.Project != rr.FirstFailed.Project {
		return nil, fmt.Errorf("RepoURL for last pass and first failed commits must be same, but aren't: %v and %v", rr.LastPassed, rr.FirstFailed)
	}
	repoUrl := gitiles.GetRepoUrl(c, rr.LastPassed)
	return gitiles.GetChangeLogs(c, repoUrl, rr.LastPassed.Id, rr.FirstFailed.Id)
}
