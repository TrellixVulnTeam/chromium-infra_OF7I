// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package manufacturingconfig

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"

	"go.chromium.org/chromiumos/infra/proto/go/manufacturing"
	"infra/libs/skylab/inventory"
)

const fullMC = `
{
	"value": [
		{
			"manufacturingId": {
				"value": "test_mid"
			},
			"devicePhase": "PHASE_PVT",
			"cr50Phase": "CR50_PHASE_PVT",
			"cr50KeyEnv": "CR50_KEYENV_PROD",
			"wifiChip": "test_wifichip",
			"hwidComponent": ["battery/testbattery", "storage/teststorage"]
		},
		{
			"manufacturingId": {
				"value": "test_mid"
			},
			"cr50Phase": "CR50_PHASE_PVT",
			"cr50KeyEnv": "CR50_KEYENV_PROD",
			"wifiChip": "test_wifichip",
			"hwidComponent": ["battery/testbattery", "storage/teststorage"]
		},
		{
			"manufacturingId": {
				"value": "test_mid"
			},
			"devicePhase": "PHASE_PVT",
			"cr50KeyEnv": "CR50_KEYENV_PROD",
			"wifiChip": "test_wifichip",
			"hwidComponent": ["battery/testbattery", "storage/teststorage"]
		},
		{
			"manufacturingId": {
				"value": "test_mid"
			},
			"devicePhase": "PHASE_PVT",
			"cr50Phase": "CR50_PHASE_PVT",
			"wifiChip": "test_wifichip",
			"hwidComponent": ["battery/testbattery", "storage/teststorage"]
		},
		{
			"manufacturingId": {
				"value": "test_mid"
			},
			"devicePhase": "PHASE_PVT",
			"cr50Phase": "CR50_PHASE_PVT",
			"cr50KeyEnv": "CR50_KEYENV_PROD"
		},
		{
			"manufacturingId": {
				"value": "test_mid"
			},
			"devicePhase": "PHASE_PVT",
			"cr50Phase": "CR50_PHASE_PVT",
			"cr50KeyEnv": "CR50_KEYENV_PROD",
			"wifiChip": "test_wifichip"
		}
	]
}
`

const testV1Spec = `
id: "id1"
hostname: "host1"
labels: {
}
`

var (
	unmarshaler = jsonpb.Unmarshaler{AllowUnknownFields: false}
)

func TestConvertMCToV1Labels(t *testing.T) {
	t.Parallel()

	allConfigs := manufacturing.ConfigList{}
	unmarshaler.Unmarshal(bytes.NewReader([]byte(fullMC)), &allConfigs)
	fmt.Println(allConfigs.GetValue())
	var got inventory.CommonDeviceSpecs

	Convey("Verify happy path", t, func() {
		err := proto.UnmarshalText(testV1Spec, &got)
		So(err, ShouldBeNil)
		ConvertMCToV1Labels(allConfigs.GetValue()[0], got.GetLabels())
		l := got.GetLabels()
		So(l.GetPhase(), ShouldEqual, inventory.SchedulableLabels_PHASE_PVT)
		So(l.GetCr50Phase(), ShouldEqual, inventory.SchedulableLabels_CR50_PHASE_PVT)
		So(l.GetCr50RoKeyid(), ShouldEqual, "prod")
		So(l.GetWifiChip(), ShouldEqual, "test_wifichip")
		So(l.GetHwidComponent(), ShouldResemble, []string{"battery/testbattery", "storage/teststorage"})
	})

	Convey("Verify empty manufacturing config", t, func() {
		err := proto.UnmarshalText(testV1Spec, &got)
		So(err, ShouldBeNil)
		// No panic
		ConvertMCToV1Labels(nil, got.GetLabels())
		l := got.GetLabels()
		So(l.GetPhase(), ShouldEqual, inventory.SchedulableLabels_PHASE_INVALID)
		So(l.GetCr50Phase(), ShouldEqual, inventory.SchedulableLabels_CR50_PHASE_INVALID)
		So(l.GetCr50RoKeyid(), ShouldBeEmpty)
		So(l.GetWifiChip(), ShouldBeEmpty)
		So(l.GetHwidComponent(), ShouldBeEmpty)
	})

	Convey("Verify empty labels", t, func() {
		// No panic
		ConvertMCToV1Labels(allConfigs.GetValue()[0], nil)
	})

	Convey("Verify empty phase", t, func() {
		err := proto.UnmarshalText(testV1Spec, &got)
		So(err, ShouldBeNil)
		ConvertMCToV1Labels(allConfigs.GetValue()[1], got.GetLabels())
		l := got.GetLabels()
		So(l.GetPhase(), ShouldEqual, inventory.SchedulableLabels_PHASE_INVALID)
		So(l.GetCr50Phase(), ShouldEqual, inventory.SchedulableLabels_CR50_PHASE_PVT)
		So(l.GetCr50RoKeyid(), ShouldEqual, "prod")
		So(l.GetWifiChip(), ShouldEqual, "test_wifichip")
		So(l.GetHwidComponent(), ShouldResemble, []string{"battery/testbattery", "storage/teststorage"})
	})

	Convey("Verify empty cr50 phase", t, func() {
		err := proto.UnmarshalText(testV1Spec, &got)
		So(err, ShouldBeNil)
		ConvertMCToV1Labels(allConfigs.GetValue()[2], got.GetLabels())
		l := got.GetLabels()
		So(l.GetPhase(), ShouldEqual, inventory.SchedulableLabels_PHASE_PVT)
		So(l.GetCr50Phase(), ShouldEqual, inventory.SchedulableLabels_CR50_PHASE_INVALID)
		So(l.GetCr50RoKeyid(), ShouldEqual, "prod")
		So(l.GetWifiChip(), ShouldEqual, "test_wifichip")
		So(l.GetHwidComponent(), ShouldResemble, []string{"battery/testbattery", "storage/teststorage"})
	})

	Convey("Verify empty cr50 env", t, func() {
		err := proto.UnmarshalText(testV1Spec, &got)
		So(err, ShouldBeNil)
		ConvertMCToV1Labels(allConfigs.GetValue()[3], got.GetLabels())
		l := got.GetLabels()
		So(l.GetPhase(), ShouldEqual, inventory.SchedulableLabels_PHASE_PVT)
		So(l.GetCr50Phase(), ShouldEqual, inventory.SchedulableLabels_CR50_PHASE_PVT)
		So(l.GetCr50RoKeyid(), ShouldBeEmpty)
		So(l.GetWifiChip(), ShouldEqual, "test_wifichip")
		So(l.GetHwidComponent(), ShouldResemble, []string{"battery/testbattery", "storage/teststorage"})
	})

	Convey("Verify empty wifi chip", t, func() {
		err := proto.UnmarshalText(testV1Spec, &got)
		So(err, ShouldBeNil)
		ConvertMCToV1Labels(allConfigs.GetValue()[4], got.GetLabels())
		l := got.GetLabels()
		So(l.GetPhase(), ShouldEqual, inventory.SchedulableLabels_PHASE_PVT)
		So(l.GetCr50Phase(), ShouldEqual, inventory.SchedulableLabels_CR50_PHASE_PVT)
		So(l.GetCr50RoKeyid(), ShouldEqual, "prod")
		So(l.GetWifiChip(), ShouldBeEmpty)
		So(l.GetHwidComponent(), ShouldBeEmpty)
	})

	Convey("Verify empty hwid component", t, func() {
		err := proto.UnmarshalText(testV1Spec, &got)
		So(err, ShouldBeNil)
		ConvertMCToV1Labels(allConfigs.GetValue()[5], got.GetLabels())
		l := got.GetLabels()
		So(l.GetPhase(), ShouldEqual, inventory.SchedulableLabels_PHASE_PVT)
		So(l.GetCr50Phase(), ShouldEqual, inventory.SchedulableLabels_CR50_PHASE_PVT)
		So(l.GetCr50RoKeyid(), ShouldEqual, "prod")
		So(l.GetWifiChip(), ShouldEqual, "test_wifichip")
		So(l.GetHwidComponent(), ShouldBeEmpty)
	})
}
