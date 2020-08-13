// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package utils

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	ufspb "infra/unifiedfleet/api/v1/proto"
)

func TestIsLocation(t *testing.T) {
	Convey("test standard chromeos location", t, func() {
		location := []string{
			"chromeos1-row2-rack3-host4",
			"chromeos1-row2-rack3-hostxxx",
			"ROW1-RACK2-HOST3",
			"chromeos6-floor",
			"chromeos6-rack1",
			"chromeos6-storage1",
			"2081-storage1",
			"em25-desk-stagenut",
			"chromeos6-row2-rack23-labstation1",
			"chromeos6-row2-rack23-labstation",
		}
		for _, l := range location {
			So(IsLocation(l), ShouldBeTrue)
		}
	})

	Convey("test invalid chromeos location", t, func() {
		location := "chromeos1-row2-rack3"
		So(IsLocation(location), ShouldBeFalse)
	})
}

func TestUFSZoneCoverage(t *testing.T) {
	Convey("test the ufs zone mapping covers all UFS zone enum", t, func() {
		got := make(map[string]bool, len(StrToUFSZone))
		for _, v := range StrToUFSZone {
			got[v] = true
		}
		for l := range ufspb.Zone_value {
			if l == ufspb.Zone_ZONE_UNSPECIFIED.String() {
				continue
			}
			_, ok := got[l]
			So(ok, ShouldBeTrue)
		}
	})

	Convey("test the ufs zone mapping doesn't cover any non-UFS zone enum", t, func() {
		for _, v := range StrToUFSZone {
			_, ok := ufspb.Zone_value[v]
			So(ok, ShouldBeTrue)
		}
	})
}

func TestReplaceZoneNames(t *testing.T) {
	Convey("TestReplaceZoneNames zonename error", t, func() {
		filter := "machine=mac-1,mac-2 & zone=XXX & nic=nic-1"
		_, err := ReplaceZoneNames(filter)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "Invalid zone name XXX for filtering.")
	})

	Convey("TestReplaceZoneNames zonename - happy path", t, func() {
		filter := "machine=mac-1,mac-2 & zone=atl97,mtv96 & nic=nic-1"
		filter, _ = ReplaceZoneNames(filter)
		So(filter, ShouldNotBeNil)
		So(filter, ShouldContainSubstring, "machine=mac-1,mac-2&zone=ZONE_ATL97,ZONE_MTV96&nic=nic-1")
	})
}
