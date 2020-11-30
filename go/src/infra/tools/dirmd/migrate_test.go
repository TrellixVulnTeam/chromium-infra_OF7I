// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dirmd

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMigrate(t *testing.T) {
	t.Parallel()

	Convey(`FilterEmptyLines`, t, func() {
		Convey(`Works`, func() {
			actual := filterEmptyLines([]string{
				"",
				"",
				"joe@example.com",
				"",
				"",
				"",
				"doe@example.com",
				"",
				"",
			})
			So(actual, ShouldResemble, []string{
				"joe@example.com",
				"",
				"doe@example.com",
				"",
			})
		})
		Convey(`Empty`, func() {
			actual := filterEmptyLines([]string{
				"",
				"",
			})
			So(actual, ShouldResemble, []string{})
		})
	})
}
