// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestIsLocation(t *testing.T) {
	Convey("test standard chromeos location", t, func() {
		location := []string{
			"chromeos1-row2-rack3-host4",
			"chromeos1-row2-rack3-hostxxx",
			"ROW1-RACK2-HOST3",
		}
		for _, l := range location {
			So(IsLocation(l), ShouldBeTrue)
		}
	})

	Convey("test invalid chromeos location", t, func() {
		location := "chromeos1-row2-rack3"
		So(IsLocation(location), ShouldBeFalse)
	})
}
