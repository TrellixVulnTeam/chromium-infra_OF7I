// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dirmd

import (
	"strings"
	"testing"

	dirmdpb "infra/tools/dirmd/proto"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestParseOwners(t *testing.T) {
	t.Parallel()

	Convey(`ParseOwners`, t, func() {
		Convey(`Works`, func() {
			actual, filtered, err := parseOwners(strings.NewReader(`
# TEAM: team-email@chromium.org
someone@example.com

# Some comments

# OS: iOS
# COMPONENT: Some>Component
# Internal Component: b/components/1234
# WPT-NOTIFY: true`))
			So(err, ShouldBeNil)
			So(filtered, ShouldResemble, []string{
				"",
				"someone@example.com",
				"",
				"# Some comments",
				"",
			})
			So(actual, ShouldResembleProto, &dirmdpb.Metadata{
				TeamEmail: "team-email@chromium.org",
				Os:        dirmdpb.OS_IOS,
				Monorail: &dirmdpb.Monorail{
					Project:   "chromium",
					Component: "Some>Component",
				},
				Wpt: &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},
				Buganizer: &dirmdpb.Buganizer{
					ComponentId: 1234,
				},
			})
		})

		Convey(`ChromeOS`, func() {
			actual, filtered, err := parseOwners(strings.NewReader(`# OS: ChromeOS`))
			So(err, ShouldBeNil)
			So(filtered, ShouldEqual, []string(nil))
			So(actual.Os, ShouldEqual, dirmdpb.OS_CHROME_OS)
		})

	})
}
