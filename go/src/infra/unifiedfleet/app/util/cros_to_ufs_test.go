// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"testing"

	"github.com/golang/protobuf/proto"
	. "github.com/smartystreets/goconvey/convey"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/frontend/fake"
)

func TestCopyDUT(t *testing.T) {
	Convey("Verify copyDUT", t, func() {
		Convey("happy path", func() {
			mockLabConfig := fake.GetMockDUT()
			newDUT := copyDUT(mockLabConfig.GetConfig().GetDut())
			nb, err := proto.Marshal(newDUT)
			So(err, ShouldBeNil)
			ob, err := proto.Marshal(mockLabConfig.Config.GetDut())
			So(err, ShouldBeNil)
			So(nb, ShouldResemble, ob)
		})
	})
}

func TestCopyLabstation(t *testing.T) {
	Convey("Verify copyLabstation", t, func() {
		Convey("happy path", func() {
			mockLabConfig := fake.GetMockLabstation()
			newL := copyLabstation(mockLabConfig.GetConfig().GetLabstation())
			nb, err := proto.Marshal(newL)
			So(err, ShouldBeNil)
			ob, err := proto.Marshal(mockLabConfig.GetConfig().GetLabstation())
			So(err, ShouldBeNil)
			So(nb, ShouldResemble, ob)
		})
	})
}

func TestParseATLTopology(t *testing.T) {
	Convey("Verify ParseATLTopology", t, func() {
		Convey("happy path", func() {
			resp, err := fake.SheetData("../frontend/fake/sheet_data.json")
			So(err, ShouldBeNil)
			topology, _ := ParseATLTopology(resp)
			So(topology, ShouldHaveLength, 2)
			So(topology, ShouldContainKey, "100.115.224.0")
			So(topology, ShouldContainKey, "100.115.226.0")
			for k, vlan := range topology {
				switch k {
				case "100.115.224.0":
					So(vlan.GetName(), ShouldEqual, "atl:201")
					So(vlan.GetCapacityIp(), ShouldEqual, 510)
					So(vlan.GetVlanAddress(), ShouldEqual, "100.115.224.0/23")
					So(vlan.GetDescription(), ShouldEqual, "ATL-DUT-Row1_2")
				case "100.115.226.0":
					So(vlan.GetName(), ShouldEqual, "atl:202")
					So(vlan.GetCapacityIp(), ShouldEqual, 510)
					So(vlan.GetVlanAddress(), ShouldEqual, "100.115.226.0/23")
					So(vlan.GetDescription(), ShouldEqual, "ATL-DUT-Row3_4")
				}
			}

		})
	})
}

func TestParseOSDhcpdConf(t *testing.T) {
	Convey("Verify ParseOSDhcpdConf", t, func() {
		Convey("happy path", func() {
			resp, err := fake.SheetData("../frontend/fake/sheet_data.json")
			So(err, ShouldBeNil)
			topology, _ := ParseATLTopology(resp)
			b, err := fake.GitData("../frontend/fake/dhcp_test.conf")
			So(err, ShouldBeNil)

			parsed, err := ParseOSDhcpdConf(string(b), topology)
			So(err, ShouldBeNil)
			So(parsed.ValidVlans, ShouldHaveLength, 2)
			So(parsed.ValidIPs, ShouldHaveLength, 510)
			ipMaps := make(map[string]*ufspb.IP, 0)
			for _, ip := range parsed.ValidIPs {
				ipMaps[ip.GetId()] = ip
			}
			So(len(ipMaps), ShouldEqual, 510)
			ip, ok := ipMaps["atl:201/100.115.224.1"]
			So(ok, ShouldBeTrue)
			So(ip.GetOccupied(), ShouldBeTrue)
			So(ip.GetVlan(), ShouldEqual, "atl:201")
			ip2, ok := ipMaps["atl:201/100.115.224.2"]
			So(ok, ShouldBeTrue)
			So(ip2.GetOccupied(), ShouldBeTrue)
			So(ip2.GetVlan(), ShouldEqual, "atl:201")
			ip3, ok := ipMaps["atl:201/100.115.224.3"]
			So(ok, ShouldBeTrue)
			So(ip3.GetOccupied(), ShouldBeTrue)
			So(ip3.GetVlan(), ShouldEqual, "atl:201")

			So(parsed.ValidDHCPs, ShouldHaveLength, 3)
			for _, dhcp := range parsed.ValidDHCPs {
				So([]string{"host1", "host2", "host3"}, ShouldContain, dhcp.GetHostname())
				switch dhcp.GetHostname() {
				case "host1":
					So(dhcp.GetIp(), ShouldEqual, "100.115.224.1")
					So(dhcp.GetMacAddress(), ShouldEqual, "aa:00:00:00:00:00")
				case "host2":
					So(dhcp.GetIp(), ShouldEqual, "100.115.224.2")
					So(dhcp.GetMacAddress(), ShouldEqual, "")
				case "host3":
					So(dhcp.GetIp(), ShouldEqual, "100.115.224.3")
					So(dhcp.GetMacAddress(), ShouldEqual, "")
				}
			}
		})
	})
}
