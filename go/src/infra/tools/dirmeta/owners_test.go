// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dirmeta

import (
	dirmetapb "infra/tools/dirmeta/proto"
	"strings"
	"testing"

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
			So(actual, ShouldResembleProto, &dirmetapb.Metadata{
				TeamEmail: "team-email@chromium.org",
				Os:        dirmetapb.OS_IOS,
				Monorail: &dirmetapb.Monorail{
					Project:   "chromium",
					Component: "Some>Component",
				},
				Wpt: &dirmetapb.WPT{Notify: dirmetapb.Trinary_YES},
			})
		})

		Convey(`ChromeOS`, func() {
			actual, err := parseOwners(strings.NewReader(`# OS: ChromeOS`))
			So(err, ShouldBeNil)
			So(actual.Os, ShouldEqual, dirmetapb.OS_CHROME_OS)
		})

	})
}
