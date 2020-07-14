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
			actual, err := parseOwners(strings.NewReader(`
someone@example.com

# Some comments

# TEAM: team-email@chromium.org
# OS: iOS
# COMPONENT: Some>Component
# WPT-NOTIFY: true
			`))
			So(err, ShouldBeNil)
			So(actual, ShouldResembleProto, &dirmdpb.Metadata{
				TeamEmail: "team-email@chromium.org",
				Os:        dirmdpb.OS_IOS,
				Monorail: &dirmdpb.Monorail{
					Project:   "chromium",
					Component: "Some>Component",
				},
				Wpt: &dirmdpb.WPT{Notify: dirmdpb.Trinary_YES},
			})
		})

		Convey(`ChromeOS`, func() {
			actual, err := parseOwners(strings.NewReader(`# OS: ChromeOS`))
			So(err, ShouldBeNil)
			So(actual.Os, ShouldEqual, dirmdpb.OS_CHROME_OS)
		})

	})
}
