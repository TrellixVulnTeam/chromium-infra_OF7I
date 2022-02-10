// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package nthsection

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
	rr *gfipb.RegressionRange) (*gfim.CompileNthSectionAnalysis, error) {
	// Create a new CompileNthSectionAnalysis Entity
	nth_section_analysis := &gfim.CompileNthSectionAnalysis{
		ParentAnalysis: datastore.KeyForObj(c, cfa),
		StartTime:      clock.Now(c),
		Status:         gfipb.AnalysisStatus_CREATED,
	}

	if err := datastore.Put(c, nth_section_analysis); err != nil {
		return nil, err
	}

	// TODO (nqmtuan) implement nth section analysis
	return nth_section_analysis, nil
}
