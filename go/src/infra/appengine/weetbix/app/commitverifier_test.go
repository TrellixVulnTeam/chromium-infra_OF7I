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

	. "github.com/smartystreets/goconvey/convey"
	bbv1 "go.chromium.org/luci/common/api/buildbucket/buildbucket/v1"
	"go.chromium.org/luci/common/clock"
	. "go.chromium.org/luci/common/testing/assertions"
	cvv0 "go.chromium.org/luci/cv/api/v0"
	cvv1 "go.chromium.org/luci/cv/api/v1"
	"go.chromium.org/luci/server/tq"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/cv"
	_ "infra/appengine/weetbix/internal/services/resultingester" // Needed to ensure task class is registered.
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"
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
		Convey(`Chromium CV run is processed`, func() {
			ctx, skdr := tq.TestingContext(ctx, nil)
			rID := "id_full_run"
			fID := fullRunID(rID)

			processCVRun := func(run *cvv0.Run) (processed bool, tasks []*taskspb.IngestTestResults) {
				existingTaskCount := len(skdr.Tasks().Payloads())

				runs := map[string]*cvv0.Run{
					fID: run,
				}
				ctx = cv.UseFakeClient(ctx, runs)
				r := &http.Request{Body: makeCVChromiumRunReq(fID)}
				processed, err := cvPubSubHandlerImpl(ctx, r)
				So(err, ShouldBeNil)

				tasks = make([]*taskspb.IngestTestResults, 0,
					len(skdr.Tasks().Payloads())-existingTaskCount)
				for _, pl := range skdr.Tasks().Payloads()[existingTaskCount:] {
					tasks = append(tasks, pl.(*taskspb.IngestTestResults))
				}
				return processed, tasks
			}

			run := &cvv0.Run{
				Id:         fID,
				Mode:       "FULL_RUN",
				CreateTime: timestamppb.New(clock.Now(ctx)),
				Owner:      "cl-owner@google.com",
				Tryjobs: []*cvv0.Tryjob{
					tryjob(buildIDs[0]),
					tryjob(2), // This build has not been ingested yet.
					tryjob(buildIDs[1]),
				},
				Cls: []*cvv0.GerritChange{
					{
						Host:     "chromium-review.googlesource.com",
						Change:   12345,
						Patchset: 1,
					},
				},
				Status: cvv0.Run_SUCCEEDED,
			}
			expectedTaskTemplate := &taskspb.IngestTestResults{
				PartitionTime: run.CreateTime,
				PresubmitRunId: &pb.PresubmitRunId{
					System: "luci-cv",
					Id:     chromiumProject + "/" + strings.Split(run.Id, "/")[3],
				},
				PresubmitRunCls: []*pb.Changelist{
					{
						Host:     "chromium-review.googlesource.com",
						Change:   12345,
						Patchset: 1,
					},
				},
				PresubmitRunSucceeded: true,
				PresubmitRunOwner:     "user",
			}
			Convey(`Baseline`, func() {
				processed, tasks := processCVRun(run)
				So(processed, ShouldBeTrue)
				So(sortTasks(tasks), ShouldResembleProto,
					sortTasks(expectedTasks(expectedTaskTemplate, buildIDs)))

				Convey(`Re-processing CV run should not result in further ingestion tasks`, func() {
					processed, tasks = processCVRun(run)
					So(processed, ShouldBeTrue)
					So(tasks, ShouldBeEmpty)
				})
			})
			Convey(`Dry run is ignored`, func() {
				run.Mode = "DRY_RUN"

				processed, tasks := processCVRun(run)
				So(processed, ShouldBeFalse)
				So(tasks, ShouldBeEmpty)
			})
			Convey(`CV Run owned by Automation`, func() {
				run.Owner = "chromium-autoroll@skia-public.iam.gserviceaccount.com"
				expectedTaskTemplate.PresubmitRunOwner = "automation"

				processed, tasks := processCVRun(run)
				So(processed, ShouldBeTrue)
				So(sortTasks(tasks), ShouldResembleProto,
					sortTasks(expectedTasks(expectedTaskTemplate, buildIDs)))
			})
			Convey(`With non-buildbucket tryjob`, func() {
				// Should be ignored.
				run.Tryjobs = append(run.Tryjobs, &cvv0.Tryjob{
					Result: &cvv0.Tryjob_Result{},
				})

				processed, tasks := processCVRun(run)
				So(processed, ShouldBeTrue)
				So(sortTasks(tasks), ShouldResembleProto,
					sortTasks(expectedTasks(expectedTaskTemplate, buildIDs)))
			})
			Convey(`Failing Run`, func() {
				run.Status = cvv0.Run_FAILED
				expectedTaskTemplate.PresubmitRunSucceeded = false

				processed, tasks := processCVRun(run)
				So(processed, ShouldBeTrue)
				So(sortTasks(tasks), ShouldResembleProto,
					sortTasks(expectedTasks(expectedTaskTemplate, buildIDs)))
			})
			Convey(`Cancelled Run`, func() {
				run.Status = cvv0.Run_CANCELLED
				expectedTaskTemplate.PresubmitRunSucceeded = false

				processed, tasks := processCVRun(run)
				So(processed, ShouldBeTrue)
				So(sortTasks(tasks), ShouldResembleProto,
					sortTasks(expectedTasks(expectedTaskTemplate, buildIDs)))
			})
			Convey(`Multi-CL`, func() {
				run.Cls = []*cvv0.GerritChange{
					{
						Host:     "host2",
						Change:   100,
						Patchset: 1,
					}, {
						Host:     "host1",
						Change:   201,
						Patchset: 2,
					}, {
						Host:     "host1",
						Change:   200,
						Patchset: 3,
					}, {
						Host:     "host1",
						Change:   200,
						Patchset: 4,
					},
				}
				// Must appear in sorted order.
				expectedTaskTemplate.PresubmitRunCls = []*pb.Changelist{
					{
						Host:     "host1",
						Change:   200,
						Patchset: 3,
					},
					{
						Host:     "host1",
						Change:   200,
						Patchset: 4,
					},
					{
						Host:     "host1",
						Change:   201,
						Patchset: 2,
					},
					{
						Host:     "host2",
						Change:   100,
						Patchset: 1,
					},
				}

				processed, tasks := processCVRun(run)
				So(processed, ShouldBeTrue)
				So(sortTasks(tasks), ShouldResembleProto,
					sortTasks(expectedTasks(expectedTaskTemplate, buildIDs)))
			})
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

func expectedTasks(taskTemplate *taskspb.IngestTestResults, buildIDs []int64) []*taskspb.IngestTestResults {
	res := make([]*taskspb.IngestTestResults, 0, len(buildIDs))
	for _, buildID := range buildIDs {
		t := proto.Clone(taskTemplate).(*taskspb.IngestTestResults)
		t.Build = &taskspb.Build{
			Host: bbHost,
			Id:   buildID,
		}
		res = append(res, t)
	}
	return res
}

func sortTasks(tasks []*taskspb.IngestTestResults) []*taskspb.IngestTestResults {
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Build.Id < tasks[j].Build.Id })
	return tasks
}
