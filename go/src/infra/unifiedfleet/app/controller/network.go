// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/util"
)

// A helper function to get free IP based on a given vlan name. The vlan name's format is: <lab-prefix>:XXX
func getFreeIP(ctx context.Context, vlanName string, pageSize int) ([]*ufspb.IP, error) {
	ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"vlan": vlanName, "occupied": "false", "reserve": "false"})
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no non-occupied ips in vlan %s", vlanName)
	}
	res := make([]*ufspb.IP, pageSize)
	count := 0
	for _, ip := range ips {
		// Double check for now to ensure IP is not occupied
		dhcps, err := configuration.QueryDHCPConfigByPropertyName(ctx, "ipv4", util.IPv4IntToStr(ip.GetIpv4()))
		if err != nil {
			return nil, err
		}
		if len(dhcps) == 0 {
			res[count] = ip
			count++
			if count >= pageSize {
				break
			}
		}
	}
	return res, nil
}
