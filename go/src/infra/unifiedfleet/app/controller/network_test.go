// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/util"
)

func TestGetFreeIP(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("Get free IPs", t, func() {
		res, err := ImportVlans(ctx, []*crimsonconfig.VLAN{
			{
				Id:        40,
				Alias:     "free-vlan",
				CidrBlock: "192.168.40.0/22",
			},
		}, 25)
		So(err, ShouldBeNil)
		So(res.Failed(), ShouldHaveLength, 0)

		occupiedIPs := []*ufspb.IP{
			util.FormatIP("browser:40", "192.168.40.13", true),
			util.FormatIP("browser:40", "192.168.40.15", true),
		}
		res, err = configuration.ImportIPs(ctx, occupiedIPs)
		So(err, ShouldBeNil)
		So(res.Failed(), ShouldHaveLength, 0)

		newIPs, err := getFreeIP(ctx, "browser:40", 100)
		So(err, ShouldBeNil)
		So(newIPs, ShouldHaveLength, 100)
		ipStrs := make([]string, len(newIPs))
		for i, ip := range newIPs {
			ipStrs[i] = util.IPv4IntToStr(ip.GetIpv4())
		}
		So(ipStrs, ShouldNotContain, "192.168.40.13")
		So(ipStrs, ShouldNotContain, "192.168.40.15")
		So(ipStrs, ShouldContain, "192.168.40.14")
		// The last ip, 192.168.40.11 - 192.168.40.111
		So(ipStrs, ShouldContain, "192.168.40.111")
	})
}
