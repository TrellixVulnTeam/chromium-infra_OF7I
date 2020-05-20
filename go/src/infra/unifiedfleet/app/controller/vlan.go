// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"go.chromium.org/luci/common/logging"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

// ImportVlans implements the logic of importing vlans and related info to backend storage.
//
// The function will return:
//      * all of the results of the operations that already run
//      * the first error that it meets
//
// The function will stop at the very first error.
func ImportVlans(ctx context.Context, vlans []*crimsonconfig.VLAN, pageSize int) (*datastore.OpResults, error) {
	logging.Debugf(ctx, "processing vlans")
	IPs := make([]*ufspb.IP, 0)
	vs := make([]*ufspb.Vlan, len(vlans))
	for i, vlan := range vlans {
		vlanName := util.GetBrowserLabVlanName(vlan.GetId())
		ips, length, err := util.ParseVlan(vlan)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			ip.Vlan = vlanName
			IPs = append(IPs, ip)
		}
		vs[i] = &ufspb.Vlan{
			Name:        vlanName,
			Description: vlan.GetAlias(),
			CapacityIp:  int32(length),
			VlanAddress: vlan.GetCidrBlock(),
		}
	}
	allRes := make(datastore.OpResults, 0)
	logging.Debugf(ctx, "Importing %d vlans", len(vs))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(vs))
		logging.Debugf(ctx, "importing vlan %dth - %dth", i, end-1)
		res, err := registration.ImportVlans(ctx, vs[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(vs) {
			break
		}
	}

	logging.Debugf(ctx, "Importing %d ips", len(IPs))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(IPs))
		logging.Debugf(ctx, "importing ip %dth - %dth", i, end-1)
		res, err := configuration.ImportIPs(ctx, IPs[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(IPs) {
			break
		}
	}
	return &allRes, nil
}
