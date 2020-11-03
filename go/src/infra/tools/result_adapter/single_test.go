// Copyright 2020 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package main

import (
	"context"
	"strings"
	"testing"

	pb "go.chromium.org/luci/resultdb/proto/v1"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestSingleConversions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	Convey(`From JSON works`, t, func() {
		str := `{
				"failures": [
					"Failed to run content_shell."
				],
				"valid": true
			}`

		results := &SingleResult{}
		err := results.ConvertFromJSON(strings.NewReader(str))
		So(err, ShouldBeNil)
		So(results.Failures, ShouldResemble, []string{"Failed to run content_shell."})
		So(results.Valid, ShouldBeTrue)
	})

	Convey(`ToProtos`, t, func() {
		Convey("test passes", func() {
			results := &SingleResult{
				Failures: []string{},
				Valid:    true,
			}

			testResults, err := results.ToProtos(ctx)
			So(err, ShouldBeNil)

			expected := &sinkpb.TestResult{
				TestId:   "",
				Expected: true,
				Status:   pb.TestStatus_PASS,
			}
			So(testResults, ShouldHaveLength, 1)
			So(testResults[0], ShouldResembleProto, expected)
		})

		Convey("test fails", func() {
			results := &SingleResult{
				Failures: []string{"Failed to run content_shell."},
				Valid:    true,
			}

			testResults, err := results.ToProtos(ctx)
			So(err, ShouldBeNil)

			expected := &sinkpb.TestResult{
				TestId:      "",
				Expected:    false,
				Status:      pb.TestStatus_FAIL,
				SummaryHtml: "<pre>Failed to run content_shell.</pre>",
			}
			So(testResults, ShouldHaveLength, 1)
			So(testResults[0], ShouldResembleProto, expected)
		})

		Convey("test result invalid", func() {
			results := &SingleResult{
				Failures: []string{},
				Valid:    false,
			}

			testResults, err := results.ToProtos(ctx)
			So(err, ShouldBeNil)

			expected := &sinkpb.TestResult{
				TestId:   "",
				Expected: false,
				Status:   pb.TestStatus_ABORT,
			}
			So(testResults, ShouldHaveLength, 1)
			So(testResults[0], ShouldResembleProto, expected)
		})
	})
}
