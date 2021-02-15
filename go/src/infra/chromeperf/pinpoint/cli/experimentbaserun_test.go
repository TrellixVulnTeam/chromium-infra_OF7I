// Copyright 2020 The Chromium Authors.
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

package cli

import (
	"flag"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCLFlagParsing(t *testing.T) {
	t.Parallel()
	Convey("When provided a valid input", t, func() {
		fs := flag.NewFlagSet("cl-flag-parsing", flag.PanicOnError)
		clFlag := clValue{}
		fs.Var(&clFlag, "cl", "a gerrit CL")
		So(fs.Parse([]string{"-cl", "1234/12"}), ShouldBeNil)
		So(clFlag.clNum, ShouldEqual, 1234)
		So(clFlag.patchSet, ShouldEqual, 12)
	})
	Convey("When provided some invalid cases", t, func() {
		fs := flag.NewFlagSet("cl-error-flag-parsing", flag.ContinueOnError)
		clFlag := clValue{}
		fs.Var(&clFlag, "cl", "a gerrit CL")
		Convey("CL/0", func() {
			err := fs.Parse([]string{"-cl", "1234/0"})
			So(err, ShouldNotBeNil)
		})
		Convey("CL/01", func() {
			err := fs.Parse([]string{"-cl", "1234/01"})
			So(err, ShouldNotBeNil)
		})
		Convey("CL/", func() {
			err := fs.Parse([]string{"-cl", "1234/"})
			So(err, ShouldNotBeNil)
		})
		Convey("0/0", func() {
			err := fs.Parse([]string{"-cl", "0/0"})
			So(err, ShouldNotBeNil)
		})
		Convey("0/", func() {
			err := fs.Parse([]string{"-cl", "0/"})
			So(err, ShouldNotBeNil)
		})
		Convey("0", func() {
			err := fs.Parse([]string{"-cl", "0"})
			So(err, ShouldNotBeNil)
		})
	})
}

func TestBugFlagParsing(t *testing.T) {
	t.Parallel()
	Convey("When provided a valid case", t, func() {
		fs := flag.NewFlagSet("bug-flag-parsing", flag.ContinueOnError)
		bug := &bugValue{}
		fs.Var(bug, "bug", "a Monorail issue in the form <project>:<id>")
		err := fs.Parse([]string{"-bug", "chromium:1234"})
		So(err, ShouldBeNil)
		So(bug.project, ShouldEqual, "chromium")
		So(bug.issueID, ShouldEqual, 1234)
	})
	Convey("When provided some invalid cases", t, func() {
		fs := flag.NewFlagSet("errors-flag-parsing", flag.ContinueOnError)
		bug := &bugValue{}
		fs.Var(bug, "bug", "a Monorail issue in the form <project>:<id>")
		Convey(":<id>", func() {
			So(fs.Parse([]string{"-bug", ":1"}), ShouldNotBeNil)
		})
		Convey("project:0", func() {
			So(fs.Parse([]string{"-bug", "project:0"}), ShouldNotBeNil)
		})
		Convey("project:01", func() {
			So(fs.Parse([]string{"-bug", "project:01"}), ShouldNotBeNil)
		})
	})
}
