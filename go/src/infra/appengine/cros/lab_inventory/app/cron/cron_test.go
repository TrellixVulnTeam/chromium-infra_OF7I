// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cron

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetHiveForDut(t *testing.T) {
	Convey("Test GetHiveForDut", t, func() {
		So(GetHiveForDut("satlab-xx12sha23-chromeos1-row2-rack3-host4"), ShouldEqual, "satlab-xx12sha23")
		So(GetHiveForDut("satlab-12:sha45-em25-desk-noogler"), ShouldEqual, "satlab-12:sha45")
		So(GetHiveForDut("chromeos1-row2-rack3-host4"), ShouldEqual, "")
	})
}
