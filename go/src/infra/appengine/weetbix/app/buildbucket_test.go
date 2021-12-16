// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package app

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	bbv1 "go.chromium.org/luci/common/api/buildbucket/buildbucket/v1"
	cvv0 "go.chromium.org/luci/cv/api/v0"
	"go.chromium.org/luci/server/tq"

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

func TestHandleBuild(t *testing.T) {
	Convey(`With Spanner Test Database`, t, func() {
		ctx := testutil.SpannerTestContext(t)
		ctx, skdr := tq.TestingContext(ctx, nil)

		Convey(`Test BuildbucketPubSubHandler`, func() {
			Convey(`non chromium build is ignored`, func() {
				buildExp := bbv1.LegacyApiCommonBuildMessage{
					Project:   "fake",
					Bucket:    "luci.fake.bucket",
					Id:        87654321,
					Status:    bbv1.StatusCompleted,
					CreatedTs: bbv1.FormatTimestamp(time.Now()),
				}
				r := &http.Request{Body: makeBBReq(buildExp, "bb-hostname")}
				err := bbPubSubHandlerImpl(ctx, r)
				So(err, ShouldBeNil)
				So(len(skdr.Tasks().Payloads()), ShouldEqual, 0)
			})

			Convey(`CI build is processed`, func() {
				// Buildbucket timestamps are only in microsecond precision.
				t := time.Now().Truncate(time.Nanosecond * 1000)

				buildExp := bbv1.LegacyApiCommonBuildMessage{
					Project:   "chromium",
					Bucket:    chromiumCIBucket,
					Id:        87654321,
					Status:    bbv1.StatusCompleted,
					CreatedTs: bbv1.FormatTimestamp(t),
				}
				r := &http.Request{Body: makeBBReq(buildExp, "bb-hostname")}

				err := bbPubSubHandlerImpl(ctx, r)
				So(err, ShouldBeNil)

				So(len(skdr.Tasks().Payloads()), ShouldEqual, 1)
				task := skdr.Tasks().Payloads()[0].(*taskspb.IngestTestResults)
				So(task, ShouldResembleProto, &taskspb.IngestTestResults{
					Build: &taskspb.Build{
						Host: "bb-hostname",
						Id:   87654321,
					},
					PartitionTime: timestamppb.New(t),
				})

				Convey(`repeated processing does not lead to further ingestion tasks`, func() {
					r := &http.Request{Body: makeBBReq(buildExp, "bb-hostname")}
					err := bbPubSubHandlerImpl(ctx, r)
					So(err, ShouldBeNil)
					So(len(skdr.Tasks().Payloads()), ShouldEqual, 1)
				})
			})

			Convey(`Try build is processed`, func() {
				t := time.Date(2025, time.April, 1, 2, 3, 4, 0, time.UTC)

				buildExp := bbv1.LegacyApiCommonBuildMessage{
					Project:   "chromium",
					Bucket:    "luci.chromium.try",
					Id:        14141414,
					Status:    bbv1.StatusCompleted,
					CreatedTs: bbv1.FormatTimestamp(t),
				}

				Convey(`With presubmit run processed previously`, func() {
					partitionTime := time.Now()
					run := &cvv0.Run{
						Id:         "projects/chromium/runs/123e4567-e89b-12d3-a456-426614174000",
						Mode:       "FULL_RUN",
						CreateTime: timestamppb.New(partitionTime),
						Tryjobs: []*cvv0.Tryjob{
							tryjob(2),
							tryjob(14141414),
						},
					}
					runs := map[string]*cvv0.Run{
						run.Id: run,
					}
					ctx = cv.UseFakeClient(ctx, runs)

					// Process presubmit run.
					r := &http.Request{Body: makeCVChromiumRunReq(run.Id)}
					processed, err := cvPubSubHandlerImpl(ctx, r)
					So(err, ShouldBeNil)
					So(processed, ShouldBeTrue)

					So(len(skdr.Tasks().Payloads()), ShouldEqual, 0)

					// Process build.
					r = &http.Request{Body: makeBBReq(buildExp, bbHost)}
					err = bbPubSubHandlerImpl(ctx, r)
					So(err, ShouldBeNil)

					So(len(skdr.Tasks().Payloads()), ShouldEqual, 1)
					task := skdr.Tasks().Payloads()[0].(*taskspb.IngestTestResults)
					So(task, ShouldResembleProto, &taskspb.IngestTestResults{
						Build: &taskspb.Build{
							Host: bbHost,
							Id:   14141414,
						},
						PartitionTime: timestamppb.New(partitionTime),
						PresubmitRunId: &pb.PresubmitRunId{
							System: "luci-cv",
							Id:     "chromium/123e4567-e89b-12d3-a456-426614174000",
						},
						PresubmitRunSucceeded: false,
					})
				})
				Convey(`Without presubmit run processed previously`, func() {
					r := &http.Request{Body: makeBBReq(buildExp, bbHost)}
					err := bbPubSubHandlerImpl(ctx, r)
					So(err, ShouldBeNil)
					So(len(skdr.Tasks().Payloads()), ShouldEqual, 0)
				})
			})
		})
	})
}

func makeBBReq(build bbv1.LegacyApiCommonBuildMessage, hostname string) io.ReadCloser {
	bmsg := struct {
		Build    bbv1.LegacyApiCommonBuildMessage `json:"build"`
		Hostname string                           `json:"hostname"`
	}{build, hostname}
	bm, _ := json.Marshal(bmsg)
	return makeReq(bm)
}
