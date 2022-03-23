// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package model

import (
	"testing"

	gofinditpb "infra/appengine/gofindit/proto"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/clock"
	"go.chromium.org/luci/common/clock/testclock"
	"go.chromium.org/luci/gae/service/datastore"
)

func TestDatastoreModel(t *testing.T) {
	t.Parallel()

	Convey("Datastore Model", t, func() {
		c := gaetesting.TestingContext()
		cl := testclock.New(testclock.TestTimeUTC)
		c = clock.Set(c, cl)

		Convey("Can create datastore models", func() {
			failed_build := &LuciFailedBuild{
				Id: 88128398584903,
				LuciBuild: LuciBuild{
					BuildId:     88128398584903,
					Project:     "chromium",
					Bucket:      "ci",
					Builder:     "android",
					BuildNumber: 123,
					Status:      buildbucketpb.Status_FAILURE,
					StartTime:   cl.Now(),
					EndTime:     cl.Now(),
					CreateTime:  cl.Now(),
				},
				FailureType: BuildFailureType_Compile,
			}
			So(datastore.Put(c, failed_build), ShouldBeNil)

			compile_failure := &CompileFailure{
				Build:         datastore.KeyForObj(c, failed_build),
				OutputTargets: []string{"abc.xyx"},
				Rule:          "CXX",
				Dependencies:  []string{"dep"},
			}
			So(datastore.Put(c, compile_failure), ShouldBeNil)

			compile_failure_analysis := &CompileFailureAnalysis{
				CompileFailure:     datastore.KeyForObj(c, compile_failure),
				CreateTime:         cl.Now(),
				StartTime:          cl.Now(),
				EndTime:            cl.Now(),
				Status:             gofinditpb.AnalysisStatus_FOUND,
				FirstFailedBuildId: 88000998778,
				LastPassedBuildId:  873929392903,
				InitialRegressionRange: &gofinditpb.RegressionRange{
					LastPassed: &buildbucketpb.GitilesCommit{
						Host:    "host",
						Project: "proj",
						Ref:     "ref",
						Id:      "id1",
					},
					FirstFailed: &buildbucketpb.GitilesCommit{
						Host:    "host",
						Project: "proj",
						Ref:     "ref",
						Id:      "id2",
					},
					NumberOfRevisions: 10,
				},
			}
			So(datastore.Put(c, compile_failure_analysis), ShouldBeNil)

			heuristic_analysis := &CompileHeuristicAnalysis{
				ParentAnalysis: datastore.KeyForObj(c, compile_failure_analysis),
				StartTime:      cl.Now(),
				EndTime:        cl.Now(),
				Status:         gofinditpb.AnalysisStatus_CREATED,
			}
			So(datastore.Put(c, heuristic_analysis), ShouldBeNil)

			nthsection_analysis := &CompileNthSectionAnalysis{
				ParentAnalysis: datastore.KeyForObj(c, compile_failure_analysis),
				StartTime:      cl.Now(),
				EndTime:        cl.Now(),
				Status:         gofinditpb.AnalysisStatus_CREATED,
			}
			So(datastore.Put(c, nthsection_analysis), ShouldBeNil)

			rerun_build := &CompileRerunBuild{
				ParentAnalysis: datastore.KeyForObj(c, nthsection_analysis),
				LuciBuild: LuciBuild{
					BuildId:     88128398584903,
					Project:     "chromium",
					Bucket:      "ci",
					Builder:     "android",
					BuildNumber: 123,
					Status:      buildbucketpb.Status_SUCCESS,
					StartTime:   cl.Now(),
					EndTime:     cl.Now(),
					CreateTime:  cl.Now(),
				},
			}
			So(datastore.Put(c, rerun_build), ShouldBeNil)

			culprit := &Culprit{
				ParentAnalysis: datastore.KeyForObj(c, compile_failure_analysis),
				GitilesCommit: buildbucketpb.GitilesCommit{
					Project:  "my project",
					Host:     "host",
					Ref:      "ref",
					Id:       "id",
					Position: 3433,
				},
			}
			So(datastore.Put(c, culprit), ShouldBeNil)

			suspect := &Suspect{
				ParentAnalysis: datastore.KeyForObj(c, compile_failure_analysis),
				GitilesCommit: buildbucketpb.GitilesCommit{
					Project:  "my project",
					Host:     "host",
					Ref:      "ref",
					Id:       "id",
					Position: 3433,
				},
				ReviewUrl:     "http://review-url.com",
				Score:         100,
				Justification: "The CL touch the file abc.cc, and it is in the log",
			}
			So(datastore.Put(c, suspect), ShouldBeNil)
		})
	})
}
