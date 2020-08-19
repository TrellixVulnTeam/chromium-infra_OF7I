// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	ufspb "infra/unifiedfleet/api/v1/proto"
	chromeosLab "infra/unifiedfleet/api/v1/proto/chromeos/lab"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/util"
)

// CreateMachineLSE creates a new machinelse in datastore.
func CreateMachineLSE(ctx context.Context, machinelse *ufspb.MachineLSE, machineNames []string, nwOpt *ufsAPI.NetworkOption) (*ufspb.MachineLSE, error) {
	// MachineLSE name and hostname must always be the same
	// Overwrite the name with hostname
	machinelse.Name = machinelse.GetHostname()

	// Overwrite the OUTPUT_ONLY fields
	// This is output only field. User is not allowed to set its value.
	// machine association to machinelse and machine indexing for machinelse table
	machinelse.Machines = machineNames

	// Labstation
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Hostname = machinelse.GetHostname()
		return createLabstation(ctx, machinelse, machineNames)
	}

	// DUT
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		// ChromeOSMachineLSE for a DUT
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Hostname = machinelse.GetHostname()
		return createDUT(ctx, machinelse, machineNames)
	}

	// Browser lab servers
	return createBrowserServer(ctx, machinelse, machineNames, nwOpt)
}

func createLabstation(ctx context.Context, lse *ufspb.MachineLSE, machineNames []string) (*ufspb.MachineLSE, error) {
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(lse)

		// Validate input
		err := validateCreateMachineLSE(ctx, lse, machineNames, nil)
		if err != nil {
			return errors.Annotate(err, "Validation error - Failed to create MachineLSE").Err()
		}

		// Get machine to get zone and rack info for machinelse table indexing
		machine, err := GetMachine(ctx, machineNames[0])
		if err != nil {
			return errors.Annotate(err, "unable to get machine %s", machineNames[0]).Err()
		}
		// Fill the rack/zone OUTPUT only fields for indexing machinelse table/vm table
		setOutputField(ctx, machine, lse)

		if err := hc.stUdt.addLseStateHelper(ctx, lse); err != nil {
			return errors.Annotate(err, "Fail to update host state").Err()
		}

		// Create the machinelse
		if _, err := inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{lse}); err != nil {
			return errors.Annotate(err, "Failed to BatchUpdate MachineLSEs %s", lse.Name).Err()
		}
		hc.LogMachineLSEChanges(nil, lse)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create machinelse in datastore: %s", err)
		return nil, err
	}
	return lse, nil
}

func createBrowserServer(ctx context.Context, lse *ufspb.MachineLSE, machineNames []string, nwOpt *ufsAPI.NetworkOption) (*ufspb.MachineLSE, error) {
	vms := lse.GetChromeBrowserMachineLse().GetVms()
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(lse)

		// Validate input
		err := validateCreateMachineLSE(ctx, lse, machineNames, nwOpt)
		if err != nil {
			return errors.Annotate(err, "Validation error - Failed to create MachineLSE").Err()
		}

		// Get machine to get zone and rack info for machinelse table indexing
		machine, err := GetMachine(ctx, machineNames[0])
		if err != nil {
			return errors.Annotate(err, "unable to get machine %s", machineNames[0]).Err()
		}
		// Fill the rack/zone OUTPUT only fields for indexing machinelse table/vm table
		setOutputField(ctx, machine, lse)

		// Assign ip configs
		if (nwOpt.GetVlan() != "" || nwOpt.GetIp() != "") && nwOpt.GetNic() != "" {
			if err := hc.netUdt.addLseHostHelper(ctx, nwOpt, lse); err != nil {
				return errors.Annotate(err, "Fail to assign ip to host %s", lse.GetName()).Err()
			}
		}

		lse.State = ufspb.State_STATE_DEPLOYED_PRE_SERVING.String()
		if err := hc.stUdt.addLseStateHelper(ctx, lse); err != nil {
			return errors.Annotate(err, "Fail to update host state").Err()
		}

		// Create the machinelse
		if vms != nil {
			if _, err := inventory.BatchUpdateVMs(ctx, vms); err != nil {
				return errors.Annotate(err, "Failed to BatchUpdate vms for host %s", lse.Name).Err()
			}
			for _, vm := range vms {
				hc.LogVMChanges(nil, vm)
			}
			lse.GetChromeBrowserMachineLse().Vms = nil
		}
		if _, err := inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{lse}); err != nil {
			return errors.Annotate(err, "Failed to BatchUpdate MachineLSEs %s", lse.Name).Err()
		}
		hc.LogMachineLSEChanges(nil, lse)
		if machine.GetChromeBrowserMachine() != nil {
			// We fill the machinelse object with newly created vms
			lse.GetChromeBrowserMachineLse().Vms = vms
		}
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create machinelse in datastore: %s", err)
		return nil, err
	}
	return lse, nil
}

// UpdateMachineLSE updates machinelse in datastore.
func UpdateMachineLSE(ctx context.Context, machinelse *ufspb.MachineLSE, machineNames []string, s ufspb.State, mask *field_mask.FieldMask) (*ufspb.MachineLSE, error) {
	// MachineLSEs name and hostname must always be the same
	// Overwrite the hostname with name as partial updates get only name
	machinelse.Hostname = machinelse.GetName()

	// If its a labstation, make the Hostname of the Labstation same as the machinelse name
	// Labstation hostname must be same as the machinelse hostname
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Hostname = machinelse.GetHostname()
	}

	// Overwrite the OUTPUT_ONLY fields
	// This is output only field. User is not allowed to set its value.
	if machineNames != nil && len(machineNames) > 0 {
		machinelse.Machines = machineNames
	}

	// If its a DUT
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		// ChromeOSMachineLSE for a DUT
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Hostname = machinelse.GetHostname()
		return updateDUT(ctx, machinelse, machineNames)
	}

	var oldMachinelse *ufspb.MachineLSE
	// If its a Chrome browser host, ChromeOS server or a ChormeOS labstation
	// ChromeBrowserMachineLSE, ChromeOSMachineLSE for a Server and Labstation
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(machinelse)

		// Validate the input
		err := validateUpdateMachineLSE(ctx, machinelse, machineNames, mask)
		if err != nil {
			return errors.Annotate(err, "Validation error - Failed to update MachineLSE").Err()
		}

		if machinelse.GetChromeBrowserMachineLse() != nil {
			// We dont update the vms in UpdateMachineLSE call.
			// We dont store vm object inside MachineLSE object in MachineLSE table.
			// vm objects are stored in separate VM table
			// user has to use VM CRUD apis to update vm
			machinelse.GetChromeBrowserMachineLse().Vms = nil
		}

		// Get the old machinelse
		// getting oldmachinelse for change history logging
		oldMachinelse, err = inventory.GetMachineLSE(ctx, machinelse.GetName())
		if err != nil {
			return errors.Annotate(err, "Failed to get old MachineLSE").Err()
		}
		oldMachinelseCopy := proto.Clone(oldMachinelse).(*ufspb.MachineLSE)
		// Copy the rack/state/zone/manufacturer/machines to machinelse OUTPUT only fields from already existing machinelse
		machinelse.Rack = oldMachinelse.GetRack()
		machinelse.Zone = oldMachinelse.GetZone()
		machinelse.Machines = oldMachinelse.GetMachines()
		machinelse.Manufacturer = oldMachinelse.GetManufacturer()
		machinelse.State = oldMachinelse.GetState()
		machinelse.Nic = oldMachinelse.GetNic()

		// Do not let updating from browser to os or vice versa change for MachineLSE.
		if oldMachinelse.GetChromeBrowserMachineLse() != nil && machinelse.GetChromeosMachineLse() != nil {
			return status.Error(codes.InvalidArgument, "UpdateMachineLSE - cannot update a browser host to chrome os host. Please delete the browser host and create a new os host")
		}
		if oldMachinelse.GetChromeosMachineLse() != nil && machinelse.GetChromeBrowserMachineLse() != nil {
			return status.Error(codes.InvalidArgument, "UpdateMachine - cannot update an os host to browser host. Please delete the os host and create a new browser host")
		}

		// check if user is trying to associate this host with a different browser machine.
		if len(machineNames) > 0 && machineNames[0] != "" && len(machinelse.GetMachines()) > 0 && machineNames[0] != machinelse.GetMachines()[0] {
			// Get machine to get zone and rack info for machinelse table indexing
			machine, err := GetMachine(ctx, machineNames[0])
			if err != nil {
				return errors.Annotate(err, "Unable to get machine %s", machineNames[0]).Err()
			}
			setOutputField(ctx, machine, machinelse)
			if err := updateIndexingForMachineLSEResources(ctx, oldMachinelse, map[string]string{"zone": machine.GetLocation().GetZone().String()}); err != nil {
				return errors.Annotate(err, "failed to update zone indexing").Err()
			}
		}

		// check if user provided a new state for the host
		if s != ufspb.State_STATE_UNSPECIFIED && machinelse.State != s.String() {
			machinelse.State = s.String()
			if err := hc.stUdt.updateStateHelper(ctx, s); err != nil {
				return errors.Annotate(err, "Fail to update state to host %s", machinelse.GetName()).Err()
			}
		}

		// Partial update by field mask
		if mask != nil && len(mask.Paths) > 0 {
			machinelse, err = processMachineLSEUpdateMask(ctx, oldMachinelse, machinelse, mask)
			if err != nil {
				return errors.Annotate(err, "UpdateMachineLSE - processing update mask failed").Err()
			}
		}

		// Update machinelse entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction. Datastore doesnt allow nested transactions.
		if _, err = inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{machinelse}); err != nil {
			return errors.Annotate(err, "Unable to batch update MachineLSE %s", machinelse.Name).Err()
		}
		hc.LogMachineLSEChanges(oldMachinelseCopy, machinelse)

		/* Comment this part for now
		// TODO(eshwarn): Add support for labstation state in the future, have a separate updatelabstation func.
		// Update states
		if osLSE := machinelse.GetChromeosMachineLse(); osLSE != nil {
			// Update labstation state to needs_deploy
			if osLSE.GetDeviceLse().GetLabstation() != nil {
				if err := hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_DEPLOYED_PRE_SERVING); err != nil {
					return err
				}
			}
		}*/
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to update entity in datastore: %s", err)
		return nil, err
	}
	if oldMachinelse.GetChromeBrowserMachineLse() != nil {
		// We fill the machinelse object with its vm objects from vm table
		setMachineLSE(ctx, machinelse)
	}
	return machinelse, nil
}

// updateIndexingForMachineLSEResources updates indexing for vm table
// can be used inside a transaction
func updateIndexingForMachineLSEResources(ctx context.Context, oldlse *ufspb.MachineLSE, indexMap map[string]string) error {
	if oldlse.GetChromeBrowserMachineLse() != nil {
		vms, err := inventory.QueryVMByPropertyName(ctx, "host_id", oldlse.GetName(), false)
		if err != nil {
			return errors.Annotate(err, "failed to query vms for host %s", oldlse.GetName()).Err()
		}
		for k, v := range indexMap {
			// These are output only fields used for indexing vm table
			switch k {
			case "zone":
				for _, vm := range vms {
					vm.Zone = v
				}
			}
		}
		if _, err := inventory.BatchUpdateVMs(ctx, vms); err != nil {
			return errors.Annotate(err, "updateIndexing - unable to batch update vms").Err()
		}
	}
	return nil
}

// processMachineLSEUpdateMask process update field mask to get only specific update
// fields and return a complete machinelse object with updated and existing fields
func processMachineLSEUpdateMask(ctx context.Context, oldMachinelse *ufspb.MachineLSE, machinelse *ufspb.MachineLSE, mask *field_mask.FieldMask) (*ufspb.MachineLSE, error) {
	// update the fields in the existing machinelse
	for _, path := range mask.Paths {
		switch path {
		case "machine":
			// In the previous step we have already checked for machine != ""
			// and got the new values for OUTPUT only fields in new machinelse object,
			// assign them to oldMachinelse.
			oldMachinelse.Machines = machinelse.GetMachines()
			oldMachinelse.Zone = machinelse.GetZone()
			oldMachinelse.Rack = machinelse.GetRack()
			if err := updateIndexingForMachineLSEResources(ctx, oldMachinelse, map[string]string{"zone": machinelse.GetZone()}); err != nil {
				return oldMachinelse, errors.Annotate(err, "failed to update zone indexing").Err()
			}
		case "mlseprototype":
			oldMachinelse.MachineLsePrototype = machinelse.GetMachineLsePrototype()
		case "osVersion":
			if oldMachinelse.GetChromeBrowserMachineLse() == nil {
				oldMachinelse.Lse = &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				}
			}
			if oldMachinelse.GetChromeBrowserMachineLse().GetOsVersion() == nil {
				oldMachinelse.GetChromeBrowserMachineLse().OsVersion = &ufspb.OSVersion{
					Value: machinelse.GetChromeBrowserMachineLse().GetOsVersion().GetValue(),
				}
			} else {
				oldMachinelse.GetChromeBrowserMachineLse().GetOsVersion().Value = machinelse.GetChromeBrowserMachineLse().GetOsVersion().GetValue()
			}
		case "vmCapacity":
			if oldMachinelse.GetChromeBrowserMachineLse() == nil {
				oldMachinelse.Lse = &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				}
			}
			oldMachinelse.GetChromeBrowserMachineLse().VmCapacity = machinelse.GetChromeBrowserMachineLse().GetVmCapacity()
		case "state":
			// In the previous step we have already checked for state != ufspb.State_STATE_UNSPECIFIED
			// and got the new values for OUTPUT only fields in new machinelse object,
			// assign them to oldMachinelse.
			oldMachinelse.State = machinelse.GetState()
		case "tags":
			oldMachinelse.Tags = mergeTags(oldMachinelse.GetTags(), machinelse.GetTags())
		}
	}
	// return existing/old machinelse with new updated values
	return oldMachinelse, nil
}

// GetMachineLSE returns machinelse for the given id from datastore.
func GetMachineLSE(ctx context.Context, id string) (*ufspb.MachineLSE, error) {
	lse, err := inventory.GetMachineLSE(ctx, id)
	if err != nil {
		return nil, err
	}
	setMachineLSE(ctx, lse)
	return lse, nil
}

// ListMachineLSEs lists the machinelses
func ListMachineLSEs(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.MachineLSE, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, inventory.GetMachineLSEIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing hosts").Err()
		}
	}
	filterMap = resetStateFilter(filterMap)
	filterMap = resetOSFilter(filterMap)
	filterMap = resetZoneFilter(filterMap)
	if _, ok := filterMap[util.FreeVMFilterName]; ok {
		delete(filterMap, util.FreeVMFilterName)
		allVMs, err := inventory.GetAllVMs(ctx)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to get all vms").Err()
		}
		capacityMap := make(map[string]int, 0)
		for _, r := range allVMs.Passed() {
			vm := r.Data.(*ufspb.VM)
			if vm.GetMachineLseId() != "" {
				capacityMap[vm.GetMachineLseId()]++
			}
		}
		lses, _, err := inventory.ListFreeMachineLSEs(ctx, pageSize, filterMap, capacityMap)
		if err != nil {
			return nil, "", err
		}
		res := make([]*ufspb.MachineLSE, 0)
		var total int32
		for _, lse := range lses {
			res = append(res, lse)
			freeSlots := lse.GetChromeBrowserMachineLse().GetVmCapacity() - int32(capacityMap[lse.GetName()])
			logging.Infof(ctx, "Found %d free slots on host %s", freeSlots, lse.GetName())
			lse.GetChromeBrowserMachineLse().VmCapacity = freeSlots
			total += freeSlots
			logging.Infof(ctx, "Already get %d (require %d)", total, pageSize)
			if total >= pageSize {
				break
			}
		}
		return res, "", nil
	}
	return inventory.ListMachineLSEs(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteMachineLSE deletes the machinelse in datastore
//
// For referential data intergrity,
// Delete if this MachineLSE is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteMachineLSE(ctx context.Context, id string) error {
	err := validateDeleteMachineLSE(ctx, id)
	if err != nil {
		return err
	}
	existingMachinelse := &ufspb.MachineLSE{}
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(&ufspb.MachineLSE{
			Name: id,
		})
		existingMachinelse, err = inventory.GetMachineLSE(ctx, id)
		if err != nil {
			return err
		}
		// Check if it is a DUT MachineLSE and has servo info.
		// Update corresponding Labstation MachineLSE.
		if existingMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
			existingServo := existingMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
			if existingServo != nil {
				// 1. remove the existingServo entry of DUT form existingLabstationMachinelse
				existingLabstationMachinelse, err := inventory.GetMachineLSE(ctx, existingServo.GetServoHostname())
				if err != nil {
					return err
				}
				removeServoEntryFromLabstation(existingServo, existingLabstationMachinelse)
				// BatchUpdate Labstation - Using Batch update and not UpdateMachineLSE,
				// because we cant have nested transaction in datastore
				_, err = inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{existingLabstationMachinelse})
				if err != nil {
					logging.Errorf(ctx, "Failed to BatchUpdate Labstation MachineLSE %s", err)
					return err
				}
			}
		}

		vms, err := inventory.QueryVMByPropertyName(ctx, "host_id", id, false)
		if err != nil {
			return err
		}
		setVMsToLSE(existingMachinelse, vms)

		// Delete states
		if err := hc.stUdt.deleteLseStateHelper(ctx, existingMachinelse); err != nil {
			return errors.Annotate(err, "Fail to delete lse-related states").Err()
		}

		// Delete dhcps
		if err := hc.netUdt.deleteLseHostHelper(ctx, existingMachinelse); err != nil {
			return errors.Annotate(err, "Fail to delete lse-related dhcps").Err()
		}

		// Delete vms
		vmIDs := make([]string, 0, len(vms))
		for _, vm := range vms {
			vmIDs = append(vmIDs, vm.GetName())
			hc.LogVMChanges(&ufspb.VM{Name: vm.GetName()}, nil)
		}
		if vmIDs != nil && len(vmIDs) > 0 {
			if err := inventory.BatchDeleteVMs(ctx, vmIDs); err != nil {
				return err
			}
		}

		if err := inventory.DeleteMachineLSE(ctx, id); err != nil {
			return err
		}
		hc.LogMachineLSEChanges(existingMachinelse, nil)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete MachineLSE in datastore: %s", err)
		return err
	}
	return nil
}

// ImportOSMachineLSEs implements the logic of importing ChromeOS machine lses and related info to backend storage.
//
// The function will return:
//      * all of the results of the operations that already run
//      * the first error that it meets
//
// The function will stop at the very first error.
func ImportOSMachineLSEs(ctx context.Context, labConfigs []*invV2Api.ListCrosDevicesLabConfigResponse_LabConfig, pageSize int) (*ufsds.OpResults, error) {
	allRes := make(ufsds.OpResults, 0)
	logging.Infof(ctx, "Importing the machine lse prototypes for OS lab")
	res, err := configuration.ImportMachineLSEPrototypes(ctx, util.GetOSMachineLSEPrototypes())
	if err != nil {
		return res, err
	}
	allRes = append(allRes, *res...)

	lses := util.ToOSMachineLSEs(labConfigs)
	deleteNonExistingMachineLSEs(ctx, lses, pageSize, "os-lab")
	logging.Infof(ctx, "Importing %d lses", len(lses))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(lses))
		res, err := inventory.ImportMachineLSEs(ctx, lses[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(lses) {
			break
		}
	}
	return &allRes, nil
}

// ImportMachineLSEs implements the logic of importing machine lses and related info to backend storage.
//
// The function will return:
//      * all of the results of the operations that already run
//      * the first error that it meets
//
// The function will stop at the very first error.
func ImportMachineLSEs(ctx context.Context, hosts []*crimson.PhysicalHost, vms []*crimson.VM, machines []*crimson.Machine, platforms []*crimson.Platform, pageSize int) (*ufsds.OpResults, error) {
	allRes := make(ufsds.OpResults, 0)
	logging.Infof(ctx, "Importing the basic lse prototypes for browser lab")
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
			Tags: []string{"browser", "no-vm"},
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
			Tags: []string{"browser", "vm"},
		},
	}
	res, err := configuration.ImportMachineLSEPrototypes(ctx, lps)
	if err != nil {
		return res, err
	}
	allRes = append(allRes, *res...)

	lses, ufsVMs, ips, dhcps := util.ToMachineLSEs(hosts, vms, machines, platforms)
	deleteNonExistingMachineLSEs(ctx, lses, pageSize, "browser-lab")
	logging.Infof(ctx, "Importing %d lses", len(lses))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(lses))
		res, err := inventory.ImportMachineLSEs(ctx, lses[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(lses) {
			break
		}
	}

	deleteNonExistingVMs(ctx, ufsVMs, pageSize)
	logging.Infof(ctx, "Importing %d vms", len(ufsVMs))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(ufsVMs))
		res, err := inventory.ImportVMs(ctx, ufsVMs[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(ufsVMs) {
			break
		}
	}

	logging.Infof(ctx, "Importing %d ips", len(ips))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(ips))
		res, err := configuration.ImportIPs(ctx, ips[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(ips) {
			break
		}
	}

	logging.Infof(ctx, "Importing %d dhcps", len(dhcps))
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

func deleteNonExistingMachineLSEs(ctx context.Context, machineLSEs []*ufspb.MachineLSE, pageSize int, lseType string) (*ufsds.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range machineLSEs {
		resMap[r.GetName()] = true
	}
	resp, err := inventory.GetAllMachineLSEs(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	var toDeleteDHCPHost []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.MachineLSE)
		if lseType == "browser-lab" && s.GetChromeosMachineLse() != nil {
			continue
		}
		if lseType == "os-lab" && s.GetChromeBrowserMachineLse() != nil {
			continue
		}
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
			toDeleteDHCPHost = append(toDeleteDHCPHost, s.GetName())
		}
		if s.GetChromeBrowserMachineLse() != nil {
			for _, vm := range s.GetChromeBrowserMachineLse().GetVms() {
				toDeleteDHCPHost = append(toDeleteDHCPHost, vm.GetHostname())
			}
		}
	}
	logging.Infof(ctx, "Deleting %d non-existing machine lses", len(toDelete))
	allRes := *deleteByPage(ctx, toDelete, pageSize, inventory.DeleteMachineLSEs)
	logging.Infof(ctx, "Deleting %d non-existing host and vm-related dhcps", len(toDelete))
	allRes = append(allRes, *deleteByPage(ctx, toDelete, pageSize, configuration.DeleteDHCPs)...)
	return &allRes, nil
}

func deleteNonExistingVMs(ctx context.Context, vms []*ufspb.VM, pageSize int) (*ufsds.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range vms {
		resMap[r.GetName()] = true
	}
	resp, err := inventory.GetAllVMs(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	var toDeleteDHCPHost []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.VM)
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
			toDeleteDHCPHost = append(toDeleteDHCPHost, s.GetName())
		}
	}
	logging.Infof(ctx, "Deleting %d non-existing vms", len(toDelete))
	allRes := *deleteByPage(ctx, toDelete, pageSize, inventory.DeleteVMs)
	logging.Infof(ctx, "Deleting %d vm-related dhcps", len(toDelete))
	allRes = append(allRes, *deleteByPage(ctx, toDelete, pageSize, configuration.DeleteDHCPs)...)
	return &allRes, nil
}

// validateServoInfoForDUT Checks if the DUT Machinelse has ServoHostname and ServoPort
// already used by a different deployed DUT
func validateServoInfoForDUT(ctx context.Context, servo *chromeosLab.Servo, DUTHostname string) (string, error) {
	// Validating the DUT's servo is already occupied
	servoID := ufsds.GetServoID(servo.GetServoHostname(), servo.GetServoPort())
	dutMachinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "servo_id", servoID, true)
	if err != nil {
		return "", err
	}
	if dutMachinelses != nil && dutMachinelses[0].GetName() != DUTHostname {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Port: %d in Labstation: %s is already "+
			"in use by DUT: %s. Please provide a different ServoPort.\n",
			servo.GetServoPort(), servo.GetServoHostname(), dutMachinelses[0].GetName()))
		logging.Errorf(ctx, errorMsg.String())
		return dutMachinelses[0].GetName(), status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	if dutMachinelses != nil {
		return dutMachinelses[0].GetName(), nil
	}
	return "", nil
}

// getLabstationMachineLSE get the Labstation MachineLSE
func getLabstationMachineLSE(ctx context.Context, labstationMachinelseName string) (*ufspb.MachineLSE, error) {
	labstationMachinelse, err := inventory.GetMachineLSE(ctx, labstationMachinelseName)
	if status.Code(err) == codes.Internal {
		return nil, err
	}
	if labstationMachinelse == nil {
		// There is no Labstation MachineLSE existing in the system
		errorMsg := fmt.Sprintf("Labstation %s not found in the system. "+
			"Please deploy the Labstation %s before deploying the DUT.",
			labstationMachinelseName, labstationMachinelseName)
		logging.Errorf(ctx, errorMsg)
		return nil, status.Errorf(codes.FailedPrecondition, errorMsg)
	}
	return labstationMachinelse, nil
}

// appendServoEntryToLabstation append servo entry to the Labstation
func appendServoEntryToLabstation(newServo *chromeosLab.Servo, labstationMachinelse *ufspb.MachineLSE) {
	existingServos := labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	existingServos = append(existingServos, newServo)
	labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = existingServos
}

// replaceServoEntryInLabstation replaces oldServo entry with newServo entry in the Labstation
func replaceServoEntryInLabstation(oldServo, newServo *chromeosLab.Servo, labstationMachinelse *ufspb.MachineLSE) {
	servos := labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	for i, s := range servos {
		if s.GetServoHostname() == oldServo.GetServoHostname() && s.GetServoPort() == oldServo.GetServoPort() {
			servos[i] = newServo
			break
		}
	}
	labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = servos
}

// removeServoEntryFromLabstation removes servo entry from the Labstation
func removeServoEntryFromLabstation(servo *chromeosLab.Servo, labstationMachinelse *ufspb.MachineLSE) {
	servos := labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	for i, s := range servos {
		if s.GetServoHostname() == servo.GetServoHostname() && s.GetServoPort() == servo.GetServoPort() {
			servos[i] = servos[len(servos)-1]
			servos = servos[:len(servos)-1]
			break
		}
	}
	labstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = servos
}

// validateCreateMachineLSE validates if a machinelse can be created in the datastore.
func validateCreateMachineLSE(ctx context.Context, machinelse *ufspb.MachineLSE, machineNames []string, nwOpt *ufsAPI.NetworkOption) error {
	//1. Check for servos for Labstation deployment
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		newServos := machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
		if len(newServos) != 0 {
			return status.Errorf(codes.FailedPrecondition, "Servos are not allowed "+
				"to be added in deploying labstations")
		}
	}

	// 2. Check if machinelse already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetMachineLSEResource(machinelse.Name)}, nil); err != nil {
		return err
	}

	// 3. Check if resources does not exist
	var resourcesNotfound []*Resource
	// Aggregate resource to check if machines does not exist
	for _, machineName := range machineNames {
		resourcesNotfound = append(resourcesNotfound, GetMachineResource(machineName))
	}
	if (nwOpt.GetVlan() != "" || nwOpt.GetIp() != "") && nwOpt.GetNic() != "" {
		if nwOpt.GetVlan() != "" {
			resourcesNotfound = append(resourcesNotfound, GetVlanResource(nwOpt.GetVlan()))
		}
		resourcesNotfound = append(resourcesNotfound, GetNicResource(nwOpt.GetNic()))
	}
	// Aggregate resources referenced by the machinelse to check if they do not exist
	if machineLSEPrototypeID := machinelse.GetMachineLsePrototype(); machineLSEPrototypeID != "" {
		resourcesNotfound = append(resourcesNotfound, GetMachineLSEProtoTypeResource(machineLSEPrototypeID))
	}
	if rpmID := machinelse.GetChromeosMachineLse().GetDeviceLse().GetRpmInterface().GetRpm(); rpmID != "" {
		resourcesNotfound = append(resourcesNotfound, GetRPMResource(rpmID))
	}
	if err := ResourceExist(ctx, resourcesNotfound, nil); err != nil {
		return err
	}

	// 4. Check if any machine is already associated with another MachineLSE
	// A machine cannot be associated with multiple hosts/machinelses
	for _, machineName := range machineNames {
		machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", machineName, true)
		if err != nil {
			return errors.Annotate(err, "Failed to query machinelses for machine %s", machineName).Err()
		}
		if len(machinelses) > 1 {
			logging.Warningf(ctx, "More than one machinelse associated with the "+
				"machine %s. Data discrepancy error.\n", machineName)
		}
		if len(machinelses) > 0 {
			var errorMsg strings.Builder
			errorMsg.WriteString(fmt.Sprintf("Host %s cannot be created because "+
				"there are other hosts which are referring this machine %s. "+
				"A machine cannot be associated with multiple hosts.",
				machinelse.Name, machineName))
			errorMsg.WriteString(fmt.Sprintf("\nHosts referring the machine %s:\n", machineName))
			for _, mlse := range machinelses {
				errorMsg.WriteString(mlse.Name + ", ")
			}
			errorMsg.WriteString(fmt.Sprintf("\nPlease delete the hosts and then "+
				"add this host %s.\n", machinelse.Name))
			logging.Errorf(ctx, errorMsg.String())
			return status.Errorf(codes.FailedPrecondition, errorMsg.String())
		}
	}
	return nil
}

// UpdateMachineLSEHost updates the machinelse host(update ip assignment).
func UpdateMachineLSEHost(ctx context.Context, machinelseName string, nwOpt *ufsAPI.NetworkOption) (*ufspb.MachineLSE, error) {
	var oldMachinelse *ufspb.MachineLSE
	var machinelse *ufspb.MachineLSE
	var err error
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(&ufspb.MachineLSE{Name: machinelseName})
		// Validate the input
		if err := validateUpdateMachineLSEHost(ctx, machinelseName, nwOpt); err != nil {
			return err
		}

		// Since we update the nic, we have to get machinelse within the transaction
		machinelse, err = GetMachineLSE(ctx, machinelseName)
		if err != nil {
			return err
		}
		// this is for logging changes
		oldMachinelse = proto.Clone(machinelse).(*ufspb.MachineLSE)

		// Find free ip, set IP and DHCP config
		if err := hc.netUdt.addLseHostHelper(ctx, nwOpt, machinelse); err != nil {
			return errors.Annotate(err, "Fail to assign ip to host %s", machinelse.Name).Err()
		}

		// Update machinelse with new nic info which set/updated in prev func addLseHostHelper
		if _, err = inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{machinelse}); err != nil {
			return errors.Annotate(err, "Unable to batch update MachineLSE %s", machinelse.Name).Err()
		}

		hc.LogMachineLSEChanges(oldMachinelse, machinelse)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to assign IP to the MachineLSE: %s", err)
		return nil, err
	}
	if oldMachinelse.GetChromeBrowserMachineLse() != nil {
		// We fill the machinelse object with its vm objects from vm table
		setMachineLSE(ctx, machinelse)
	}
	return machinelse, nil
}

// validateUpdateMachineLSEHost validates if an ip can be assigned to the MachineLSE
func validateUpdateMachineLSEHost(ctx context.Context, machinelseName string, nwOpt *ufsAPI.NetworkOption) error {
	// Aggregate resource to check if machinelse does not exist
	resourcesNotFound := []*Resource{GetMachineLSEResource(machinelseName)}
	if nwOpt.GetVlan() != "" {
		resourcesNotFound = append(resourcesNotFound, GetVlanResource(nwOpt.GetVlan()))
	}
	if nwOpt.GetNic() != "" {
		resourcesNotFound = append(resourcesNotFound, GetNicResource(nwOpt.GetNic()))
	}
	// Check if resources does not exist
	return ResourceExist(ctx, resourcesNotFound, nil)
}

// DeleteMachineLSEHost deletes the dhcp/ip of a machinelse in datastore.
func DeleteMachineLSEHost(ctx context.Context, machinelseName string) error {
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(&ufspb.MachineLSE{Name: machinelseName})
		if err := hc.netUdt.deleteDHCPHelper(ctx); err != nil {
			return err
		}
		lse, err := inventory.GetMachineLSE(ctx, machinelseName)
		if err != nil {
			return err
		}
		lseCopy := proto.Clone(lse).(*ufspb.MachineLSE)
		lse.Nic = ""
		lse.State = ufspb.State_STATE_DEPLOYED_PRE_SERVING.String()
		if _, err := inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{lse}); err != nil {
			return errors.Annotate(err, "Failed to update host %q", machinelseName).Err()
		}
		hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
		hc.LogMachineLSEChanges(lseCopy, lse)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete the machinelse dhcp/ip: %s", err)
		return err
	}
	return nil
}

// validateUpdateMachineLSE validates if a machinelse can be updated in the datastore.
func validateUpdateMachineLSE(ctx context.Context, machinelse *ufspb.MachineLSE, machineNames []string, mask *field_mask.FieldMask) error {
	// 1. This check is only for a Labstation
	// Check if labstation MachineLSE is updating any servo information
	// It is also not allowed to update the servo Hostname and servo Port of any servo.
	// Servo info is added/updated into Labstation only when a DUT(MachineLSE) is added/updated
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		newServos := machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
		existingLabstationMachinelse, err := inventory.GetMachineLSE(ctx, machinelse.GetName())
		if status.Code(err) == codes.Internal {
			return err
		}
		if existingLabstationMachinelse != nil {
			existingServos := existingLabstationMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
			if !testServoEq(newServos, existingServos) {
				return status.Errorf(codes.FailedPrecondition, "Servos are not allowed to be updated in redeploying labstations")
			}
		}
	}

	// 2. Check if resources does not exist
	// Aggregate resource to check if machinelse does not exist
	resourcesNotfound := []*Resource{GetMachineLSEResource(machinelse.Name)}
	// Aggregate resource to check if machines does not exist
	for _, machineName := range machineNames {
		if machineName != "" {
			resourcesNotfound = append(resourcesNotfound, GetMachineResource(machineName))
		}
	}

	// Aggregate resources referenced by the machinelse to check if they do not exist
	if machineLSEPrototypeID := machinelse.GetMachineLsePrototype(); machineLSEPrototypeID != "" {
		resourcesNotfound = append(resourcesNotfound, GetMachineLSEProtoTypeResource(machineLSEPrototypeID))
	}
	if vlanID := machinelse.GetChromeosMachineLse().GetServerLse().GetSupportedRestrictedVlan(); vlanID != "" {
		resourcesNotfound = append(resourcesNotfound, GetVlanResource(vlanID))
	}
	if rpmID := machinelse.GetChromeosMachineLse().GetDeviceLse().GetRpmInterface().GetRpm(); rpmID != "" {
		resourcesNotfound = append(resourcesNotfound, GetRPMResource(rpmID))
	}
	if err := ResourceExist(ctx, resourcesNotfound, nil); err != nil {
		//return err
		return errors.Annotate(err, "HELLO %s", machinelse.Name).Err()
	}

	// 3. Check if any machine is already associated with another MachineLSE
	// A machine cannot be associated with multiple hosts/machinelses
	for _, machineName := range machineNames {
		machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", machineName, true)
		if err != nil {
			return errors.Annotate(err, "Failed to query machinelses for machine %s", machineName).Err()
		}
		if len(machinelses) == 1 && machinelses[0].GetName() != machinelse.Name {
			errorMsg := fmt.Sprintf("Host %s cannot be updated because "+
				"there is another host %s which is referring this machine %s. "+
				"A machine cannot be associated with multiple hosts. "+
				"Please delete the other host and then update this host.\n",
				machinelse.Name, machinelses[0].Name, machineName)
			return status.Errorf(codes.FailedPrecondition, errorMsg)
		}
		if len(machinelses) > 1 {
			var errorMsg strings.Builder
			errorMsg.WriteString(fmt.Sprintf("More than one host associated "+
				"with the machine %s. Data discrepancy error. Host %s cannot be "+
				"updated because there are other hosts which are referring this "+
				"machine %s. A machine cannot be associated with multiple hosts. ",
				machineName, machinelse.Name, machineName))
			errorMsg.WriteString(fmt.Sprintf("Hosts referring the machine %s:\n", machineName))
			for _, mlse := range machinelses {
				errorMsg.WriteString(mlse.Name + ", ")
			}
			errorMsg.WriteString(fmt.Sprintf("\nPlease delete the hosts and then "+
				"add this host %s.\n", machinelse.Name))
			logging.Errorf(ctx, errorMsg.String())
			return status.Errorf(codes.FailedPrecondition, errorMsg.String())
		}
	}

	// validate update mask
	return validateMachineLSEUpdateMask(machinelse, mask)
}

// validateMachineLSEUpdateMask validates the update mask for machinelse update
func validateMachineLSEUpdateMask(machinelse *ufspb.MachineLSE, mask *field_mask.FieldMask) error {
	if mask != nil {
		// validate the give field mask
		for _, path := range mask.Paths {
			switch path {
			case "name":
				return status.Error(codes.InvalidArgument, "validateMachineLSEUpdateMask - name cannot be updated, delete and create a new machinelse instead")
			case "update_time":
				return status.Error(codes.InvalidArgument, "validateMachineLSEUpdateMask - update_time cannot be updated, it is a Output only field")
			case "machine":
			case "mlseprototype":
			case "osVersion":
				if machinelse.GetChromeBrowserMachineLse() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineLSEUpdateMask - browser machine lse cannot be empty/nil.")
				}
				if machinelse.GetChromeBrowserMachineLse().GetOsVersion() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineLSEUpdateMask - Osverison cannot be empty/nil.")
				}
			case "vmCapacity":
				if machinelse.GetChromeBrowserMachineLse() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineLSEUpdateMask - browser machine lse cannot be empty/nil.")
				}
			case "tags":
			case "state":
				// valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "validateMachineLSEUpdateMask - unsupported update mask path %q", path)
			}
		}
	}
	return nil
}

// validateDeleteMachineLSE validates if a MachineLSE can be deleted
func validateDeleteMachineLSE(ctx context.Context, id string) error {
	existingMachinelse, err := inventory.GetMachineLSE(ctx, id)
	if err != nil {
		return err
	}
	if existingMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		existingServos := existingMachinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
		nonDeletedHosts := make([]string, 0, len(existingServos))
		for _, servo := range existingServos {
			dutHostName, err := validateServoInfoForDUT(ctx, servo, "")
			if err != nil {
				nonDeletedHosts = append(nonDeletedHosts, dutHostName)
			}
		}
		if len(nonDeletedHosts) != 0 {
			errorMsg := fmt.Sprintf("Labstation %s cannot be deleted because "+
				"there are servos in the labstation referenced by other DUTs: %s.",
				id, strings.Join(nonDeletedHosts, ", "))
			logging.Errorf(ctx, errorMsg)
			return status.Errorf(codes.FailedPrecondition, errorMsg)
		}
	}
	return nil
}

func setOutputField(ctx context.Context, machine *ufspb.Machine, lse *ufspb.MachineLSE) error {
	lse.Rack = machine.GetLocation().GetRack()
	lse.Zone = machine.GetLocation().GetZone().String()
	lse.Machines = []string{machine.GetName()}
	for _, vm := range lse.GetChromeBrowserMachineLse().GetVms() {
		vm.Zone = machine.GetLocation().GetZone().String()
		vm.MachineLseId = lse.GetName()
		vm.State = ufspb.State_STATE_DEPLOYED_PRE_SERVING.String()
	}
	if pName := machine.GetChromeBrowserMachine().GetChromePlatform(); pName != "" {
		platform, err := configuration.GetChromePlatform(ctx, pName)
		if err != nil {
			return errors.Annotate(err, "invalid chrome platform name attached to machine %s", machine.GetName()).Err()
		}
		lse.Manufacturer = strings.ToLower(platform.GetManufacturer())
	}
	return nil
}

func setMachineLSE(ctx context.Context, machinelse *ufspb.MachineLSE) {
	vms, err := inventory.QueryVMByPropertyName(ctx, "host_id", machinelse.GetName(), false)
	if err != nil {
		// Just log a warning message and dont fail operation
		logging.Warningf(ctx, "setMachineLSE - failed to query vms for host %s: %s", machinelse.GetName(), err)
	}
	setVMsToLSE(machinelse, vms)
}

func setVMsToLSE(lse *ufspb.MachineLSE, vms []*ufspb.VM) {
	if len(vms) <= 0 {
		return
	}
	if lse.GetChromeBrowserMachineLse() == nil {
		lse.Lse = &ufspb.MachineLSE_ChromeBrowserMachineLse{
			ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{
				Vms: vms,
			},
		}
	} else {
		lse.GetChromeBrowserMachineLse().Vms = vms
	}
}

func getHostHistoryClient(m *ufspb.MachineLSE) *HistoryClient {
	return &HistoryClient{
		stUdt: &stateUpdater{
			ResourceName: util.AddPrefix(util.HostCollection, m.Name),
		},
		netUdt: &networkUpdater{
			Hostname: m.Name,
		},
	}
}
