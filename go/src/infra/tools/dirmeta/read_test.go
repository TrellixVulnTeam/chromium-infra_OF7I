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

func TestMappingReader(t *testing.T) {
	t.Parallel()

	Convey(`MappingReader`, t, func() {
		r := &MappingReader{
			Root: "testdata/root",
		}

		Convey(`Original`, func() {
			err := r.ReadAll(dirmetapb.MappingForm_ORIGINAL)
			So(err, ShouldBeNil)
			So(r.Mapping.Proto(), ShouldResembleProto, &dirmetapb.Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					".": {
						TeamEmail: "chromium-review@chromium.org",
						Os:        dirmetapb.OS_LINUX,
					},
					"subdir_with_owners": {
						TeamEmail: "team-email@chromium.org",
						// OS was not inherited
						Monorail: &dirmetapb.Monorail{
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
			err := r.ReadAll(dirmetapb.MappingForm_FULL)
			So(err, ShouldBeNil)
			So(r.Mapping.Proto(), ShouldResembleProto, &dirmetapb.Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					".": {
						TeamEmail: "chromium-review@chromium.org",
						Os:        dirmetapb.OS_LINUX,
					},
					"subdir_with_owners": {
						TeamEmail: "team-email@chromium.org",
						Os:        dirmetapb.OS_LINUX,
						Monorail: &dirmetapb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
					},
					"subdir_with_owners/empty_subdir": {
						TeamEmail: "team-email@chromium.org",
						Os:        dirmetapb.OS_LINUX,
						Monorail: &dirmetapb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
					},
				},
			})
		})

		Convey(`Computed`, func() {
			err := r.ReadAll(dirmetapb.MappingForm_COMPUTED)
			So(err, ShouldBeNil)
			So(r.Mapping.Proto(), ShouldResembleProto, &dirmetapb.Mapping{
				Dirs: map[string]*dirmetapb.Metadata{
					".": {
						TeamEmail: "chromium-review@chromium.org",
						Os:        dirmetapb.OS_LINUX,
					},
					"subdir_with_owners": {
						TeamEmail: "team-email@chromium.org",
						Os:        dirmetapb.OS_LINUX,
						Monorail: &dirmetapb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
					},
				},
			})

			Convey(`Reduced`, func() {
				err := r.ReadAll(dirmetapb.MappingForm_REDUCED)
				So(err, ShouldBeNil)
				So(r.Mapping.Proto(), ShouldResembleProto, &dirmetapb.Mapping{
					Dirs: map[string]*dirmetapb.Metadata{
						".": {
							TeamEmail: "chromium-review@chromium.org",
							Os:        dirmetapb.OS_LINUX,
						},
						"subdir_with_owners": {
							TeamEmail: "team-email@chromium.org",
							Monorail: &dirmetapb.Monorail{
								Project:   "chromium",
								Component: "Some>Component",
							},
						},
					},
				})
			})
		})
	})
}
