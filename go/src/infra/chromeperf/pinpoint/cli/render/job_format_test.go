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

package render

import (
	"testing"
	"time"

	"github.com/google/uuid"
	. "github.com/smartystreets/goconvey/convey"
	"google.golang.org/protobuf/types/known/timestamppb"

	"infra/chromeperf/pinpoint/proto"
)

func TestJobRenderingLegacyURL(t *testing.T) {
	Convey("Given a Job proto", t, func() {
		j := &proto.Job{
			Name:           "",
			State:          0,
			CreatedBy:      "",
			CreateTime:     timestamppb.New(time.Now().Add(-time.Hour)),
			LastUpdateTime: timestamppb.Now(),
			JobSpec: &proto.JobSpec{
				MonorailIssue: &proto.MonorailIssue{
					Project: "chromium",
					IssueId: 1234,
				},
			},
			CancellationReason: "",
			Results:            nil,
		}
		Convey("When we have a monorail issue", func() {
			s := renderMonorailIssue(j)
			So(s, ShouldEqual, "https://bugs.chromium.org/p/chromium/issues/detail?id=1234")
		})
		Convey("When we have a legacy ID", func() {
			j.Name = "jobs/legacy-1234567"
			Convey("Then we can generate a URL for the job", func() {
				u, err := legacyJobURL(j)
				So(err, ShouldBeNil)
				So(u, ShouldEqual, "https://pinpoint-dot-chromeperf.appspot.com/job/1234567")
			})
			Convey("Then we can generate a ID for the job", func() {
				u, err := JobID(j)
				So(err, ShouldBeNil)
				So(u, ShouldEqual, "1234567")
			})
		})
		Convey("When the legacy service does not have a trailing /", func() {
			j.Name = "jobs/legacy-1234"
			Convey("Then we can generate a valid URL for the job", func() {
				u, err := legacyJobURL(j)
				So(err, ShouldBeNil)
				So(u, ShouldEqual, "https://pinpoint-dot-chromeperf.appspot.com/job/1234")
			})
			Convey("Then we can generate a valid ID for the job", func() {
				u, err := JobID(j)
				So(err, ShouldBeNil)
				So(u, ShouldEqual, "1234")
			})
		})
		Convey("When we have a non-legacy ID", func() {
			uID, err := uuid.NewRandom()
			So(err, ShouldBeNil)
			j.Name = uID.String()
			So(j.Name, ShouldNotEqual, "")
			Convey("Then we cannot generate a valid URL for the job", func() {
				u, err := legacyJobURL(j)
				So(u, ShouldEqual, "")
				So(err, ShouldNotBeNil)
			})
		})
	})
}
