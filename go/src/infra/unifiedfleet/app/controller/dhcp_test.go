// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	. "go.chromium.org/luci/common/testing/assertions"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
)

func TestBatchGetDHCPs(t *testing.T) {
	t.Parallel()
	ctx := testingContext()
	Convey("BatchGetDHCPs", t, func() {
		Convey("Batch get dhcps - happy path", func() {
			dhcps := make([]*ufspb.DHCPConfig, 4)
			for i := 0; i < 4; i++ {
				dhcp := &ufspb.DHCPConfig{
					Hostname: fmt.Sprintf("dhcp-batchGet-%d", i),
					Ip:       fmt.Sprintf("%d", i),
				}
				dhcps[i] = dhcp
			}
			_, err := configuration.BatchUpdateDHCPs(ctx, dhcps)
			So(err, ShouldBeNil)
			resp, err := configuration.BatchGetDHCPConfigs(ctx, []string{"dhcp-batchGet-0", "dhcp-batchGet-1", "dhcp-batchGet-2", "dhcp-batchGet-3"})
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 4)
			So(resp, ShouldResembleProto, dhcps)
		})
		Convey("Batch get dhcps - missing id", func() {
			resp, err := configuration.BatchGetDHCPConfigs(ctx, []string{"dhcp-batchGet-non-existing"})
			So(err, ShouldNotBeNil)
			So(resp, ShouldBeNil)
			So(err.Error(), ShouldContainSubstring, "dhcp-batchGet-non-existing")
		})
		Convey("Batch get dhcps - empty input", func() {
			resp, err := configuration.BatchGetDHCPConfigs(ctx, nil)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)

			input := make([]string, 0)
			resp, err = configuration.BatchGetDHCPConfigs(ctx, input)
			So(err, ShouldBeNil)
			So(resp, ShouldHaveLength, 0)
		})
	})
}
