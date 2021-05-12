// Copyright 2021 The Chromium Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fakelegacy_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http/httptest"
	"sort"
	"testing"

	"infra/chromeperf/pinpoint"
	"infra/chromeperf/pinpoint/fakelegacy"
	"infra/chromeperf/pinpoint/proto"
	"infra/chromeperf/pinpoint/server"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	. "infra/chromeperf/pinpoint/assertions"

	. "github.com/smartystreets/goconvey/convey"
)

// Path to the directory which contains templates for API responses.
const templateDir = "templates/"

func TestStaticUsage(t *testing.T) {
	const (
		user   = "user@example.com"
		jobID0 = "00000000000000"
		jobID1 = "11111111111111"
	)
	legacyName0 := pinpoint.LegacyJobName(jobID0)
	legacyName1 := pinpoint.LegacyJobName(jobID1)
	fake, err := fakelegacy.NewServer(
		templateDir,
		map[string]*fakelegacy.Job{
			jobID0: {
				ID:        jobID0,
				Status:    fakelegacy.CompletedStatus,
				UserEmail: user,
			},
			jobID1: {
				ID:        jobID1,
				Status:    fakelegacy.CompletedStatus,
				UserEmail: user,
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(fake.Handler())
	defer ts.Close()

	grpcPinpoint := server.New(ts.URL, ts.Client())

	ctx := context.Background()
	Convey("GetJob should return known job", t, func() {
		j, err := grpcPinpoint.GetJob(ctx, &proto.GetJobRequest{Name: legacyName0})
		So(err, ShouldBeNil)
		So(j.Name, ShouldEqual, legacyName0)
		So(j.State, ShouldEqual, proto.Job_SUCCEEDED)
	})
	Convey("GetJob should return NotFound for unknown job", t, func() {
		_, err := grpcPinpoint.GetJob(ctx, &proto.GetJobRequest{Name: pinpoint.LegacyJobName("86753098675309")})
		So(err, ShouldBeStatusError, codes.NotFound)
	})
	Convey("ListJobs should return both known jobs", t, func() {
		list, err := grpcPinpoint.ListJobs(ctx, &proto.ListJobsRequest{})
		So(err, ShouldBeNil)
		So(list.Jobs, ShouldHaveLength, 2)

		sort.Slice(list.Jobs, func(i, j int) bool {
			return list.Jobs[i].Name < list.Jobs[j].Name
		})
		So(list.Jobs[0].Name, ShouldEqual, legacyName0)
		So(list.Jobs[1].Name, ShouldEqual, legacyName1)
	})
}

func TestAddJob(t *testing.T) {
	Convey("Given a fresh fakelegacy server", t, func() {
		const userEmail = "user@example.com"

		fake, err := fakelegacy.NewServer(
			templateDir,
			map[string]*fakelegacy.Job{},
		)
		So(err, ShouldBeNil)
		ts := httptest.NewServer(fake.Handler())
		defer ts.Close()

		grpcPinpoint := server.New(ts.URL, ts.Client())

		ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{
			server.EndpointsHeader: []string{
				base64.RawURLEncoding.EncodeToString(
					[]byte(fmt.Sprintf(`{"email": %q}`, userEmail)),
				),
			},
		})
		Convey("Users can schedule a gtest benchmark", func() {
			job, err := grpcPinpoint.ScheduleJob(ctx, &proto.ScheduleJobRequest{
				Job: &proto.JobSpec{
					Config: "some-config",
					Target: "some-target",
					Arguments: &proto.JobSpec_GtestBenchmark{
						GtestBenchmark: &proto.GTestBenchmark{
							Benchmark:   "benchmark",
							Test:        "test",
							Measurement: "measurement",
						},
					},
				},
			})
			So(err, ShouldBeNil)
			name := job.Name

			Convey("Users can immediately GetJob", func() {
				job, err := grpcPinpoint.GetJob(ctx, &proto.GetJobRequest{Name: name})
				So(err, ShouldBeNil)
				So(job.Name, ShouldEqual, name)
				So(job.State, ShouldEqual, proto.Job_PENDING)
				So(job.CreatedBy, ShouldEqual, userEmail)
			})

			Convey("The new job shows up in ListJobs", func() {
				list, err := grpcPinpoint.ListJobs(ctx, &proto.ListJobsRequest{})
				So(err, ShouldBeNil)
				So(list.Jobs, ShouldHaveLength, 1)
				So(list.Jobs[0].Name, ShouldEqual, name)
			})
		})
	})
}
