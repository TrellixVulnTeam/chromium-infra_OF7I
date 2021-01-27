// Copyright 2019 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package deviceconfig

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/chromiumos/infra/proto/go/device"
	"go.chromium.org/luci/appengine/gaetesting"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/proto/gitiles"
	"go.chromium.org/luci/common/proto/gitiles/mock_gitiles"
	"go.chromium.org/luci/gae/service/datastore"
)

var deviceConfigJSON = `
{
	"configs": [
		{
			"unkonwnField": "hahaha",
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

func TestUpdateDatastore(t *testing.T) {
	Convey("Test update device config cache", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		ctl := gomock.NewController(t)
		defer ctl.Finish()

		gitilesMock := mock_gitiles.NewMockGitilesClient(ctl)
		gitilesMock.EXPECT().DownloadFile(gomock.Any(), gomock.Any()).Return(
			&gitiles.DownloadFileResponse{
				Contents: deviceConfigJSON,
			},
			nil,
		)
		Convey("Happy path", func() {
			err := UpdateDatastore(ctx, gitilesMock, "", "", "")
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
			{ID: "platform.model.variant1"},
			{ID: "platform.model.variant2"},
			{
				ID:        "platform.model.variant3",
				DevConfig: []byte("bad data"),
			},
		})
		So(err, ShouldBeNil)

		Convey("Happy path", func() {
			devcfg, err := GetCachedConfig(ctx, []*device.ConfigId{
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant1"},
					BrandId:    &device.BrandId{Value: "brand1"},
				},
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant2"},
					BrandId:    &device.BrandId{Value: "brand2"},
				},
			})
			So(err, ShouldBeNil)
			So(devcfg, ShouldHaveLength, 2)
		})

		Convey("Device id is case insensitive", func() {
			devcfg, err := GetCachedConfig(ctx, []*device.ConfigId{
				{
					PlatformId: &device.PlatformId{Value: "PLATFORM"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant1"},
					BrandId:    &device.BrandId{Value: "brand1"},
				},
			})
			So(err, ShouldBeNil)
			So(devcfg, ShouldHaveLength, 1)
		})

		Convey("Data unmarshal error", func() {
			_, err := GetCachedConfig(ctx, []*device.ConfigId{
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant3"},
					BrandId:    &device.BrandId{Value: "brand3"},
				},
			})
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "unmarshal config data")
		})

		Convey("Get nonexisting data", func() {
			resp, err := GetCachedConfig(ctx, []*device.ConfigId{
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant-nonexisting"},
					BrandId:    &device.BrandId{Value: "nonexisting"},
				},
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant1"},
					BrandId:    &device.BrandId{Value: "brand1"},
				},
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant-nonexisting2"},
					BrandId:    &device.BrandId{Value: "nonexisting"},
				},
			})
			So(err, ShouldNotBeNil)
			errs := err.(errors.MultiError)
			So(errs, ShouldHaveLength, 3)
			So(resp, ShouldHaveLength, 3)
			So(errs[0].Error(), ShouldContainSubstring, "no such entity")
			So(resp[0], ShouldBeNil)
			So(errs[1], ShouldBeNil)
			So(resp[1].(*device.Config), ShouldNotBeNil)
			So(errs[2].Error(), ShouldContainSubstring, "no such entity")
			So(resp[2], ShouldBeNil)
		})
	})
}

func TestGetAllCachedConfig(t *testing.T) {
	Convey("Test get all device config cache", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		datastore.GetTestable(ctx).Consistent(true)
		err := datastore.Put(ctx, []devcfgEntity{
			{ID: "platform.model.variant1"},
			{ID: "platform.model.variant2"},
			{
				ID:        "platform.model.variant3",
				DevConfig: []byte("bad data"),
			},
		})
		So(err, ShouldBeNil)

		devConfigs, err := GetAllCachedConfig(ctx)
		So(err, ShouldBeNil)
		So(devConfigs, ShouldHaveLength, 2)
		for dc := range devConfigs {
			So(dc.GetId(), ShouldBeNil)
		}
	})
}

func TestDeviceConfigsExists(t *testing.T) {
	ctx := gaetesting.TestingContextWithAppID("go-test")

	Convey("Test exists device config in datastore", t, func() {
		err := datastore.Put(ctx, []devcfgEntity{
			{ID: "kunimitsu.lars.variant1"},
			{ID: "arcada.arcada.variant2"},
			{
				ID:        "platform.model.variant3",
				DevConfig: []byte("bad data"),
			},
		})
		So(err, ShouldBeNil)

		Convey("Happy path", func() {
			exists, err := DeviceConfigsExists(ctx, []*device.ConfigId{
				{
					PlatformId: &device.PlatformId{Value: "kunimitsu"},
					ModelId:    &device.ModelId{Value: "lars"},
					VariantId:  &device.VariantId{Value: "variant1"},
				},
				{
					PlatformId: &device.PlatformId{Value: "arcada"},
					ModelId:    &device.ModelId{Value: "arcada"},
					VariantId:  &device.VariantId{Value: "variant2"},
				},
			})
			So(err, ShouldBeNil)
			So(exists[0], ShouldBeTrue)
			So(exists[1], ShouldBeTrue)
		})

		Convey("check for nonexisting data", func() {
			exists, err := DeviceConfigsExists(ctx, []*device.ConfigId{
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant-nonexisting"},
					BrandId:    &device.BrandId{Value: "nonexisting"},
				},
			})
			So(err, ShouldBeNil)
			So(exists[0], ShouldBeFalse)
		})

		Convey("check for existing and nonexisting data", func() {
			exists, err := DeviceConfigsExists(ctx, []*device.ConfigId{
				{
					PlatformId: &device.PlatformId{Value: "platform"},
					ModelId:    &device.ModelId{Value: "model"},
					VariantId:  &device.VariantId{Value: "variant-nonexisting"},
				},
				{
					PlatformId: &device.PlatformId{Value: "arcada"},
					ModelId:    &device.ModelId{Value: "arcada"},
					VariantId:  &device.VariantId{Value: "variant2"},
				},
			})
			So(err, ShouldBeNil)
			So(exists[0], ShouldBeFalse)
			So(exists[1], ShouldBeTrue)
		})
	})
}

type fakeGitClient struct {
	project string
}

type fakeGSClient struct{}

func (gc *fakeGitClient) GetFile(ctx context.Context, path string) (string, error) {
	if path != "generated/configs.jsonproto" {
		return "", nil
	}
	b, err := ioutil.ReadFile("test_device_config_v2.jsonproto")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (gc *fakeGitClient) SwitchProject(ctx context.Context, project string) error {
	gc.project = project
	return nil
}

func (gsClient *fakeGSClient) GetFile(ctx context.Context, path string) ([]byte, error) {
	b, err := ioutil.ReadFile("test_program_configs.json")
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func TestUpdateDatastoreFromBoxter(t *testing.T) {
	Convey("Test update device config from boxster", t, func() {
		ctx := gaetesting.TestingContextWithAppID("go-test")
		gitilesMock := &fakeGitClient{}
		Convey("Happy path", func() {
			err := UpdateDatastoreFromBoxster(ctx, gitilesMock, "generated/configs.jsonproto", nil, "", "", "")
			So(err, ShouldBeNil)
			// There should be 6 entities created in datastore as
			// test_device_config_v2.jsonproto contains 6 device configs.
			var cfgs []*devcfgEntity
			datastore.GetTestable(ctx).Consistent(true)
			err = datastore.GetAll(ctx, datastore.NewQuery(entityKind), &cfgs)
			So(err, ShouldBeNil)
			So(cfgs, ShouldHaveLength, 6)
		})
	})
}
