// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package deviceconfig

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/chromiumos/config/go/payload"
	"go.chromium.org/chromiumos/infra/proto/go/device"
)

func TestParseConfigBundle(t *testing.T) {
	Convey("Test config bundle parsing", t, func() {
		var payloads payload.ConfigBundleList
		unmarshaller := &jsonpb.Unmarshaler{AllowUnknownFields: false}
		// Refer to https://chromium.googlesource.com/chromiumos/config/+/refs/heads/master/test/project/fake/fake/config.star for unittest check
		b, err := ioutil.ReadFile("test_device_config_v2.jsonproto")
		So(err, ShouldBeNil)
		buf, err := fixFieldMaskForConfigBundleList([]byte(b))
		So(err, ShouldBeNil)
		err = unmarshaller.Unmarshal(bytes.NewBuffer(buf), &payloads)
		So(err, ShouldBeNil)
		Convey("Happy path", func() {
			So(payloads.GetValues(), ShouldHaveLength, 1)
			dcs := parseConfigBundle(payloads.GetValues()[0])
			So(dcs, ShouldHaveLength, 6)
			for _, dc := range dcs {
				So(dc.GetId().GetPlatformId().GetValue(), ShouldEqual, "FAKE_PROGRAM")
				modelWithSku := fmt.Sprintf("%s:%s", dc.GetId().GetModelId().GetValue(), dc.GetId().GetVariantId().GetValue())
				switch modelWithSku {
				case "FAKE-REF-DESIGN:2147483647":
					So(dc.GetFormFactor(), ShouldEqual, device.Config_FORM_FACTOR_CLAMSHELL)
					So(dc.GetHardwareFeatures(), ShouldResemble, []device.Config_HardwareFeature{
						device.Config_HARDWARE_FEATURE_BLUETOOTH,
						device.Config_HARDWARE_FEATURE_INTERNAL_DISPLAY,
						device.Config_HARDWARE_FEATURE_STYLUS,
						device.Config_HARDWARE_FEATURE_TOUCHPAD,
						device.Config_HARDWARE_FEATURE_TOUCHSCREEN,
						device.Config_HARDWARE_FEATURE_DETACHABLE_KEYBOARD,
						device.Config_HARDWARE_FEATURE_FINGERPRINT,
					})
					So(dc.GetStorage(), ShouldEqual, device.Config_STORAGE_MMC)
					So(dc.GetSoc(), ShouldEqual, device.Config_SOC_UNSPECIFIED)
					So(dc.GetCpu(), ShouldEqual, device.Config_ARCHITECTURE_UNDEFINED)
				case "FAKE-REF-DESIGN:0":
					fallthrough
				case "FAKE-REF-DESIGN:2":
					So(dc.GetFormFactor(), ShouldEqual, device.Config_FORM_FACTOR_CLAMSHELL)
					So(dc.GetHardwareFeatures(), ShouldResemble, []device.Config_HardwareFeature{
						device.Config_HARDWARE_FEATURE_BLUETOOTH,
						device.Config_HARDWARE_FEATURE_INTERNAL_DISPLAY,
						device.Config_HARDWARE_FEATURE_STYLUS,
						device.Config_HARDWARE_FEATURE_TOUCHPAD,
						device.Config_HARDWARE_FEATURE_TOUCHSCREEN,
						device.Config_HARDWARE_FEATURE_DETACHABLE_KEYBOARD,
					})
					So(dc.GetStorage(), ShouldEqual, device.Config_STORAGE_MMC)
					So(dc.GetSoc(), ShouldEqual, device.Config_SOC_UNSPECIFIED)
					So(dc.GetCpu(), ShouldEqual, device.Config_ARCHITECTURE_UNDEFINED)
					So(dc.GetEc(), ShouldEqual, device.Config_EC_CHROME)
				case "PROJECT-A:32":
					So(dc.GetFormFactor(), ShouldEqual, device.Config_FORM_FACTOR_CONVERTIBLE)
					So(dc.GetHardwareFeatures(), ShouldResemble, []device.Config_HardwareFeature{
						device.Config_HARDWARE_FEATURE_BLUETOOTH,
						device.Config_HARDWARE_FEATURE_INTERNAL_DISPLAY,
						device.Config_HARDWARE_FEATURE_STYLUS,
						device.Config_HARDWARE_FEATURE_TOUCHPAD,
						device.Config_HARDWARE_FEATURE_TOUCHSCREEN,
						device.Config_HARDWARE_FEATURE_DETACHABLE_KEYBOARD,
						device.Config_HARDWARE_FEATURE_FINGERPRINT,
					})
					So(dc.GetStorage(), ShouldEqual, device.Config_STORAGE_MMC)
					So(dc.GetSoc(), ShouldEqual, device.Config_SOC_UNSPECIFIED)
					So(dc.GetCpu(), ShouldEqual, device.Config_ARCHITECTURE_UNDEFINED)
				case "PROJECT-B:33":
					So(dc.GetFormFactor(), ShouldEqual, device.Config_FORM_FACTOR_CONVERTIBLE)
					So(dc.GetHardwareFeatures(), ShouldResemble, []device.Config_HardwareFeature{
						device.Config_HARDWARE_FEATURE_BLUETOOTH,
						device.Config_HARDWARE_FEATURE_INTERNAL_DISPLAY,
						device.Config_HARDWARE_FEATURE_STYLUS,
						device.Config_HARDWARE_FEATURE_TOUCHPAD,
						device.Config_HARDWARE_FEATURE_TOUCHSCREEN,
						device.Config_HARDWARE_FEATURE_DETACHABLE_KEYBOARD,
					})
					So(dc.GetStorage(), ShouldEqual, device.Config_STORAGE_MMC)
					So(dc.GetSoc(), ShouldEqual, device.Config_SOC_UNSPECIFIED)
					So(dc.GetCpu(), ShouldEqual, device.Config_ARCHITECTURE_UNDEFINED)
				case "PROJECT-C:34":
					So(dc.GetFormFactor(), ShouldEqual, device.Config_FORM_FACTOR_CLAMSHELL)
					So(dc.GetHardwareFeatures(), ShouldResemble, []device.Config_HardwareFeature{
						device.Config_HARDWARE_FEATURE_BLUETOOTH,
						device.Config_HARDWARE_FEATURE_INTERNAL_DISPLAY,
						device.Config_HARDWARE_FEATURE_STYLUS,
						device.Config_HARDWARE_FEATURE_TOUCHPAD,
						device.Config_HARDWARE_FEATURE_TOUCHSCREEN,
						device.Config_HARDWARE_FEATURE_DETACHABLE_KEYBOARD,
					})
					So(dc.GetStorage(), ShouldEqual, device.Config_STORAGE_MMC)
					So(dc.GetSoc(), ShouldEqual, device.Config_SOC_UNSPECIFIED)
					So(dc.GetCpu(), ShouldEqual, device.Config_ARCHITECTURE_UNDEFINED)
				case "PROJECT-WL:64":
					So(dc.GetFormFactor(), ShouldEqual, device.Config_FORM_FACTOR_CLAMSHELL)
					So(dc.GetHardwareFeatures(), ShouldResemble, []device.Config_HardwareFeature{
						device.Config_HARDWARE_FEATURE_BLUETOOTH,
						device.Config_HARDWARE_FEATURE_INTERNAL_DISPLAY,
						device.Config_HARDWARE_FEATURE_TOUCHPAD,
						device.Config_HARDWARE_FEATURE_TOUCHSCREEN,
						device.Config_HARDWARE_FEATURE_DETACHABLE_KEYBOARD,
					})
					So(dc.GetStorage(), ShouldEqual, device.Config_STORAGE_MMC)
					So(dc.GetSoc(), ShouldEqual, device.Config_SOC_UNSPECIFIED)
					So(dc.GetCpu(), ShouldEqual, device.Config_ARCHITECTURE_UNDEFINED)
				default:
					t.Errorf("Invalid model:sku: %s", modelWithSku)
				}
			}
		})
	})
}
