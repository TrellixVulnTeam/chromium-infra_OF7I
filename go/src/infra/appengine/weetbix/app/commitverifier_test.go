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
	"go.chromium.org/luci/gae/impl/memory"
	"go.chromium.org/luci/server/tq"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/appengine/weetbix/internal/config"
	configpb "infra/appengine/weetbix/internal/config/proto"
	"infra/appengine/weetbix/internal/cv"
	controlpb "infra/appengine/weetbix/internal/ingestion/control/proto"
	_ "infra/appengine/weetbix/internal/services/resultingester" // Needed to ensure task class is registered.
	"infra/appengine/weetbix/internal/tasks/taskspb"
	"infra/appengine/weetbix/internal/testutil"
	pb "infra/appengine/weetbix/proto/v1"
)

// bbCreateTime is the create time assigned to buildbucket builds, for testing.
// Must be in microsecond precision as that is the precision of buildbucket.
var bbCreateTime = time.Date(2025, time.December, 1, 2, 3, 4, 5000, time.UTC)

func TestHandleCVRun(t *testing.T) {
	Convey(`Test CVRunPubSubHandler`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx, skdr := tq.TestingContext(ctx, nil)
		ctx = memory.Use(ctx) // For test config.

		// Builds and CV runs can come from different projects
		// and still join. We test this by using two projects,
		// one for builds, one for cv runs. Only the project
		// for builds needs to be configured, as that is the
		// project where data is ingested into.
		configs := map[string]*configpb.ProjectConfig{
			"buildproject": config.CreatePlaceholderProjectConfig(),
		}

		err := config.SetTestProjectConfig(ctx, configs)
		So(err, ShouldBeNil)

		// Setup two ingested tryjob builds.
		buildIDs := []int64{87654321, 87654322}
		for _, buildID := range buildIDs {
			buildExp := bbv1.LegacyApiCommonBuildMessage{
				Project:   "buildproject",
				Bucket:    "luci.buildproject.bucket",
				Id:        buildID,
				Status:    bbv1.StatusCompleted,
				CreatedTs: bbv1.FormatTimestamp(bbCreateTime),
				Tags:      []string{"user_agent:cq"},
			}
			r := &http.Request{Body: makeBBReq(buildExp, bbHost)}
			project, processed, err := bbPubSubHandlerImpl(ctx, r)
			So(err, ShouldBeNil)
			So(processed, ShouldBeTrue)
			So(project, ShouldEqual, "buildproject")
		}
		So(len(skdr.Tasks().Payloads()), ShouldEqual, 0)

		Convey(`CV run is processed`, func() {
			ctx, skdr := tq.TestingContext(ctx, nil)
			rID := "id_full_run"
			fullRunID := fullRunID("cvproject", rID)

			processCVRun := func(run *cvv0.Run) (processed bool, tasks []*taskspb.IngestTestResults) {
				existingTaskCount := len(skdr.Tasks().Payloads())

				runs := map[string]*cvv0.Run{
					fullRunID: run,
				}
				ctx = cv.UseFakeClient(ctx, runs)
				r := &http.Request{Body: makeCVChromiumRunReq(fullRunID)}
				project, processed, err := cvPubSubHandlerImpl(ctx, r)
				So(err, ShouldBeNil)
				So(project, ShouldEqual, "cvproject")

				tasks = make([]*taskspb.IngestTestResults, 0,
					len(skdr.Tasks().Payloads())-existingTaskCount)
				for _, pl := range skdr.Tasks().Payloads()[existingTaskCount:] {
					tasks = append(tasks, pl.(*taskspb.IngestTestResults))
				}
				return processed, tasks
			}

			run := &cvv0.Run{
				Id:         fullRunID,
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
				PresubmitRun: &controlpb.PresubmitResult{
					PresubmitRunId: &pb.PresubmitRunId{
						System: "luci-cv",
						Id:     "cvproject/" + strings.Split(run.Id, "/")[3],
					},
					Cls: []*pb.Changelist{
						{
							Host:     "chromium-review.googlesource.com",
							Change:   12345,
							Patchset: 1,
						},
					},
					PresubmitRunSucceeded: true,
					Mode:                  "FULL_RUN",
					Owner:                 "user",
					CreationTime:          run.CreateTime,
				},
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
			Convey(`Dry run`, func() {
				run.Mode = "DRY_RUN"
				expectedTaskTemplate.PresubmitRun.Mode = "DRY_RUN"

				processed, tasks := processCVRun(run)
				So(processed, ShouldBeTrue)
				So(sortTasks(tasks), ShouldResembleProto,
					sortTasks(expectedTasks(expectedTaskTemplate, buildIDs)))
			})
			Convey(`CV Run owned by Automation`, func() {
				run.Owner = "chromium-autoroll@skia-public.iam.gserviceaccount.com"
				expectedTaskTemplate.PresubmitRun.Owner = "automation"

				processed, tasks := processCVRun(run)
				So(processed, ShouldBeTrue)
				So(sortTasks(tasks), ShouldResembleProto,
					sortTasks(expectedTasks(expectedTaskTemplate, buildIDs)))
			})
			Convey(`CV Run owned by Automation 2`, func() {
				run.Owner = "3su6n15k.default@developer.gserviceaccount.com"
				expectedTaskTemplate.PresubmitRun.Owner = "automation"

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
			Convey(`With re-used tryjob`, func() {
				// Assume that this tryjob was created by another CV run,
				// so should not be ingested with this CV run.
				run.Tryjobs[0].Reuse = true

				processed, tasks := processCVRun(run)
				So(processed, ShouldBeTrue)
				So(sortTasks(tasks), ShouldResembleProto,
					sortTasks(expectedTasks(expectedTaskTemplate, buildIDs[1:])))
			})
			Convey(`Failing Run`, func() {
				run.Status = cvv0.Run_FAILED
				expectedTaskTemplate.PresubmitRun.PresubmitRunSucceeded = false

				processed, tasks := processCVRun(run)
				So(processed, ShouldBeTrue)
				So(sortTasks(tasks), ShouldResembleProto,
					sortTasks(expectedTasks(expectedTaskTemplate, buildIDs)))
			})
			Convey(`Cancelled Run`, func() {
				run.Status = cvv0.Run_CANCELLED
				expectedTaskTemplate.PresubmitRun.PresubmitRunSucceeded = false

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
				expectedTaskTemplate.PresubmitRun.Cls = []*pb.Changelist{
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
		Critical: (bID % 2) == 0,
	}
}

func fullRunID(project, runID string) string {
	return fmt.Sprintf("projects/%s/runs/%s", project, runID)
}

func expectedTasks(taskTemplate *taskspb.IngestTestResults, buildIDs []int64) []*taskspb.IngestTestResults {
	res := make([]*taskspb.IngestTestResults, 0, len(buildIDs))
	for _, buildID := range buildIDs {
		t := proto.Clone(taskTemplate).(*taskspb.IngestTestResults)
		t.PresubmitRun.Critical = ((buildID % 2) == 0)
		t.Build = &controlpb.BuildResult{
			Host:         bbHost,
			Id:           buildID,
			CreationTime: timestamppb.New(bbCreateTime),
		}
		res = append(res, t)
	}
	return res
}

func sortTasks(tasks []*taskspb.IngestTestResults) []*taskspb.IngestTestResults {
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Build.Id < tasks[j].Build.Id })
	return tasks
}
