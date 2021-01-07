// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dirmd

import (
	"testing"

	dirmdpb "infra/tools/dirmd/proto"

	"go.chromium.org/luci/resultdb/pbutil"
	sinkpb "go.chromium.org/luci/resultdb/sink/proto/v1"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestLocationTag(t *testing.T) {
	t.Parallel()

	Convey(`ToLocationTags`, t, func() {
		mapping := &dirmdpb.Mapping{
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
			},
		}
		tags, err := ToLocationTags((*Mapping)(mapping))
		for _, dir := range tags.Dirs {
			pbutil.SortStringPairs(dir.Tags)
		}

		expected := &sinkpb.LocationTags_Repo{
			Dirs: map[string]*sinkpb.LocationTags_Dir{
				".": {
					Tags: pbutil.StringPairs(
						"os", dirmdpb.OS_LINUX.String(),
						"team_email", "chromium-review@chromium.org"),
				},
				"subdir": {
					Tags: pbutil.StringPairs(
						"feature", "another-one",
						"feature", "read-later",
						"monorail_component", "Some>Component",
						"team_email", "team-email@chromium.org"),
				},
				"subdir_with_owners": {
					Tags: pbutil.StringPairs(
						"monorail_component", "Some>Component",
						"team_email", "team-email@chromium.org"),
				},
			},
		}

		So(err, ShouldBeNil)
		So(tags, ShouldResembleProto, expected)
	})
}
