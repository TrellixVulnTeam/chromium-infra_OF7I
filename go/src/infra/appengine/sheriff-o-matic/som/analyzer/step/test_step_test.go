// Copyright 2016 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package step

import (
	"net/url"
	"testing"

	"infra/monitoring/messages"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetTestSuite(t *testing.T) {
	Convey("GetTestSuite", t, func() {
		s := &messages.BuildStep{
			Step: &messages.Step{
				Name: "thing_tests",
			},
		}
		url, err := url.Parse("https://build.chromium.org/p/chromium.linux")
		So(err, ShouldBeNil)
		s.Master = &messages.MasterLocation{
			URL: *url,
		}
		Convey("basic", func() {
			So(GetTestSuite(s), ShouldEqual, "thing_tests")
		})
		Convey("with suffixes", func() {
			s.Step.Name = "thing_tests on Intel GPU on Linux"
			So(GetTestSuite(s), ShouldEqual, "thing_tests")
		})
		Convey("on perf", func() {
			url, err = url.Parse("https://build.chromium.org/p/chromium.perf")
			So(err, ShouldBeNil)
			s.Master = &messages.MasterLocation{
				URL: *url,
			}
			s.Step.Logs = [][]interface{}{
				{
					"chromium_swarming.summary",
					"foo",
				},
			}
			Convey("with suffixes", func() {
				s.Step.Name = "battor.power_cases on Intel GPU on Linux"
				So(GetTestSuite(s), ShouldEqual, "battor.power_cases")
			})
			Convey("C++ test with suffixes", func() {
				s.Step.Name = "performance_browser_tests on Intel GPU on Linux"
				So(GetTestSuite(s), ShouldEqual, "performance_browser_tests")
			})
			Convey("not a test", func() {
				s.Step.Logs = nil
				s.Step.Name = "something_random"
				So(GetTestSuite(s), ShouldEqual, "")
			})
		})
	})
}
