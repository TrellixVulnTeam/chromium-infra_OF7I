// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	fleet "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/util"

	"go.chromium.org/luci/common/logging"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
)

// CreateMachineLSE creates a new machinelse in datastore.
//
// Checks if the resources referenced by the MachineLSE input already exists
// in the system before creating a new MachineLSE
func CreateMachineLSE(ctx context.Context, machinelse *fleet.MachineLSE) (*fleet.MachineLSE, error) {
	err := validateMachineLSE(ctx, machinelse)
	if err != nil {
		return nil, err
	}
	return inventory.CreateMachineLSE(ctx, machinelse)
}

// UpdateMachineLSE updates machinelse in datastore.
//
// Checks if the resources referenced by the MachineLSE input already exists
// in the system before updating a MachineLSE
func UpdateMachineLSE(ctx context.Context, machinelse *fleet.MachineLSE) (*fleet.MachineLSE, error) {
	err := validateMachineLSE(ctx, machinelse)
	if err != nil {
		return nil, err
	}
	return inventory.UpdateMachineLSE(ctx, machinelse)
}

// GetMachineLSE returns machinelse for the given id from datastore.
func GetMachineLSE(ctx context.Context, id string) (*fleet.MachineLSE, error) {
	return inventory.GetMachineLSE(ctx, id)
}

// ListMachineLSEs lists the machinelses
func ListMachineLSEs(ctx context.Context, pageSize int32, pageToken string) ([]*fleet.MachineLSE, string, error) {
	return inventory.ListMachineLSEs(ctx, pageSize, pageToken)
}

// DeleteMachineLSE deletes the machinelse in datastore
//
// For referential data intergrity,
// Delete if this MachineLSE is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteMachineLSE(ctx context.Context, id string) error {
	return inventory.DeleteMachineLSE(ctx, id)
}

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
	lps := []*fleet.MachineLSEPrototype{
		{
			Name: "browser-lab:no-vm",
			VirtualRequirements: []*fleet.VirtualRequirement{
				{
					VirtualType: fleet.VirtualType_VIRTUAL_TYPE_VM,
					Min:         0,
					Max:         0,
				},
			},
		},
		{
			Name: "browser-lab:vm",
			VirtualRequirements: []*fleet.VirtualRequirement{
				{
					VirtualType: fleet.VirtualType_VIRTUAL_TYPE_VM,
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

// validateMachineLSE validates if a machinelse can be created/updated in the datastore.
//
// Checks if the resources referenced by the given MachineLSE input already exists
// in the system. Returns an error if any resource referenced by the MachineLSE input
// does not exist in the system.
func validateMachineLSE(ctx context.Context, machinelse *fleet.MachineLSE) error {
	var resources []*Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create MachineLSE %s:\n", machinelse.Name))

	machineIDs := machinelse.GetMachines()
	machineLSEPrototypeID := machinelse.GetMachineLsePrototype()
	vlanID := machinelse.GetChromeosMachineLse().GetServer().GetSupportedRestrictedVlan()
	rpmID := machinelse.GetChromeosMachineLse().GetDut().GetRpmInterface().GetRpm()
	switchID := machinelse.GetChromeosMachineLse().GetDut().GetNetworkDeviceInterface().GetSwitch()

	if len(machineIDs) != 0 {
		for _, machineID := range machineIDs {
			resources = append(resources, GetMachineResource(machineID))
		}
	}
	if machineLSEPrototypeID != "" {
		resources = append(resources, GetMachineLSEProtoTypeResource(machineLSEPrototypeID))
	}
	if vlanID != "" {
		resources = append(resources, GetVlanResource(vlanID))
	}
	if rpmID != "" {
		resources = append(resources, GetRPMResource(rpmID))
	}
	if switchID != "" {
		resources = append(resources, GetSwitchResource(switchID))
	}

	return ResourceExist(ctx, resources, &errorMsg)
}
