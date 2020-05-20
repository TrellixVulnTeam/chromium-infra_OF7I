// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"encoding/binary"
	"fmt"
	"net"

	"go.chromium.org/luci/common/errors"

	ufspb "infra/unifiedfleet/api/v1/proto"

	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
)

// ParseVlan parses vlan to a list of IPs
func ParseVlan(vlan *crimsonconfig.VLAN) ([]*ufspb.IP, int, error) {
	ip, subnet, err := net.ParseCIDR(vlan.CidrBlock)
	if err != nil {
		return nil, 0, errors.Reason("invalid CIDR block %q for vlan %d", vlan.CidrBlock, vlan.GetId()).Err()
	}
	ipv4 := ip.Mask(subnet.Mask).To4()
	if ipv4 == nil {
		return nil, 0, errors.Reason("invalid IPv4 CIDR block %q for vlan %d", vlan.CidrBlock, vlan.GetId()).Err()
	}
	ones, _ := subnet.Mask.Size()
	length := 1 << uint32(32-ones)
	ips := make([]*ufspb.IP, length)
	startIP := binary.BigEndian.Uint32(ipv4)
	for i := 0; i < length; i++ {
		ips[i] = &ufspb.IP{
			Id:   getIPName(vlan.GetId(), startIP),
			Ipv4: startIP,
		}
		startIP++
	}
	return ips, length, nil
}

func getIPName(vlan int64, ipv4 uint32) string {
	return fmt.Sprintf("vlan-%d/%d", vlan, ipv4)
}
