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
func ImportDatacenter(ctx context.Context, dcs []*crimsonconfig.Datacenter, pageSize int) (*datastore.OpResults, error) {
	allRes := make(datastore.OpResults, 0)
	logging.Debugf(ctx, "Importing the basic lse prototypes for browser lab")
	lps := []*ufspb.RackLSEPrototype{
		{
			Name: "browser-lab:normal",
			PeripheralRequirements: []*ufspb.PeripheralRequirement{
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
			Tags: []string{"browser", "normal"},
		},
	}
	res, err := configuration.ImportRackLSEPrototypes(ctx, lps)
	if err != nil {
		return res, err
	}
	allRes = append(allRes, *res...)

	racks := make([]*ufspb.Rack, 0)
	rackLSEs := make([]*ufspb.RackLSE, 0)
	kvms := make([]*ufspb.KVM, 0)
	switches := make([]*ufspb.Switch, 0)
	dhcps := make([]*ufspb.DHCPConfig, 0)
	for _, dc := range dcs {
		rackRes, lseRes, kvmRes, switchRes, dhcpRes := util.ProcessDatacenters(dc)
		racks = append(racks, rackRes...)
		rackLSEs = append(rackLSEs, lseRes...)
		kvms = append(kvms, kvmRes...)
		switches = append(switches, switchRes...)
		dhcps = append(dhcps, dhcpRes...)
	}

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
	deleteNonExistingRacks(ctx, racks, pageSize)
	logging.Debugf(ctx, "Importing %d racks", len(racks))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(racks))
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
		res, err := inventory.ImportRackLSEs(ctx, rackLSEs[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(rackLSEs) {
			break
		}
	}
	deleteNonExistingKVMs(ctx, kvms, pageSize)
	logging.Debugf(ctx, "Importing %d kvms", len(kvms))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(kvms))
		res, err := registration.ImportKVMs(ctx, kvms[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(kvms) {
			break
		}
	}
	deleteNonExistingSwitches(ctx, switches, pageSize)
	logging.Debugf(ctx, "Importing %d switches", len(switches))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(switches))
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

func deleteNonExistingRacks(ctx context.Context, racks []*ufspb.Rack, pageSize int) (*datastore.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range racks {
		resMap[r.GetName()] = true
	}
	rackRes, err := registration.GetAllRacks(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	for _, sr := range rackRes.Passed() {
		s := sr.Data.(*ufspb.Rack)
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
		}
	}
	logging.Debugf(ctx, "Deleting %d non-existing racks", len(toDelete))
	return deleteByPage(ctx, toDelete, pageSize, registration.DeleteRacks), nil
}

func deleteNonExistingKVMs(ctx context.Context, kvms []*ufspb.KVM, pageSize int) (*datastore.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range kvms {
		resMap[r.GetName()] = true
	}
	resp, err := registration.GetAllKVMs(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.KVM)
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
		}
	}
	logging.Debugf(ctx, "Deleting %d non-existing kvms", len(toDelete))
	allRes := *deleteByPage(ctx, toDelete, pageSize, registration.DeleteKVMs)
	logging.Debugf(ctx, "Deleting %d non-existing kvm-related dhcps", len(toDelete))
	allRes = append(allRes, *deleteByPage(ctx, toDelete, pageSize, configuration.DeleteDHCPs)...)
	return &allRes, nil
}

func deleteNonExistingSwitches(ctx context.Context, switches []*ufspb.Switch, pageSize int) (*datastore.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range switches {
		resMap[r.GetName()] = true
	}
	resp, err := registration.GetAllSwitches(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.Switch)
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
		}
	}
	logging.Debugf(ctx, "Deleting %d non-existing switches", len(toDelete))
	return deleteByPage(ctx, toDelete, pageSize, registration.DeleteSwitches), nil
}
