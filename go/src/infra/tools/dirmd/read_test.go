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

func TestRead(t *testing.T) {
	t.Parallel()

	Convey(`ReadMapping`, t, func() {
		Convey(`Original`, func() {
			m, err := ReadMapping("testdata/root", dirmdpb.MappingForm_ORIGINAL)
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": {
						TeamEmail: "chromium-review@chromium.org",
						Os:        dirmdpb.OS_LINUX,
					},
					"subdir_with_owners": {
						TeamEmail: "team-email@chromium.org",
						// OS was not inherited
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
					},
					// "subdir_with_owners/empty_subdir" is not present because it has
					// no metadata.
				},
			})
		})

		Convey(`Full`, func() {
			m, err := ReadMapping("testdata/root", dirmdpb.MappingForm_FULL)
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": {
						TeamEmail: "chromium-review@chromium.org",
						Os:        dirmdpb.OS_LINUX,
					},
					"subdir_with_owners": {
						TeamEmail: "team-email@chromium.org",
						Os:        dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
					},
					"subdir_with_owners/empty_subdir": {
						TeamEmail: "team-email@chromium.org",
						Os:        dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
					},
				},
			})
		})

		Convey(`Computed`, func() {
			m, err := ReadMapping("testdata/root", dirmdpb.MappingForm_COMPUTED)
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": {
						TeamEmail: "chromium-review@chromium.org",
						Os:        dirmdpb.OS_LINUX,
					},
					"subdir_with_owners": {
						TeamEmail: "team-email@chromium.org",
						Os:        dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
					},
				},
			})
		})

		Convey(`Reduced`, func() {
			m, err := ReadMapping("testdata/root", dirmdpb.MappingForm_REDUCED)
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					".": {
						TeamEmail: "chromium-review@chromium.org",
						Os:        dirmdpb.OS_LINUX,
					},
					"subdir_with_owners": {
						TeamEmail: "team-email@chromium.org",
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
					},
				},
			})
		})
	})

	Convey(`ReadComputed`, t, func() {
		m, err := ReadComputed("testdata/inheritance", "testdata/inheritance/a", "testdata/inheritance/a/b")
		So(err, ShouldBeNil)
		So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
			Dirs: map[string]*dirmdpb.Metadata{
				"a": {
					TeamEmail: "chromium-review@chromium.org",
					Monorail: &dirmdpb.Monorail{
						Project:   "chromium",
						Component: "Component",
					},
				},
				"a/b": {
					TeamEmail: "chromium-review@chromium.org",
					Monorail: &dirmdpb.Monorail{
						Project:   "chromium",
						Component: "Component>Child",
					},
				},
			},
		})
	})
}
