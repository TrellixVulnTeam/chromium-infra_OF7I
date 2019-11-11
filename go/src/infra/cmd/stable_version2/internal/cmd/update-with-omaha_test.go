// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package cmd

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	sv "go.chromium.org/chromiumos/infra/proto/go/lab_platform"
	"infra/cmd/stable_version2/internal/utils"
)

func TestCompareCrosSV(t *testing.T) {
	Convey("CompareCrosSV", t, func() {
		ctx := context.Background()
		new := []*sv.StableCrosVersion{
			utils.MakeCrOSSV("newboard", "R1-2.3.4"),
			utils.MakeCrOSSV("invalid_version", "R49-2.3.4"),
			utils.MakeCrOSSV("older", "R50-2.3.4"),
			utils.MakeCrOSSV("newer", "R49-2.3.4"),
		}
		old := []*sv.StableCrosVersion{
			utils.MakeCrOSSV("invalid_version", "R50-2.3.4abc"),
			utils.MakeCrOSSV("older", "R49-2.2.4"),
			utils.MakeCrOSSV("newer", "R50-2.3.4"),
		}
		updated := compareCrosSV(ctx, new, old)

		Convey("new board is added", func() {
			So(utils.GetCrOSSVByBuildtarget(updated, "newboard"), ShouldEqual, "R1-2.3.4")
		})

		Convey("invalid version is overwritten by new", func() {
			So(utils.GetCrOSSVByBuildtarget(updated, "invalid_version"), ShouldEqual, "R49-2.3.4")
		})

		Convey("version is overwritten if older", func() {
			So(utils.GetCrOSSVByBuildtarget(updated, "older"), ShouldEqual, "R50-2.3.4")
		})

		Convey("version is not updated if newer", func() {
			So(utils.GetCrOSSVByBuildtarget(updated, "newer"), ShouldEqual, "")
		})
	})
}
