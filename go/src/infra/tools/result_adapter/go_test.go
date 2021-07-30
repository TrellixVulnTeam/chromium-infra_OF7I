// Copyright 2021 The LUCI Authors.
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

package main

import (
	"context"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestEnsureArgsValid(t *testing.T) {
	t.Parallel()

	r := &goRun{}
	Convey(`Does not alter correct command`, t, func() {
		args := strings.Split("go test -json infra/tools/result_adapter", " ")
		validArgs, err := r.ensureArgsValid(args)
		So(err, ShouldBeNil)
		So(validArgs, ShouldResemble, args)
	})
	Convey(`Adds json flag`, t, func() {
		args := strings.Split("go test infra/tools/result_adapter", " ")
		validArgs, err := r.ensureArgsValid(args)
		So(err, ShouldBeNil)
		So(validArgs, ShouldResemble, strings.Split("go test -json infra/tools/result_adapter", " "))
	})
	Convey(`detects bad command`, t, func() {
		args := strings.Split("result_adapter.test -json -v", " ")
		_, err := r.ensureArgsValid(args)
		So(err, ShouldErrLike, "Expected command to be an invocation of `go test`")
	})
}

func TestGenerateTestResults(t *testing.T) {
	t.Parallel()

	r := &goRun{}

	Convey(`parses output`, t, func() {
		trs, err := r.generateTestResults(context.Background(),
			[]byte(`{"Time":"2021-06-17T15:59:10.536706-07:00","Action":"run","Package":"infra/tools/result_adapter","Test":"TestEnsureArgsValid"}
			{"Time":"2021-06-17T15:59:10.537037-07:00","Action":"output","Package":"infra/tools/result_adapter","Test":"TestEnsureArgsValid","Output":"=== RUN   TestEnsureArgsValid\n"}
			{"Time":"2021-06-17T15:59:10.537058-07:00","Action":"output","Package":"infra/tools/result_adapter","Test":"TestEnsureArgsValid","Output":"=== PAUSE TestEnsureArgsValid\n"}
			{"Time":"2021-06-17T15:59:10.537064-07:00","Action":"pause","Package":"infra/tools/result_adapter","Test":"TestEnsureArgsValid"}
			{"Time":"2021-06-17T15:59:10.537178-07:00","Action":"cont","Package":"infra/tools/result_adapter","Test":"TestEnsureArgsValid"}
			{"Time":"2021-06-17T15:59:10.537183-07:00","Action":"output","Package":"infra/tools/result_adapter","Test":"TestEnsureArgsValid","Output":"=== CONT  TestEnsureArgsValid\n"}
			{"Time":"2021-06-17T15:59:10.537309-07:00","Action":"output","Package":"infra/tools/result_adapter","Test":"TestEnsureArgsValid","Output":"--- PASS: TestEnsureArgsValid (0.00s)\n"}
			{"Time":"2021-06-17T15:59:10.537672-07:00","Action":"pass","Package":"infra/tools/result_adapter","Test":"TestEnsureArgsValid","Elapsed":0}
			{"Time":"2021-06-17T15:59:10.540475-07:00","Action":"output","Package":"infra/tools/result_adapter","Output":"PASS\n"}
			{"Time":"2021-06-17T15:59:10.541301-07:00","Action":"output","Package":"infra/tools/result_adapter","Output":"ok  \tinfra/tools/result_adapter\t0.143s\n"}
			{"Time":"2021-06-17T15:59:10.541324-07:00","Action":"pass","Package":"infra/tools/result_adapter","Elapsed":0.143}`),
		)
		So(err, ShouldBeNil)
		So(trs, ShouldHaveLength, 1)
		So(trs[0], ShouldResembleProtoText,
			`test_id:  "infra/tools/result_adapter.TestEnsureArgsValid"
			expected:  true
			status:  PASS
			summary_html:  "<p><text-artifact artifact-id=\"output\"></p>"
			start_time:  {
		  		seconds:  1623970750
		  		nanos:  536706000
			}
			duration:  {}
			artifacts:  {
		  		key:  "output"
		  		value:  {
					contents:  "=== RUN   TestEnsureArgsValid\n=== PAUSE TestEnsureArgsValid\n=== CONT  TestEnsureArgsValid\n--- PASS: TestEnsureArgsValid (0.00s)\nPASS\nok  \tinfra/tools/result_adapter\t0.143s\n"
		  		}
			}`)
	})

	Convey(`parses skipped package`, t, func() {
		trs, err := r.generateTestResults(context.Background(),
			[]byte(`{"Time":"2021-06-17T16:11:01.086366-07:00","Action":"output","Package":"go.chromium.org/luci/resultdb/internal/permissions","Output":"?   \tgo.chromium.org/luci/resultdb/internal/permissions\t[no test files]\n"}
			{"Time":"2021-06-17T16:11:01.086381-07:00","Action":"skip","Package":"go.chromium.org/luci/resultdb/internal/permissions","Elapsed":0}`),
		)
		So(err, ShouldBeNil)
		So(trs, ShouldHaveLength, 0)
	})
}
