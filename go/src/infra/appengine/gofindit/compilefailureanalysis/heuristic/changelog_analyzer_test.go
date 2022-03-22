// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristic

import (
	"context"
	gfim "infra/appengine/gofindit/model"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestChangeLogAnalyzer(t *testing.T) {
	t.Parallel()

	Convey("AreRelelatedExtensions", t, func() {
		So(AreRelelatedExtensions("c", "cpp"), ShouldBeTrue)
		So(AreRelelatedExtensions("py", "pyc"), ShouldBeTrue)
		So(AreRelelatedExtensions("gyp", "gypi"), ShouldBeTrue)
		So(AreRelelatedExtensions("c", "py"), ShouldBeFalse)
		So(AreRelelatedExtensions("abc", "xyz"), ShouldBeFalse)
	})

	Convey("NormalizeObjectFilePath", t, func() {
		data := map[string]string{
			"obj/a/T.x.o":   "a/x.o",
			"obj/a/T.x.y.o": "a/x.y.o",
			"x.o":           "x.o",
			"obj/a/x.obj":   "a/x.obj",
			"a.cc.obj":      "a.cc.obj",
			"T.a.c.o":       "a.c.o",
			"T.a.o":         "a.o",
			"T.a.b.c":       "T.a.b.c",
		}
		for k, v := range data {
			So(NormalizeObjectFilePath(k), ShouldEqual, v)
		}
	})

	Convey("StripExtensionAndCommonSuffix", t, func() {
		data := map[string]string{
			"a_file_impl_mac_test.cc": "a_file",
			"src/b_file_x11_ozone.h":  "src/b_file",
			"c_file.cc":               "c_file",
		}
		for k, v := range data {
			So(StripExtensionAndCommonSuffix(k), ShouldEqual, v)
		}
	})

	Convey("AnalyzeOneChangeLog", t, func() {
		c := context.Background()
		signal := &gfim.CompileFailureSignal{
			Files: map[string][]int{
				"src/a/b/x.cc":       {27},
				"obj/content/util.o": {},
			},
			Edges: []*gfim.CompileFailureEdge{
				{
					Dependencies: []string{
						"x/y/aa_impl_mac.cc",
					},
				},
			},
		}
		Convey("Changelog from a non-blamable email", func() {
			cl := &gfim.ChangeLog{
				Author: gfim.ChangeLogActor{
					Email: "chrome-release-bot@chromium.org",
				},
			}

			justification, err := AnalyzeOneChangeLog(c, signal, cl)
			So(err, ShouldBeNil)
			So(justification, ShouldResemble, &gfim.SuspectJustification{IsNonBlamable: true})
		})

		Convey("Changelog did not touch any file", func() {
			cl := &gfim.ChangeLog{
				ChangeLogDiffs: []gfim.ChangeLogDiff{
					{
						Type:    gfim.ChangeType_ADD,
						NewPath: "some_file.cc",
					},
				},
			}
			justification, err := AnalyzeOneChangeLog(c, signal, cl)
			So(err, ShouldBeNil)
			So(justification, ShouldResemble, &gfim.SuspectJustification{})
		})

		Convey("Changelog touched relevant files", func() {
			cl := &gfim.ChangeLog{
				ChangeLogDiffs: []gfim.ChangeLogDiff{
					{
						Type:    gfim.ChangeType_MODIFY,
						NewPath: "content/util.c",
					},
					{
						Type:    gfim.ChangeType_ADD,
						NewPath: "dir/a/b/x.cc",
					},
					{
						Type:    gfim.ChangeType_RENAME,
						OldPath: "unrelated_file_1.cc",
						NewPath: "unrelated_file_2.cc",
					},
					{
						Type:    gfim.ChangeType_DELETE,
						OldPath: "x/y/aa.h",
					},
				},
			}
			justification, err := AnalyzeOneChangeLog(c, signal, cl)
			So(err, ShouldBeNil)
			So(justification, ShouldResemble, &gfim.SuspectJustification{
				Items: []*gfim.SuspectJustificationItem{
					{
						Score:    5,
						FilePath: "dir/a/b/x.cc",
						Reason:   `The file "dir/a/b/x.cc" was added and it was in the failure log.`,
					},
					{
						Score:    2,
						FilePath: "content/util.c",
						Reason:   "The file \"content/util.c\" was modified. It was related to the file obj/content/util.o which was in the failure log.",
					},
					{
						Score:    1,
						FilePath: "x/y/aa.h",
						Reason:   "The file \"x/y/aa.h\" was deleted. It was related to the file x/y/aa_impl_mac.cc which was in the failure log.",
					},
				},
			})
		})
	})

	Convey("AnalyzeChangeLogs", t, func() {
		c := context.Background()
		signal := &gfim.CompileFailureSignal{
			Files: map[string][]int{
				"src/a/b/x.cc":       {27},
				"obj/content/util.o": {},
			},
		}

		Convey("Results should be sorted", func() {
			cls := []*gfim.ChangeLog{
				{
					Commit:  "abcd",
					Message: "blah blah\nReviewed-on: https://chromium-review.googlesource.com/c/chromium/src/+/123\n bla",
					ChangeLogDiffs: []gfim.ChangeLogDiff{
						{
							Type:    gfim.ChangeType_MODIFY,
							NewPath: "content/util.c",
						},
					},
				},
				{
					Commit:  "efgh",
					Message: "blah blah\nReviewed-on: https://chromium-review.googlesource.com/c/chromium/src/+/456\n bla",
					ChangeLogDiffs: []gfim.ChangeLogDiff{
						{
							Type:    gfim.ChangeType_RENAME,
							OldPath: "unrelated_file_1.cc",
							NewPath: "unrelated_file_2.cc",
						},
					},
				},
				{
					Commit:  "wxyz",
					Message: "blah blah\nReviewed-on: https://chromium-review.googlesource.com/c/chromium/src/+/789\n bla",
					ChangeLogDiffs: []gfim.ChangeLogDiff{
						{
							Type:    gfim.ChangeType_ADD,
							NewPath: "dir/a/b/x.cc",
						},
					},
				},
			}

			analysisResult, err := AnalyzeChangeLogs(c, signal, cls)
			So(err, ShouldBeNil)
			So(analysisResult, ShouldResemble, &gfim.HeuristicAnalysisResult{
				Items: []*gfim.HeuristicAnalysisResultItem{
					{
						Commit:    "wxyz",
						ReviewUrl: "https://chromium-review.googlesource.com/c/chromium/src/+/789",
						Justification: &gfim.SuspectJustification{
							Items: []*gfim.SuspectJustificationItem{
								{
									Score:    5,
									FilePath: "dir/a/b/x.cc",
									Reason:   `The file "dir/a/b/x.cc" was added and it was in the failure log.`,
								},
							},
						},
					},
					{
						Commit:    "abcd",
						ReviewUrl: "https://chromium-review.googlesource.com/c/chromium/src/+/123",
						Justification: &gfim.SuspectJustification{
							Items: []*gfim.SuspectJustificationItem{
								{
									Score:    2,
									FilePath: "content/util.c",
									Reason:   "The file \"content/util.c\" was modified. It was related to the file obj/content/util.o which was in the failure log.",
								},
							},
						},
					},
				},
			})
		})
	})

}
