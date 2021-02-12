// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package git

import (
	"bufio"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestRead(t *testing.T) {
	t.Parallel()

	Convey(`Read`, t, func() {
		parseGraph := func(tokens ...string) *Graph {
			g := &Graph{}
			g.ensureInitialized()
			input := strings.Join(tokens, "\n") + "\n"
			r := &reader{
				r:        bufio.NewReader(strings.NewReader(input)),
				textMode: true,
			}
			err := r.readGraph(g)
			So(err, ShouldBeNil)
			return g
		}

		Convey(`Zero`, func() {
			g := parseGraph(
				"54", // header
				"0",  // version
				"",   // commit hash
				"0",  // number of root commits
				"0",  // number of root children
				"0",  // total number of edges
				"0",  // number of root edges
			)
			So(g.Commit, ShouldResemble, "")
			So(g.root, ShouldResemble, node{name: "//"})
		})

		Convey(`Two direct children`, func() {
			g := parseGraph(
				"54",       // header
				"0",        // version
				"deadbeef", // commit hash

				"0", // root's sumProbDenomiator.
				"2", // number of root children

				"bar", // name of a root child
				"2",   // bar's sumProbDenomiator
				"0",   // number of bar children

				"foo", // name of a root child
				"1",   // foo's sumProbDenomiator
				"0",   // number of foo children

				"2", // total number of edges

				"0", // number of root edges

				"1",        // number of bar edges
				"2",        // index of foo
				"16777216", // probSum for bar->foo

				"1",        // number of foo edges
				"1",        // index bar
				"16777216", // probSum for foo->bar
			)

			So(g.Commit, ShouldResemble, "deadbeef")
			So(g.root, ShouldResemble, node{
				name: "//",
				children: map[string]*node{
					"foo": {
						name:               "//foo",
						parent:             &g.root,
						probSumDenominator: 1,
						copyEdgesOnAppend:  true,
						edges: []edge{{
							to:      g.root.children["bar"],
							probSum: probOne,
						}},
					},
					"bar": {
						name:               "//bar",
						parent:             &g.root,
						probSumDenominator: 2,
						copyEdgesOnAppend:  true,
						edges: []edge{{
							to:      g.root.children["foo"],
							probSum: probOne,
						}},
					},
				},
			})
		})

		Convey(`Descendant name`, func() {
			g := parseGraph(
				"54",       // header
				"0",        // version
				"deadbeef", // commit hash

				"0", // root's probSumDenominator
				"1", // number of root children

				"dir", // name of a root child
				"0",   // dir's probSumDenominator
				"1",   // number of dir children

				"foo", // name of a dir child
				"1",   // foo's probSumDenominator
				"0",   // number of foo children

				"0", // total number of edges

				"0", // number of root edges
				"0", // number of dir edges
				"0", // number of foo edges
			)

			So(g.Commit, ShouldResemble, "deadbeef")
			So(g.root, ShouldResemble, node{
				name: "//",
				children: map[string]*node{
					"dir": {
						name:   "//dir",
						parent: &g.root,
						children: map[string]*node{
							"foo": {
								name:               "//dir/foo",
								parent:             g.node("//dir"),
								probSumDenominator: 1,
							},
						},
					},
				},
			})
		})
	})
}
