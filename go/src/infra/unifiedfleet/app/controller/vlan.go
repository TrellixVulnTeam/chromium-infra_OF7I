// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	crimsonconfig "go.chromium.org/luci/machine-db/api/config/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/libs/cros/git"
	"infra/libs/cros/sheet"
	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/config"
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/util"
)

// CreateVlan creates a new vlan in datastore.
func CreateVlan(ctx context.Context, vlan *ufspb.Vlan) (*ufspb.Vlan, error) {
	return configuration.CreateVlan(ctx, vlan)
}

// UpdateVlan updates vlan in datastore.
func UpdateVlan(ctx context.Context, vlan *ufspb.Vlan) (*ufspb.Vlan, error) {
	return configuration.UpdateVlan(ctx, vlan)
}

// GetVlan returns vlan for the given id from datastore.
func GetVlan(ctx context.Context, id string) (*ufspb.Vlan, error) {
	return configuration.GetVlan(ctx, id)
}

// ListVlans lists the vlans
func ListVlans(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.Vlan, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, configuration.GetVlanIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing vlans").Err()
		}
	}
	return configuration.ListVlans(ctx, pageSize, pageToken, filterMap, keysOnly)
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
	IPs := make([]*ufspb.IP, 0)
	vs := make([]*ufspb.Vlan, len(vlans))
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
		vs[i] = &ufspb.Vlan{
			Name:        vlanName,
			Description: vlan.GetAlias(),
			CapacityIp:  int32(length),
			VlanAddress: vlan.GetCidrBlock(),
		}
	}
	deleteNonExistingVlans(ctx, vs, pageSize)
	allRes := make(datastore.OpResults, 0)
	logging.Debugf(ctx, "Importing %d vlans", len(vs))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(vs))
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

func deleteNonExistingVlans(ctx context.Context, vlans []*ufspb.Vlan, pageSize int) (*datastore.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range vlans {
		resMap[r.GetName()] = true
	}
	resp, err := configuration.GetAllVlans(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	var toDeleteIP []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.Vlan)
		if util.IsInBrowserLab(s.GetName()) {
			if _, ok := resMap[s.GetName()]; !ok {
				toDelete = append(toDelete, s.GetName())
				ips, err := configuration.QueryIPByPropertyName(ctx, map[string]string{"vlan": s.GetName()})
				if err != nil {
					return nil, err
				}
				for _, ip := range ips {
					toDeleteIP = append(toDeleteIP, ip.GetId())
				}
			}
		}
	}

	logging.Debugf(ctx, "Deleting %d non-existing ips ", len(toDeleteIP))
	deleteByPage(ctx, toDeleteIP, pageSize, configuration.DeleteIPs)
	logging.Debugf(ctx, "Deleting %d non-existing vlans ", len(toDelete))
	return deleteByPage(ctx, toDelete, pageSize, configuration.DeleteVlans), nil
}

// ImportOSVlans imports the logic of parse and save network infos.
func ImportOSVlans(ctx context.Context, sheetClient sheet.ClientInterface, gitClient git.ClientInterface, pageSize int) (*datastore.OpResults, error) {
	networkCfg := config.Get(ctx).GetCrosNetworkConfig()
	allVlans := make([]*ufspb.Vlan, 0)
	allIPs := make([]*ufspb.IP, 0)
	allDhcps := make([]*ufspb.DHCPConfig, 0)

	for _, cfg := range networkCfg.GetCrosNetworkTopology() {
		logging.Debugf(ctx, "########### Parse %s ###########", cfg.GetName())
		resp, err := sheetClient.Get(ctx, cfg.GetSheetId(), []string{"VLANs and Netblocks"})
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		topology, dupcatedVlan := util.ParseATLTopology(resp)
		logging.Debugf(ctx, "Topology length %d", len(topology))
		logging.Debugf(ctx, "Duplicated vlans found in topology:")
		logVlans(ctx, dupcatedVlan)
		conf, err := gitClient.GetFile(ctx, cfg.GetRemotePath())
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		parsed, err := util.ParseOSDhcpdConf(conf, topology)
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		logging.Debugf(ctx, "Duplicated vlans found in dhcp conf file:")
		logVlans(ctx, parsed.DuplicatedVlans)
		logging.Debugf(ctx, "Vlans not existing in pre-defined topology:")
		logVlans(ctx, parsed.MismatchedVlans)
		logging.Debugf(ctx, "Invalid dhcps without vlan:")
		logDHCPs(ctx, parsed.DHCPsWithoutVlan)
		logging.Debugf(ctx, "Duplicated ips found in dhcp conf file:")
		logIPs(ctx, parsed.DuplicatedIPs)

		logging.Debugf(ctx, "Get %d vlans, %d ips, %d dhcps for %s", len(parsed.ValidVlans), len(parsed.ValidIPs), len(parsed.ValidDHCPs), cfg.GetName())
		allVlans = append(allVlans, parsed.ValidVlans...)
		allIPs = append(allIPs, parsed.ValidIPs...)
		allDhcps = append(allDhcps, parsed.ValidDHCPs...)
	}

	allRes := make(datastore.OpResults, 0)
	logging.Debugf(ctx, "Importing %d vlans", len(allVlans))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(allVlans))
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

func logVlans(ctx context.Context, vlans []*ufspb.Vlan) {
	if vlans != nil && len(vlans) > 0 {
		for _, v := range vlans {
			logging.Debugf(ctx, "\tVlan %s (%s)", v.GetName(), v.GetVlanAddress())
		}
		return
	}
	logging.Debugf(ctx, "\tNot found")
}

func logDHCPs(ctx context.Context, dhcps []*ufspb.DHCPConfig) {
	if dhcps != nil && len(dhcps) > 0 {
		logging.Debugf(ctx, "enter")
		for _, v := range dhcps {
			logging.Debugf(ctx, "\tHost %s (%s)", v.GetHostname(), v.GetIp())
		}
		return
	}
	logging.Debugf(ctx, "\tNot found")

}

func logIPs(ctx context.Context, ips []*ufspb.IP) {
	if ips != nil && len(ips) > 0 {
		for _, v := range ips {
			logging.Debugf(ctx, "\tIP %s", v.GetId())
		}
		return
	}
	logging.Debugf(ctx, "\tNot found")
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
func ReplaceVlan(ctx context.Context, oldVlan *ufspb.Vlan, newVlan *ufspb.Vlan) (*ufspb.Vlan, error) {
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
