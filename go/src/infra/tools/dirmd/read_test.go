// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dirmd

import (
	"context"
	"path/filepath"
	"testing"

	dirmdpb "infra/tools/dirmd/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestRead(t *testing.T) {
	t.Parallel()

	testDataKey := "go/src/infra/tools/dirmd/testdata"

	Convey(`ReadMapping`, t, func() {
		ctx := context.Background()
		rootKey := testDataKey + "/root"

		Convey(`Original`, func() {
			m, err := ReadMapping(ctx, dirmdpb.MappingForm_ORIGINAL, "testdata/root")
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					rootKey: {
						TeamEmail: "chromium-review@chromium.org",
						Os:        dirmdpb.OS_LINUX,
					},
					rootKey + "/subdir": {
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
					rootKey + "/subdir_with_owners": {
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

		Convey(`Original with two dirs`, func() {
			m, err := ReadMapping(ctx, dirmdpb.MappingForm_ORIGINAL, "testdata/root/subdir", "testdata/root/subdir_with_owners")
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					rootKey + "/subdir": {
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
					rootKey + "/subdir_with_owners": {
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
			m, err := ReadMapping(ctx, dirmdpb.MappingForm_FULL, "testdata/root")
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					rootKey: {
						TeamEmail: "chromium-review@chromium.org",
						Os:        dirmdpb.OS_LINUX,
					},
					rootKey + "/subdir": {
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
					rootKey + "/subdir_with_owners": {
						TeamEmail: "team-email@chromium.org",
						Os:        dirmdpb.OS_LINUX,
						Monorail: &dirmdpb.Monorail{
							Project:   "chromium",
							Component: "Some>Component",
						},
					},
					rootKey + "/subdir_with_owners/empty_subdir": {
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
			m, err := ReadMapping(ctx, dirmdpb.MappingForm_COMPUTED, "testdata/root")
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					rootKey: {
						TeamEmail: "chromium-review@chromium.org",
						Os:        dirmdpb.OS_LINUX,
					},
					rootKey + "/subdir": {
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
					rootKey + "/subdir_with_owners": {
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

		Convey(`Sparse`, func() {
			m, err := ReadMapping(ctx, dirmdpb.MappingForm_SPARSE, "testdata/root/subdir")
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					rootKey + "/subdir": {
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
				},
			})
		})

		Convey(`Reduced`, func() {
			m, err := ReadMapping(ctx, dirmdpb.MappingForm_REDUCED, "testdata/root")
			So(err, ShouldBeNil)
			So(m.Proto(), ShouldResembleProto, &dirmdpb.Mapping{
				Dirs: map[string]*dirmdpb.Metadata{
					rootKey: {
						TeamEmail: "chromium-review@chromium.org",
						Os:        dirmdpb.OS_LINUX,
					},
					rootKey + "/subdir": {
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
					rootKey + "/subdir_with_owners": {
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
}

func TestRemoveRedundantDirs(t *testing.T) {
	t.Parallel()

	Convey("TestRemoveRedundantDirs", t, func() {
		actual := removeRedundantDirs(
			filepath.FromSlash("x/y2/z"),
			filepath.FromSlash("a"),
			filepath.FromSlash("a/b"),
			filepath.FromSlash("x/y1"),
			filepath.FromSlash("x/y2"),
		)
		So(actual, ShouldResemble, []string{
			filepath.FromSlash("a"),
			filepath.FromSlash("x/y1"),
			filepath.FromSlash("x/y2"),
		})
	})
}
