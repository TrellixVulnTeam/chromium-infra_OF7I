// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/logging"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/libs/cros/sheet"
	fleet "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/util"
)

// CreateVlan creates a new vlan in datastore.
func CreateVlan(ctx context.Context, vlan *fleet.Vlan) (*fleet.Vlan, error) {
	return configuration.CreateVlan(ctx, vlan)
}

// UpdateVlan updates vlan in datastore.
func UpdateVlan(ctx context.Context, vlan *fleet.Vlan) (*fleet.Vlan, error) {
	return configuration.UpdateVlan(ctx, vlan)
}

// GetVlan returns vlan for the given id from datastore.
func GetVlan(ctx context.Context, id string) (*fleet.Vlan, error) {
	return configuration.GetVlan(ctx, id)
}

// ListVlans lists the vlans
func ListVlans(ctx context.Context, pageSize int32, pageToken string) ([]*fleet.Vlan, string, error) {
	return configuration.ListVlans(ctx, pageSize, pageToken)
}

// DeleteVlan deletes the vlan in datastore
//
// For referential data intergrity,
// Delete if this Vlan is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteVlan(ctx context.Context, id string) error {
	err := validateDeleteVlan(ctx, id)
	if err != nil {
		return err
	}
	return configuration.DeleteVlan(ctx, id)
}

// ImportVlans implements the logic of importing vlans and related info to backend storage.
//
// The function will return:
//      * all of the results of the operations that already run
//      * the first error that it meets
//
// The function will stop at the very first error.
func ImportVlans(ctx context.Context, vlans []*crimsonconfig.VLAN, pageSize int) (*datastore.OpResults, error) {
	logging.Debugf(ctx, "processing vlans")
	IPs := make([]*fleet.IP, 0)
	vs := make([]*fleet.Vlan, len(vlans))
	for i, vlan := range vlans {
		vlanName := util.GetBrowserLabName(util.Int64ToStr(vlan.GetId()))
		ips, length, err := util.ParseVlan(vlan)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			ip.Vlan = vlanName
			IPs = append(IPs, ip)
		}
		vs[i] = &fleet.Vlan{
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
		res, err := configuration.ImportVlans(ctx, vs[i:end])
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

// ImportOSVlans imports the logic of parse and save network infos.
func ImportOSVlans(ctx context.Context, sheetClient sheet.ClientInterface, pageSize int) (*datastore.OpResults, error) {
	networkCfg := config.Get(ctx).GetCrosNetworkConfig()
	allVlans := make([]*fleet.Vlan, 0)
	allIPs := make([]*fleet.IP, 0)
	allDhcps := make([]*fleet.DHCPConfig, 0)

	// TODO: add logic to parse vlans
	for _, cfg := range networkCfg.GetCrosNetworkTopology() {
		resp, err := sheetClient.Get(ctx, cfg.GetSheetId(), []string{"VLANs and Netblocks"})
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		util.ParseATLTopology(resp)
	}

	allRes := make(datastore.OpResults, 0)
	logging.Debugf(ctx, "Importing %d vlans", len(allVlans))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(allVlans))
		logging.Debugf(ctx, "importing vlan %dth - %dth", i, end-1)
		res, err := configuration.ImportVlans(ctx, allVlans[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(allVlans) {
			break
		}
	}

	logging.Debugf(ctx, "Importing %d ips", len(allIPs))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(allIPs))
		logging.Debugf(ctx, "importing ip %dth - %dth", i, end-1)
		res, err := configuration.ImportIPs(ctx, allIPs[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(allIPs) {
			break
		}
	}

	logging.Debugf(ctx, "Importing %d ips", len(allDhcps))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(allDhcps))
		logging.Debugf(ctx, "importing ip %dth - %dth", i, end-1)
		res, err := configuration.ImportDHCPConfigs(ctx, allDhcps[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(allDhcps) {
			break
		}
	}
	return &allRes, nil
}

// ReplaceVlan replaces an old Vlan with new Vlan in datastore
//
// It does a delete of old vlan and create of new Vlan.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old Vlan, it will get all the resources referencing
// the old Vlan. It will update all the resources which were referencing
// the old Vlan(got in the last step) with new Vlan.
// Deletes the old Vlan.
// Creates the new Vlan.
// This will preserve data integrity in the system.
func ReplaceVlan(ctx context.Context, oldVlan *fleet.Vlan, newVlan *fleet.Vlan) (*fleet.Vlan, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateDeleteVlan validates if a Vlan can be deleted
//
// Checks if this Vlan(VlanID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteVlan(ctx context.Context, id string) error {
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "vlan_id", id, true)
	if err != nil {
		return err
	}
	if len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Vlan %s cannot be deleted because there are other resources which are referring this Vlan.", id))
		errorMsg.WriteString(fmt.Sprintf("\nMachineLSEs referring the Vlan:\n"))
		for _, machinelse := range machinelses {
			errorMsg.WriteString(machinelse.Name + ", ")
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}
