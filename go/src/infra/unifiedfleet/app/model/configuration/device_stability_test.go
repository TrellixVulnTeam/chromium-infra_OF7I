// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/luci/appengine/gaetesting"

	"go.chromium.org/chromiumos/config/go/test/dut"
)

func TestDeviceStability(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	Convey("Test Device Stability", t, func() {
		b, err := ioutil.ReadFile("test_device_stability.cfg")
		So(err, ShouldBeNil)
		unmarshaller := &jsonpb.Unmarshaler{AllowUnknownFields: false}
		var dsList dut.DeviceStabilityList
		err = unmarshaller.Unmarshal(bytes.NewBuffer(b), &dsList)
		So(err, ShouldBeNil)
		for _, ds := range dsList.GetValues() {
			for _, id := range ds.GetDutCriteria()[0].GetValues() {
				err := UpdateDeviceStability(ctx, id, ds)
				So(err, ShouldBeNil)
			}
		}
		Convey("GetDeviceStability", func() {
			resp, err := GetDeviceStability(ctx, "milkyway")
			So(err, ShouldBeNil)
			So(resp.GetStability().String(), ShouldEqual, "UNSTABLE")
		})
	})
}
