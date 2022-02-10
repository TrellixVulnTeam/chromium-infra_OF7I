// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristic

import (
	"context"
	gfim "infra/appengine/gofindit/model"
	gfipb "infra/appengine/gofindit/proto"

	"go.chromium.org/luci/common/clock"
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

	// TODO (nqmtuan) implement heuristic analysis
	return heuristic_analysis, nil
}
