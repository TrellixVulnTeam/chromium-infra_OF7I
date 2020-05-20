// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/util"

	"go.chromium.org/luci/common/logging"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"

	ufspb "infra/unifiedfleet/api/v1/proto"
)

// ImportMachineLSEs implements the logic of importing machine lses and related info to backend storage.
//
// The function will return:
//      * all of the results of the operations that already run
//      * the first error that it meets
//
// The function will stop at the very first error.
func ImportMachineLSEs(ctx context.Context, hosts []*crimson.PhysicalHost, vms []*crimson.VM, pageSize int) (*datastore.OpResults, error) {
	allRes := make(datastore.OpResults, 0)
	logging.Debugf(ctx, "Importing the basic lse prototypes for browser lab")
	lps := []*ufspb.MachineLSEPrototype{
		{
			Name: "browser-lab:no-vm",
			VirtualRequirements: []*ufspb.VirtualRequirement{
				{
					VirtualType: ufspb.VirtualType_VIRTUAL_TYPE_VM,
					Min:         0,
					Max:         0,
				},
			},
		},
		{
			Name: "browser-lab:vm",
			VirtualRequirements: []*ufspb.VirtualRequirement{
				{
					VirtualType: ufspb.VirtualType_VIRTUAL_TYPE_VM,
					Min:         1,
					// A random number, not true.
					Max: 100,
				},
			},
		},
	}
	res, err := configuration.ImportMachineLSEPrototypes(ctx, lps)
	if err != nil {
		return res, err
	}
	allRes = append(allRes, *res...)

	lses, ips, dhcps := util.ToMachineLSEs(hosts, vms)
	logging.Debugf(ctx, "Importing %d lses", len(lses))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(lses))
		logging.Debugf(ctx, "importing lses %dth - %dth", i, end-1)
		res, err := inventory.ImportMachineLSEs(ctx, lses[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(lses) {
			break
		}
	}

	logging.Debugf(ctx, "Importing %d ips", len(ips))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(ips))
		logging.Debugf(ctx, "importing ips %dth - %dth", i, end-1)
		res, err := configuration.ImportIPs(ctx, ips[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(ips) {
			break
		}
	}

	logging.Debugf(ctx, "Importing %d dhcps", len(dhcps))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(dhcps))
		logging.Debugf(ctx, "importing dhcps %dth - %dth", i, end-1)
		res, err := configuration.ImportDHCPConfigs(ctx, dhcps[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(dhcps) {
			break
		}
	}
	return &allRes, nil
}
