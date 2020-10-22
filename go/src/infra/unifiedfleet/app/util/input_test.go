// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	ufspb "infra/unifiedfleet/api/v1/models"
)

func TestUFSStateCoverage(t *testing.T) {
	Convey("test the ufs state mapping covers all UFS state enum", t, func() {
		got := make(map[string]bool, len(StrToUFSState))
		for _, v := range StrToUFSState {
			got[v] = true
		}
		for l := range ufspb.State_value {
			if l == ufspb.State_STATE_UNSPECIFIED.String() {
				continue
			}
			_, ok := got[l]
			So(ok, ShouldBeTrue)
		}
	})

	Convey("test the ufs state mapping doesn't cover any non-UFS state enum", t, func() {
		for _, v := range StrToUFSState {
			_, ok := ufspb.State_value[v]
			So(ok, ShouldBeTrue)
		}
	})
}

func TestGetResourcePrefix(t *testing.T) {
	Convey("Test various proto message", t, func() {
		Convey("Test machine proto", func() {
			machine := &ufspb.Machine{}
			res, err := GetResourcePrefix(machine)
			So(err, ShouldBeNil)
			So(res, ShouldEqual, "machines")
		})
		Convey("Test asset proto", func() {
			asset := &ufspb.Asset{}
			res, err := GetResourcePrefix(asset)
			So(err, ShouldBeNil)
			So(res, ShouldEqual, "assets")
		})
		Convey("Test event proto", func() {
			event := &ufspb.ChangeEvent{}
			res, err := GetResourcePrefix(event)
			So(err, ShouldBeNil)
			So(res, ShouldEqual, "events")
		})
		Convey("Test chrome platform proto", func() {
			platform := &ufspb.ChromePlatform{}
			res, err := GetResourcePrefix(platform)
			So(err, ShouldBeNil)
			So(res, ShouldEqual, "chromePlatforms")
		})
		Convey("Test nic proto", func() {
			nic := &ufspb.Nic{}
			res, err := GetResourcePrefix(nic)
			So(err, ShouldBeNil)
			So(res, ShouldEqual, "nics")
		})
		Convey("Test vlan proto", func() {
			vlan := &ufspb.Vlan{}
			res, err := GetResourcePrefix(vlan)
			So(err, ShouldBeNil)
			So(res, ShouldEqual, "vlans")
		})
		Convey("Test kvm proto", func() {
			kvm := &ufspb.KVM{}
			res, err := GetResourcePrefix(kvm)
			So(err, ShouldBeNil)
			So(res, ShouldEqual, "kvms")
		})
		Convey("Test rpm proto", func() {
			rpm := &ufspb.RPM{}
			res, err := GetResourcePrefix(rpm)
			So(err, ShouldBeNil)
			So(res, ShouldEqual, "rpms")
		})
		Convey("Test switch proto", func() {
			swch := &ufspb.Switch{}
			res, err := GetResourcePrefix(swch)
			So(err, ShouldBeNil)
			So(res, ShouldEqual, "switches")
		})
		Convey("Test rack proto", func() {
			rack := &ufspb.Rack{}
			res, err := GetResourcePrefix(rack)
			So(err, ShouldBeNil)
			So(res, ShouldEqual, "racks")
		})
	})
}
