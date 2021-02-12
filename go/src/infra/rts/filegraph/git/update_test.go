// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package git

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestApply(t *testing.T) {
	t.Parallel()

	Convey(`apply`, t, func() {
		g := &Graph{}
		g.ensureInitialized()

		applyChanges := func(changes []fileChange) {
			err := g.apply(changes, 100)
			So(err, ShouldBeNil)
		}

		Convey(`Empty change`, func() {
			applyChanges(nil)
			So(g.root, ShouldResemble, node{name: "//"})
		})

		Convey(`Add one file`, func() {
			applyChanges([]fileChange{
				{Path: "a", Status: 'A'},
			})
			// The file is registered, but the commit is otherwise ignored.
			So(g.root, ShouldResemble, node{
				name: "//",
				children: map[string]*node{
					"a": {
						name:   "//a",
						parent: &g.root,
					},
				},
			})
		})

		Convey(`Add two files`, func() {
			applyChanges([]fileChange{
				{Path: "a", Status: 'A'},
				{Path: "b", Status: 'A'},
			})
			So(g.root, ShouldResemble, node{
				name: "//",
				children: map[string]*node{
					"a": {
						name:               "//a",
						parent:             &g.root,
						probSumDenominator: 1,
						edges:              []edge{{to: g.node("//b"), probSum: probOne}},
					},
					"b": {
						name:               "//b",
						parent:             &g.root,
						probSumDenominator: 1,
						edges:              []edge{{to: g.node("//a"), probSum: probOne}},
					},
				},
			})

			Convey(`Add two more`, func() {
				applyChanges([]fileChange{
					{Path: "b", Status: 'A'},
					{Path: "c/d", Status: 'A'},
				})
				So(g.root, ShouldResemble, node{
					name: "//",
					children: map[string]*node{
						"a": {
							name:               "//a",
							parent:             &g.root,
							probSumDenominator: 1,
							edges:              []edge{{to: g.node("//b"), probSum: probOne}},
						},
						"b": {
							name:               "//b",
							parent:             &g.root,
							probSumDenominator: 2,
							edges: []edge{
								{to: g.node("//a"), probSum: probOne},
								{to: g.node("//c/d"), probSum: probOne},
							},
						},
						"c": {
							name:   "//c",
							parent: &g.root,
							children: map[string]*node{
								"d": {
									name:               "//c/d",
									parent:             g.node("//c"),
									probSumDenominator: 1,
									edges:              []edge{{to: g.node("//b"), probSum: probOne}},
								},
							},
						},
					},
				})
			})

			Convey(`Modify them again`, func() {
				applyChanges([]fileChange{
					{Path: "a", Status: 'M'},
					{Path: "b", Status: 'M'},
				})
				So(g.root, ShouldResemble, node{
					name: "//",
					children: map[string]*node{
						"a": {
							name:               "//a",
							parent:             &g.root,
							probSumDenominator: 2,
							edges:              []edge{{to: g.node("//b"), probSum: 2 * probOne}},
						},
						"b": {
							name:               "//b",
							parent:             &g.root,
							probSumDenominator: 2,
							edges:              []edge{{to: g.node("//a"), probSum: 2 * probOne}},
						},
					},
				})

			})

			Convey(`Modify one and add another`, func() {
				applyChanges([]fileChange{
					{Path: "b", Status: 'M'},
					{Path: "c", Status: 'M'},
				})
				So(g.root, ShouldResemble, node{
					name: "//",
					children: map[string]*node{
						"a": {
							name:               "//a",
							parent:             &g.root,
							probSumDenominator: 1,
							edges:              []edge{{to: g.node("//b"), probSum: probOne}},
						},
						"b": {
							name:               "//b",
							parent:             &g.root,
							probSumDenominator: 2,
							edges: []edge{
								{to: g.node("//a"), probSum: probOne},
								{to: g.node("//c"), probSum: probOne},
							},
						},
						"c": {
							name:               "//c",
							parent:             &g.root,
							probSumDenominator: 1,
							edges:              []edge{{to: g.node("//b"), probSum: probOne}},
						},
					},
				})
			})

			Convey(`Rename one`, func() {
				applyChanges([]fileChange{
					{Path: "b", Path2: "c", Status: 'R'},
				})
				So(g.root, ShouldResemble, node{
					name: "//",
					children: map[string]*node{
						"a": {
							name:               "//a",
							parent:             &g.root,
							probSumDenominator: 1,
							edges:              []edge{{to: g.node("//b"), probSum: probOne}},
						},
						"b": {
							name:               "//b",
							parent:             &g.root,
							probSumDenominator: 1,
							edges: []edge{
								{to: g.node("//a"), probSum: probOne},
								{to: g.node("//c")},
							},
						},
						"c": {
							name:   "//c",
							parent: &g.root,
							edges:  []edge{{to: g.node("//b")}},
						},
					},
				})
			})

			Convey(`Remove one`, func() {
				applyChanges([]fileChange{
					{Path: "b", Status: 'D'},
				})
				So(g.root, ShouldResemble, node{
					name: "//",
					children: map[string]*node{
						"a": {
							name:               "//a",
							parent:             &g.root,
							probSumDenominator: 1,
							edges:              []edge{{to: g.node("//b"), probSum: probOne}},
						},
						"b": {
							name:               "//b",
							parent:             &g.root,
							probSumDenominator: 1,
							edges:              []edge{{to: g.node("//a"), probSum: probOne}},
						},
					},
				})
			})
		})

		Convey(`Great migration`, func() {
			addFiles := make([]fileChange, 1000)
			for i := range addFiles {
				addFiles[i] = fileChange{Path: fmt.Sprintf("%d", i), Status: 'A'}
			}
			applyChanges(addFiles)

			greatMigration := make([]fileChange, len(addFiles))
			for i, add := range addFiles {
				greatMigration[i] = fileChange{
					Path:   add.Path,
					Path2:  "new/" + add.Path,
					Status: 'R',
				}
			}
			applyChanges(greatMigration)

			old54 := g.node("//54")
			new54 := g.node("//new/54")
			So(new54, ShouldNotBeNil)
			So(new54.edges, ShouldResemble, []edge{{to: old54}})
		})
	})
}
