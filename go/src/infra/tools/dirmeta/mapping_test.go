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

func TestMerge(t *testing.T) {
	Convey(`Merge`, t, func() {
		Convey(`wpt.notify false value is not overwritten/ignored`, func() {
			inherited := &dirmetapb.Metadata{
				Wpt: &dirmetapb.WPT{Notify: dirmetapb.Trinary_YES},
			}
			own := &dirmetapb.Metadata{
				Wpt: &dirmetapb.WPT{Notify: dirmetapb.Trinary_NO},
			}
			Merge(inherited, own)
			So(inherited.Wpt.Notify, ShouldEqual, dirmetapb.Trinary_NO)
		})
	})
}

func TestComputeAll(t *testing.T) {
	t.Parallel()

	Convey(`Nearest ancestor`, t, func() {
		m := &Mapping{
			Dirs: map[string]*dirmetapb.Metadata{
				".": {TeamEmail: "0"},
			},
		}
		So(m.nearestAncestor("a/b/c").TeamEmail, ShouldEqual, "0")
		So(m.nearestAncestor("."), ShouldBeNil)
	})

	Convey(`ComputeAll`, t, func() {
		Convey(`Works`, func() {
			m := &Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					".": {
						TeamEmail: "team@example.com",
						// Will be inherited entirely.
						Wpt: &dirmetapb.WPT{Notify: dirmetapb.Trinary_YES},

						// Will be inherited partially.
						Monorail: &dirmetapb.Monorail{
							Project: "chromium",
						},
					},
					"a": {
						TeamEmail: "team-email@chromium.org",
						Monorail: &dirmetapb.Monorail{
							Component: "Component",
						},
					},
				},
			}
			m.ComputeAll()
			So(m.Proto(), ShouldResembleProto, &dirmetapb.Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					".": m.Dirs["."], // did not change
					"a": {
						TeamEmail: "team-email@chromium.org",
						Wpt:       &dirmetapb.WPT{Notify: dirmetapb.Trinary_YES},
						Monorail: &dirmetapb.Monorail{
							Project:   "chromium",
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
					"a":   {},
					"a/b": {},
				},
			}
			m.ComputeAll()
			So(m.Dirs["a/b"].TeamEmail, ShouldEqual, "team@example.com")
		})

		Convey(`No root`, func() {
			input := &Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					"a": {TeamEmail: "a"},
					"b": {TeamEmail: "b"},
				},
			}

			actual := input.Clone()
			actual.ComputeAll()
			So(input.Proto(), ShouldResembleProto, input.Proto())
		})
	})
}
