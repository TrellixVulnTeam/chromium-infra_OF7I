// Copyright 2018 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUtils(t *testing.T) {
	t.Parallel()
	Convey("Ensure token (un)escaping works as intended", t, func() {
		for _, tc := range []string{
			"n\nn",
			"n\\nn",
			"n\\\nn",
			"n\\\\nn",
			"n\\\\\nn",
			"n\\\\\\nn",
			"n\nn",
			"n\n\n",
			"n\n\\n",
			"n\n\\\n",
			"n\n\\\\n",
			"n\n\\\\\n",
			"n\n\\\\\\n",
			"n\n:n",
			"n\\n:n",
			"n\\\n:n",
			"n\\\\n:n",
			"n\\\\\n:n",
			"n\\\\\\n:n",
			"n:\nn",
			"n:\n\n",
			"n:\n\\n",
			"n:\n\\\n",
			"n:\n\\\\n",
			"n:\n\\\\\n",
			"n:\n\\\\\\n",
			"c\n:c",
			"c\\c:c",
			"c\\\n:c",
			"c\\\\c:c",
			"c\\\\\n:c",
			"c\\\\\\c:c",
			"c:\nc",
			"c:\n\n",
			"c:\n\\c",
			"c:\n\\\n",
			"c:\n\\\\c",
			"c:\n\\\\\n",
			"c:\n\\\\\\c",
		} {
			So(tc, ShouldEqual, unescapeToken(escapeToken(tc)))
		}
	})
	prefixList := []string{
		"Bug:",
		"BUG=",
		"Fixed:",
	}
	Convey("One bug", t, func() {
		for _, prefix := range prefixList {
			commitMsg := fmt.Sprintf("Test\n\n%s 123456", prefix)
			bugID, _ := bugIDFromCommitMessage(commitMsg)
			So(bugID, ShouldEqual, "123456")
		}
	})
	Convey("Multiple bugs", t, func() {
		for _, prefix := range prefixList {
			commitMsg := fmt.Sprintf("Test\n\n%s 123456, 234567, 345678", prefix)
			bugID, _ := bugIDFromCommitMessage(commitMsg)
			So(bugID, ShouldEqual, "123456,234567,345678")
		}
	})
	Convey("Chromium and buganizer bugs", t, func() {
		for _, prefix := range prefixList {
			commitMsg := fmt.Sprintf("Test\n\n%s chromium: 123456, b: 234567, chromium:345678, b:435", prefix)
			bugID, _ := bugIDFromCommitMessage(commitMsg)
			So(bugID, ShouldEqual, "123456,345678")

			commitMsg = fmt.Sprintf("Test\n\n%s b: 123456, chromium: 234567, b:345678, b:435", prefix)
			bugID, _ = bugIDFromCommitMessage(commitMsg)
			So(bugID, ShouldEqual, "234567")
		}
	})
	Convey("Invalid bugs", t, func() {
		commitMsg := "Fixed aaa."
		bugID, _ := bugIDFromCommitMessage(commitMsg)
		So(bugID, ShouldEqual, "")

		commitMsg = "Bug chromium:aaa."
		bugID, _ = bugIDFromCommitMessage(commitMsg)
		So(bugID, ShouldEqual, "")
	})
}
