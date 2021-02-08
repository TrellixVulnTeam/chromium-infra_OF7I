// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package eval

import (
	"math"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestScoreString(t *testing.T) {
	t.Parallel()

	Convey(`ScoreString`, t, func() {
		Convey("NaN", func() {
			So(scoreString(float32(math.NaN())), ShouldEqual, "?")
		})
		Convey("0%", func() {
			So(scoreString(0), ShouldEqual, "0.00%")
		})
		Convey("0.0001%", func() {
			So(scoreString(0.000001), ShouldEqual, "<0.01%")
		})
		Convey("50%", func() {
			So(scoreString(0.5), ShouldEqual, "50.00%")
		})
		Convey("99.999%", func() {
			So(scoreString(0.99999), ShouldEqual, ">99.99%")
		})
		Convey("100%", func() {
			So(scoreString(1), ShouldEqual, "100.00%")
		})
	})
}
