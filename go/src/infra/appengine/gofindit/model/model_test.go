// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package model

import (
	"testing"

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

			compile_failure_keys := []*datastore.Key{datastore.KeyForObj(c, compile_failure)}
			compile_failure_analysis := &CompileFailureAnalysis{
				CompileFailures:    compile_failure_keys,
				CreateTime:         cl.Now(),
				StartTime:          cl.Now(),
				EndTime:            cl.Now(),
				Status:             AnalysisStatus_Completed,
				FirstFailedBuildId: 88000998778,
				LastPassedBuildId:  873929392903,
			}
			So(datastore.Put(c, compile_failure_analysis), ShouldBeNil)

			rerun_build := &CompileRerunBuild{
				ParentAnalysis: datastore.KeyForObj(c, compile_failure_analysis),
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
				GitilesCommit: GitilesCommit{
					GitilesProject:        "my project",
					GitilesHost:           "host",
					GitilesRef:            "ref",
					GitilesCommitID:       "id",
					GitilesCommitPosition: 3433,
				},
			}
			So(datastore.Put(c, culprit), ShouldBeNil)

			suspect := &Suspect{
				ParentAnalysis: datastore.KeyForObj(c, compile_failure_analysis),
				GitilesCommit: GitilesCommit{
					GitilesProject:        "my project",
					GitilesHost:           "host",
					GitilesRef:            "ref",
					GitilesCommitID:       "id",
					GitilesCommitPosition: 3433,
				},
				Hint: SuspectHint{
					Content: "The CL touch the file abc.cc, and it is in the log",
					Score:   100,
				},
			}
			So(datastore.Put(c, suspect), ShouldBeNil)

		})
	})
}
