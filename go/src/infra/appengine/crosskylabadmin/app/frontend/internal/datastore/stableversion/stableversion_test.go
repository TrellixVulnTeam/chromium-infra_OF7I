// Copyright 2019 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stableversion

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/impl/memory"
)

func TestUpdateAndGet(t *testing.T) {
	// TODO(gregorynisbet): convert to test fixture
	ctx := context.Background()
	ctx = memory.Use(ctx)

	model := "xxx-model"
	buildTarget := "xxx-buildTarget"
	crosVersion := "xxx-cros-version"
	firmwareVersion := "xxx-firmware-version"
	faftVersion := "xxx-faft-version"

	Convey("StableVersion datastore", t, func() {
		Convey("Cros", func() {
			Convey("Cros not present initially", func() {
				item, err := GetCrosStableVersion(ctx, buildTarget)
				So(err, ShouldNotBeNil)
				So(item, ShouldEqual, "")
			})
			Convey("Cros write should succeed", func() {
				err := PutCrosStableVersion(ctx, buildTarget, crosVersion)
				So(err, ShouldBeNil)
			})
			Convey("Cros present after write", func() {
				item, err := GetCrosStableVersion(ctx, buildTarget)
				So(err, ShouldBeNil)
				So(item, ShouldEqual, crosVersion)
			})
		})
		Convey("Faft", func() {
			Convey("Faft not present initially", func() {
				item, err := GetFaftStableVersion(ctx, buildTarget, model)
				So(err, ShouldNotBeNil)
				So(item, ShouldEqual, "")
			})
			Convey("Faft write should succeed", func() {
				err := PutFaftStableVersion(ctx, buildTarget, model, faftVersion)
				So(err, ShouldBeNil)
			})
			Convey("Faft present after write", func() {
				item, err := GetFaftStableVersion(ctx, buildTarget, model)
				So(err, ShouldBeNil)
				So(item, ShouldEqual, faftVersion)
			})
		})
		Convey("Firmware", func() {
			Convey("Firmware not present initially", func() {
				item, err := GetFirmwareStableVersion(ctx, buildTarget, model)
				So(err, ShouldNotBeNil)
				So(item, ShouldEqual, "")
			})
			Convey("Firmware write should succeed", func() {
				err := PutFirmwareStableVersion(ctx, buildTarget, model, firmwareVersion)
				So(err, ShouldBeNil)
			})
			Convey("Firmware present after write", func() {
				item, err := GetFirmwareStableVersion(ctx, buildTarget, model)
				So(err, ShouldBeNil)
				So(item, ShouldEqual, firmwareVersion)
			})
		})
	})
}
