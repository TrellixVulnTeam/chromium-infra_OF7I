// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetHiveForDut(t *testing.T) {
	Convey("Test GetHiveForDut", t, func() {
		So(GetHiveForDut("satlab-xx12sha23-chromeos1-row2-rack3-host4"), ShouldEqual, "satlab-xx12sha23")
		So(GetHiveForDut("satlab-12:sha45-em25-desk-noogler"), ShouldEqual, "satlab-12:sha45")
		So(GetHiveForDut("chromeos1-row2-rack3-host4"), ShouldEqual, "")
		So(GetHiveForDut("cros-mtv1950-144-rack204-host1"), ShouldEqual, "cros-mtv1950-144")
		So(GetHiveForDut("cros-mtv1950-144-rack204-host2"), ShouldEqual, "cros-mtv1950-144")
		So(GetHiveForDut("cros-mtv1950-144-rack204-labstation1"), ShouldEqual, "cros-mtv1950-144")
		So(GetHiveForDut("chromeos8-foo"), ShouldEqual, "e")
	})
}

func TestAppendUnique(t *testing.T) {
	Convey("Test AppendUnique", t, func() {
		So(AppendUniqueStrings([]string{"eeny", "meeny", "miny", "moe"}, "catch", "a", "tiger", "by", "the", "toe"), ShouldHaveLength, 10)
		So(AppendUniqueStrings([]string{"london", "bridge", "is", "falling", "down"}, "falling", "down", "falling", "down"), ShouldHaveLength, 5)
		So(AppendUniqueStrings([]string{}, "twinkle", "twinkle", "little", "star"), ShouldHaveLength, 3)
		So(AppendUniqueStrings([]string{"humpty", "dumpty", "sat", "on", "a", "wall"}), ShouldHaveLength, 6)
		So(AppendUniqueStrings([]string{"row", "row", "row", "your", "boat"}), ShouldHaveLength, 3)
	})
}
