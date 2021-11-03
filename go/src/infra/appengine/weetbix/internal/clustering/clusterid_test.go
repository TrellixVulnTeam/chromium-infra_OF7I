// Copyright 2021 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package clustering

import (
	"encoding/hex"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"
)

func TestValidate(t *testing.T) {
	Convey(`Validate`, t, func() {
		id := ClusterID{
			Algorithm: "blah-v2",
			ID:        hex.EncodeToString([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}),
		}
		Convey(`Algorithm missing`, func() {
			id.Algorithm = ""
			err := id.Validate()
			So(err, ShouldErrLike, `algorithm not valid`)
		})
		Convey("Algorithm invalid", func() {
			id.Algorithm = "!!!"
			err := id.Validate()
			So(err, ShouldErrLike, `algorithm not valid`)
		})
		Convey("ID missing", func() {
			id.ID = ""
			err := id.Validate()
			So(err, ShouldErrLike, `ID is empty`)
		})
		Convey("ID invalid", func() {
			id.ID = "!!!"
			err := id.Validate()
			So(err, ShouldErrLike, `ID is not valid hexadecimal`)
		})
		Convey("ID not lowercase", func() {
			id.ID = "AA"
			err := id.Validate()
			So(err, ShouldErrLike, `ID must be in lowercase`)
		})
		Convey("ID too long", func() {
			id.ID = hex.EncodeToString([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17})
			err := id.Validate()
			So(err, ShouldErrLike, `ID is too long (got 17 bytes, want at most 16 bytes)`)
		})
	})
}
