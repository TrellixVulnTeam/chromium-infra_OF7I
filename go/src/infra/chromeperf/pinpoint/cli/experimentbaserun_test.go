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
	"fmt"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCLFlagParsing(t *testing.T) {
	t.Parallel()
	Convey("When provided a valid input", t, func() {
		fs := flag.NewFlagSet("cl-flag-parsing", flag.PanicOnError)
		clFlag := clValue{}
		fs.Var(&clFlag, "cl", "a gerrit CL")
		Convey("/c/<repo>/+/<CL>", func() {
			s := "https://chromium-review.googlesource.com/c/d/d/+/1234"
			So(fs.Parse([]string{"-cl", s}), ShouldBeNil)
			So(clFlag.clNum, ShouldEqual, 1234)
			So(clFlag.patchSet, ShouldEqual, 0)
		})
		Convey("/c/<repo>/+/<CL>/<patch>", func() {
			s := "https://chromium-review.googlesource.com/c/d/d/+/1234/12"
			So(fs.Parse([]string{"-cl", s}), ShouldBeNil)
			So(clFlag.clNum, ShouldEqual, 1234)
			So(clFlag.patchSet, ShouldEqual, 12)
		})
		Convey("/c/<repo>/+/<CL>/<patch> short repo", func() {
			s := "https://chromium-review.googlesource.com/c/d/+/1234/12"
			So(fs.Parse([]string{"-cl", s}), ShouldBeNil)
			So(clFlag.clNum, ShouldEqual, 1234)
			So(clFlag.patchSet, ShouldEqual, 12)
		})
		Convey("/c/<CL>", func() {
			s := "https://chromium-review.googlesource.com/c/1234"
			So(fs.Parse([]string{"-cl", s}), ShouldBeNil)
			So(clFlag.clNum, ShouldEqual, 1234)
			So(clFlag.patchSet, ShouldEqual, 0)
		})
		Convey("/c/<CL>/<patch>", func() {
			s := "https://chromium-review.googlesource.com/c/1234/12"
			So(fs.Parse([]string{"-cl", s}), ShouldBeNil)
			So(clFlag.clNum, ShouldEqual, 1234)
			So(clFlag.patchSet, ShouldEqual, 12)
		})
		Convey("crrev.com", func() {
			s := "https://crrev.com/c/1234/12"
			So(fs.Parse([]string{"-cl", s}), ShouldBeNil)
			So(clFlag.clNum, ShouldEqual, 1234)
			So(clFlag.patchSet, ShouldEqual, 12)
		})
	})
	Convey("When provided some invalid cases", t, func() {
		fs := flag.NewFlagSet("cl-error-flag-parsing", flag.ContinueOnError)
		clFlag := clValue{}
		fs.Var(&clFlag, "cl", "a gerrit CL")
		Convey("<CL>/<patch> without host", func() {
			s := "1234/12"
			So(fs.Parse([]string{"-cl", s}), ShouldNotBeNil)
		})
		Convey("/c/<repo>/+/", func() {
			s := "https://chromium-review.googlesource.com/c/d/d/+"
			So(fs.Parse([]string{"-cl", s}), ShouldNotBeNil)
		})
		Convey("/c/<repo>/+/0/<patch>", func() {
			s := "https://chromium-review.googlesource.com/c/d/d/+/0/01"
			So(fs.Parse([]string{"-cl", s}), ShouldNotBeNil)
		})
		Convey("/c/<repo>/+/<CL>/", func() {
			s := "https://chromium-review.googlesource.com/c/d/d/+/1234/"
			So(fs.Parse([]string{"-cl", s}), ShouldNotBeNil)
		})
		Convey("/c/+/<CL>", func() {
			s := "https://chromium-review.googlesource.com/c/+/1234"
			So(fs.Parse([]string{"-cl", s}), ShouldNotBeNil)
		})
		Convey("<CL>/<patch> wrong host", func() {
			s := "https://crev.com/c/1234/12"
			So(fs.Parse([]string{"-cl", s}), ShouldNotBeNil)
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

func hardcodedCommandOutput(data string) writeGitCLJSON {
	return func(intoFile string) error {
		if err := os.WriteFile(intoFile, []byte(data), 0666); err != nil {
			panic(fmt.Sprintf("unexpected error while writing out fake data to %q: %v", intoFile, err))
		}
		return nil
	}
}

const (
	// The output from running `git cl issue --json $FILE` inside the pinpoint
	// directory as of 2021-03-23.
	infraGitClIssueOutput = `{"gerrit_host": "chromium-review.googlesource.com", "gerrit_project": "infra/infra", "issue_url": null, "issue": null}`
	// Constants to represent the values in infraGitClIssueOutput
	infraGerritHost  = "chromium-review.googlesource.com"
	infraGitilesHost = "chromium.googlesource.com"
	infraRepository  = "infra/infra"

	// The output from running `git cl issue --json $FILE` before
	// https://crrev.com/c/2766153 was applied.
	oldGitClIssueOutput = `{"issue_url": null, "issue": null}`
)

func TestGuessRepositoryDefaults(t *testing.T) {
	t.Parallel()
	Convey("When provided appropriate JSON data", t, func() {
		gitiles, gerrit, repo, err := guessRepositoryDefaults(hardcodedCommandOutput(infraGitClIssueOutput))

		Convey("should return infra information", func() {
			So(err, ShouldBeNil)
			So(gitiles, ShouldEqual, infraGitilesHost)
			So(gerrit, ShouldEqual, infraGerritHost)
			So(repo, ShouldEqual, infraRepository)
		})
	})

	Convey("When provided outdated JSON data", t, func() {
		gitiles, gerrit, repo, err := guessRepositoryDefaults(hardcodedCommandOutput(oldGitClIssueOutput))

		Convey("should return default information", func() {
			So(err, ShouldBeError)
			So(gitiles, ShouldEqual, defaultGitilesHost)
			So(gerrit, ShouldEqual, defaultGerritHost)
			So(repo, ShouldEqual, defaultRepository)
		})
	})

	Convey("When provided invalid JSON data", t, func() {
		gitiles, gerrit, repo, err := guessRepositoryDefaults(hardcodedCommandOutput("invalid json"))

		Convey("should return default information", func() {
			So(err, ShouldBeError)
			So(gitiles, ShouldEqual, defaultGitilesHost)
			So(gerrit, ShouldEqual, defaultGerritHost)
			So(repo, ShouldEqual, defaultRepository)
		})
	})
}
