// Copyright 2022 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package heuristic

import (
	"context"
	"infra/appengine/gofindit/model"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestExtractSignal(t *testing.T) {
	t.Parallel()
	c := context.Background()
	Convey("Extract Ninja Log", t, func() {
		Convey("No Log Should throw error", func() {
			failureLog := &model.CompileLogs{}
			_, e := ExtractSignals(c, failureLog)
			So(e, ShouldNotBeNil)
		})
		Convey("No output", func() {
			failureLog := &model.CompileLogs{
				NinjaLog: &model.NinjaLog{
					Failures: []*model.NinjaLogFailure{
						{
							Rule:         "CXX",
							OutputNodes:  []string{"n1", "n2"},
							Dependencies: []string{"a/b/../c.d", "../../a.b"},
						},
						{
							Rule:         "CC",
							OutputNodes:  []string{"n3", "n4"},
							Dependencies: []string{"x/y/./z", "d\\e\\f"},
						},
					},
				},
			}
			signal, e := ExtractSignals(c, failureLog)
			So(e, ShouldBeNil)
			So(signal, ShouldResemble, &model.CompileFailureSignal{
				Nodes: []string{"n1", "n2", "n3", "n4"},
				Edges: []*model.CompileFailureEdge{
					{
						Rule:         "CXX",
						OutputNodes:  []string{"n1", "n2"},
						Dependencies: []string{"a/c.d", "a.b"},
					},
					{
						Rule:         "CC",
						OutputNodes:  []string{"n3", "n4"},
						Dependencies: []string{"x/y/z", "d/e/f"},
					},
				},
			})
			Convey("Python patterns", func() {
				failureLog := &model.CompileLogs{
					NinjaLog: &model.NinjaLog{
						Failures: []*model.NinjaLogFailure{
							{
								Rule:         "CXX",
								OutputNodes:  []string{"n1", "n2"},
								Dependencies: []string{"a/b/../c.d", "../../a.b"},
								Output: `
method1 at path/a.py:1
message1
method2 at ../path/b.py:2
message2
method3 at path/a.py:3
message3
blablaError: blabla...

blabla

File "path/a.py", line 14, in method1
message1
File "path/c.py", line 34, in method2
message2
blabalError: blabla...`,
							},
						},
					},
				}
				signal, e := ExtractSignals(c, failureLog)
				So(e, ShouldBeNil)
				So(signal, ShouldResemble, &model.CompileFailureSignal{
					Nodes: []string{"n1", "n2"},
					Edges: []*model.CompileFailureEdge{
						{
							Rule:         "CXX",
							OutputNodes:  []string{"n1", "n2"},
							Dependencies: []string{"a/c.d", "a.b"},
						},
					},
					Files: map[string][]int{
						"path/a.py": {1, 3, 14},
						"path/b.py": {2},
						"path/c.py": {34},
					},
				})
			})

			Convey("NonPython patterns", func() {
				failureLog := &model.CompileLogs{
					NinjaLog: &model.NinjaLog{
						Failures: []*model.NinjaLogFailure{
							{
								Rule:         "CXX",
								OutputNodes:  []string{"obj/a/b/test.c.o"},
								Dependencies: []string{"../../a/b/c.c", "../../a.b"},
								Output: `/b/build/goma/gomacc blabla ... -c ../../a/b/c.c -o obj/a/b/test.c.o
../../a/b/c.c:307:44: error:no member 'kEnableExtensionInfoDialog' ...
Error
../../d.cpp error:Undeclared variable ...
Error
blah blah c:\\a\\b.txt:12 error
Error c:\a\b.txt(123) blah blah
D:\\x\\y.cc[line 456]
//BUILD.gn
1 error generated.`,
							},
						},
					},
				}
				signal, e := ExtractSignals(c, failureLog)
				So(e, ShouldBeNil)
				So(signal, ShouldResemble, &model.CompileFailureSignal{
					Nodes: []string{"obj/a/b/test.c.o"},
					Edges: []*model.CompileFailureEdge{
						{
							Rule:         "CXX",
							OutputNodes:  []string{"obj/a/b/test.c.o"},
							Dependencies: []string{"a/b/c.c", "a.b"},
						},
					},
					Files: map[string][]int{
						"a/b/c.c":    {307},
						"d.cpp":      {},
						"c:/a/b.txt": {12, 123},
						"D:/x/y.cc":  {456},
						"BUILD.gn":   {},
					},
				})
			})
		})
	})
}
