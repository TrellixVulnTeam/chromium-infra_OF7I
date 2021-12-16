// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	bbv1 "go.chromium.org/luci/common/api/buildbucket/buildbucket/v1"
	"go.chromium.org/luci/common/clock"
	cvv0 "go.chromium.org/luci/cv/api/v0"
	cvv1 "go.chromium.org/luci/cv/api/v1"
	"go.chromium.org/luci/server/tq"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/cv"
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	// Needed to ensure task class is registered.
	_ "infra/appengine/weetbix/internal/services/resultingester"
)

func TestHandleCVRun(t *testing.T) {
	Convey(`Test CVRunPubSubHandler`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx, skdr := tq.TestingContext(ctx, nil)

		// Setup two ingested builds.
		buildIDs := []int64{87654321, 87654322}
		for _, buildID := range buildIDs {
			buildExp := bbv1.LegacyApiCommonBuildMessage{
				Project:   "chromium",
				Bucket:    "luci.chromium.try",
				Id:        buildID,
				Status:    bbv1.StatusCompleted,
				CreatedTs: bbv1.FormatTimestamp(time.Now()),
			}
			r := &http.Request{Body: makeBBReq(buildExp, bbHost)}
			err := bbPubSubHandlerImpl(ctx, r)
			So(err, ShouldBeNil)
		}
		So(len(skdr.Tasks().Payloads()), ShouldEqual, 0)

		Convey(`non chromium cv run is ignored`, func() {
			psRun := &cvv1.PubSubRun{
				Id:     "projects/fake/runs/run_id",
				Status: cvv1.Run_SUCCEEDED,
			}
			r := &http.Request{Body: makeCVRunReq(psRun)}
			processed, err := cvPubSubHandlerImpl(ctx, r)
			So(err, ShouldBeNil)
			So(processed, ShouldBeFalse)
			So(len(skdr.Tasks().Payloads()), ShouldEqual, 0)
		})

		Convey(`chromium cv dry_run is ignored`, func() {
			ctx, skdr := tq.TestingContext(ctx, nil)
			rID := "id_dry_run"
			fID := fullRunID(rID)
			runs := map[string]*cvv0.Run{
				fID: {
					Id:         fID,
					Mode:       "DRY_RUN",
					CreateTime: timestamppb.New(clock.Now(ctx)),
					Tryjobs: []*cvv0.Tryjob{
						tryjob(buildIDs[0]),
						tryjob(buildIDs[1]),
					},
				},
			}
			ctx = cv.UseFakeClient(ctx, runs)
			r := &http.Request{Body: makeCVChromiumRunReq(fID)}
			processed, err := cvPubSubHandlerImpl(ctx, r)
			So(err, ShouldBeNil)
			So(processed, ShouldBeFalse)
			So(len(skdr.Tasks().Payloads()), ShouldEqual, 0)
		})

		Convey(`successful chromium cv full_run is processed`, func() {
			ctx, skdr := tq.TestingContext(ctx, nil)
			rID := "id_full_run"
			fID := fullRunID(rID)
			run := &cvv0.Run{
				Id:         fID,
				Mode:       "FULL_RUN",
				CreateTime: timestamppb.New(clock.Now(ctx)),
				Tryjobs: []*cvv0.Tryjob{
					tryjob(buildIDs[0]),
					tryjob(2),
					tryjob(buildIDs[1]),
				},
			}
			runs := map[string]*cvv0.Run{
				fID: run,
			}
			ctx = cv.UseFakeClient(ctx, runs)
			r := &http.Request{Body: makeCVChromiumRunReq(fID)}
			processed, err := cvPubSubHandlerImpl(ctx, r)
			So(err, ShouldBeNil)
			So(processed, ShouldBeTrue)
			So(len(skdr.Tasks().Payloads()), ShouldEqual, 2)

			actTasks := make([]*taskspb.IngestTestResults, 0, len(skdr.Tasks().Payloads()))
			for _, pl := range skdr.Tasks().Payloads() {
				actTasks = append(actTasks, pl.(*taskspb.IngestTestResults))
			}
			So(sortTasks(actTasks), ShouldResembleProto, sortTasks(expectedTasks(run, buildIDs)))

			Convey(`re-processing CV run should not result in further ingestion tasks`, func() {
				r := &http.Request{Body: makeCVChromiumRunReq(fID)}
				processed, err := cvPubSubHandlerImpl(ctx, r)
				So(err, ShouldBeNil)
				So(processed, ShouldBeTrue)
				So(len(skdr.Tasks().Payloads()), ShouldEqual, 2)
			})
		})

		Convey(`partial success`, func() {
			ctx, skdr := tq.TestingContext(ctx, nil)
			rID := "id_with_invalid_result"
			fID := fullRunID(rID)
			run := &cvv0.Run{
				Id:         fID,
				Mode:       "FULL_RUN",
				CreateTime: timestamppb.New(clock.Now(ctx)),
				Tryjobs: []*cvv0.Tryjob{
					{
						// Should be ignored.
						Result: &cvv0.Tryjob_Result{},
					},
					tryjob(buildIDs[0]),
				},
			}
			runs := map[string]*cvv0.Run{
				fID: run,
			}
			ctx = cv.UseFakeClient(ctx, runs)
			r := &http.Request{Body: makeCVChromiumRunReq(fID)}
			processed, err := cvPubSubHandlerImpl(ctx, r)
			So(err, ShouldBeNil)
			So(processed, ShouldBeTrue)
			So(len(skdr.Tasks().Payloads()), ShouldEqual, 1)
			So(skdr.Tasks().Payloads()[0].(*taskspb.IngestTestResults), ShouldResembleProto, expectedTasks(run, buildIDs[:1])[0])
		})
	})
}

func makeCVRunReq(psRun *cvv1.PubSubRun) io.ReadCloser {
	blob, _ := protojson.Marshal(psRun)
	return makeReq(blob)
}

func makeCVChromiumRunReq(runID string) io.ReadCloser {
	return makeCVRunReq(&cvv1.PubSubRun{
		Id:       runID,
		Status:   cvv1.Run_SUCCEEDED,
		Hostname: "cvhost",
	})
}

func tryjob(bID int64) *cvv0.Tryjob {
	return &cvv0.Tryjob{
		Result: &cvv0.Tryjob_Result{
			Backend: &cvv0.Tryjob_Result_Buildbucket_{
				Buildbucket: &cvv0.Tryjob_Result_Buildbucket{
					Id: int64(bID),
				},
			},
		},
	}
}

func fullRunID(runID string) string {
	return fmt.Sprintf("projects/%s/runs/%s", chromiumProject, runID)
}

func expectedTasks(run *cvv0.Run, buildIDs []int64) []*taskspb.IngestTestResults {
	res := make([]*taskspb.IngestTestResults, 0, len(run.Tryjobs))
	for _, buildID := range buildIDs {
		t := &taskspb.IngestTestResults{
			Build: &taskspb.Build{
				Host: bbHost,
				Id:   buildID,
			},
			PartitionTime: run.CreateTime,
			PresubmitRunId: &pb.PresubmitRunId{
				System: "luci-cv",
				Id:     chromiumProject + "/" + strings.Split(run.Id, "/")[3],
			},
			PresubmitRunSucceeded: run.Status == cvv0.Run_SUCCEEDED,
		}
		res = append(res, t)
	}
	return res
}

func sortTasks(tasks []*taskspb.IngestTestResults) []*taskspb.IngestTestResults {
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Build.Id < tasks[j].Build.Id })
	return tasks
}
