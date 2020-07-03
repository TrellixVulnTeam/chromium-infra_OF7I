// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"

	ufspb "infra/unifiedfleet/api/v1/proto"
	api "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

// ImportDatacenter imports the datacenters and the related resources.
//
// The function will return:
//      * all of the results of the operations that already run
//      * the first error that it meets
//
// The function will stop at the very first error.
func ImportDatacenter(ctx context.Context, dc *crimsonconfig.Datacenter, pageSize int) (*datastore.OpResults, error) {
	allRes := make(datastore.OpResults, 0)
	logging.Debugf(ctx, "Importing the basic lse prototypes for browser lab")
	lps := []*ufspb.RackLSEPrototype{
		{
			Name: "browser-lab:normal",
			PeripheralRequirement: []*ufspb.PeripheralRequirement{
				{
					PeripheralType: ufspb.PeripheralType_PERIPHERAL_TYPE_KVM,
					Min:            0,
					Max:            2,
				},
				{
					PeripheralType: ufspb.PeripheralType_PERIPHERAL_TYPE_SWITCH,
					Min:            0,
					// A manual setup with no consideration, the current maximum
					// switches per rack is 3 in crimson.
					Max: 5,
				},
			},
		},
	}
	res, err := configuration.ImportRackLSEPrototypes(ctx, lps)
	if err != nil {
		return res, err
	}
	allRes = append(allRes, *res...)

	racks, rackLSEs, kvms, switches, dhcps := util.ProcessDatacenters(dc)
	logging.Debugf(ctx, "Got %d racks, %d fack LSEs, %d kvms, %d switches, %d dhcp configs", len(racks), len(rackLSEs), len(kvms), len(switches), len(dhcps))
	if err := api.ValidateResourceKey(racks, "Name"); err != nil {
		return nil, errors.Annotate(err, "racks has invalid chars").Err()
	}
	if err := api.ValidateResourceKey(rackLSEs, "Name"); err != nil {
		return nil, errors.Annotate(err, "rackLSEs has invalid chars").Err()
	}
	if err := api.ValidateResourceKey(kvms, "Name"); err != nil {
		return nil, errors.Annotate(err, "kvms has invalid chars").Err()
	}
	if err := api.ValidateResourceKey(switches, "Name"); err != nil {
		return nil, errors.Annotate(err, "switches has invalid chars").Err()
	}
	if err := api.ValidateResourceKey(dhcps, "Hostname"); err != nil {
		return nil, errors.Annotate(err, "dhcps has invalid chars").Err()
	}

	// Please note that the importing here is not in one transaction, which
	// actually may cause data incompleteness. But as the importing job
	// will be triggered periodically, such incompleteness that's caused by
	// potential failure will be ignored.
	logging.Debugf(ctx, "Importing %d racks", len(racks))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(racks))
		logging.Debugf(ctx, "importing rack %dth - %dth", i, end-1)
		res, err := registration.ImportRacks(ctx, racks[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(racks) {
			break
		}
	}
	logging.Debugf(ctx, "Importing %d rack LSEs", len(rackLSEs))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(rackLSEs))
		logging.Debugf(ctx, "importing rack LSE %dth - %dth", i, end-1)
		res, err := inventory.ImportRackLSEs(ctx, rackLSEs[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(rackLSEs) {
			break
		}
	}
	logging.Debugf(ctx, "Importing %d kvms", len(kvms))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(kvms))
		logging.Debugf(ctx, "importing kvm %dth - %dth", i, end-1)
		res, err := registration.ImportKVMs(ctx, kvms[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(kvms) {
			break
		}
	}
	logging.Debugf(ctx, "Importing %d switches", len(switches))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(switches))
		logging.Debugf(ctx, "importing switch %dth - %dth", i, end-1)
		res, err := registration.ImportSwitches(ctx, switches[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(switches) {
			break
		}
	}
	logging.Debugf(ctx, "Importing %d DHCP configs", len(dhcps))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(dhcps))
		logging.Debugf(ctx, "importing dhcp configs %dth - %dth", i, end-1)
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
