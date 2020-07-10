// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dirmeta

import (
	dirmetapb "infra/tools/dirmeta/proto"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestExpand(t *testing.T) {
	t.Parallel()

	Convey(`Expand`, t, func() {
		Convey(`Works`, func() {
			input := &Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					".": {
						TeamEmail: "team@example.com",
						// Will be inherited entirely.
						Wpt: &dirmetapb.WPT{Notify: true},

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
			actual := input.Expand()
			So(actual.Proto(), ShouldResembleProto, &dirmetapb.Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					".": input.Dirs["."], // did not change
					"a": {
						TeamEmail: "team-email@chromium.org",
						Wpt:       &dirmetapb.WPT{Notify: true},
						Monorail: &dirmetapb.Monorail{
							Project:   "chromium",
							Component: "Component",
						},
					},
				},
			})
		})

		Convey(`Deep nesting`, func() {
			input := &Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					".":   {TeamEmail: "team@example.com"},
					"a":   {},
					"a/b": {},
				},
			}
			actual := input.Expand()
			So(actual.Dirs["a/b"].TeamEmail, ShouldEqual, "team@example.com")
		})

		Convey(`No root`, func() {
			input := &Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					"a": {TeamEmail: "a"},
					"b": {TeamEmail: "b"},
				},
			}
			actual := input.Expand()
			So(actual.Proto(), ShouldResembleProto, input.Proto())
		})
	})
}
