// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package manufacturingconfig

import (
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/proto/gitiles"
)

var manufacturingConfigJSON = `
{
	"value": [
		{
			"manufacturingId": {
				"value": "TERRA D25-E4C-A2I-A6A-A6L"
			},
			"devicePhase": "PHASE_PVT"
		},
		{
			"manufacturingId": {
				"value": "BARLA C3B-A4D-B3K-A4F-S34"
			},
			"devicePhase": "PHASE_PVT",
			"cr50Phase": "CR50_PHASE_PVT",
			"cr50KeyEnv": "CR50_KEYENV_PROD"
		}
	]
}
`

func TestUpdateDatastore(t *testing.T) {
	Convey("Test update device config cache", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		ctl := gomock.NewController(t)
		defer ctl.Finish()

		gitilesMock := gitiles.NewMockGitilesClient(ctl)
		gitilesMock.EXPECT().DownloadFile(gomock.Any(), gomock.Any()).Return(
			&gitiles.DownloadFileResponse{
				Contents: manufacturingConfigJSON,
			},
			nil,
		)
		Convey("Happy path", func() {
			err := UpdateDatastore(ctx, gitilesMock, "", "", "")
			So(err, ShouldBeNil)
			// There should be 2 entities created in datastore.
			var cfgs []*manufacturingCfgEntity
			datastore.GetTestable(ctx).Consistent(true)
			err = datastore.GetAll(ctx, datastore.NewQuery(entityKind), &cfgs)
			So(err, ShouldBeNil)
			So(cfgs, ShouldHaveLength, 2)
		})
	})
}

func TestGetCachedManufacturingConfig(t *testing.T) {
	ctx := gaetesting.TestingContextWithAppID("go-test")

	Convey("Test get manufacturing config from datastore", t, func() {
		err := datastore.Put(ctx, []manufacturingCfgEntity{
			{ID: "FOO"},
			{ID: "BAR"},
			{
				ID:     "BAZ",
				Config: []byte("bad data"),
			},
		})
		So(err, ShouldBeNil)

		Convey("Happy path", func() {
			devcfg, err := GetCachedConfig(ctx, []*manufacturing.ConfigID{
				{Value: "FOO"},
				{Value: "BAR"},
			})
			So(err, ShouldBeNil)
			So(devcfg, ShouldHaveLength, 2)
		})

		Convey("Data unmarshal error", func() {
			_, err := GetCachedConfig(ctx, []*manufacturing.ConfigID{
				{Value: "BAZ"},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "unmarshal config data")
		})

		Convey("Get nonexisting data", func() {
			_, err := GetCachedConfig(ctx, []*manufacturing.ConfigID{
				{Value: "GHOST"},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "get cached config data")
		})
	})
}
