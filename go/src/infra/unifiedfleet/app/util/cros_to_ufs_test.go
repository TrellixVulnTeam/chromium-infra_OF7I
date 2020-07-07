// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"testing"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"

	"infra/unifiedfleet/app/frontend/fake"
)

func TestCopyDUT(t *testing.T) {
	Convey("Verify copyDUT", t, func() {
		Convey("happy path", func() {
			mockLabConfig := fake.GetMockDUT()
			newDUT := copyDUT(mockLabConfig.GetConfig().GetDut())
			nb, err := proto.Marshal(newDUT)
			So(err, ShouldBeNil)
			ob, err := proto.Marshal(mockLabConfig.Config.GetDut())
			So(err, ShouldBeNil)
			So(nb, ShouldResemble, ob)
		})
	})
}

func TestCopyLabstation(t *testing.T) {
	Convey("Verify copyLabstation", t, func() {
		Convey("happy path", func() {
			mockLabConfig := fake.GetMockLabstation()
			newL := copyLabstation(mockLabConfig.GetConfig().GetLabstation())
			nb, err := proto.Marshal(newL)
			So(err, ShouldBeNil)
			ob, err := proto.Marshal(mockLabConfig.GetConfig().GetLabstation())
			So(err, ShouldBeNil)
			So(nb, ShouldResemble, ob)
		})
	})
}
