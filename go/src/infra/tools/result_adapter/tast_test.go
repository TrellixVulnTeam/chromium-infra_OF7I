// Copyright 2021 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
	pb "go.chromium.org/luci/resultdb/proto/v1"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func parseTime(s string) time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05.99Z", s)
	return t
}

func mockCollect(s string) (map[string]string, error) {
	return map[string]string{
		"foo": s + "/foo",
	}, nil
}

func genJSONLine(m map[string]string) string {
	base := map[string]string{
		"name":   "lacros.Basic",
		"bundle": "cros",
		"start":  "2021-07-26T18:53:33.983328614Z",
		"end":    "2021-07-26T18:53:34.983328614Z",
		"outDir": "/usr/local/autotest/results/lxc_job_folder/tast/results/tests/lacros.Basic",
	}
	for k, v := range m {
		base[k] = v
	}
	jsonStr := ""
	for k, v := range base {
		if k == "errors" {
			jsonStr += fmt.Sprintf("\"%s\":%s,", k, v)
			continue
		}
		jsonStr += fmt.Sprintf("\"%s\":\"%s\",", k, v)
	}
	return fmt.Sprintf("{%s}", strings.TrimSuffix(jsonStr, ","))
}

func TestTastConversions(t *testing.T) {
	t.Parallel()

	Convey(`From JSON works`, t, func() {
		r := &TastResults{}
		Convey(`Basic`, func() {
			jsonLine := genJSONLine(map[string]string{
				"skipReason": "dummy skipped",
			})
			err := r.ConvertFromJSON(strings.NewReader(jsonLine))
			So(err, ShouldBeNil)
			So(r.Cases[0], ShouldResemble, TastCase{
				Name:       "lacros.Basic",
				OutDir:     "/usr/local/autotest/results/lxc_job_folder/tast/results/tests/lacros.Basic",
				SkipReason: "dummy skipped",
				Errors:     nil,
				Start:      parseTime("2021-07-26T18:53:33.983328614Z"),
				End:        parseTime("2021-07-26T18:53:34.983328614Z"),
			})
		})
		Convey(`Errors`, func() {
			jsonLine := genJSONLine(map[string]string{
				"errors": `[{ "time": "2021-07-26T18:54:38.153491776Z", "file": "dummy.go", "reason": "Failed due to dummy error", "stack": "Dummy Failure" }]`,
			})
			err := r.ConvertFromJSON(strings.NewReader(jsonLine))
			So(err, ShouldBeNil)
			So(r.Cases[0].Errors[0], ShouldResemble, TastError{
				parseTime("2021-07-26T18:54:38.153491776Z"),
				"Failed due to dummy error",
				"dummy.go",
				"Dummy Failure",
			})
		})
	})

	Convey(`ToProtos works`, t, func() {
		ctx := context.Background()
		Convey(`Basic`, func() {
			jsonLine := genJSONLine(map[string]string{})
			r := &TastResults{
				BaseDir: "/usr/local/autotest/results/swarming-55970dfb3e7ef210/1/autoserv_test",
			}
			r.ConvertFromJSON(strings.NewReader(jsonLine))
			got, err := r.ToProtos(ctx, mockCollect)
			So(err, ShouldBeNil)
			So(got[0], ShouldResembleProto, &sinkpb.TestResult{
				TestId:   "lacros.Basic",
				Expected: true,
				Status:   pb.TestStatus_PASS,
				Artifacts: map[string]*sinkpb.Artifact{
					"foo": {
						Body: &sinkpb.Artifact_FilePath{FilePath: "/usr/local/autotest/results/swarming-55970dfb3e7ef210/1/autoserv_test/tast/results/tests/lacros.Basic/foo"},
					},
				},
				TestMetadata: &pb.TestMetadata{
					Name: "lacros.Basic",
				},
				StartTime: timestamppb.New(parseTime("2021-07-26T18:53:33.983328614Z")),
				Duration:  &duration.Duration{Seconds: 1},
			})
		})
		Convey(`Skipped`, func() {
			jsonLine := genJSONLine(map[string]string{
				"skipReason": "dummy skipped",
				"outDir":     "",
			})
			r := &TastResults{
				BaseDir: "/usr/local/autotest/results/swarming-55970dfb3e7ef210/1/autoserv_test",
			}
			r.ConvertFromJSON(strings.NewReader(jsonLine))
			got, err := r.ToProtos(ctx, mockCollect)
			So(err, ShouldBeNil)
			So(got[0], ShouldResembleProto, &sinkpb.TestResult{
				TestId:      "lacros.Basic",
				Expected:    true,
				Status:      pb.TestStatus_SKIP,
				SummaryHtml: "<text-artifact artifact-id=\"Skip Reason\" />",
				Artifacts: map[string]*sinkpb.Artifact{
					"Skip Reason": {
						Body: &sinkpb.Artifact_Contents{
							Contents: []byte("dummy skipped"),
						},
						ContentType: "text/plain",
					},
				},
				TestMetadata: &pb.TestMetadata{
					Name: "lacros.Basic",
				},
			})
		})
		Convey(`Errors`, func() {
			jsonLine := genJSONLine(map[string]string{
				"errors": `[{ "time": "2021-07-26T18:54:38.153491776Z", "file": "dummy.go", "reason": "Failed due to dummy error", "stack": "Dummy Failure" }]`,
			})
			r := &TastResults{
				BaseDir: "/usr/local/autotest/results/swarming-55970dfb3e7ef210/1/autoserv_test",
			}
			r.ConvertFromJSON(strings.NewReader(jsonLine))
			got, err := r.ToProtos(ctx, mockCollect)
			So(err, ShouldBeNil)
			So(got[0].Duration, ShouldResemble, &duration.Duration{Seconds: 1})
			So(got[0], ShouldResembleProto, &sinkpb.TestResult{
				TestId:      "lacros.Basic",
				Expected:    false,
				Status:      pb.TestStatus_FAIL,
				SummaryHtml: "<text-artifact artifact-id=\"Test Log\" />",
				Artifacts: map[string]*sinkpb.Artifact{
					"foo": {
						Body: &sinkpb.Artifact_FilePath{FilePath: "/usr/local/autotest/results/swarming-55970dfb3e7ef210/1/autoserv_test/tast/results/tests/lacros.Basic/foo"},
					},
					"Test Log": {
						Body: &sinkpb.Artifact_Contents{
							Contents: []byte("Dummy Failure\n"),
						},
						ContentType: "text/plain",
					},
				},
				TestMetadata: &pb.TestMetadata{
					Name: "lacros.Basic",
				},
				FailureReason: &pb.FailureReason{
					PrimaryErrorMessage: "Failed due to dummy error",
				},
				StartTime: timestamppb.New(parseTime("2021-07-26T18:53:33.983328614Z")),
				Duration:  &duration.Duration{Seconds: 1},
			})
		})
	})
}
