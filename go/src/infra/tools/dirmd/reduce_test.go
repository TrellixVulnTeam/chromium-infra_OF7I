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

func TestReduce(t *testing.T) {
	t.Parallel()

	Convey(`Reduce`, t, func() {
		Convey(`Works`, func() {
			input := &Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": {
						TeamEmail: "team@example.com",
						Monorail: &dirmdpb.Monorail{
							Project: "chromium",
						},
					},
					"a": {
						TeamEmail: "team@example.com", // redundant
						Wpt:       &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium", // redundant
							Component: "Component",
						},
					},
				},
			}

			actual := input.Clone()
			actual.Reduce()
			So(actual.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": input.Dirs["."], // did not change
					"a": {
						Wpt: &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},
						Monorail: &dirmdpb.Monorail{
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
				Dirs: map[string]*dirmdpb.Metadata{
					"a": {TeamEmail: "team@example.com"},
					"b": {TeamEmail: "team@example.com"},
				},
			}
			m.Reduce()
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a": {TeamEmail: "team@example.com"},
					"b": {TeamEmail: "team@example.com"},
				},
			})
		})
	})
}
