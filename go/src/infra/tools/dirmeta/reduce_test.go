// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dirmeta

import (
	"testing"

	dirmetapb "infra/tools/dirmeta/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestReduce(t *testing.T) {
	t.Parallel()

	Convey(`Reduce`, t, func() {
		Convey(`Works`, func() {
			input := &Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					".": {
						TeamEmail: "team@example.com",
						Monorail: &dirmetapb.Monorail{
							Project: "chromium",
						},
					},
					"a": {
						TeamEmail: "team@example.com", // redundant
						Wpt:       &dirmetapb.WPT{Notify: dirmetapb.Trinary_YES},
						Monorail: &dirmetapb.Monorail{
							Project:   "chromium", // redundant
							Component: "Component",
						},
					},
				},
			}

			actual := input.Clone()
			actual.Reduce()
			So(actual.Proto(), ShouldResembleProto, &dirmetapb.Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					".": input.Dirs["."], // did not change
					"a": {
						Wpt: &dirmetapb.WPT{Notify: dirmetapb.Trinary_YES},
						Monorail: &dirmetapb.Monorail{
							Component: "Component",
						},
					},
				},
			})
		})

		Convey(`Deep nesting`, func() {
			m := &Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					".":   {TeamEmail: "team@example.com"},
					"a":   {TeamEmail: "team@example.com"},
					"a/b": {TeamEmail: "team@example.com"},
				},
			}
			m.Reduce()
			So(m.Dirs, ShouldNotContainKey, "a")
			So(m.Dirs, ShouldNotContainKey, "a/b")
		})

		Convey(`Nothing to reduce`, func() {
			m := &Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					"a": {TeamEmail: "team@example.com"},
					"b": {TeamEmail: "team@example.com"},
				},
			}
			m.Reduce()
			So(m.Proto(), ShouldResembleProto, &dirmetapb.Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					"a": {TeamEmail: "team@example.com"},
					"b": {TeamEmail: "team@example.com"},
				},
			})
		})
	})
}
