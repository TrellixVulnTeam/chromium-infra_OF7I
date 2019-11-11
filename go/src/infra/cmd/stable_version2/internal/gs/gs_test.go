// Copyright 2019 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package gs

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"infra/cmd/stable_version2/internal/utils"
)

func TestParseOmahaStatus(t *testing.T) {
	Convey("Parse omaha status file", t, func() {
		ctx := context.Background()
		bt, err := ioutil.ReadFile(testDataPath("omaha_status.json"))
		So(err, ShouldBeNil)

		res, err := ParseOmahaStatus(ctx, bt)
		So(err, ShouldBeNil)

		Convey("Parse normal board", func() {
			v := utils.GetCrOSSVByBuildtarget(res, "normal")
			So(v, ShouldEqual, "R59-9414.0.0")
		})

		Convey("Parse board name with dash, dash should be replaced by underscore", func() {
			v := utils.GetCrOSSVByBuildtarget(res, "board-with-dash")
			So(v, ShouldEqual, "")

			v = utils.GetCrOSSVByBuildtarget(res, "board_with_dash")
			So(v, ShouldEqual, "R59-9414.0.0")
		})

		Convey("Parse board with 2 chromeos versions, shoud return the new one", func() {
			v := utils.GetCrOSSVByBuildtarget(res, "board_with_new_version")
			So(v, ShouldEqual, "R59-9514.0.0")
		})

		Convey("Parse board with 2 milestones, shoud return the new one", func() {
			v := utils.GetCrOSSVByBuildtarget(res, "board_with_new_milestone")
			So(v, ShouldEqual, "R60-9414.0.0")
		})

		Convey("Parse non-beta channel board, no return", func() {
			v := utils.GetCrOSSVByBuildtarget(res, "canaryboard")
			So(v, ShouldEqual, "")
		})
	})
}

func TestParseMetaData(t *testing.T) {
	Convey("Parse meta data file", t, func() {

		Convey("Parse non-unibuild", func() {
			bt, err := ioutil.ReadFile((testDataPath("meta_data.json")))
			So(err, ShouldBeNil)

			res, err := ParseMetadata(bt)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 1)
			So(res[0].GetKey().GetBuildTarget().GetName(), ShouldEqual, "arkham")
			So(res[0].GetKey().GetModelId().GetValue(), ShouldEqual, "arkham")
			So(res[0].GetVersion(), ShouldEqual, "v1")
		})

		Convey("Parse unibuild", func() {
			bt, err := ioutil.ReadFile((testDataPath("meta_data_unibuild.json")))
			So(err, ShouldBeNil)

			res, err := ParseMetadata(bt)
			So(err, ShouldBeNil)
			So(res, ShouldHaveLength, 2)
			So(res[0].GetKey().GetBuildTarget().GetName(), ShouldEqual, "reef")
			So(res[1].GetKey().GetBuildTarget().GetName(), ShouldEqual, "reef")
			models := map[string]string{
				res[0].GetKey().GetModelId().GetValue(): res[0].GetVersion(),
				res[1].GetKey().GetModelId().GetValue(): res[1].GetVersion(),
			}
			So(models["model1"], ShouldEqual, "v1")
			So(models["model2"], ShouldEqual, "v2")
		})
	})
}

func testDataPath(p string) string {
	return filepath.Join("testdata", p)
}
