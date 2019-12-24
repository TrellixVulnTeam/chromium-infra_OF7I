// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package deviceconfig

import (
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/proto/gitiles"
)

var deviceConfigJSON = `
{
	"configs": [
		{
			"id": {
				"platformId": {
					"value": "Arcada"
				},
				"modelId": {
					"value": "arcada"
				},
				"variantId": {}
			},
			"hardwareFeatures": [
				"HARDWARE_FEATURE_BLUETOOTH",
				"HARDWARE_FEATURE_TOUCHSCREEN"
			],
			"power": "POWER_SUPPLY_BATTERY",
			"storage": "STORAGE_NVME",
			"videoAccelerationSupports": [
				"VIDEO_ACCELERATION_H264",
				"VIDEO_ACCELERATION_ENC_MJPG"
			],
			"soc": "SOC_WHISKEY_LAKE_U"
		},
		{
			"id": {
				"platformId": {
					"value": "Arcada"
				},
				"modelId": {
					"value": "arcada"
				},
				"variantId": {
					"value": "2"
				}
			},
			"hardwareFeatures": [
				"HARDWARE_FEATURE_TOUCHPAD",
				"HARDWARE_FEATURE_TOUCHSCREEN"
			],
			"power": "POWER_SUPPLY_BATTERY",
			"storage": "STORAGE_NVME",
			"videoAccelerationSupports": [
				"VIDEO_ACCELERATION_MJPG",
				"VIDEO_ACCELERATION_ENC_MJPG"
			],
			"soc": "SOC_WHISKEY_LAKE_U"
		}
	]
}
`

func TestUpdateDeviceConfigCache(t *testing.T) {
	Convey("Test update device config cache", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		ctl := gomock.NewController(t)
		defer ctl.Finish()

		gitilesMock := gitiles.NewMockGitilesClient(ctl)
		gitilesMock.EXPECT().DownloadFile(gomock.Any(), gomock.Any()).Return(
			&gitiles.DownloadFileResponse{
				Contents: deviceConfigJSON,
			},
			nil,
		)
		Convey("Happy path", func() {
			err := UpdateDeviceConfigCache(ctx, gitilesMock, "", "", "")
			So(err, ShouldBeNil)
			// There should be 2 entities created in datastore.
			var cfgs []*devcfgEntity
			datastore.GetTestable(ctx).Consistent(true)
			err = datastore.GetAll(ctx, datastore.NewQuery(entityKind), &cfgs)
			So(err, ShouldBeNil)
			So(cfgs, ShouldHaveLength, 2)
		})
	})
}

func TestGetCachedDeviceConfig(t *testing.T) {
	ctx := gaetesting.TestingContextWithAppID("go-test")

	Convey("Test get device config from datastore", t, func() {
		err := datastore.Put(ctx, []devcfgEntity{
			{ID: "platform.model.variant.brand1"},
			{ID: "platform.model.variant.brand2"},
			{
				ID:        "platform.model.variant.brand3",
				DevConfig: []byte("bad data"),
			},
		})
		So(err, ShouldBeNil)

		Convey("Happy path", func() {
			devcfg, err := GetCachedDeviceConfig(ctx, []*device.ConfigId{
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant"},
					BrandId:    &device.BrandId{Value: "brand1"},
				},
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant"},
					BrandId:    &device.BrandId{Value: "brand2"},
				},
			})
			So(err, ShouldBeNil)
			So(devcfg, ShouldHaveLength, 2)
		})

		Convey("Data unmarshal error", func() {
			_, err := GetCachedDeviceConfig(ctx, []*device.ConfigId{
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant"},
					BrandId:    &device.BrandId{Value: "brand3"},
				},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "unmarshal device config data")
		})

		Convey("Get nonexisting data", func() {
			_, err := GetCachedDeviceConfig(ctx, []*device.ConfigId{
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant"},
					BrandId:    &device.BrandId{Value: "nonexisting"},
				},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "get cached device config data")
		})
	})
}
