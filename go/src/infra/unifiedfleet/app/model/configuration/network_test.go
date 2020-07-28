// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/appengine/gaetesting"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
)

func TestImportDHCPConfigs(t *testing.T) {
	t.Parallel()
	ctx := gaetesting.TestingContextWithAppID("go-test")
	datastore.GetTestable(ctx).Consistent(true)
	Convey("import nics", t, func() {
		dhcps := []*ufspb.DHCPConfig{
			mockDHCPConfig("hostname1", "ip1"),
			mockDHCPConfig("hostname2", "ip2"),
		}
		Convey("happy path", func() {
			resp, err := ImportDHCPConfigs(ctx, dhcps)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, len(dhcps))
			getRes, _, err := ListDHCPConfigs(ctx, 100, "", nil, false)
			So(err, ShouldBeNil)
			So(getRes, ShouldResembleProto, dhcps)
		})
		Convey("happy path also for importing existing dhcp configs", func() {
			dhcps1 := []*ufspb.DHCPConfig{
				mockDHCPConfig("hostname1", "ip1-1"),
			}
			resp, err := ImportDHCPConfigs(ctx, dhcps1)
			So(err, ShouldBeNil)
			So(resp.Passed(), ShouldHaveLength, len(dhcps1))
			s, err := GetDHCPConfig(ctx, "hostname1")
			So(err, ShouldBeNil)
			So(s.GetIp(), ShouldEqual, "ip1-1")
		})
	})
}

func mockDHCPConfig(hostname, ipv4 string) *ufspb.DHCPConfig {
	return &ufspb.DHCPConfig{
		Hostname: hostname,
		Ip:       ipv4,
	}
}
