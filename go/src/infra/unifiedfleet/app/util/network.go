// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package util

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"go.chromium.org/luci/common/errors"

	ufspb "infra/unifiedfleet/api/v1/proto"
)

// The first ip is the subnet ID
// The next first 10 ips are reserved
const reserveFirst = 11

// The last ip is broadcast address.
const reserveLast = 1

// ParseVlan parses vlan to a list of IPs
//
// vlanName here is a full vlan name, e.g. browser:123
// The first 10 and last 1 ip of this cidr block will be reserved and not returned to users
// for further operations
func ParseVlan(vlanName, cidr, freeStartIP, freeEndIP string) ([]*ufspb.IP, int, string, string, error) {
	ip, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, 0, "", "", errors.Reason("invalid CIDR block %q for vlan %s", cidr, vlanName).Err()
	}
	ipv4 := ip.Mask(subnet.Mask).To4()
	if ipv4 == nil {
		return nil, 0, "", "", errors.Reason("invalid IPv4 CIDR block %q for vlan %s", cidr, vlanName).Err()
	}
	ones, _ := subnet.Mask.Size()
	length := 1 << uint32(32-ones)
	ips := make([]*ufspb.IP, length)
	startIP := binary.BigEndian.Uint32(ipv4)
	freeStartIPInt := startIP + reserveFirst
	freeEndIPInt := startIP + uint32(length-reserveLast-1)
	if freeStartIP != "" {
		ipInt, err := IPv4StrToInt(freeStartIP)
		if err != nil {
			return nil, 0, "", "", errors.Reason("invalid free start IP %q for vlan %s", freeStartIP, vlanName).Err()
		}
		freeStartIPInt = ipInt
	} else {
		freeStartIP = IPv4IntToStr(uint32(startIP + reserveFirst))
	}
	if freeEndIP != "" {
		ipInt, err := IPv4StrToInt(freeEndIP)
		if err != nil {
			return nil, 0, "", "", errors.Reason("invalid free end IP %q for vlan %s", freeEndIP, vlanName).Err()
		}
		freeEndIPInt = ipInt
	} else {
		freeEndIP = IPv4IntToStr(startIP + uint32(length-reserveLast-1))
	}
	for i := 0; i < length; i++ {
		if startIP < freeStartIPInt || startIP > freeEndIPInt {
			ips[i] = &ufspb.IP{
				Id:      GetIPName(vlanName, Int64ToStr(int64(startIP))),
				Ipv4:    startIP,
				Ipv4Str: IPv4IntToStr(startIP),
				Vlan:    vlanName,
				Reserve: true,
			}
		} else {
			ips[i] = &ufspb.IP{
				Id:      GetIPName(vlanName, Int64ToStr(int64(startIP))),
				Ipv4:    startIP,
				Ipv4Str: IPv4IntToStr(startIP),
				Vlan:    vlanName,
			}
		}
		startIP++
	}
	return ips, length, freeStartIP, freeEndIP, nil
}

// FormatIP initialize an IP object
func FormatIP(vlanName, ipAddress string, reserve, occupied bool) *ufspb.IP {
	ipv4, err := IPv4StrToInt(ipAddress)
	if err != nil {
		return nil
	}
	return &ufspb.IP{
		Id:       GetIPName(vlanName, Int64ToStr(int64(ipv4))),
		Ipv4:     ipv4,
		Ipv4Str:  ipAddress,
		Vlan:     vlanName,
		Occupied: occupied,
		Reserve:  reserve,
	}
}

// IPv4StrToInt returns an uint32 address from the given ip address string.
func IPv4StrToInt(ipAddress string) (uint32, error) {
	ip := net.ParseIP(ipAddress)
	if ip != nil {
		ip = ip.To4()
	}
	if ip == nil {
		return 0, errors.Reason("invalid IPv4 address %q", ipAddress).Err()
	}
	return binary.BigEndian.Uint32(ip), nil
}

// IPv4IntToStr returns a string ip address
func IPv4IntToStr(ipAddress uint32) string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, ipAddress)
	return ip.String()
}

// parseCidrBlock returns a tuple of (cidr_block, capacity of this block)
func parseCidrBlock(subnet, mask string) (string, int) {
	maskIP := net.ParseIP(mask)
	maskAddr := maskIP.To4()
	ones, sz := net.IPv4Mask(maskAddr[0], maskAddr[1], maskAddr[2], maskAddr[3]).Size()
	return fmt.Sprintf("%s/%d", subnet, ones), 1 << uint32(sz-ones)
}

// ParseMac returns a valid mac address after parsing user input.
func ParseMac(userMac string) (string, error) {
	newUserMac := formatMac(userMac)
	m, err := net.ParseMAC(newUserMac)
	if err != nil || len(m) != 6 {
		return "", errors.Reason("invalid mac address %q (before parsing %q)", newUserMac, userMac).Err()
	}
	bytes := make([]byte, 8)
	copy(bytes[2:], m)
	mac := make(net.HardwareAddr, 8)
	binary.BigEndian.PutUint64(mac, binary.BigEndian.Uint64(bytes))
	return mac[2:].String(), nil
}

func formatMac(userMac string) string {
	if strings.Contains(userMac, ":") {
		return userMac
	}

	var newMac string
	for i := 0; ; i += 2 {
		if i+2 > len(userMac)-1 {
			newMac += userMac[i:]
			break
		}
		newMac += userMac[i:i+2] + ":"
	}
	return newMac
}
