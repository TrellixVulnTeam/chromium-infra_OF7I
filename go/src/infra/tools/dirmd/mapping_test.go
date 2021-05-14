// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dirmd

import (
	"testing"

	dirmdpb "infra/tools/dirmd/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestMerge(t *testing.T) {
	Convey(`Merge`, t, func() {
		Convey(`wpt.notify false value is not overwritten/ignored`, func() {
			inherited := &dirmdpb.Metadata{
				Wpt: &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},
			}
			own := &dirmdpb.Metadata{
				Wpt: &dirmdpb.WPT{Notify: dirmdpb.Trinary_NO},
			}
			Merge(inherited, own)
			So(inherited.Wpt.Notify, ShouldEqual, dirmdpb.Trinary_NO)
		})
	})
}

func TestComputeAll(t *testing.T) {
	t.Parallel()

	Convey(`Nearest ancestor`, t, func() {
		m := &Mapping{
			Dirs: map[string]*dirmdpb.Metadata{
				".": {TeamEmail: "0"},
			},
		}
		So(m.nearestAncestor("a/b/c").TeamEmail, ShouldEqual, "0")
		So(m.nearestAncestor("."), ShouldBeNil)
	})

	Convey(`ComputeAll`, t, func() {
		Convey(`Works`, func() {
			m := &Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": {
						TeamEmail: "team@example.com",
						// Will be inherited entirely.
						Wpt: &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},

						// Will be inherited partially.
						Monorail: &dirmdpb.Monorail{
							Project: "chromium",
						},
					},
					"a": {
						TeamEmail: "team-email@chromium.org",
						Monorail: &dirmdpb.Monorail{
							Component: "Component",
						},
					},
				},
			}
			So(m.ComputeAll(), ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": m.Dirs["."], // did not change
					"a": {
						TeamEmail: "team-email@chromium.org",
						Wpt:       &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Component",
						},
					},
				},
			})
		})

		Convey(`Deep nesting`, func() {
			m := &Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".":   {TeamEmail: "team@example.com"},
					"a":   {},
					"a/b": {},
				},
			}
			So(m.ComputeAll(), ShouldBeNil)
			So(m.Dirs["a/b"].TeamEmail, ShouldEqual, "team@example.com")
		})

		Convey(`No root`, func() {
			input := &Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a": {TeamEmail: "a"},
					"b": {TeamEmail: "b"},
				},
			}

			actual := input.Clone()
			So(actual.ComputeAll(), ShouldBeNil)
			So(input.Proto(), ShouldResembleProto, input.Proto())
		})
	})
}
