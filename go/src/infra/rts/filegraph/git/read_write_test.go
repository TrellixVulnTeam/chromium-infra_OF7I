// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package git

import (
	"bufio"
	"bytes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestReadWrite(t *testing.T) {
	t.Parallel()

	Convey(`ReadWrite`, t, func() {
		test := func(g *Graph) {
			g.ensureInitialized()

			buf := &bytes.Buffer{}
			w := writer{w: buf}
			err := w.writeGraph(g)
			So(err, ShouldBeNil)

			r := reader{r: bufio.NewReader(buf)}
			g2 := &Graph{}
			g2.ensureInitialized()
			err = r.readGraph(g2)
			So(err, ShouldBeNil)

			g2.root.visit(func(n *node) bool {
				n.copyEdgesOnAppend = false
				return true
			})
			So(g, ShouldResemble, g2)
		}

		Convey(`Zero`, func() {
			test(&Graph{})
		})

		Convey(`Two direct children`, func() {
			g := &Graph{
				Commit: "deadbeef",
				root:   node{name: "//"},
			}
			foo := &node{parent: &g.root, name: "//foo", probSumDenominator: 1}
			bar := &node{parent: &g.root, name: "//bar", probSumDenominator: 2}
			foo.edges = []edge{{to: bar, probSum: probOne}}
			bar.edges = []edge{{to: foo, probSum: probOne}}
			g.root.children = map[string]*node{
				"foo": foo,
				"bar": bar,
			}
			test(g)
		})
	})
}
