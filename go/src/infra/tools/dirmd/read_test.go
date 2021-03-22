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
					"subdir": {
						TeamEmail: "team-email@chromium.org",
						// OS was not inherited
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
						Resultdb: &dirmdpb.ResultDB{
							Tags: []string{
								"feature:read-later",
								"feature:another-one",
							},
						},
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
					"inherit_from": {
						InheritFrom: "//subdir",
						Monorail: &dirmdpb.Monorail{
							Component: "Another>Component",
						},
					},
					"inherit_from/x": {
						TeamEmail: "x@chromium.org",
					},
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
					"subdir": {
						TeamEmail: "team-email@chromium.org",
						Os:        dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
						Resultdb: &dirmdpb.ResultDB{
							Tags: []string{
								"feature:read-later",
								"feature:another-one",
							},
						},
					},
					"subdir/no-own-meta": {
						TeamEmail: "team-email@chromium.org",
						Os:        dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
						Resultdb: &dirmdpb.ResultDB{
							Tags: []string{
								"feature:read-later",
								"feature:another-one",
							},
						},
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
					"inherit_from": {
						InheritFrom: "//subdir",
						TeamEmail:   "team-email@chromium.org",
						Os:          dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Another>Component",
						},
						Resultdb: &dirmdpb.ResultDB{
							Tags: []string{
								"feature:read-later",
								"feature:another-one",
							},
						},
					},
					"inherit_from/x": {
						TeamEmail: "x@chromium.org",
						Os:        dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Another>Component",
						},
						Resultdb: &dirmdpb.ResultDB{
							Tags: []string{
								"feature:read-later",
								"feature:another-one",
							},
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
					"subdir": {
						TeamEmail: "team-email@chromium.org",
						Os:        dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
						Resultdb: &dirmdpb.ResultDB{
							Tags: []string{
								"feature:read-later",
								"feature:another-one",
							},
						},
					},
					"subdir_with_owners": {
						TeamEmail: "team-email@chromium.org",
						Os:        dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
					},
					"inherit_from": {
						InheritFrom: "//subdir",
						TeamEmail:   "team-email@chromium.org",
						Os:          dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Another>Component",
						},
						Resultdb: &dirmdpb.ResultDB{
							Tags: []string{
								"feature:read-later",
								"feature:another-one",
							},
						},
					},
					"inherit_from/x": {
						TeamEmail: "x@chromium.org",
						Os:        dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Another>Component",
						},
						Resultdb: &dirmdpb.ResultDB{
							Tags: []string{
								"feature:read-later",
								"feature:another-one",
							},
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
					"subdir": {
						TeamEmail: "team-email@chromium.org",
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
						Resultdb: &dirmdpb.ResultDB{
							Tags: []string{
								"feature:read-later",
								"feature:another-one",
							},
						},
					},
					"subdir_with_owners": {
						TeamEmail: "team-email@chromium.org",
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
					},
					"inherit_from": {
						InheritFrom: "//subdir",
						Monorail: &dirmdpb.Monorail{
							Component: "Another>Component",
						},

						Resultdb: &dirmdpb.ResultDB{
							// This is a bug. These tags shouldn't be here.
							// TODO(crbug.com/1188314): remove this.
							Tags: []string{
								"feature:read-later",
								"feature:another-one",
							},
						},
					},
					"inherit_from/x": {
						TeamEmail: "x@chromium.org",
						Resultdb: &dirmdpb.ResultDB{
							// This is a bug. These tags shouldn't be here.
							// TODO(crbug.com/1188314): remove this.
							Tags: []string{
								"feature:read-later",
								"feature:another-one",
							},
						},
					},
				},
			})
		})
	})

	Convey(`ReadComputed`, t, func() {
		Convey(`a, a/b`, func() {
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
		Convey(`inherit_from`, func() {
			m, err := ReadComputed("testdata/inheritance", "testdata/inheritance/inherit_from_ab")
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"inherit_from_ab": {
						InheritFrom: "//a/b",
						TeamEmail:   "chromium-review@chromium.org",
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Component>Child",
						},
						Os: dirmdpb.OS_LINUX,
					},
				},
			})
		})
		Convey(`no_inheritance`, func() {
			m, err := ReadComputed("testdata/inheritance", "testdata/inheritance/a/b/no_inheritance")
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					"a/b/no_inheritance": {
						InheritFrom: "-",
						Monorail: &dirmdpb.Monorail{
							Component: "No>Inheritance",
						},
					},
				},
			})
		})
	})
}
