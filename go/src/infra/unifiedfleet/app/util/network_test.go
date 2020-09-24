// Copyright 2020 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestParseVlan(t *testing.T) {
	Convey("ParseVlan - happy path", t, func() {
		ips, l, freeStartIP, freeEndIP, err := ParseVlan("fake_vlan", "192.168.40.0/22")
		So(err, ShouldBeNil)
		So(l, ShouldEqual, 1024)
		So(ips, ShouldHaveLength, 1024)
		for i, ip := range ips {
			if i >= 0 && i < 11 {
				So(ip.GetReserve(), ShouldBeTrue)
			} else if i >= 1023 {
				So(ip.GetReserve(), ShouldBeTrue)
			} else {
				So(ip.GetReserve(), ShouldBeFalse)
			}
		}
		So(freeStartIP, ShouldEqual, "192.168.40.11")
		So(freeEndIP, ShouldEqual, "192.168.43.254")
	})
}

func TestParseMac(t *testing.T) {
	Convey("ParseMac - happy path", t, func() {
		mac, err := ParseMac("12:34:56:78:90:ab")
		So(err, ShouldBeNil)
		So(mac, ShouldEqual, "12:34:56:78:90:ab")
	})

	Convey("ParseMac - happy path without colon separators", t, func() {
		mac, err := ParseMac("1234567890ab")
		So(err, ShouldBeNil)
		So(mac, ShouldEqual, "12:34:56:78:90:ab")
	})

	Convey("ParseMac - invalid characters", t, func() {
		invalidMacs := []string{
			"1234567890,b",
			"hello world",
			"123455678901234567890",
		}
		for _, userMac := range invalidMacs {
			mac, err := ParseMac(userMac)
			So(err, ShouldNotBeNil)
			So(mac, ShouldBeEmpty)
		}
	})
}

func TestFormatMac(t *testing.T) {
	Convey("formatMac - happy path with colon separators", t, func() {
		So(formatMac("12:34:56:78:90:ab"), ShouldEqual, "12:34:56:78:90:ab")
	})

	Convey("formatMac - happy path without colon separators", t, func() {
		So(formatMac("1234567890ab"), ShouldEqual, "12:34:56:78:90:ab")
	})

	Convey("formatMac - odd length", t, func() {
		So(formatMac("1234567890abcde"), ShouldEqual, "12:34:56:78:90:ab:cd:e")
	})
}
