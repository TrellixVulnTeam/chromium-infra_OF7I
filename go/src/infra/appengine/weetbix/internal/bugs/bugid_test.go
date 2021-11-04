// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package bugs

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestValidate(t *testing.T) {
	Convey(`Validate`, t, func() {
		id := BugID{
			System: "monorail",
			ID:     "chromium/123",
		}
		Convey(`System missing`, func() {
			id.System = ""
			err := id.Validate()
			So(err, ShouldErrLike, `invalid bug tracking system`)
		})
		Convey("ID invalid", func() {
			id.ID = "!!!"
			err := id.Validate()
			So(err, ShouldErrLike, `invalid monorail bug ID`)
		})
	})
}
