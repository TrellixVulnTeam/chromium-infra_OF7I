// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

// MachineRegistration creates a new machine, new nic and a new drac in datastore.
func MachineRegistration(ctx context.Context, machine *ufspb.Machine) (*ufspb.Machine, error) {
	// Older clients of Shivas send OldBrowserLabAdminRealm, which needs to be replaced with BrowserLabAdminRealm
	machine.Realm = util.GetValidRealmName(machine.GetRealm())

	nics := machine.GetChromeBrowserMachine().GetNicObjects()
	drac := machine.GetChromeBrowserMachine().GetDracObject()
	f := func(ctx context.Context) error {
		hc := GetMachineHistoryClient(machine)
		// Validate input
		if err := validateMachineRegistration(ctx, machine); err != nil {
			return errors.Annotate(err, "validation failed").Err()
		}

		// Create nics
		if nics != nil {
			for _, nic := range nics {
				// Fill the machine/rack/zone to nic OUTPUT only fields
				nic.Machine = machine.GetName()
				nic.Rack = machine.GetLocation().GetRack()
				nic.Zone = machine.GetLocation().GetZone().String()
			}
			if _, err := registration.BatchUpdateNics(ctx, nics); err != nil {
				return errors.Annotate(err, "unable to batch update nics").Err()
			}
			// We do not save nic objects in machine table
			machine.GetChromeBrowserMachine().NicObjects = nil
		}

		// Create drac
		if drac != nil {
			// Fill the machine/rack/zone to drac OUTPUT only fields
			drac.Machine = machine.GetName()
			drac.Rack = machine.GetLocation().GetRack()
			drac.Zone = machine.GetLocation().GetZone().String()
			if _, err := registration.BatchUpdateDracs(ctx, []*ufspb.Drac{drac}); err != nil {
				return errors.Annotate(err, "unable to batch update drac").Err()
			}
			// We do not save drac object in machine table
			machine.GetChromeBrowserMachine().DracObject = nil
		}

		// Create the machine
		machine.ResourceState = ufspb.State_STATE_REGISTERED
		if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
			return errors.Annotate(err, "unable to batch update machine").Err()
		}

		if machine.GetChromeBrowserMachine() != nil {
			// Update deployment record
			if machine.GetSerialNumber() != "" {
				// If the deployment record already exist, it usually means sth weird happened, but we will keep
				// the existing deployment record anyway.
				_, err := inventory.GetMachineLSEDeployment(ctx, machine.GetSerialNumber())
				if util.IsNotFoundError(err) {
					dr := util.FormatDeploymentRecord("", machine.GetSerialNumber())
					if _, err := inventory.UpdateMachineLSEDeployments(ctx, []*ufspb.MachineLSEDeployment{dr}); err != nil {
						return errors.Annotate(err, "unable to update deployment record").Err()
					}
					hc.LogMachineLSEDeploymentChanges(nil, dr)
				}
			}
			// We fill the machine object with newly created nics/drac from nic/drac table
			// This will have all the extra information for nics/drac(machine name, updated time.. etc)
			machine.GetChromeBrowserMachine().NicObjects = nics
			machine.GetChromeBrowserMachine().DracObject = drac
		}
		hc.LogAddMachineChanges(machine, nics, drac)
		hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_REGISTERED)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "MachineRegistration").Err()
	}
	return machine, nil
}

// UpdateMachine updates machine in datastore.
//
// Checks if the resources referenced by the Machine input already exists
// in the system before updating a Machine
func UpdateMachine(ctx context.Context, machine *ufspb.Machine, mask *field_mask.FieldMask) (*ufspb.Machine, error) {
	// Older clients of Shivas send OldBrowserLabAdminRealm, which needs to be replaced with BrowserLabAdminRealm
	machine.Realm = util.GetValidRealmName(machine.GetRealm())

	var oldMachine *ufspb.Machine
	var err error
	f := func(ctx context.Context) error {
		hc := GetMachineHistoryClient(machine)

		// Get the existing/old machine
		oldMachine, err = registration.GetMachine(ctx, machine.GetName())
		if err != nil {
			return errors.Annotate(err, "get machine %s failed", machine.GetName()).Err()
		}

		// Validate input
		if err := validateUpdateMachine(ctx, oldMachine, machine, mask); err != nil {
			return errors.Annotate(err, "validation failed").Err()
		}

		// Copy for logging
		oldMachineCopy := proto.Clone(oldMachine).(*ufspb.Machine)

		if machine.GetChromeBrowserMachine() != nil {
			// nics and dracs are not allowed to update in UpdateMachine call.
			// We dont store nics/drac object inside Machine object in Machine table.
			// nics/drac objects are stored in their separate tables
			// user has to use nic/drac CRUD apis to update nic/drac
			machine.GetChromeBrowserMachine().NicObjects = nil
			machine.GetChromeBrowserMachine().DracObject = nil
		}

		if machine.GetChromeosMachine() != nil {
			// OUTPUT_ONLY fields for chrome os machine.
			// copy back original values.
			machine.SerialNumber = oldMachine.GetSerialNumber()
			machine.GetChromeosMachine().Sku = oldMachine.GetChromeosMachine().GetSku()

			// Allow users to modify hwid before we can get authoritative source from HaRT
			// Don't allow users to modify it to empty
			if machine.GetChromeosMachine().Hwid == "" {
				machine.GetChromeosMachine().Hwid = oldMachine.GetChromeosMachine().GetHwid()
			}
		}

		// Do not let updating from browser to os or vice versa change for machine.
		if oldMachine.GetChromeBrowserMachine() != nil && machine.GetChromeosMachine() != nil {
			return status.Error(codes.InvalidArgument, "cannot update a browser machine to os machine. Please delete the browser machine and create a new os machine")
		}
		if oldMachine.GetChromeosMachine() != nil && machine.GetChromeBrowserMachine() != nil {
			return status.Error(codes.InvalidArgument, "cannot update an os machine to browser machine. Please delete the os machine and create a new browser machine")
		}

		// Partial update by field mask
		if mask != nil && len(mask.Paths) > 0 {
			machine, err = processMachineUpdateMask(ctx, oldMachine, machine, mask, hc)
			if err != nil {
				return errors.Annotate(err, "processing update mask failed").Err()
			}
		} else if machine.GetLocation().GetRack() != oldMachine.GetLocation().GetRack() ||
			machine.GetLocation().GetZone() != oldMachine.GetLocation().GetZone() {
			// this check is for json input with complete update machine
			// Check if machine zone/rack information is changed/updated
			indexMap := map[string]string{
				"zone": machine.GetLocation().GetZone().String(), "rack": machine.GetLocation().GetRack()}
			oldIndexMap := map[string]string{
				"zone": oldMachine.GetLocation().GetZone().String(), "rack": oldMachine.GetLocation().GetRack()}
			if err = updateIndexingForMachineResources(ctx, oldMachine, indexMap, oldIndexMap, hc); err != nil {
				return errors.Annotate(err, "update zone and rack indexing failed").Err()
			}
		}

		// Update state
		if err := hc.stUdt.updateStateHelper(ctx, machine.GetResourceState()); err != nil {
			return errors.Annotate(err, "Fail to update state of machine %s", machine.GetName()).Err()
		}

		// Update deployment record if serial number is updated
		if machine.GetChromeBrowserMachine() != nil && oldMachineCopy.GetSerialNumber() != machine.GetSerialNumber() {
			// If the deployment record for newly updated machine's serial number already exist, it usually
			// means sth weird happened, but we will keep the existing deployment record anyway, even if
			// the attached hostname may be not correct.
			_, err := inventory.GetMachineLSEDeployment(ctx, machine.GetSerialNumber())
			if util.IsNotFoundError(err) {
				hostname := ""
				lses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", machine.GetName(), true)
				if err != nil {
					logging.Infof(ctx, "fail to get hosts for machine %s", machine.GetName())
				} else if len(lses) > 0 {
					// It's possible that when the host is added to UFS, the corresponding machine
					// doesn't have any serial number. Later users add the serial numbers back for
					// the machine, then the deployment record should be updated also.
					hostname = lses[0].GetName()
				}
				dr := util.FormatDeploymentRecord(hostname, machine.GetSerialNumber())
				if _, err := inventory.UpdateMachineLSEDeployments(ctx, []*ufspb.MachineLSEDeployment{dr}); err != nil {
					return errors.Annotate(err, "unable to update deployment record").Err()
				}
				hc.LogMachineLSEDeploymentChanges(nil, dr)
			}
			if oldMachineCopy.GetSerialNumber() != "" {
				if err := inventory.DeleteDeployment(ctx, oldMachineCopy.GetSerialNumber()); err != nil {
					if util.IsNotFoundError(err) {
						logging.Infof(ctx, "no deployment record for %s", oldMachineCopy.GetSerialNumber())
					} else {
						return errors.Annotate(err, "unable to delete deployment record").Err()
					}
				}
				hc.LogMachineLSEDeploymentChanges(&ufspb.MachineLSEDeployment{SerialNumber: oldMachineCopy.GetSerialNumber()}, nil)
			}
		}

		// update the machine
		if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
			return errors.Annotate(err, "unable to batch update machine %s", machine.Name).Err()
		}
		hc.LogMachineChanges(oldMachineCopy, machine)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "UpdateMachine").Err()
	}
	if oldMachine.GetChromeBrowserMachine() != nil {
		// We fill the machine object with its nics/drac from nic/drac table
		setMachine(ctx, machine)
	}
	return machine, nil
}

// UpdateDutMeta updates only duta meta data for a given ChromeOS DUT.
//
// It's a temporary method to correct serial number & HWID.
// Will remove once HaRT could provide us the correct info.
func UpdateDutMeta(ctx context.Context, meta *ufspb.DutMeta) error {
	if meta == nil {
		return nil
	}
	f := func(ctx context.Context) error {
		machine, err := registration.GetMachine(ctx, meta.GetChromeosDeviceId())
		if err != nil {
			return err
		}
		hc := GetMachineHistoryClient(machine)

		osMachine := machine.GetChromeosMachine()
		if osMachine == nil {
			logging.Warningf(ctx, "%s is not a valid Chromeos machine", meta.GetChromeosDeviceId())
			return nil
		}
		// Copy for logging
		oldMachine := proto.Clone(machine).(*ufspb.Machine)

		if machine.GetSerialNumber() == meta.GetSerialNumber() &&
			machine.GetChromeosMachine().GetHwid() == meta.GetHwID() &&
			machine.GetChromeosMachine().GetSku() == meta.GetDeviceSku() {
			logging.Warningf(ctx, "nothing to update: old serial number %q, old hwid %q, old device-sku %q", meta.GetSerialNumber(), meta.GetHwID(), meta.GetDeviceSku())
			return nil
		}

		machine.SerialNumber = meta.GetSerialNumber()
		machine.GetChromeosMachine().Hwid = meta.GetHwID()
		machine.GetChromeosMachine().Sku = meta.GetDeviceSku()
		// Update the machine
		// Won't update asset as asset will be updated from Inv2 in pre-launch period, which will definitely be correct.
		// Later when the asset info is updated from HaRT, UpdateDutMeta will be abandoned.
		if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
			return errors.Annotate(err, "Unable to update dut meta for %s", machine.Name).Err()
		}
		hc.LogMachineChanges(oldMachine, machine)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "UpdateDutMeta (%s, %s) - %s", meta.GetChromeosDeviceId(), meta.GetHostname(), err.Error())
		return err
	}
	return nil
}

// updateRecoveryDutData is a duplicate of UpdateDutMeta with new request API
// Remove UpdateDutMeta when recovery tool migrate to updateRecoveryDutData and no other tools is using it.
//
// It's a temporary method to correct serial number & HWID.
// Will remove once HaRT could provide us the correct info.
func updateRecoveryDutData(ctx context.Context, dutId string, dutData *ufsAPI.UpdateDeviceRecoveryDataRequest_DutData) error {
	if dutData == nil {
		logging.Warningf(ctx, "Empty dut data (%s)", dutId)
		return nil
	}
	f := func(ctx context.Context) error {
		machine, err := registration.GetMachine(ctx, dutId)
		if err != nil {
			logging.Errorf(ctx, "updateRecoveryMachineHelper machine not found(%s) - %s", dutId, err.Error())
			return err
		}
		osMachine := machine.GetChromeosMachine()
		if osMachine == nil {
			logging.Warningf(ctx, "updateRecoveryMachineHelper %s is not a valid Chromeos machine", dutId)
			return nil
		}
		if err := updateRecoveryMachineHelper(ctx, machine, dutData); err != nil {
			logging.Errorf(ctx, "updateRecoveryDutData fail update machine(%s) - %s", dutId, err.Error())
			return err
		}
		asset, err := registration.GetAsset(ctx, dutId)
		if err != nil {
			logging.Errorf(ctx, "updateRecoveryAssetData Asset not found", dutId)
			return err
		}
		if err := updateRecoveryAssetHelper(ctx, asset, dutData); err != nil {
			logging.Errorf(ctx, "updateRecoveryDutData fail update asset(%s) - %s", dutId, err.Error())
			return err
		}
		return nil
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return errors.Annotate(err, "updatRecoveryeDutData (%s),  ", dutId).Err()
	}
	return nil
}

// updateRecoveryMachineHelper is a helper function to update machine
func updateRecoveryMachineHelper(ctx context.Context, machine *ufspb.Machine, dutData *ufsAPI.UpdateDeviceRecoveryDataRequest_DutData) error {
	hc := GetMachineHistoryClient(machine)
	// Copy for logging
	oldMachine := proto.Clone(machine).(*ufspb.Machine)

	if machine.GetSerialNumber() == dutData.GetSerialNumber() &&
		machine.GetChromeosMachine().GetHwid() == dutData.GetHwID() &&
		machine.GetChromeosMachine().GetSku() == dutData.GetDeviceSku() {
		logging.Warningf(ctx, "nothing to update: old serial number %q, old hwid %q, old device-sku %q", dutData.GetSerialNumber(), dutData.GetHwID(), dutData.GetDeviceSku())
		return nil
	}

	machine.SerialNumber = dutData.GetSerialNumber()
	machine.GetChromeosMachine().Hwid = dutData.GetHwID()
	machine.GetChromeosMachine().Sku = dutData.GetDeviceSku()
	if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
		return errors.Annotate(err, "Unable to update dut dutData for %s", machine.Name).Err()
	}
	hc.LogMachineChanges(oldMachine, machine)
	return hc.SaveChangeEvents(ctx)

}

// updateRecoveryAssetHelper is a helper function to update asset
func updateRecoveryAssetHelper(ctx context.Context, asset *ufspb.Asset, dutData *ufsAPI.UpdateDeviceRecoveryDataRequest_DutData) error {
	hc := &HistoryClient{}
	// Copy for logging
	oldAsset := proto.Clone(asset).(*ufspb.Asset)
	if asset.GetInfo() == nil {
		asset.Info = &ufspb.AssetInfo{}
	}
	if asset.GetInfo().GetSerialNumber() == dutData.GetSerialNumber() &&
		asset.GetInfo().GetHwid() == dutData.GetHwID() &&
		asset.GetInfo().GetSku() == dutData.GetDeviceSku() {
		logging.Warningf(ctx, "nothing to update: old serial number %q, old hwid %q, old device-sku %q", dutData.GetSerialNumber(), dutData.GetHwID(), dutData.GetDeviceSku())
		return nil
	}
	asset.GetInfo().SerialNumber = dutData.GetSerialNumber()
	asset.GetInfo().Hwid = dutData.GetHwID()
	asset.GetInfo().Sku = dutData.GetDeviceSku()
	// Update the asset
	if _, err := registration.BatchUpdateAssets(ctx, []*ufspb.Asset{asset}); err != nil {
		return errors.Annotate(err, "Unable to update dut asset for %s", asset.Name).Err()
	}
	hc.LogAssetChanges(oldAsset, asset)
	return hc.SaveChangeEvents(ctx)
}

// processMachineUpdateMask process update field mask to get only specific update
// fields and return a complete machine object with updated and existing fields
func processMachineUpdateMask(ctx context.Context, oldMachine, machine *ufspb.Machine, mask *field_mask.FieldMask, hc *HistoryClient) (*ufspb.Machine, error) {
	// If we are updating zone. We need to reset all the fields in the Location
	if util.ContainsAnyStrings(mask.Paths, "zone") && oldMachine.GetLocation().GetZone() != machine.GetLocation().GetZone() {
		oldMachine.Location = &ufspb.Location{}
	}
	// update the fields in the existing nic
	for _, path := range mask.Paths {
		switch path {
		case "zone":
			if machine.GetLocation().GetZone().String() == oldMachine.GetLocation().GetZone().String() {
				// If the zone is not updated, then don't do anything.
				continue
			}
			indexMap := map[string]string{"zone": machine.GetLocation().GetZone().String()}
			oldIndexMap := map[string]string{"zone": oldMachine.GetLocation().GetZone().String()}
			if err := updateIndexingForMachineResources(ctx, oldMachine, indexMap, oldIndexMap, hc); err != nil {
				return oldMachine, errors.Annotate(err, "processMachineUpdateMask - failed to update zone indexing").Err()
			}
			if oldMachine.GetLocation() == nil {
				oldMachine.Location = &ufspb.Location{}
			}
			oldMachine.GetLocation().Zone = machine.GetLocation().GetZone()
			oldMachine.Realm = machine.GetRealm()
		case "rack":
			if machine.GetLocation().GetRack() == oldMachine.GetLocation().GetRack() {
				continue
			}
			indexMap := map[string]string{"rack": machine.GetLocation().GetRack()}
			oldIndexMap := map[string]string{"rack": oldMachine.GetLocation().GetRack()}
			if err := updateIndexingForMachineResources(ctx, oldMachine, indexMap, oldIndexMap, hc); err != nil {
				return oldMachine, errors.Annotate(err, "processMachineUpdateMask - failed to update rack indexing").Err()
			}
			if oldMachine.GetLocation() == nil {
				oldMachine.Location = &ufspb.Location{}
			}
			oldMachine.GetLocation().Rack = machine.GetLocation().GetRack()
		case "platform":
			if oldMachine.GetChromeBrowserMachine() == nil {
				oldMachine.Device = &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				}
			}
			oldMachine.GetChromeBrowserMachine().ChromePlatform = machine.GetChromeBrowserMachine().GetChromePlatform()
		case "kvm":
			if oldMachine.GetChromeBrowserMachine() == nil {
				oldMachine.Device = &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				}
			}
			if oldMachine.GetChromeBrowserMachine().GetKvmInterface() == nil {
				oldMachine.GetChromeBrowserMachine().KvmInterface = &ufspb.KVMInterface{}
			}
			oldMachine.GetChromeBrowserMachine().GetKvmInterface().Kvm = machine.GetChromeBrowserMachine().GetKvmInterface().GetKvm()
		case "kvmport":
			if oldMachine.GetChromeBrowserMachine() == nil {
				oldMachine.Device = &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				}
			}
			if oldMachine.GetChromeBrowserMachine().GetKvmInterface() == nil {
				oldMachine.GetChromeBrowserMachine().KvmInterface = &ufspb.KVMInterface{}
			}
			oldMachine.GetChromeBrowserMachine().GetKvmInterface().PortName = machine.GetChromeBrowserMachine().GetKvmInterface().GetPortName()
		case "deploymentTicket":
			if oldMachine.GetChromeBrowserMachine() == nil {
				oldMachine.Device = &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				}
			}
			oldMachine.GetChromeBrowserMachine().DeploymentTicket = machine.GetChromeBrowserMachine().GetDeploymentTicket()
		case "tags":
			oldMachine.Tags = mergeTags(oldMachine.GetTags(), machine.GetTags())
		case "serialNumber":
			oldMachine.SerialNumber = machine.GetSerialNumber()
		case "resourceState":
			oldMachine.ResourceState = machine.GetResourceState()
		case "description":
			oldMachine.GetChromeBrowserMachine().Description = machine.GetChromeBrowserMachine().GetDescription()
		case "admManufacturer":
			oldMachine.GetAttachedDevice().Manufacturer = machine.GetAttachedDevice().GetManufacturer()
		case "admDeviceType":
			oldMachine.GetAttachedDevice().DeviceType = machine.GetAttachedDevice().GetDeviceType()
		case "admBuildTarget":
			oldMachine.GetAttachedDevice().BuildTarget = machine.GetAttachedDevice().GetBuildTarget()
		case "admModel":
			oldMachine.GetAttachedDevice().Model = machine.GetAttachedDevice().GetModel()
		}
	}
	// For partial update, validate kvm interface just before updating in case
	// before we checks the incompleted interface
	if err := validateKVMPort(ctx, oldMachine.GetName(), oldMachine.GetChromeBrowserMachine().GetKvmInterface()); err != nil {
		return oldMachine, err
	}
	// return existing/old machine with new updated values
	return oldMachine, nil
}

// updateIndexingForMachineResources updates indexing for machinelse/nic/drac tables
// can be used inside a transaction
func updateIndexingForMachineResources(ctx context.Context, oldMachine *ufspb.Machine, indexMap map[string]string, oldIndexMap map[string]string, hc *HistoryClient) error {
	// get MachineLSEs for indexing
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", oldMachine.GetName(), false)
	if err != nil {
		return errors.Annotate(err, "updateIndexing - failed to query machinelses/hosts for machine %s", oldMachine.GetName()).Err()
	}
	var nics []*ufspb.Nic
	var dracs []*ufspb.Drac
	// update indexing for nic/drac table only for chrome browser machines.
	if oldMachine.GetChromeBrowserMachine() != nil {
		var err error
		// get Nics for indexing
		nics, err = registration.QueryNicByPropertyName(ctx, "machine", oldMachine.GetName(), false)
		if err != nil {
			return errors.Annotate(err, "updateIndexing - failed to query nics for machine %s", oldMachine.GetName()).Err()
		}
		// get Dracs for indexing
		dracs, err = registration.QueryDracByPropertyName(ctx, "machine", oldMachine.GetName(), false)
		if err != nil {
			return errors.Annotate(err, "updateIndexing - failed to query dracs for machine %s", oldMachine.GetName()).Err()
		}
	}
	var vms []*ufspb.VM
	if len(machinelses) > 0 {
		// get VMs for indexing
		vms, err = inventory.QueryVMByPropertyName(ctx, "host_id", machinelses[0].GetName(), false)
		if err != nil {
			return errors.Annotate(err, "updateIndexing - failed to query vms for host %s", machinelses[0].GetName()).Err()
		}
	}

	for k, v := range indexMap {
		// These are output only fields used for indexing machinelse/drac/nic table
		switch k {
		case "rack":
			for _, machinelse := range machinelses {
				machinelse.Rack = v
			}
			for _, nic := range nics {
				nic.Rack = v
			}
			for _, drac := range dracs {
				drac.Rack = v
			}
		case "zone":
			for _, machinelse := range machinelses {
				machinelse.Zone = v
			}
			for _, nic := range nics {
				nic.Zone = v
			}
			for _, drac := range dracs {
				drac.Zone = v
			}
			for _, vm := range vms {
				vm.Zone = v
			}
		}
	}

	if _, err := inventory.BatchUpdateMachineLSEs(ctx, machinelses); err != nil {
		return errors.Annotate(err, "updateIndexing - unable to batch update machinelses").Err()
	}
	// update indexing for nic/drac table only for chrome browser machines.
	if oldMachine.GetChromeBrowserMachine() != nil {
		if _, err := registration.BatchUpdateNics(ctx, nics); err != nil {
			return errors.Annotate(err, "updateIndexing - unable to batch update nics").Err()
		}
		if _, err := registration.BatchUpdateDracs(ctx, dracs); err != nil {
			return errors.Annotate(err, "updateIndexing - unable to batch update dracs").Err()
		}
	}
	if _, err := inventory.BatchUpdateVMs(ctx, vms); err != nil {
		return errors.Annotate(err, "updateIndexing - unable to batch update vms").Err()
	}
	hc.LogMachineLocationChanges(machinelses, nics, dracs, vms, indexMap, oldIndexMap)
	return nil
}

// GetMachine returns machine for the given id from datastore.
func GetMachine(ctx context.Context, id string) (*ufspb.Machine, error) {
	machine, err := registration.GetMachine(ctx, id)
	if err != nil {
		return nil, err
	}
	// Nics or Drac info not associated with CrOS machines, yet.
	if machine.GetChromeBrowserMachine() != nil {
		setMachine(ctx, machine)
	}
	return machine, nil
}

// BatchGetMachines returns a batch of machines from datastore.
func BatchGetMachines(ctx context.Context, ids []string) ([]*ufspb.Machine, error) {
	return registration.BatchGetMachines(ctx, ids)
}

// ListMachines lists the machines
func ListMachines(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly, full bool) ([]*ufspb.Machine, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, registration.GetMachineIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "ListMachines - failed to read filter for listing machines").Err()
		}
	}
	filterMap = resetStateFilter(filterMap)
	filterMap = resetZoneFilter(filterMap)
	filterMap = resetDeviceTypeFilter(filterMap)
	machines, nextPageToken, err := registration.ListMachines(ctx, pageSize, pageToken, filterMap, keysOnly)
	if full && !keysOnly {
		for _, machine := range machines {
			// Nics or Drac info not associated with CrOS machines, yet.
			if machine.GetChromeBrowserMachine() != nil {
				setMachine(ctx, machine)
			}
		}
	}
	return machines, nextPageToken, err
}

// GetAllMachines returns all machines in datastore.
func GetAllMachines(ctx context.Context) (*ufsds.OpResults, error) {
	return registration.GetAllMachines(ctx)
}

// DeleteMachine deletes the machine and its associated nics and drac in datastore
//
// For referential data intergrity,
// Delete if this Machine is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteMachine(ctx context.Context, id string) error {
	f := func(ctx context.Context) error {
		// 1. Get the machine
		machine, err := registration.GetMachine(ctx, id)
		if status.Code(err) == codes.Internal {
			return errors.Annotate(err, "failed to get machine %s", id).Err()
		}
		if machine == nil {
			return status.Errorf(codes.NotFound, ufsds.NotFound)
		}

		// 2. Check if any other resource references this machine.
		if err = validateDeleteMachine(ctx, machine); err != nil {
			return errors.Annotate(err, "validation failed").Err()
		}

		//Only for a browser machine
		var nicIDs []string
		var dracID string
		if machine.GetChromeBrowserMachine() != nil {
			nicIDs, err = getDeleteNicIDs(ctx, machine.GetName())
			if err != nil {
				return err
			}
			dracID, err = getDeleteDracID(ctx, machine.GetName())
			if err != nil {
				return err
			}
			for _, nicID := range nicIDs {
				if err := deleteNicHelper(ctx, nicID, false); err != nil {
					return errors.Annotate(err, "failed to delete associated nic %s", nicID).Err()
				}
			}
			if dracID != "" {
				if err := deleteDracHelper(ctx, dracID, false); err != nil {
					return errors.Annotate(err, "failed to delete associated drac %s", dracID).Err()
				}
			}
		}

		if err := registration.DeleteMachine(ctx, id); err != nil {
			return err
		}
		hc := GetMachineHistoryClient(&ufspb.Machine{Name: id})
		hc.stUdt.deleteStateHelper(ctx)
		hc.LogMachineChanges(&ufspb.Machine{Name: id}, nil)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return errors.Annotate(err, "DeleteMachine").Err()
	}
	return nil
}

// ImportMachines creates or updates a batch of machines in datastore
func ImportMachines(ctx context.Context, machines []*ufspb.Machine, pageSize int) (*ufsds.OpResults, error) {
	deleteNonExistingMachines(ctx, machines, pageSize)
	allRes := make(ufsds.OpResults, 0)
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(machines))
		res, err := registration.ImportMachines(ctx, machines[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(machines) {
			break
		}
	}
	return &allRes, nil
}

func deleteNonExistingMachines(ctx context.Context, machines []*ufspb.Machine, pageSize int) (*ufsds.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range machines {
		resMap[r.GetName()] = true
	}
	resp, err := registration.GetAllMachines(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.Machine)
		if s.GetChromeBrowserMachine() != nil {
			if _, ok := resMap[s.GetName()]; !ok {
				toDelete = append(toDelete, s.GetName())
			}
		}
	}
	logging.Infof(ctx, "Deleting %d non-existing machines", len(toDelete))
	return deleteByPage(ctx, toDelete, pageSize, registration.DeleteMachines), nil
}

// RenameMachine renames the machine and updates the associated nics, drac and machinelse in datastore
func RenameMachine(ctx context.Context, oldMachineName, newMachineName string) (machine *ufspb.Machine, err error) {
	f := func(ctx context.Context) error {
		machine, err = renameMachineInner(ctx, oldMachineName, newMachineName)
		return err
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "RenameMachine - failed to rename machine in datastore").Err()
	}
	// Nics or Drac info not associated with CrOS machines, yet.
	if machine.GetChromeBrowserMachine() != nil {
		setMachine(ctx, machine)
	}
	return machine, nil
}

// renameMachineInner renames the machine to the given name. Use inside a transaction
func renameMachineInner(ctx context.Context, oldMachineName, newMachineName string) (machine *ufspb.Machine, err error) {
	// Get the old machine
	machine, err = registration.GetMachine(ctx, oldMachineName)
	if err != nil {
		return
	}

	// Validate
	if err = validateRenameMachine(ctx, machine, newMachineName); err != nil {
		return
	}

	// Copy for logging
	oldMachineCopy := proto.Clone(machine).(*ufspb.Machine)
	hc := GetMachineHistoryClient(oldMachineCopy)

	if err = renameMachineHelper(ctx, oldMachineName, newMachineName, hc); err != nil {
		return
	}

	// Delete old machine
	err = registration.DeleteMachine(ctx, oldMachineName)
	if err != nil {
		return
	}

	// Create new machine
	machine.Name = newMachineName
	if machine.GetChromeBrowserMachine() != nil {
		machine.GetChromeBrowserMachine().DisplayName = newMachineName
	}
	_, err = registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine})
	if err != nil {
		return
	}

	// Log history change events
	hc.LogMachineChanges(oldMachineCopy, machine)
	err = hc.SaveChangeEvents(ctx)
	if err != nil {
		return
	}
	newHc := GetMachineHistoryClient(machine)
	newHc.stUdt.replaceStateHelper(ctx, util.AddPrefix(util.MachineCollection, oldMachineName))
	err = newHc.SaveChangeEvents(ctx)
	return
}

func renameMachineHelper(ctx context.Context, oldMachineName, newMachineName string, hc *HistoryClient) error {
	// Rename the Nic(s) for the machine
	newNics := make([]*ufspb.Nic, 0, 0)
	// Nic have machine name in their "Name/ID"
	// Also Update the machine name in all the nics
	nics, err := registration.QueryNicByPropertyName(ctx, "machine", oldMachineName, false)
	if err != nil {
		return errors.Annotate(err, "failed to query nics for machine %s", oldMachineName).Err()
	}
	for _, nic := range nics {
		// Copy for logging
		oldNicCopy := proto.Clone(nic).(*ufspb.Nic)
		if strings.HasPrefix(nic.GetName(), oldMachineName+":") {
			// Delete the nic
			if err := registration.DeleteNic(ctx, nic.GetName()); err != nil {
				return errors.Annotate(err, "unable to delete nic %s", nic.GetName()).Err()
			}
			nic.Name = util.GetNewNicNameForRenameMachine(nic.GetName(), oldMachineName, newMachineName)
		}
		nic.Machine = newMachineName
		hc.LogNicChanges(oldNicCopy, nic)
		newNics = append(newNics, nic)
	}
	// Create/Update the Nic(s) for the machine with new name
	if _, err = registration.BatchUpdateNics(ctx, newNics); err != nil {
		return errors.Annotate(err, "unable to batch update nics").Err()
	}

	// Update the MachineLSE with new machine name and new nic name
	if err := updateIndexingForMachineLSE(ctx, "machine", oldMachineName, newMachineName, hc); err != nil {
		return errors.Annotate(err, "failed to update indexing for hosts").Err()
	}

	// Update Drac with new Machine name
	if err := updateIndexingForDrac(ctx, "machine", oldMachineName, newMachineName, hc); err != nil {
		return errors.Annotate(err, "failed to update indexing for dracs").Err()
	}
	return nil
}

// ReplaceMachine replaces an old Machine with new Machine in datastore
//
// It does a delete of old machine and create of new Machine.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old Machine, it will get all the resources referencing
// the old Machine. It will update all the resources which were referencing
// the old Machine(got in the last step) with new Machine.
// Deletes the old Machine.
// Creates the new Machine.
// This will preserve data integrity in the system.
func ReplaceMachine(ctx context.Context, oldMachine *ufspb.Machine, newMachine *ufspb.Machine) (*ufspb.Machine, error) {
	f := func(ctx context.Context) error {
		hc := GetMachineHistoryClient(newMachine)
		machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", oldMachine.Name, false)
		if err != nil {
			return err
		}
		if machinelses != nil {
			for _, machinelse := range machinelses {
				machines := machinelse.GetMachines()
				for i := range machines {
					if machines[i] == oldMachine.Name {
						machines[i] = newMachine.Name
						break
					}
				}
				machinelse.Machines = machines
			}
			_, err := inventory.BatchUpdateMachineLSEs(ctx, machinelses)
			if err != nil {
				return err
			}
		}

		err = registration.DeleteMachine(ctx, oldMachine.Name)
		if err != nil {
			return err
		}

		err = validateMachineReferences(ctx, newMachine)
		if err != nil {
			return err
		}
		entity := &registration.MachineEntity{
			ID: newMachine.Name,
		}
		existsResults, err := datastore.Exists(ctx, entity)
		if err == nil {
			if existsResults.All() {
				return status.Errorf(codes.AlreadyExists, ufsds.AlreadyExists)
			}
		} else {
			logging.Errorf(ctx, "Failed to check existence: %s", err)
			return status.Errorf(codes.Internal, ufsds.InternalError)
		}

		_, err = registration.BatchUpdateMachines(ctx, []*ufspb.Machine{newMachine})
		if err != nil {
			return err
		}
		hc.LogMachineChanges(oldMachine, nil)
		hc.LogMachineChanges(nil, newMachine)
		hc.stUdt.replaceStateHelper(ctx, util.AddPrefix(util.MachineCollection, oldMachine.GetName()))
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to replace entity in datastore: %s", err)
		return nil, err
	}
	return newMachine, nil
}

// validateMachineReferences validates if the resources referenced by the machine
// are in the system.
//
// Checks if the resources referenced by the given Machine input already exists
// in the system. Returns an error if any resource referenced by the Machine input
// does not exist in the system.
func validateMachineReferences(ctx context.Context, machine *ufspb.Machine) error {
	var resources []*Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create Machine %s:\n", machine.Name))

	if kvmID := machine.GetChromeBrowserMachine().GetKvmInterface().GetKvm(); kvmID != "" {
		resources = append(resources, GetKVMResource(kvmID))
	}
	if rpmID := machine.GetChromeBrowserMachine().GetRpmInterface().GetRpm(); rpmID != "" {
		resources = append(resources, GetRPMResource(rpmID))
	}
	if chromePlatformID := machine.GetChromeBrowserMachine().GetChromePlatform(); chromePlatformID != "" {
		resources = append(resources, GetChromePlatformResource(chromePlatformID))
	}

	return ResourceExist(ctx, resources, &errorMsg)
}

// validateDeleteMachine validates if a Machine can be deleted
//
// Checks if this Machine(MachineID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteMachine(ctx context.Context, machine *ufspb.Machine) error {
	if err := util.CheckPermission(ctx, util.RegistrationsDelete, machine.GetRealm()); err != nil {
		return err
	}
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", machine.GetName(), true)
	if err != nil {
		return err
	}
	if len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Machine %s is occupied.", machine.GetName()))
		errorMsg.WriteString(fmt.Sprintf("\nHosts referring the Machine:\n"))
		for _, machinelse := range machinelses {
			errorMsg.WriteString(machinelse.Name + ", ")
		}
		errorMsg.WriteString(fmt.Sprintf("\nPlease delete the hosts first.\n"))
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

// getBrowserMachine gets the browser machine
func getBrowserMachine(ctx context.Context, machineName string) (*ufspb.Machine, error) {
	machine, err := registration.GetMachine(ctx, machineName)
	if err != nil {
		return nil, errors.Annotate(err, "getBrowserMachine - unable to get browser machine %s", machineName).Err()
	}
	if machine.GetChromeBrowserMachine() == nil {
		errorMsg := fmt.Sprintf("Machine %s is not a browser machine.", machineName)
		return nil, status.Errorf(codes.FailedPrecondition, errorMsg)
	}
	return machine, nil
}

// validateMachineRegistration validates if a machine/nics/drac can be created
func validateMachineRegistration(ctx context.Context, machine *ufspb.Machine) error {
	if err := util.CheckPermission(ctx, util.RegistrationsCreate, machine.GetRealm()); err != nil {
		return err
	}
	var resourcesNotFound []*Resource
	var resourcesAlreadyExists []*Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create machine %s:\n", machine.Name))
	var nics []*ufspb.Nic
	var drac *ufspb.Drac
	if machine.GetChromeBrowserMachine() != nil {
		// Only check serial number for browser machines. OS machines' serial number is auto-detected.
		if err := validateUniqueSerial(ctx, machine.GetSerialNumber()); err != nil {
			return err
		}
		nics = machine.GetChromeBrowserMachine().GetNicObjects()
		drac = machine.GetChromeBrowserMachine().GetDracObject()
	}

	// Validate that corresponding asset exists for chromeos machine.
	if machine.GetChromeosMachine() != nil {
		resourcesNotFound = append(resourcesNotFound, GetAssetResource(machine.GetName()))
	}

	// Aggregate resources to check if machine already exists
	resourcesAlreadyExists = append(resourcesAlreadyExists, GetMachineResource(machine.Name))
	switchToNicMap := make(map[string]string)
	for _, nic := range nics {
		// Aggregate resources to check if nic already exists
		resourcesAlreadyExists = append(resourcesAlreadyExists, GetNicResource(nic.Name))

		// Aggregate resources to check if resources referenced by the nic exists
		if switchID := nic.GetSwitchInterface().GetSwitch(); switchID != "" {
			resourcesNotFound = append(resourcesNotFound, GetSwitchResource(switchID))

			if switchPortName := nic.GetSwitchInterface().GetPortName(); switchPortName != "" {
				prevNicID, ok := switchToNicMap[switchID+switchPortName]
				if ok {
					return status.Errorf(codes.InvalidArgument, "nic %s and %s share the same switch %s and port %s", prevNicID, nic.GetName(), switchID, switchPortName)
				}
				switchToNicMap[switchID+switchPortName] = nic.GetName()
			}
		}

		if err := validateMacAddress(ctx, nic.GetName(), nic.GetMacAddress()); err != nil {
			return err
		}
		if err := validateNicSwitchPort(ctx, nic.GetName(), machine.Name, nic.GetSwitchInterface()); err != nil {
			return err
		}
	}
	if drac != nil {
		// Aggregate resources to check if drac already exists
		resourcesAlreadyExists = append(resourcesAlreadyExists, GetDracResource(drac.Name))

		// Aggregate resources to check if resources referenced by the drac exists
		if switchID := drac.GetSwitchInterface().GetSwitch(); switchID != "" {
			resourcesNotFound = append(resourcesNotFound, GetSwitchResource(switchID))
		}

		if err := validateDracSwitchPort(ctx, drac.GetName(), machine.Name, drac.GetSwitchInterface()); err != nil {
			return err
		}
		if err := validateMacAddress(ctx, drac.GetName(), drac.GetMacAddress()); err != nil {
			return err
		}
	}
	// Aggregate resources referenced by the machine to check if they do not exist
	if kvmID := machine.GetChromeBrowserMachine().GetKvmInterface().GetKvm(); kvmID != "" {
		resourcesNotFound = append(resourcesNotFound, GetKVMResource(kvmID))
		if err := validateKVMPort(ctx, machine.GetName(), machine.GetChromeBrowserMachine().GetKvmInterface()); err != nil {
			return err
		}
	}
	if rpmID := machine.GetChromeBrowserMachine().GetRpmInterface().GetRpm(); rpmID != "" {
		resourcesNotFound = append(resourcesNotFound, GetRPMResource(rpmID))
	}
	if chromePlatformID := machine.GetChromeBrowserMachine().GetChromePlatform(); chromePlatformID != "" {
		resourcesNotFound = append(resourcesNotFound, GetChromePlatformResource(chromePlatformID))
	}
	if rackID := machine.GetLocation().GetRack(); rackID != "" {
		resourcesNotFound = append(resourcesNotFound, GetRackResource(rackID))
	}
	// Check if machine/nics/drac already exists
	if err := resourceAlreadyExists(ctx, resourcesAlreadyExists, nil); err != nil {
		return err
	}
	// check if resources does not exist
	return ResourceExist(ctx, resourcesNotFound, &errorMsg)
}

// validateUpdateMachine validates if a machine can be updated
//
// checks if the machine and the resources referenced  by the machine
// does not exist in the system.
func validateUpdateMachine(ctx context.Context, oldMachine *ufspb.Machine, machine *ufspb.Machine, mask *field_mask.FieldMask) error {
	if err := util.CheckPermission(ctx, util.RegistrationsUpdate, oldMachine.GetRealm()); err != nil {
		return err
	}
	if machine.GetRealm() != "" && oldMachine.GetRealm() != machine.GetRealm() {
		if err := util.CheckPermission(ctx, util.RegistrationsUpdate, machine.GetRealm()); err != nil {
			return err
		}
	}
	// Only check serial number for browser machines. OS machines' serial number is auto-detected.
	if machine.GetChromeBrowserMachine() != nil && oldMachine.GetSerialNumber() != machine.GetSerialNumber() {
		if err := validateUniqueSerial(ctx, machine.GetSerialNumber()); err != nil {
			return err
		}
	}

	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot update machine %s:\n", machine.Name))
	var resourcesNotFound []*Resource
	// Aggregate resources referenced by the machine to check if they do not exist
	if kvmID := machine.GetChromeBrowserMachine().GetKvmInterface().GetKvm(); kvmID != "" {
		resourcesNotFound = append(resourcesNotFound, GetKVMResource(kvmID))
	}
	if rpmID := machine.GetChromeBrowserMachine().GetRpmInterface().GetRpm(); rpmID != "" {
		resourcesNotFound = append(resourcesNotFound, GetRPMResource(rpmID))
	}
	if chromePlatformID := machine.GetChromeBrowserMachine().GetChromePlatform(); chromePlatformID != "" {
		resourcesNotFound = append(resourcesNotFound, GetChromePlatformResource(chromePlatformID))
	}

	// check if resources does not exist
	if err := ResourceExist(ctx, resourcesNotFound, &errorMsg); err != nil {
		return err
	}
	// Validate partial update first to avoid unnecessary validations
	if err := validateMachineUpdateMask(machine, mask); err != nil {
		return err
	}
	if err := validateKVMPort(ctx, machine.GetName(), machine.GetChromeBrowserMachine().GetKvmInterface()); err != nil {
		return err
	}
	return nil
}

// validateMachineUpdateMask validates the update mask for machine update
func validateMachineUpdateMask(machine *ufspb.Machine, mask *field_mask.FieldMask) error {
	if mask != nil {
		// validate the give field mask
		for _, path := range mask.Paths {
			switch path {
			case "name":
				return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - name cannot be updated, delete and create a new machine instead")
			case "update_time":
				return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - update_time cannot be updated, it is a output only field")
			case "zone":
				if machine.GetLocation() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - location cannot be empty/nil.")
				} else if machine.GetLocation().GetZone() == ufspb.Zone_ZONE_UNSPECIFIED {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - zone cannot be unspecified")
				} else if machine.GetLocation().GetRack() == "" {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - Cannot update zone without updating rack")
				}
			case "rack":
				if machine.GetLocation() == nil || machine.GetLocation().GetRack() == "" {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - location/rack cannot be empty/nil.")
				}
			case "platform":
				if machine.GetChromeBrowserMachine() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - browser machine cannot be empty/nil.")
				}
			case "kvm":
				fallthrough
			case "kvmport":
				// Check kvm interface validity in processMachineUpdateMask later.
				if machine.GetChromeBrowserMachine() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - browser machine cannot be empty/nil.")
				}
				if machine.GetChromeBrowserMachine().GetKvmInterface() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - kvm interface cannot be empty/nil.")
				}
			case "deploymentTicket":
				if machine.GetChromeBrowserMachine() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - browser machine cannot be empty/nil.")
				}
			case "description":
				if machine.GetChromeBrowserMachine() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - browser machine cannot be empty/nil.")
				}
			case "admManufacturer":
				if machine.GetAttachedDevice() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - attached device machine cannot be empty/nil.")
				}
			case "admDeviceType":
				if machine.GetAttachedDevice() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - attached device machine cannot be empty/nil.")
				}
			case "admBuildTarget":
				if machine.GetAttachedDevice() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - attached device machine cannot be empty/nil.")
				}
			case "admModel":
				if machine.GetAttachedDevice() == nil {
					return status.Error(codes.InvalidArgument, "validateMachineUpdateMask - attached device machine cannot be empty/nil.")
				}
			case "tags":
			case "serialNumber":
			case "resourceState":
				// valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "validateMachineUpdateMask - unsupported update mask path %q", path)
			}
		}
	}
	return nil
}

// validateRenameMachine validates if a machine can be renamed
func validateRenameMachine(ctx context.Context, oldMachine *ufspb.Machine, newMachineName string) error {
	// Check permission
	if err := util.CheckPermission(ctx, util.RegistrationsUpdate, oldMachine.GetRealm()); err != nil {
		return err
	}
	// Check if new machine name already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetMachineResource(newMachineName)}, nil); err != nil {
		return err
	}
	return nil
}

func GetMachineHistoryClient(m *ufspb.Machine) *HistoryClient {
	return &HistoryClient{
		stUdt: &stateUpdater{
			ResourceName: util.AddPrefix(util.MachineCollection, m.Name),
		},
		netUdt: &networkUpdater{
			Hostname: m.Name,
		},
	}
}

//setMachine populates the machine object with nics and drac
func setMachine(ctx context.Context, machine *ufspb.Machine) {
	// get Nics for machine
	nics, err := registration.QueryNicByPropertyName(ctx, "machine", machine.GetName(), false)
	if err != nil {
		// Just log a warning message and dont fail operation
		logging.Warningf(ctx, "GetMachine - failed to query nics for machine %s: %s", machine.GetName(), err)
	}
	setNicsToMachine(machine, nics)

	// get Dracs for machine
	dracs, err := registration.QueryDracByPropertyName(ctx, "machine", machine.GetName(), false)
	if err != nil {
		// Just log a warning message and dont fail operation
		logging.Warningf(ctx, "GetMachine - failed to query dracs for machine %s: %s", machine.GetName(), err)
	}
	if dracs != nil && len(dracs) > 0 {
		setDracToMachine(machine, dracs[0])
	}
}

func setNicsToMachine(machine *ufspb.Machine, nics []*ufspb.Nic) {
	if len(nics) <= 0 {
		return
	}
	if machine.GetChromeBrowserMachine() == nil {
		machine.Device = &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
				NicObjects: nics,
			},
		}
	} else {
		machine.GetChromeBrowserMachine().NicObjects = nics
	}
}

func setDracToMachine(machine *ufspb.Machine, drac *ufspb.Drac) {
	if drac == nil {
		return
	}
	if machine.GetChromeBrowserMachine() == nil {
		machine.Device = &ufspb.Machine_ChromeBrowserMachine{
			ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{
				DracObject: drac,
			},
		}
	} else {
		machine.GetChromeBrowserMachine().DracObject = drac
	}
}

// can be called inside a transaction
func getDeleteNicIDs(ctx context.Context, machineName string) ([]string, error) {
	nics, err := registration.QueryNicByPropertyName(ctx, "machine", machineName, true)
	if err != nil {
		return nil, errors.Annotate(err, "failed to query nics for machine %s", machineName).Err()
	}
	nicIDs := make([]string, 0, len(nics))
	for _, nic := range nics {
		nicIDs = append(nicIDs, nic.GetName())
	}
	return nicIDs, nil
}

// can be called inside a transaction
func getDeleteDracID(ctx context.Context, machineName string) (string, error) {
	dracs, err := registration.QueryDracByPropertyName(ctx, "machine", machineName, true)
	if err != nil {
		return "", errors.Annotate(err, "failed to query dracs for machine %s", machineName).Err()
	}
	if len(dracs) > 0 {
		return dracs[0].GetName(), nil
	}
	return "", nil
}

func validateUniqueSerial(ctx context.Context, serialNumber string) error {
	if serialNumber == "" {
		return nil
	}
	if serialNumber != "" {
		res, _, err := registration.ListMachines(ctx, 1, "", map[string][]interface{}{
			"serial_number": {serialNumber},
		}, true)
		if err != nil {
			return err
		}
		if len(res) > 0 {
			return fmt.Errorf("machine %q contains the same serial number %q", res[0].Name, serialNumber)
		}
	}
	return nil
}
