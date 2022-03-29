// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package compilefailureanalysis

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	bbpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/gae/service/datastore"

	"infra/appengine/gofindit/internal/buildbucket"
	"infra/appengine/gofindit/internal/gitiles"
	"infra/appengine/gofindit/internal/logdog"
	"infra/appengine/gofindit/model"
)

func TestAnalyzeFailure(t *testing.T) {
	t.Parallel()
	c := memory.Use(context.Background())
	cl := testclock.New(testclock.TestTimeUTC)
	c = clock.Set(c, cl)

	// Setup mock for buildbucket
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mc := buildbucket.NewMockedClient(c, ctl)
	c = mc.Ctx
	c = gitiles.MockedGitilesClientContext(c)
	res := &bbpb.Build{
		Input: &bbpb.Build_Input{
			GitilesCommit: &bbpb.GitilesCommit{
				Host:    "host",
				Project: "proj",
				Id:      "id",
				Ref:     "ref",
			},
		},
		Steps: []*bbpb.Step{
			{
				Name: "compile",
				Logs: []*bbpb.Log{
					{
						Name:    "json.output[ninja_info]",
						ViewUrl: "https://logs.chromium.org/logs/ninja_log",
					},
					{
						Name:    "stdout",
						ViewUrl: "https://logs.chromium.org/logs/stdout_log",
					},
				},
			},
		},
	}
	mc.Client.EXPECT().GetBuild(gomock.Any(), gomock.Any(), gomock.Any()).Return(res, nil).AnyTimes()

	// Mock logdog
	ninjaLogJson := map[string]interface{}{
		"failures": []interface{}{},
	}
	ninjaLogStr, _ := json.Marshal(ninjaLogJson)
	c = logdog.MockClientContext(c, map[string]string{
		"https://logs.chromium.org/logs/ninja_log":  string(ninjaLogStr),
		"https://logs.chromium.org/logs/stdout_log": "stdout_log",
	})

	Convey("AnalyzeFailure analysis is created", t, func() {
		failed_build := &model.LuciFailedBuild{
			Id: 88128398584903,
			LuciBuild: model.LuciBuild{
				BuildId:     88128398584903,
				Project:     "chromium",
				Bucket:      "ci",
				Builder:     "android",
				BuildNumber: 123,
				StartTime:   cl.Now(),
				EndTime:     cl.Now(),
				CreateTime:  cl.Now(),
			},
			FailureType: model.BuildFailureType_Compile,
		}
		So(datastore.Put(c, failed_build), ShouldBeNil)

		compile_failure := &model.CompileFailure{
			Build:         datastore.KeyForObj(c, failed_build),
			OutputTargets: []string{"abc.xyx"},
			Rule:          "CXX",
			Dependencies:  []string{"dep"},
		}
		So(datastore.Put(c, compile_failure), ShouldBeNil)

		compile_failure_analysis, err := AnalyzeFailure(c, compile_failure, 123, 456)
		So(err, ShouldBeNil)
		datastore.GetTestable(c).CatchupIndexes()

		// Make sure that the analysis is created
		q := datastore.NewQuery("CompileFailureAnalysis").Eq("compile_failure", datastore.KeyForObj(c, compile_failure))
		analyses := []*model.CompileFailureAnalysis{}
		datastore.GetAll(c, q, &analyses)
		So(len(analyses), ShouldEqual, 1)

		// Make sure the heuristic analysis and nthsection analysis are run
		q = datastore.NewQuery("CompileHeuristicAnalysis").Eq("parent", datastore.KeyForObj(c, compile_failure_analysis))
		heuristic_analyses := []*model.CompileHeuristicAnalysis{}
		datastore.GetAll(c, q, &heuristic_analyses)
		So(len(heuristic_analyses), ShouldEqual, 1)

		q = datastore.NewQuery("CompileNthSectionAnalysis").Eq("parent", datastore.KeyForObj(c, compile_failure_analysis))
		nthsection_analyses := []*model.CompileNthSectionAnalysis{}
		datastore.GetAll(c, q, &nthsection_analyses)
		So(len(nthsection_analyses), ShouldEqual, 1)
	})
}
