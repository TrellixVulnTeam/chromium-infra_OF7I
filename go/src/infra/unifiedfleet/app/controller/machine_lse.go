// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"go.chromium.org/luci/grpc/grpcutil"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	invV2Api "infra/appengine/cros/lab_inventory/api/v1"
	ufspb "infra/unifiedfleet/api/v1/models"
	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/configuration"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

// CreateMachineLSE creates a new machinelse in datastore.
func CreateMachineLSE(ctx context.Context, machinelse *ufspb.MachineLSE, nwOpt *ufsAPI.NetworkOption) (*ufspb.MachineLSE, error) {
	// MachineLSE name and hostname must always be the same
	// Overwrite the name with hostname
	machinelse.Name = machinelse.GetHostname()

	// Labstation
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Hostname = machinelse.GetHostname()
		return CreateLabstation(ctx, machinelse)
	}

	// DUT
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		// ChromeOSMachineLSE for a DUT
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Hostname = machinelse.GetHostname()
		return CreateDUT(ctx, machinelse)
	}

	// Browser lab servers
	return createBrowserServer(ctx, machinelse, nwOpt)
}

func createBrowserServer(ctx context.Context, lse *ufspb.MachineLSE, nwOpt *ufsAPI.NetworkOption) (*ufspb.MachineLSE, error) {
	vms := lse.GetChromeBrowserMachineLse().GetVms()
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(lse)

		// Get machine to get zone and rack info for machinelse table indexing
		machine, err := GetMachine(ctx, lse.GetMachines()[0])
		if err != nil {
			return errors.Annotate(err, "unable to get machine %s", lse.GetMachines()[0]).Err()
		}

		// Validate input
		if err := validateCreateMachineLSE(ctx, lse, nwOpt, machine); err != nil {
			return errors.Annotate(err, "Validation error - Failed to create MachineLSE").Err()
		}

		// Copy for logging
		oldMachine := proto.Clone(machine).(*ufspb.Machine)

		machine.ResourceState = ufspb.State_STATE_SERVING
		// Fill the rack/zone OUTPUT only fields for indexing machinelse table/vm table
		setOutputField(ctx, machine, lse)

		// Assign ip configs
		if err := setNicIfNeeded(ctx, lse, machine, nwOpt); err != nil {
			return err
		}
		lse.Nic = ""
		lse.Vlan = ""
		if (nwOpt.GetVlan() != "" || nwOpt.GetIp() != "") && nwOpt.GetNic() != "" {
			if err := hc.netUdt.addLseHostHelper(ctx, nwOpt, lse); err != nil {
				return errors.Annotate(err, "Fail to assign ip to host %s", lse.GetName()).Err()
			}
			lse.ResourceState = ufspb.State_STATE_DEPLOYING
		} else {
			lse.ResourceState = ufspb.State_STATE_REGISTERED
		}

		// Create the vms, update machine, update machine lses
		if vms != nil {
			for _, vm := range vms {
				hc.LogVMChanges(nil, vm)
			}
			if _, err := inventory.BatchUpdateVMs(ctx, vms); err != nil {
				return errors.Annotate(err, "Failed to BatchUpdate vms for host %s", lse.Name).Err()
			}
			// We do not save vm objects in machinelse table
			lse.GetChromeBrowserMachineLse().Vms = nil
		}
		if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
			return errors.Annotate(err, "Fail to update machine %s", machine.GetName()).Err()
		}
		hc.LogMachineChanges(oldMachine, machine)
		if _, err := inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{lse}); err != nil {
			return errors.Annotate(err, "Failed to BatchUpdate MachineLSEs %s", lse.Name).Err()
		}
		hc.LogMachineLSEChanges(nil, lse)
		if lse.GetChromeBrowserMachineLse() != nil {
			// We fill the machinelse object with newly created vms
			lse.GetChromeBrowserMachineLse().Vms = vms
		}
		// Add/Update machine lse deployment
		if machine.GetSerialNumber() != "" {
			lseDr, err := inventory.GetMachineLSEDeployment(ctx, machine.GetSerialNumber())
			lseDrCopy := proto.Clone(lseDr).(*ufspb.MachineLSEDeployment)
			if util.IsNotFoundError(err) {
				lseDr = util.FormatDeploymentRecord(lse.GetName(), machine.GetSerialNumber())
			} else {
				if err != nil {
					return errors.Annotate(err, "fails to get deployment record for %s", machine.GetSerialNumber()).Err()
				}
			}
			if lseDr.GetHostname() != lse.GetName() {
				lseDr.Hostname = lse.GetName()
			}
			if _, err := inventory.UpdateMachineLSEDeployments(ctx, []*ufspb.MachineLSEDeployment{lseDr}); err != nil {
				return errors.Annotate(err, "unable to update deployment record").Err()
			}
			hc.LogMachineLSEDeploymentChanges(lseDrCopy, lseDr)
		}

		if err := hc.stUdt.addLseStateHelper(ctx, lse, machine); err != nil {
			return errors.Annotate(err, "Fail to update host state").Err()
		}
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "createBrowserServers: %s", err)
		return nil, err
	}
	return lse, nil
}

// UpdateMachineLSE updates machinelse in datastore.
func UpdateMachineLSE(ctx context.Context, machinelse *ufspb.MachineLSE, mask *field_mask.FieldMask) (*ufspb.MachineLSE, error) {
	// MachineLSEs name and hostname must always be the same
	// Overwrite the hostname with name as partial updates get only name
	machinelse.Hostname = machinelse.GetName()
	var err error

	// Validate Pool Names
	err = validateUpdateMachineLSEPoolNames(ctx, machinelse)
	if err != nil {
		return nil, err
	}

	// If its a labstation, make the Hostname of the Labstation same as the machinelse name
	// Labstation hostname must be same as the machinelse hostname
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Hostname = machinelse.GetHostname()
		return UpdateLabstation(ctx, machinelse, mask)
	}

	// If its a DUT
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		// ChromeOSMachineLSE for a DUT
		machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().Hostname = machinelse.GetHostname()
		return UpdateDUT(ctx, machinelse, mask)
	}

	var oldMachinelse *ufspb.MachineLSE
	// If its a Chrome browser host, ChromeOS server or a ChormeOS labstation
	// ChromeBrowserMachineLSE, ChromeOSMachineLSE for a Server and Labstation
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(machinelse)

		// Get the old machinelse
		// getting oldmachinelse for change history logging
		oldMachinelse, err = inventory.GetMachineLSE(ctx, machinelse.GetName())
		if err != nil {
			return errors.Annotate(err, "Failed to get old MachineLSE").Err()
		}

		// Validate the input
		err := validateUpdateMachineLSE(ctx, oldMachinelse, machinelse, mask)
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

		// Copy for logging
		oldMachinelseCopy := proto.Clone(oldMachinelse).(*ufspb.MachineLSE)
		// Copy the rack/zone/manufacturer/nic/vlan to machinelse OUTPUT only fields from already existing machinelse
		machinelse.Rack = oldMachinelse.GetRack()
		machinelse.Zone = oldMachinelse.GetZone()
		machinelse.Manufacturer = oldMachinelse.GetManufacturer()
		machinelse.Nic = oldMachinelse.GetNic()
		machinelse.Vlan = oldMachinelse.GetVlan()

		// Do not let updating from browser to os or vice versa change for MachineLSE.
		if oldMachinelse.GetChromeBrowserMachineLse() != nil && machinelse.GetChromeosMachineLse() != nil {
			return status.Error(codes.InvalidArgument, "UpdateMachineLSE - cannot update a browser host to chrome os host. Please delete the browser host and create a new os host")
		}
		if oldMachinelse.GetChromeosMachineLse() != nil && machinelse.GetChromeBrowserMachineLse() != nil {
			return status.Error(codes.InvalidArgument, "UpdateMachine - cannot update an os host to browser host. Please delete the os host and create a new browser host")
		}

		// Partial update by field mask
		if mask != nil && len(mask.Paths) > 0 {
			machinelse, err = processMachineLSEUpdateMask(ctx, oldMachinelse, machinelse, mask)
			if err != nil {
				return errors.Annotate(err, "UpdateMachineLSE - processing update mask failed").Err()
			}
		} else {
			// This is for the compelte object
			if machinelse.GetMachines() == nil || len(machinelse.GetMachines()) == 0 || machinelse.GetMachines()[0] == "" {
				return status.Error(codes.InvalidArgument, "machines field cannot be empty/nil.")
			}
			// check if user is trying to associate this host with a different browser machine.
			if len(oldMachinelse.GetMachines()) > 0 && len(machinelse.GetMachines()) > 0 && oldMachinelse.GetMachines()[0] != machinelse.GetMachines()[0] {
				// Get machine to get zone and rack info for machinelse table indexing
				machine, err := GetMachine(ctx, machinelse.GetMachines()[0])
				if err != nil {
					return errors.Annotate(err, "Unable to get machine %s", machinelse.GetMachines()[0]).Err()
				}

				// Check permission
				if err := util.CheckPermission(ctx, util.InventoriesUpdate, machine.GetRealm()); err != nil {
					return err
				}
				setOutputField(ctx, machine, machinelse)
				if err := updateIndexingForMachineLSEResources(ctx, oldMachinelse, map[string]string{"zone": machine.GetLocation().GetZone().String()}); err != nil {
					return errors.Annotate(err, "failed to update zone indexing").Err()
				}
			}
		}

		// Update state
		if err := hc.stUdt.updateStateHelper(ctx, machinelse.GetResourceState()); err != nil {
			return errors.Annotate(err, "Fail to update state to host %s", machinelse.GetName()).Err()
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
		case "machines":
			// Get machine to get zone and rack info for machinelse table indexing
			machine, err := GetMachine(ctx, machinelse.GetMachines()[0])
			if err != nil {
				return oldMachinelse, errors.Annotate(err, "Unable to get machine %s", machinelse.GetMachines()[0]).Err()
			}
			oldMachinelse.Machines = machinelse.GetMachines()
			// Check permission
			if err := util.CheckPermission(ctx, util.InventoriesUpdate, machine.GetRealm()); err != nil {
				return oldMachinelse, err
			}
			setOutputField(ctx, machine, oldMachinelse)
			if err := updateIndexingForMachineLSEResources(ctx, oldMachinelse, map[string]string{"zone": machine.GetLocation().GetZone().String()}); err != nil {
				return oldMachinelse, errors.Annotate(err, "failed to update zone indexing").Err()
			}
		case "mlseprototype":
			oldMachinelse.MachineLsePrototype = machinelse.GetMachineLsePrototype()
		case "osVersion":
			if oldMachinelse.GetChromeBrowserMachineLse() != nil {
				if oldMachinelse.GetChromeBrowserMachineLse().GetOsVersion() == nil {
					oldMachinelse.GetChromeBrowserMachineLse().OsVersion = &ufspb.OSVersion{
						Value: machinelse.GetChromeBrowserMachineLse().GetOsVersion().GetValue(),
					}
				} else {
					oldMachinelse.GetChromeBrowserMachineLse().GetOsVersion().Value = machinelse.GetChromeBrowserMachineLse().GetOsVersion().GetValue()
				}
			} else if oldMachinelse.GetAttachedDeviceLse() != nil {
				if oldMachinelse.GetAttachedDeviceLse().GetOsVersion() == nil {
					oldMachinelse.GetAttachedDeviceLse().OsVersion = &ufspb.OSVersion{
						Value: machinelse.GetAttachedDeviceLse().GetOsVersion().GetValue(),
					}
				} else {
					oldMachinelse.GetAttachedDeviceLse().GetOsVersion().Value = machinelse.GetAttachedDeviceLse().GetOsVersion().GetValue()
				}
			}
		case "osImage":
			if oldMachinelse.GetChromeBrowserMachineLse() == nil {
				oldMachinelse.Lse = &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				}
			}
			if oldMachinelse.GetChromeBrowserMachineLse().GetOsVersion() == nil {
				oldMachinelse.GetChromeBrowserMachineLse().OsVersion = &ufspb.OSVersion{
					Image: machinelse.GetChromeBrowserMachineLse().GetOsVersion().GetImage(),
				}
			} else {
				oldMachinelse.GetChromeBrowserMachineLse().GetOsVersion().Image = machinelse.GetChromeBrowserMachineLse().GetOsVersion().GetImage()
			}
		case "vmCapacity":
			if oldMachinelse.GetChromeBrowserMachineLse() == nil {
				oldMachinelse.Lse = &ufspb.MachineLSE_ChromeBrowserMachineLse{
					ChromeBrowserMachineLse: &ufspb.ChromeBrowserMachineLSE{},
				}
			}
			oldMachinelse.GetChromeBrowserMachineLse().VmCapacity = machinelse.GetChromeBrowserMachineLse().GetVmCapacity()
		case "resourceState":
			oldMachinelse.ResourceState = machinelse.GetResourceState()
		case "tags":
			oldMachinelse.Tags = mergeTags(oldMachinelse.GetTags(), machinelse.GetTags())
		case "description":
			oldMachinelse.Description = machinelse.Description
		case "deploymentTicket":
			oldMachinelse.DeploymentTicket = machinelse.GetDeploymentTicket()
		case "assocHostname":
			oldMachinelse.GetAttachedDeviceLse().AssociatedHostname = machinelse.GetAttachedDeviceLse().GetAssociatedHostname()
		case "assocHostPort":
			oldMachinelse.GetAttachedDeviceLse().AssociatedHostPort = machinelse.GetAttachedDeviceLse().GetAssociatedHostPort()
		case "schedulable":
			oldMachinelse.Schedulable = machinelse.GetSchedulable()
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
	if lse.GetChromeBrowserMachineLse() != nil {
		setMachineLSE(ctx, lse)
	}
	return lse, nil
}

// BatchGetMachineLSEs returns a batch of machine lses
func BatchGetMachineLSEs(ctx context.Context, ids []string) ([]*ufspb.MachineLSE, error) {
	lses, err := inventory.BatchGetMachineLSEs(ctx, ids)
	if err != nil {
		return nil, err
	}
	// Not set vms to save time
	return lses, nil
}

// ListMachineLSEs lists the machinelses
func ListMachineLSEs(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly, full bool) ([]*ufspb.MachineLSE, string, error) {
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
		if full && !keysOnly {
			for _, lse := range lses {
				setMachineLSE(ctx, lse)
			}
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
	lses, nextPageToken, err := inventory.ListMachineLSEs(ctx, pageSize, pageToken, filterMap, keysOnly)
	if full && !keysOnly {
		for _, lse := range lses {
			// VM info not associated with CrOS machinelses.
			if lse.GetChromeBrowserMachineLse() != nil {
				setMachineLSE(ctx, lse)
			}
		}
	}
	return lses, nextPageToken, err
}

// DeleteMachineLSE deletes the machinelse in datastore
//
// For referential data intergrity,
// Delete if this MachineLSE is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteMachineLSE(ctx context.Context, id string) error {
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(&ufspb.MachineLSE{
			Name: id,
		})

		existingMachinelse, err := inventory.GetMachineLSE(ctx, id)
		if err != nil {
			return err
		}

		if err := validateDeleteMachineLSE(ctx, existingMachinelse); err != nil {
			return err
		}

		// Check if it is a DUT MachineLSE and has servo info.
		// Update corresponding Labstation MachineLSE.
		if existingMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
			existingServo := existingMachinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPeripherals().GetServo()
			// Labstation update is not required if the device is a ServoV3 or servod is running inside a docker container
			if existingServo != nil && existingServo.GetServoHostname() != "" && !util.ServoV3HostnameRegex.MatchString(existingServo.GetServoHostname()) && existingServo.GetDockerContainerName() == "" {
				// remove the existingServo entry of DUT form existingLabstationMachinelse
				existingLabstationMachinelse, err := inventory.GetMachineLSE(ctx, existingServo.GetServoHostname())
				if err != nil {
					// Log error as failure to find a labstation means that DUT was misconfigured.
					logging.Errorf(ctx, "DeleteMachineLSE - Failed to get labstation %s for update. %s", existingServo.GetServoHostname(), err)
				} else {
					// Copy for logging
					oldLabstation := proto.Clone(existingLabstationMachinelse).(*ufspb.MachineLSE)

					// remove the servo entry from labstation
					if err := removeServoEntryFromLabstation(ctx, existingServo, existingLabstationMachinelse); err != nil {
						return err
					}

					// BatchUpdate Labstation - Using Batch update and not UpdateMachineLSE,
					// because we cant have nested transaction in datastore
					_, err = inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{existingLabstationMachinelse})
					if err != nil {
						logging.Errorf(ctx, "Failed to BatchUpdate Labstation MachineLSE %s", err)
						return err
					}

					// log events for labstation
					hcLabstation := getHostHistoryClient(existingLabstationMachinelse)
					hcLabstation.LogMachineLSEChanges(oldLabstation, existingLabstationMachinelse)
					hcLabstation.SaveChangeEvents(ctx)
				}
			}
		}

		vms, err := inventory.QueryVMByPropertyName(ctx, "host_id", id, false)
		if err != nil {
			return err
		}
		setVMsToLSE(existingMachinelse, vms)

		// Delete states
		var machine *ufspb.Machine
		if len(existingMachinelse.GetMachines()) > 0 {
			machine, err = GetMachine(ctx, existingMachinelse.GetMachines()[0])
			if err != nil {
				return errors.Annotate(err, "Unable to get machine %s", existingMachinelse.GetMachines()[0]).Err()
			}
			oldMachine := proto.Clone(machine).(*ufspb.Machine)
			machine.ResourceState = ufspb.State_STATE_REGISTERED
			if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
				return errors.Annotate(err, "Fail to update machine %s", machine.GetName()).Err()
			}
			hc.LogMachineChanges(oldMachine, machine)
		}

		if err := hc.stUdt.deleteLseStateHelper(ctx, existingMachinelse, machine); err != nil {
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
		if len(vmIDs) > 0 {
			if err := inventory.BatchDeleteVMs(ctx, vmIDs); err != nil {
				return err
			}
		}

		if err := inventory.DeleteMachineLSE(ctx, id); err != nil {
			return err
		}

		// Delete machine lse deployment
		if machine.GetChromeBrowserMachine() != nil && machine.GetSerialNumber() != "" {
			err := inventory.DeleteDeployment(ctx, machine.GetSerialNumber())
			if err != nil && !util.IsNotFoundError(err) {
				return errors.Annotate(err, "fails to delete deployment record for %s", machine.GetSerialNumber()).Err()
			}
			hc.LogMachineLSEDeploymentChanges(&ufspb.MachineLSEDeployment{SerialNumber: machine.GetSerialNumber()}, nil)
		}

		hc.LogMachineLSEChanges(existingMachinelse, nil)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "DeleteMachineLSE: %s", err)
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

	lses, lseToLabConfigMap := util.ToOSMachineLSEs(labConfigs)
	deleteNonExistingMachineLSEs(ctx, lses, pageSize, "os")
	createNonExistingAssetAndMachineForLSE(ctx, lseToLabConfigMap)
	populateRackForMachineLSEs(ctx, lses)
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

	dutStates := util.ToOSDutStates(labConfigs)
	deleteNonExistingDutStates(ctx, dutStates, pageSize)
	logging.Infof(ctx, "Importing %d dut states", len(dutStates))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(dutStates))
		res, err := state.ImportDutStates(ctx, dutStates[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(dutStates) {
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
			Name: "browser:no-vm",
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
			Name: "browser:vm",
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
	deleteNonExistingMachineLSEs(ctx, lses, pageSize, "browser")
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

func populateRackForMachineLSEs(ctx context.Context, lses []*ufspb.MachineLSE) {
	for _, lse := range lses {
		if len(lse.GetMachines()) != 0 {
			machine, err := registration.GetMachine(ctx, lse.GetMachines()[0])
			if err != nil {
				logging.Infof(ctx, "Failed to get machine %s", lse.GetMachines()[0])
				continue
			}
			lse.Rack = machine.GetLocation().GetRack()
		}
	}
}

func createNonExistingAssetAndMachineForLSE(ctx context.Context, mp map[*ufspb.MachineLSE]*invV2Api.ListCrosDevicesLabConfigResponse_LabConfig) {
	for lse, lc := range mp {
		if len(lse.GetMachines()) != 0 {
			_, err := registration.GetMachine(ctx, lse.GetMachines()[0])
			if err != nil && util.IsNotFoundError(err) {
				logging.Infof(ctx, "Failed to get machine %s", lse.GetMachines()[0])
				// check and create asset
				asset, err := registration.GetAsset(ctx, lse.GetMachines()[0])
				if err != nil && util.IsNotFoundError(err) {
					logging.Infof(ctx, "Failed to get asset %s", lse.GetMachines()[0])
					assetTag := lc.GetConfig().GetId().GetValue()
					assetInfo, hostname, assetType := util.DeviceToAssetMeta(lc.GetConfig())
					location := util.HostnameToLocation(hostname)
					asset = &ufspb.Asset{
						Name:     assetTag,
						Type:     assetType,
						Model:    assetInfo.Model,
						Location: location,
						Info:     assetInfo,
						Realm:    util.ToUFSRealm(location.GetZone().String()),
					}
					logging.Infof(ctx, "Creating asset %+v", asset)
					_, err := registration.BatchUpdateAssets(ctx, []*ufspb.Asset{asset})
					if err != nil {
						logging.Errorf(ctx, "Failed to create asset %s", err)
						continue
					}
				}

				// create machine
				machine := CreateMachineFromAsset(asset)
				if machine == nil {
					logging.Errorf(ctx, "Failed to create machine from asset %s", asset)
					continue
				}
				logging.Infof(ctx, "Creating machine %+v", machine)
				if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
					logging.Errorf(ctx, "Failed to create machine %s", err)
					continue
				}
			}
		}
	}
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
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.MachineLSE)
		if lseType == "browser" && s.GetChromeosMachineLse() != nil {
			continue
		}
		if lseType == "os" && s.GetChromeBrowserMachineLse() != nil {
			continue
		}
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
		}
	}
	logging.Infof(ctx, "Deleting %d non-existing machine lses", len(toDelete))
	allRes := *deleteByPage(ctx, toDelete, pageSize, inventory.DeleteMachineLSEs)
	logging.Infof(ctx, "Deleting %d non-existing host and vm-related dhcps", len(toDelete))
	allRes = append(allRes, *deleteByPage(ctx, toDelete, pageSize, configuration.DeleteDHCPs)...)
	return &allRes, nil
}

func deleteNonExistingDutStates(ctx context.Context, dutStates []*chromeosLab.DutState, pageSize int) (*ufsds.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range dutStates {
		resMap[r.GetId().GetValue()] = true
	}
	resp, err := state.GetAllDutStates(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*chromeosLab.DutState)
		if _, ok := resMap[s.GetId().GetValue()]; !ok {
			toDelete = append(toDelete, s.GetId().GetValue())
		}
	}
	logging.Infof(ctx, "Deleting %d non-existing dut states", len(toDelete))
	allRes := *deleteByPage(ctx, toDelete, pageSize, state.DeleteDutStates)
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
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.VM)
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
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
		errorMsg.WriteString(fmt.Sprintf("Port: %d in %s is already in use by %s. Please provide a different ServoPort.\n",
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

// appendServoEntryToLabstation append servo entry to the Labstation.
//
// servo => Servo to be added to the DUT.
// labstation => Current labstation configuration.
func appendServoEntryToLabstation(ctx context.Context, servo *chromeosLab.Servo, labstation *ufspb.MachineLSE) error {
	if servo == nil || servo.GetServoHostname() == "" {
		// Nothing to append.
		return status.Errorf(codes.FailedPrecondition, "Servo/ServoHost is nil")
	}
	// Check if the servo is a V3 device. They can be updated without servo serial.
	if util.ServoV3HostnameRegex.MatchString(labstation.GetHostname()) {
		return updateServoV3EntryInLabstation(ctx, servo, labstation)
	}
	// If not a servo V3 device. Servo serial should not be empty.
	if servo.GetServoSerial() == "" {
		return status.Errorf(codes.FailedPrecondition, "Missing servo serial. Cannot assign servo")
	}
	// Not a servo v3 device. Validate port in range.
	if port := servo.GetServoPort(); port > servoPortMax || port < servoPortMin {
		return status.Errorf(codes.FailedPrecondition, "Port %v, out of range for servo", port)
	}
	// Ensure we can add the servo to the labstation.
	if err := validateServoForLabstation(ctx, servo, labstation); err != nil {
		return errors.Annotate(err, "appendServoEntryToLabstation - Cannot add servo to labstation").Err()
	}
	existingServos := labstation.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	for i, s := range existingServos {
		if s.GetServoSerial() == servo.GetServoSerial() {
			// Replace the servo entry if it exists
			existingServos[i] = s
			return nil
		}
	}
	existingServos = append(existingServos, servo)
	labstation.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = existingServos
	return nil
}

// validateServoForLabstation checks if the given servo can be used on the labstation.
func validateServoForLabstation(ctx context.Context, servo *chromeosLab.Servo, labstation *ufspb.MachineLSE) error {
	if servo == nil || servo.GetServoHostname() == "" {
		// Nothing to append.
		return status.Errorf(codes.FailedPrecondition, "validateServoForLabstation - Servo/ServoHost is nil")
	}
	if labstation == nil {
		return status.Errorf(codes.FailedPrecondition, "validateServoForLabstation - Labstation is nil")
	}

	if servo.GetServoHostname() != labstation.GetHostname() {
		status.Errorf(codes.Internal, "Cannot add servo %s:%v on %s labstation", servo.GetServoHostname(), servo.GetServoPort(), labstation.GetHostname())
	}
	// Check for port/serial number conflicts.
	for _, s := range labstation.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos() {
		// Check if servo already exists on the labstation.
		if s.GetServoSerial() == servo.GetServoSerial() {
			// Verify that such a DUT actually exists
			dut, err := GetDUTConnectedToServo(ctx, s)
			if err != nil {
				// Return error
				return errors.Annotate(err, "validateServoForLabstation - (%s:%d %s) serial conflict",
					s.GetServoHostname(), s.GetServoPort(), s.GetServoSerial()).Err()
			}
			if dut != nil {
				// Return error that the servo is connected to DUT
				return status.Errorf(codes.FailedPrecondition, "Servo serial %s exists is connected to %s",
					s.GetServoSerial(), dut.GetHostname())
			}
		}
		// Check if servo port is available
		if s.GetServoPort() == servo.GetServoPort() {
			dut, err := GetDUTConnectedToServo(ctx, s)
			if err != nil {
				// Return error
				return errors.Annotate(err, "validateServoForLabstation - (%s:%d %s) port conflict",
					servo.GetServoHostname(), servo.GetServoPort(), servo.GetServoSerial()).Err()
			}
			if dut != nil {
				return status.Errorf(codes.FailedPrecondition, "Servo port %v is in use by %s", s.GetServoPort(), dut.GetHostname())
			}
		}
	}
	return nil
}

// updateServoV3EntryInLabstation adds servo entry to labstation
func updateServoV3EntryInLabstation(ctx context.Context, servo *chromeosLab.Servo, labstation *ufspb.MachineLSE) error {
	if servo == nil || servo.GetServoHostname() == "" {
		// Nothing to append.
		return status.Errorf(codes.FailedPrecondition, "updateServoV3EntryInLabstation - Servo/ServoHost is nil")
	}
	if labstation == nil {
		return status.Errorf(codes.FailedPrecondition, "updateServoV3EntryInLabstation - Labstation is nil")
	}

	if !util.ServoV3HostnameRegex.MatchString(labstation.GetHostname()) {
		return status.Errorf(codes.Internal, "Not a servo V3 device")
	}
	servos := labstation.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	if len(servos) > 1 {
		logging.Errorf(ctx, "Servo V3 host %s cannot contain more than one servo (has %v)", labstation.GetHostname(), len(servos))
	}
	// Remove the existing record to delete oldServo.
	labstation.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = nil
	if servo == nil {
		// Deleting the servos.
		return nil
	}
	// Enforce port 9999 for all servo V3
	servo.ServoPort = int32(9999)
	// Don't store servo serial for servo V3.
	servo.ServoSerial = ""
	labstation.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = []*chromeosLab.Servo{servo}
	return nil
}

// removeServoEntryFromLabstation removes servo entry from the Labstation.
//
// servo => dut record of the servo.
// labstation => lse of the labstation.
// Servo is removed from labstation by matching servo serial except for servo V3 devices.
func removeServoEntryFromLabstation(ctx context.Context, servo *chromeosLab.Servo, labstation *ufspb.MachineLSE) error {
	logging.Warningf(ctx, "Deleting %s", servo)
	if servo == nil || labstation == nil {
		return status.Errorf(codes.Internal, "removeServoEntryFromLabstation - Invalid use of API")
	}
	// Check if it's a servo v3 device.
	if util.ServoV3HostnameRegex.MatchString(labstation.GetHostname()) {
		// Need not delete a servo v3 labstation entry
		return nil
	}
	if servo.GetServoSerial() == "" {
		return status.Errorf(codes.InvalidArgument, "Cannot remove unkown servo %s:%v. Missing serial number", labstation.GetHostname(), servo.GetServoPort())
	}
	servos := labstation.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetServos()
	// Attempt to remove by comparing servo serial first.
	for i, s := range servos {
		if s.GetServoSerial() == servo.GetServoSerial() {
			// Delete the servo. Check if port is mismatched.
			if dutSP := servo.GetServoPort(); s.GetServoPort() != dutSP {
				// Mismatch on servo record in DUT and labstation.
				logging.Warningf(ctx, "servo  %s port mismatch between dut[%s] and labstation[%s] record", s.GetServoSerial(), dutSP, s.GetServoPort())
			}
			servos[i] = servos[len(servos)-1]
			servos = servos[:len(servos)-1]
			labstation.GetChromeosMachineLse().GetDeviceLse().GetLabstation().Servos = servos
			return nil
		}
	}
	logging.Errorf(ctx, "Cannot remove servo %v from labstation %s as it contains no such record. %v", servo, labstation.GetHostname(), servos)
	return nil
}

// validateCreateMachineLSE validates if a machinelse can be created in the datastore.
func validateCreateMachineLSE(ctx context.Context, machinelse *ufspb.MachineLSE, nwOpt *ufsAPI.NetworkOption, machine *ufspb.Machine) error {
	// Check permission
	if err := util.CheckPermission(ctx, util.InventoriesCreate, machine.GetRealm()); err != nil {
		return err
	}

	// 1. Check for servos in labstation
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		// Check for servos for Labstation deployment
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
	for _, machineName := range machinelse.GetMachines() {
		resourcesNotfound = append(resourcesNotfound, GetMachineResource(machineName))
	}
	if nwOpt.GetVlan() != "" {
		resourcesNotfound = append(resourcesNotfound, GetVlanResource(nwOpt.GetVlan()))
	}
	if nwOpt.GetNic() != "" {
		resourcesNotfound = append(resourcesNotfound, GetNicResource(nwOpt.GetNic()))
	}
	if nwOpt.GetIp() != "" {
		if _, err := util.IPv4StrToInt(nwOpt.GetIp()); err != nil {
			return errors.Annotate(err, "Validate create host").Tag(grpcutil.InvalidArgumentTag).Err()
		}
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
	for _, machineName := range machinelse.GetMachines() {
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

	// 5. Check if the OS MachineLSE DUT/Labstation is trying to use an already used rpm name and rpm port
	rpmName, rpmPort := getRPMNamePortForOSMachineLSE(machinelse)
	if rpmName != "" && rpmPort != "" {
		lses, err := inventory.QueryMachineLSEByPropertyNames(ctx, map[string]string{"rpm_id": rpmName, "rpm_port": rpmPort}, true)
		if err != nil {
			return errors.Annotate(err, "Failed to query machinelses for rpm name and port %s:%s", rpmName, rpmPort).Err()
		}
		if len(lses) > 0 {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("The rpm powerunit_name and powerunit_outlet is already in use by %s.", lses[0].GetName()))
		}
	}

	// 6. Check for device config if its an OS MachineLSE
	if machinelse.GetChromeosMachineLse().GetDeviceLse() != nil {
		// Validate device config
		if err := validateDeviceConfig(ctx, machine); err != nil {
			// Keep error msg shorter to avoid hitting gRPC response length limit
			return errors.Annotate(err, "device config is not valid, please verify whether it exists in http://shortn/_CLeuVYZoWt").Err()
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

		// Since we update the nic, we have to get machinelse within the transaction
		machinelse, err = GetMachineLSE(ctx, machinelseName)
		if err != nil {
			return err
		}

		// Validate the input
		if err := validateUpdateMachineLSEHost(ctx, machinelse, nwOpt); err != nil {
			return err
		}

		// this is for logging changes
		oldMachinelse = proto.Clone(machinelse).(*ufspb.MachineLSE)
		if err := setNicIfNeeded(ctx, machinelse, nil, nwOpt); err != nil {
			return err
		}

		// Find free ip, set IP and DHCP config
		if err := hc.netUdt.addLseHostHelper(ctx, nwOpt, machinelse); err != nil {
			return errors.Annotate(err, "Fail to assign ip to host %s", machinelse.Name).Err()
		}
		machinelse.ResourceState = ufspb.State_STATE_DEPLOYING
		if err := hc.stUdt.updateStateHelper(ctx, machinelse.ResourceState); err != nil {
			return errors.Annotate(err, "Fail to update state to host %s", machinelse.GetName()).Err()
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
func validateUpdateMachineLSEHost(ctx context.Context, machinelse *ufspb.MachineLSE, nwOpt *ufsAPI.NetworkOption) error {
	machine, err := registration.GetMachine(ctx, machinelse.GetMachines()[0])
	if err != nil {
		return errors.Annotate(err, "unable to get machine %s", machinelse.GetMachines()[0]).Err()
	}
	// Check permission
	if err := util.CheckPermission(ctx, util.InventoriesUpdate, machine.GetRealm()); err != nil {
		return err
	}
	// Aggregate resource to check if machinelse does not exist
	var resourcesNotFound []*Resource
	if nwOpt.GetVlan() != "" {
		resourcesNotFound = append(resourcesNotFound, GetVlanResource(nwOpt.GetVlan()))
	}
	if nwOpt.GetNic() != "" {
		resourcesNotFound = append(resourcesNotFound, GetNicResource(nwOpt.GetNic()))
	}
	if nwOpt.GetIp() != "" {
		if _, err := util.IPv4StrToInt(nwOpt.GetIp()); err != nil {
			return errors.Annotate(err, "Validate update host").Tag(grpcutil.InvalidArgumentTag).Err()
		}
	}
	// Check if resources does not exist
	return ResourceExist(ctx, resourcesNotFound, nil)
}

// validateUpdateMachineLSEPoolNames validates if a pool name can be assigned to the MachineLSE
func validateUpdateMachineLSEPoolNames(ctx context.Context, machinelse *ufspb.MachineLSE) error {
	var pools []string
	// If its a LabStation
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
		pools = machinelse.GetChromeosMachineLse().GetDeviceLse().GetLabstation().GetPools()
	}

	// If its a DUT
	if machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
		pools = machinelse.GetChromeosMachineLse().GetDeviceLse().GetDut().GetPools()
	}

	for _, poolName := range pools {
		if poolName != "" && !util.PoolNameRegex.MatchString(poolName) {
			return status.Errorf(codes.InvalidArgument, "Invalid Pool Name %s", poolName)
		}
	}
	return nil
}

// DeleteMachineLSEHost deletes the dhcp/ip of a machinelse in datastore.
func DeleteMachineLSEHost(ctx context.Context, machinelseName string) error {
	f := func(ctx context.Context) error {
		hc := getHostHistoryClient(&ufspb.MachineLSE{Name: machinelseName})

		lse, err := inventory.GetMachineLSE(ctx, machinelseName)
		if err != nil {
			return err
		}

		if err := validateDeleteMachineLSEHost(ctx, lse); err != nil {
			return err
		}

		if err := hc.netUdt.deleteDHCPHelper(ctx); err != nil {
			return err
		}

		lseCopy := proto.Clone(lse).(*ufspb.MachineLSE)
		lse.Nic = ""
		lse.Vlan = ""
		lse.Ip = ""
		lse.ResourceState = ufspb.State_STATE_REGISTERED
		if _, err := inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{lse}); err != nil {
			return errors.Annotate(err, "Failed to update host %q", machinelseName).Err()
		}
		hc.stUdt.updateStateHelper(ctx, lse.ResourceState)
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
func validateUpdateMachineLSE(ctx context.Context, oldMachinelse *ufspb.MachineLSE, machinelse *ufspb.MachineLSE, mask *field_mask.FieldMask) error {
	machine, err := registration.GetMachine(ctx, oldMachinelse.GetMachines()[0])
	if err != nil {
		return errors.Annotate(err, "unable to get machine %s", oldMachinelse.GetMachines()[0]).Err()
	}
	// Check permission
	if err := util.CheckPermission(ctx, util.InventoriesUpdate, machine.GetRealm()); err != nil {
		return err
	}

	// 1. Check if resources does not exist
	// Aggregate resource to check if machinelse does not exist
	resourcesNotfound := []*Resource{GetMachineLSEResource(machinelse.Name)}
	// Aggregate resource to check if machines does not exist
	for _, machineName := range machinelse.GetMachines() {
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
		return errors.Annotate(err, "%s", machinelse.Name).Err()
	}

	// 2. Check if any machine is already associated with another MachineLSE
	// A machine cannot be associated with multiple hosts/machinelses
	for _, machineName := range machinelse.GetMachines() {
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

	// 3. Check if the OS MachineLSE DUT/Labstation is trying to use an already used rpm name and rpm port
	rpmName, rpmPort := getRPMNamePortForOSMachineLSE(machinelse)
	if rpmName != "" && rpmPort != "" {
		lses, err := inventory.QueryMachineLSEByPropertyNames(ctx, map[string]string{"rpm_id": rpmName, "rpm_port": rpmPort}, true)
		if err != nil {
			return errors.Annotate(err, "Failed to query machinelses for rpm name and port %s:%s", rpmName, rpmPort).Err()
		}
		for _, lse := range lses {
			if lse.GetName() != machinelse.Name {
				return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("The rpm powerunit_name and powerunit_outlet is already in use by %s.", lse.GetName()))
			}
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
			case "machines":
				if machinelse.GetMachines() == nil || len(machinelse.GetMachines()) == 0 || machinelse.GetMachines()[0] == "" {
					return status.Error(codes.InvalidArgument, "machines field cannot be empty/nil.")
				}
			case "mlseprototype":
			case "osImage":
				fallthrough
			case "osVersion":
				if machinelse.GetChromeBrowserMachineLse() == nil && machinelse.GetAttachedDeviceLse() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineLSEUpdateMask - browser / attached device machine lse cannot be empty/nil.")
				}
				if (machinelse.GetChromeBrowserMachineLse() != nil && machinelse.GetChromeBrowserMachineLse().GetOsVersion() == nil) ||
					(machinelse.GetAttachedDeviceLse() != nil && machinelse.GetAttachedDeviceLse().GetOsVersion() == nil) {
					return status.Error(codes.InvalidArgument, "validateMachineLSEUpdateMask - OsVersion cannot be empty/nil.")
				}
			case "vmCapacity":
				if machinelse.GetChromeBrowserMachineLse() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineLSEUpdateMask - browser machine lse cannot be empty/nil.")
				}
			case "assocHostname":
				if machinelse.GetAttachedDeviceLse() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineLSEUpdateMask - machine is not an attached device")
				}
			case "assocHostPort":
				if machinelse.GetAttachedDeviceLse() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineLSEUpdateMask - machine is not an attached device")
				}
			case "schedulable":
			case "deploymentTicket":
			case "tags":
			case "description":
			case "resourceState":
				// valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "validateMachineLSEUpdateMask - unsupported update mask path %q", path)
			}
		}
	}
	return nil
}

// validateDeleteMachineLSE validates if a MachineLSE can be deleted
func validateDeleteMachineLSE(ctx context.Context, existingMachinelse *ufspb.MachineLSE) error {
	existingMachinelse, err := inventory.GetMachineLSE(ctx, existingMachinelse.GetName())
	if err != nil {
		return err
	}
	machine, err := registration.GetMachine(ctx, existingMachinelse.GetMachines()[0])
	if err != nil {
		return errors.Annotate(err, "unable to get machine %s", existingMachinelse.GetMachines()[0]).Err()
	}
	// Check permission
	if err := util.CheckPermission(ctx, util.InventoriesDelete, machine.GetRealm()); err != nil {
		return err
	}
	if existingMachinelse.GetChromeosMachineLse() != nil {
		schedulingUnits, err := inventory.QuerySchedulingUnitByPropertyNames(ctx, map[string]string{"machinelses": existingMachinelse.GetName()}, true)
		if err != nil {
			return errors.Annotate(err, "failed to query SchedulingUnit for machinelses %s", existingMachinelse.GetName()).Err()
		}
		if len(schedulingUnits) > 0 {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("DUT is associated with SchedulingUnit. Run `shivas update schedulingunit -name %s -removeduts %s` to remove association before deleting the DUT.", schedulingUnits[0].GetName(), existingMachinelse.GetName()))
		}
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
				existingMachinelse.GetName(), strings.Join(nonDeletedHosts, ", "))
			logging.Errorf(ctx, errorMsg)
			return status.Errorf(codes.FailedPrecondition, errorMsg)
		}
	}
	return nil
}

func setNicIfNeeded(ctx context.Context, lse *ufspb.MachineLSE, machine *ufspb.Machine, nwOpt *ufsAPI.NetworkOption) error {
	if (nwOpt.GetVlan() != "" || nwOpt.GetIp() != "") && nwOpt.GetNic() == "" {
		var err error
		if machine == nil {
			machine, err = GetMachine(ctx, lse.GetMachines()[0])
			if err != nil {
				return errors.Annotate(err, "unable to get machine of host %s", lse.GetName()).Err()
			}
		}
		nics := machine.GetChromeBrowserMachine().GetNicObjects()
		if len(nics) > 1 {
			return status.Errorf(codes.InvalidArgument,
				"The attached machine %s has more than 1 nic (%s), please specify the nic for ip assignment",
				machine.GetName(),
				strings.Join(ufsAPI.ParseResources(nics, "Name"), ","))
		}
		if len(nics) == 0 {
			return status.Errorf(codes.InvalidArgument, "The attached machine %s has no nic for ip assignment", machine.GetName())
		}
		nwOpt.Nic = machine.GetChromeBrowserMachine().GetNicObjects()[0].GetName()
	}
	return nil
}

func setOutputField(ctx context.Context, machine *ufspb.Machine, lse *ufspb.MachineLSE) error {
	lse.Rack = machine.GetLocation().GetRack()
	lse.Zone = machine.GetLocation().GetZone().String()
	for _, vm := range lse.GetChromeBrowserMachineLse().GetVms() {
		vm.Zone = machine.GetLocation().GetZone().String()
		vm.MachineLseId = lse.GetName()
		vm.ResourceState = ufspb.State_STATE_REGISTERED
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

// validateDeleteMachineLSEHost validates if a lse host can be deleted
func validateDeleteMachineLSEHost(ctx context.Context, lse *ufspb.MachineLSE) error {
	machine, err := registration.GetMachine(ctx, lse.GetMachines()[0])
	if err != nil {
		return errors.Annotate(err, "unable to get machine %s", lse.GetMachines()[0]).Err()
	}
	// Check permission
	if err := util.CheckPermission(ctx, util.InventoriesDelete, machine.GetRealm()); err != nil {
		return err
	}
	return nil
}

// updateIndexingForMachineLSE updates indexing for Machinelse table
// can be used inside a transaction
func updateIndexingForMachineLSE(ctx context.Context, property, oldValue, newValue string, hc *HistoryClient) error {
	var lses []*ufspb.MachineLSE
	var err error
	switch property {
	case "machine":
		// Update the MachineLSE with new machine name and nic name
		lses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", oldValue, false)
		if err != nil {
			return errors.Annotate(err, "failed to query machinelses/hosts for machine %s", oldValue).Err()
		}
		for _, lse := range lses {
			// Copy for logging
			oldLseCopy := proto.Clone(lse).(*ufspb.MachineLSE)
			machines := lse.GetMachines()
			for i := range machines {
				if machines[i] == oldValue {
					machines[i] = newValue
					break
				}
			}
			lse.Machines = machines
			// Update the nic name as well
			lse.Nic = util.GetNewNicNameForRenameMachine(lse.GetNic(), oldValue, newValue)
			hc.LogMachineLSEChanges(oldLseCopy, lse)
		}
		if _, err = inventory.BatchUpdateMachineLSEs(ctx, lses); err != nil {
			return errors.Annotate(err, "unable to batch update machinelses").Err()
		}
	case "nic":
		// get MachineLSEs for nic indexing
		lses, err = inventory.QueryMachineLSEByPropertyName(ctx, "nic", oldValue, false)
		if err != nil {
			return errors.Annotate(err, "failed to query machinelses/hosts for nic %s", oldValue).Err()
		}
		for _, lse := range lses {
			oldLseCopy := proto.Clone(lse).(*ufspb.MachineLSE)
			lse.Nic = newValue
			hc.LogMachineLSEChanges(oldLseCopy, lse)
		}
	case "switch":
		// get MachineLSEs for switch indexing
		lses, err = inventory.QueryMachineLSEByPropertyName(ctx, "switch_id", oldValue, false)
		if err != nil {
			return errors.Annotate(err, "failed to query machinelses/hosts for switch %s", oldValue).Err()
		}
		for _, lse := range lses {
			oldLseCopy := proto.Clone(lse).(*ufspb.MachineLSE)
			lse.GetChromeosMachineLse().GetDeviceLse().GetNetworkDeviceInterface().Switch = newValue
			hc.LogMachineLSEChanges(oldLseCopy, lse)
		}
	}
	if _, err = inventory.BatchUpdateMachineLSEs(ctx, lses); err != nil {
		return errors.Annotate(err, "unable to batch update machinelses").Err()
	}
	return nil
}

// UpdateLabMeta updates only lab meta data for a given ChromeOS DUT.
func UpdateLabMeta(ctx context.Context, meta *ufspb.LabMeta) error {
	if meta == nil {
		return nil
	}
	f := func(ctx context.Context) error {
		lse, err := inventory.GetMachineLSE(ctx, meta.GetHostname())
		if err != nil {
			return err
		}
		hc := getHostHistoryClient(lse)

		dut := lse.GetChromeosMachineLse().GetDeviceLse().GetDut()
		if dut == nil {
			logging.Warningf(ctx, "%s is not a valid Chromeos DUT", meta.GetHostname())
			return nil
		}

		// Copy for logging
		oldLSE := proto.Clone(lse).(*ufspb.MachineLSE)
		if servo := dut.GetPeripherals().GetServo(); servo != nil {
			servo.ServoType = meta.GetServoType()
			servo.ServoTopology = meta.GetServoTopology()
			servo.ServoComponent = extractServoComponents(meta.GetServoType())
		}
		// Periphrals cannot be nil for valid DUT
		if dut.GetPeripherals() == nil {
			dut.Peripherals = &chromeosLab.Peripherals{}
		}
		dut.GetPeripherals().SmartUsbhub = meta.GetSmartUsbhub()
		if _, err = inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{lse}); err != nil {
			return errors.Annotate(err, "Unable to update lab meta for %s", lse.Name).Err()
		}
		hc.LogMachineLSEChanges(oldLSE, lse)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "UpdateLabMeta (%s, %s) - %s", meta.GetChromeosDeviceId(), meta.GetHostname(), err)
		return err
	}
	return nil
}

// UpdateRecoveryLabdata updates only labdata and resource state for a given ChromeOS DUT.
func updateRecoveryLabData(ctx context.Context, hostname string, resourceState ufspb.State, labData *ufsAPI.UpdateDeviceRecoveryDataRequest_LabData) error {
	f := func(ctx context.Context) error {
		lse, err := inventory.GetMachineLSE(ctx, hostname)
		if err != nil {
			return err
		}
		hc := getHostHistoryClient(lse)
		oldLSE := proto.Clone(lse).(*ufspb.MachineLSE)
		// Apply resource_state edits
		lse.ResourceState = resourceState
		if labData == nil {
			// TODO add to Proto labdata - Not be updated if labdata is nil
			logging.Warningf(ctx, "updateRecoveryLabData: empty labData(%q)", labData)
		} else {
			dut := lse.GetChromeosMachineLse().GetDeviceLse().GetDut()
			if dut == nil {
				logging.Warningf(ctx, "%s is not a valid Chromeos DUT", lse.GetName())
			} else {
				// Periphrals cannot be nil for valid DUT
				if dut.GetPeripherals() == nil {
					dut.Peripherals = &chromeosLab.Peripherals{}
				}
				peri := dut.GetPeripherals()
				// Copy for logging
				// Apply smart usb hub edits
				peri.SmartUsbhub = labData.GetSmartUsbhub()

				// Servo cannot be nil for valid DUT
				if peri.GetServo() == nil {
					peri.Servo = &chromeosLab.Servo{}
				}
				// Apply servo edits
				if err = editRecoveryPeripheralServo(peri.GetServo(), labData); err != nil {
					return err
				}
				// Wifi cannot be nil for valid DUT
				if peri.GetWifi() == nil {
					peri.Wifi = &chromeosLab.Wifi{}
				}
				// Apply wifirouters edits
				if err = editRecoveryPeripheralWifi(ctx, peri.GetWifi(), labData); err != nil {
					return err
				}

				if err = updateBluetoothPeerStates(peri, labData.GetBlueoothPeers()); err != nil {
					return err
				}
			}
		}
		if _, err = inventory.BatchUpdateMachineLSEs(ctx, []*ufspb.MachineLSE{lse}); err != nil {
			return errors.Annotate(err, "unable to update labData for %s", lse.Name).Err()
		}
		hc.LogMachineLSEChanges(oldLSE, lse)
		return hc.SaveChangeEvents(ctx)

	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "updateRecoveryDataDeviceLSE  (%s) - %s", hostname, err)
		return err
	}
	return nil
}

// updateBluetoothPeerStates updates p.BluetoothPeers with state from btps. It returns an error if a hostname
// that is not part of p is sent in btps. It handles nil btps.
func updateBluetoothPeerStates(p *chromeosLab.Peripherals, btps []*ufsAPI.UpdateDeviceRecoveryDataRequest_BluetoothPeer) error {
	if len(btps) == 0 {
		return nil
	}
	ufsBTPs := make(map[string]*chromeosLab.BluetoothPeer)
	for _, btp := range p.GetBluetoothPeers() {
		d := btp.GetDevice()
		if _, ok := d.(*chromeosLab.BluetoothPeer_RaspberryPi); !ok {
			return errors.Reason("unsupported BTP device type %T", d).Err()
		}
		ufsBTPs[btp.GetRaspberryPi().GetHostname()] = btp
	}

	for _, btp := range btps {
		b, ok := ufsBTPs[btp.GetHostname()]
		if !ok {
			return errors.Reason("unknown BTP with hostname %q recieved from lab", btp.GetHostname()).Err()
		}
		b.GetRaspberryPi().State = btp.GetState()
	}
	return nil
}

// editRecoveryPeripheralServo edits peripherals servo
func editRecoveryPeripheralServo(servo *chromeosLab.Servo, labData *ufsAPI.UpdateDeviceRecoveryDataRequest_LabData) error {
	servo.ServoType = labData.GetServoType()
	servo.ServoTopology = labData.GetServoTopology()
	servo.ServoComponent = extractServoComponents(labData.GetServoType())
	return nil
}

// editRecoveryPeripheralServo edits peripherals Wifi
func editRecoveryPeripheralWifi(ctx context.Context, wifi *chromeosLab.Wifi, labData *ufsAPI.UpdateDeviceRecoveryDataRequest_LabData) error {
	// labDataRouterMap is Wifirouters as hostname-> wifirouter hashmap for easier individual Wifirouter update
	labDataRouterMap := make(map[string]*ufsAPI.UpdateDeviceRecoveryDataRequest_WifiRouter)
	for _, labDataRouter := range labData.GetWifiRouters() {
		labDataRouterMap[labDataRouter.GetHostname()] = labDataRouter
	}
	newRouters := []*chromeosLab.WifiRouter{}
	for _, lseRouter := range wifi.GetWifiRouters() {
		// edit wifirouter if router already exists in UFS
		if labDataRouter, ok := labDataRouterMap[lseRouter.GetHostname()]; ok {
			logging.Infof(ctx, "editRecoverPeripheralWifi - edit wifi router(%s), found in labdata.", lseRouter.GetHostname())
			lseRouter.State = labDataRouter.GetState()
			newRouters = append(newRouters, lseRouter)
			delete(labDataRouterMap, lseRouter.GetHostname())
		} else {
			// remove from UFS if not in lab data
			logging.Infof(ctx, "editRecoverPeripheralWifi - remove wifi router(%s), not found in labdata.", lseRouter.GetHostname())
		}
	}
	// add new wifirouters to UFS
	for hostname := range labDataRouterMap {
		logging.Infof(ctx, "editRecoverPeripheralWifi - add wifi router(%s) new in labdata.", hostname)
		newRouters = append(newRouters, &chromeosLab.WifiRouter{
			Hostname: hostname,
			State:    labDataRouterMap[hostname].GetState(),
		})
	}
	// assign updated routers to Wifi
	wifi.WifiRouters = newRouters
	return nil
}

// extractServoComponents extracts servo components based on servo_type.
// TODO(xianuowang): Move this function out of UFS since UFS doesn't have knowledge of
// how this should works.
func extractServoComponents(servoType string) []string {
	var servoComponents []string
	reg := regexp.MustCompile("_with_|_and_")
	for _, c := range reg.Split(servoType, -1) {
		// Sanitize empty string in case of servoType is empty or invalid.
		if len(c) > 0 {
			servoComponents = append(servoComponents, c)
		}
	}
	return servoComponents
}

// RenameMachineLSE renames the machineLSE to the new hostname.
func RenameMachineLSE(ctx context.Context, oldName, newName string) (*ufspb.MachineLSE, error) {
	var newLSE *ufspb.MachineLSE
	f := func(ctx context.Context) error {
		// Check if the host exists
		lse, err := inventory.GetMachineLSE(ctx, oldName)
		if err != nil {
			return err
		}
		if lse.GetChromeBrowserMachineLse() != nil {
			return status.Errorf(codes.Unimplemented, fmt.Sprintf("Renaming %s [browser host] is not supported yet", oldName))
		}
		machine, err := registration.GetMachine(ctx, lse.GetMachines()[0])
		if err != nil {
			return errors.Annotate(err, "unable to get machine %s. Misconfigured host?", lse.GetMachines()[0]).Err()
		}
		if err := validateRenameMachineLSE(ctx, oldName, newName, lse, machine); err != nil {
			return err
		}
		if lse.GetChromeosMachineLse().GetDeviceLse().GetDut() != nil {
			if newLSE, err = renameDUT(ctx, oldName, newName, lse, machine); err != nil {
				return err
			}
			return nil
		}
		if lse.GetChromeosMachineLse().GetDeviceLse().GetLabstation() != nil {
			if newLSE, err = renameLabstation(ctx, oldName, newName, lse, machine); err != nil {
				return err
			}
			return nil
		}
		return status.Errorf(codes.Unimplemented, fmt.Sprintf("Renaming %s is not supported yet", oldName))
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "RenameMachineLSE [%s -> %s] failed. %s", oldName, newName, err.Error())
		return nil, err
	}
	return newLSE, nil
}

func validateRenameMachineLSE(ctx context.Context, oldName, newName string, lse *ufspb.MachineLSE, machine *ufspb.Machine) error {
	if err := resourceAlreadyExists(ctx, []*Resource{GetMachineLSEResource(newName)}, nil); err != nil {
		return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("Failed to rename %s. %s already exists", oldName, newName))
	}
	// You need both delete and create permissions to do anything here
	if err := util.CheckPermission(ctx, util.InventoriesDelete, machine.GetRealm()); err != nil {
		return status.Errorf(codes.PermissionDenied, fmt.Sprintf("Need delete permission to rename %s. %s", oldName, err.Error()))
	}
	if err := util.CheckPermission(ctx, util.InventoriesCreate, machine.GetRealm()); err != nil {
		return status.Errorf(codes.PermissionDenied, fmt.Sprintf("Need create permission to rename %s. %s", oldName, err.Error()))
	}
	return nil
}

// GetAttachedDeviceData returns AttachedDeviceData for the given id/hostname from UFS.
func GetAttachedDeviceData(ctx context.Context, lse *ufspb.MachineLSE) (*ufsAPI.AttachedDeviceData, error) {
	if lse == nil {
		return nil, fmt.Errorf("host cannot be empty")
	}
	if len(lse.GetMachines()) == 0 {
		return nil, fmt.Errorf("host does not have machines registered to it")
	}
	machineId := lse.GetMachines()[0]
	dutState, err := state.GetDutState(ctx, machineId)
	if err != nil {
		logging.Warningf(ctx, "DutState for %s not found. Error: %s", machineId, err)
	}
	machine, err := GetMachine(ctx, machineId)
	if err != nil {
		logging.Errorf(ctx, "Machine for %s not found. Error: %s", machineId, err)
	}
	return &ufsAPI.AttachedDeviceData{
		LabConfig: lse,
		Machine:   machine,
		DutState:  dutState,
	}, nil
}
