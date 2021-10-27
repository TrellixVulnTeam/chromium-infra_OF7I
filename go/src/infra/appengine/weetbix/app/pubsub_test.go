// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
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

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	// Needed to ensure task class is registered.
	_ "infra/appengine/weetbix/internal/services/resultingester"
)

func makeReq(blob []byte) io.ReadCloser {
	msg := struct {
		Message struct {
			Data []byte
		}
		Attributes map[string]interface{}
	}{struct{ Data []byte }{Data: blob}, nil}
	jmsg, _ := json.Marshal(msg)
	return ioutil.NopCloser(bytes.NewReader(jmsg))
}

func makeBBReq(build bbv1.LegacyApiCommonBuildMessage) io.ReadCloser {
	bmsg := struct {
		Build    bbv1.LegacyApiCommonBuildMessage `json:"build"`
		Hostname string                           `json:"hostname"`
	}{build, "hostname"}
	bm, _ := json.Marshal(bmsg)
	return makeReq(bm)
}

func TestHandleBuild(t *testing.T) {
	t.Parallel()
	ctx := testutil.TestingContext()
	ctx, _ = tq.TestingContext(ctx, nil)

	Convey(`Test BuildbucketPubSubHandler`, t, func() {
		Convey(`non chromium build is ignored`, func() {
			buildExp := bbv1.LegacyApiCommonBuildMessage{
				Project:   "fake",
				Bucket:    "luci.fake.bucket",
				Id:        87654321,
				Status:    bbv1.StatusCompleted,
				CreatedTs: bbv1.FormatTimestamp(time.Now()),
			}
			r := &http.Request{Body: makeBBReq(buildExp)}
			err := bbPubSubHandlerImpl(ctx, r)
			So(err, ShouldBeNil)
		})

		Convey(`chromium build is processed`, func() {
			buildExp := bbv1.LegacyApiCommonBuildMessage{
				Project:   "chromium",
				Bucket:    chromiumCIBucket,
				Id:        87654321,
				Status:    bbv1.StatusCompleted,
				CreatedTs: bbv1.FormatTimestamp(time.Now()),
			}
			r := &http.Request{Body: makeBBReq(buildExp)}
			err := bbPubSubHandlerImpl(ctx, r)
			So(err, ShouldBeNil)
		})
	})
}

func makeCVRunReq(psRun *cvv1.PubSubRun) io.ReadCloser {
	blob, _ := protojson.Marshal(psRun)
	return makeReq(blob)
}

func fullRunID(runID string) string {
	return fmt.Sprintf("projects/%s/runs/%s", chromiumProject, runID)
}

func makeCVChromiumRunReq(runID string) io.ReadCloser {
	return makeCVRunReq(&cvv1.PubSubRun{
		Id:       runID,
		Status:   cvv1.Run_SUCCEEDED,
		Hostname: "cvhost",
	})
}

func tryjob(bID int) *cvv0.Tryjob {
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

func expectedTasks(run *cvv0.Run) []*taskspb.IngestTestResults {
	res := make([]*taskspb.IngestTestResults, 0, len(run.Tryjobs))
	for _, tj := range run.Tryjobs {
		if tj.GetResult() == nil {
			continue
		}
		t := &taskspb.IngestTestResults{
			CvRun: run,
			Build: &taskspb.Build{
				Host: bbHost,
				Id:   tj.GetResult().GetBuildbucket().GetId(),
			},
			PartitionTime: run.CreateTime,
		}
		res = append(res, t)
	}
	return res
}

func sortTasks(tasks []*taskspb.IngestTestResults) []*taskspb.IngestTestResults {
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Build.Id < tasks[j].Build.Id })
	return tasks
}

func TestHandleCVRun(t *testing.T) {
	t.Parallel()
	ctx := testutil.TestingContext()

	Convey(`Test CVRunPubSubHandler`, t, func() {
		Convey(`non chromium cv run is ignored`, func() {
			ctx, skdr := tq.TestingContext(ctx, nil)
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
					tryjob(1),
					tryjob(2),
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
			So(sortTasks(actTasks), ShouldResembleProto, sortTasks(expectedTasks(run)))
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
					tryjob(1),
					{
						Result: &cvv0.Tryjob_Result{},
					},
				},
			}
			runs := map[string]*cvv0.Run{
				fID: run,
			}
			ctx = cv.UseFakeClient(ctx, runs)
			r := &http.Request{Body: makeCVChromiumRunReq(fID)}
			processed, err := cvPubSubHandlerImpl(ctx, r)
			So(err, ShouldErrLike, "unrecognized CV run try job result")
			So(processed, ShouldBeTrue)
			So(len(skdr.Tasks().Payloads()), ShouldEqual, 1)
			So(skdr.Tasks().Payloads()[0].(*taskspb.IngestTestResults), ShouldResembleProto, expectedTasks(run)[0])
		})
	})
}
