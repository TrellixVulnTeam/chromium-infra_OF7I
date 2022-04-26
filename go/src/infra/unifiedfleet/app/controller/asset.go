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
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

// AssetRegistration registers the given asset to the datastore after validation
func AssetRegistration(ctx context.Context, asset *ufspb.Asset) (*ufspb.Asset, error) {

	hc := &HistoryClient{}
	f := func(ctx context.Context) error {
		if err := validateAssetRegistration(ctx, asset); err != nil {
			return err
		}
		if asset.GetType() == ufspb.AssetType_DUT || asset.GetType() == ufspb.AssetType_LABSTATION {
			//Create a new machine
			if err := addMachineHelper(ctx, asset); err != nil {
				return err
			}
		}
		_, err := registration.BatchUpdateAssets(ctx, []*ufspb.Asset{asset})
		if err != nil {
			return err
		}
		hc.LogAssetChanges(nil, asset)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "AddAsset - unable to update asset %s", asset.GetName()).Err()
	}
	if asset.GetType() == ufspb.AssetType_DUT {
		// On successful update to datastore, make an asset info request.
		herr := util.PublishHaRTAssetInfoRequest(ctx, []string{asset.GetName()})
		if herr != nil {
			// Don't return this error. Cron job will eventually get to update this.
			logging.Warningf(ctx, "AssetRegistration - Faild to publish request for asset info to HaRT. %v", herr)
		}
	}
	return asset, nil
}

// UpdateAsset updates the asset record to the datastore after validation
func UpdateAsset(ctx context.Context, asset *ufspb.Asset, mask *field_mask.FieldMask) (*ufspb.Asset, error) {
	var oldAsset *ufspb.Asset
	var err error
	hc := &HistoryClient{}
	f := func(ctx context.Context) error {
		// TODO(anshruth): Support validation of DUT/Labstation/Servo
		// created using this asset. And update them accordingly or fail.
		oldAsset, err = registration.GetAsset(ctx, asset.GetName())
		if err != nil {
			return err
		}

		err := validateUpdateAsset(ctx, oldAsset, asset, mask)
		if err != nil {
			return err
		}

		// Copy OUTPUT_ONLY fields
		if asset.GetInfo() == nil {
			asset.Info = &ufspb.AssetInfo{}
		}
		asset.GetInfo().SerialNumber = oldAsset.GetInfo().GetSerialNumber()

		// Allow users to modify hwid before we can get authoritative source from HaRT
		// Don't allow users to modify it to empty
		if asset.GetInfo().GetHwid() == "" {
			asset.GetInfo().Hwid = oldAsset.GetInfo().GetHwid()
		}
		asset.GetInfo().Sku = oldAsset.GetInfo().GetSku()

		// updatableAsset will be used to update the asset
		updatableAsset := asset
		if mask != nil && mask.Paths != nil {
			// Construct updatableAsset from mask if given
			updatableAsset = proto.Clone(proto.MessageV1(oldAsset)).(*ufspb.Asset)
			updatableAsset, err = processAssetUpdateMask(asset, updatableAsset, mask)
			if err != nil {
				return err
			}
		}
		a, err := registration.BatchUpdateAssets(ctx, []*ufspb.Asset{updatableAsset})
		if err != nil {
			return err
		}

		// Update the associated Machine from the updated asset
		if err := updateMachineHelper(ctx, a[0]); err != nil {
			return err
		}
		// Return the updated asset
		asset = a[0]
		hc.LogAssetChanges(oldAsset, updatableAsset)
		return hc.SaveChangeEvents(ctx)
	}
	if err = datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "UpdateAsset - unable to update asset %s", asset.GetName()).Err()
	}
	return asset, err
}

// GetAsset returns asset for the given name from datastore
func GetAsset(ctx context.Context, name string) (*ufspb.Asset, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "GetAsset - missing asset name")
	}
	asset, err := registration.GetAsset(ctx, name)
	if err != nil {
		return nil, errors.Annotate(err, "GetAsset - unable to get asset %s", name).Err()
	}
	return asset, nil
}

// ListAssets lists the assets
func ListAssets(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.Asset, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, registration.GetAssetIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "ListAssets - failed to read filter for listing assets").Err()
		}
	}
	filterMap = resetZoneFilter(filterMap)
	filterMap = resetAssetTypeFilter(filterMap)
	return registration.ListAssets(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteAsset deletes an asset from UFS
func DeleteAsset(ctx context.Context, name string) error {
	hc := &HistoryClient{}
	f := func(ctx context.Context) error {
		asset, err := registration.GetAsset(ctx, name)
		if err != nil {
			return errors.Annotate(err, "DeleteAsset - cannot find asset %s", name).Err()
		}
		if err := validateDeleteAsset(ctx, asset); err != nil {
			return errors.Annotate(err, "DeleteAsset - failed to delete %s", name).Err()
		}
		// delete associated machine
		if err := deleteMachineHelper(ctx, asset.GetName()); err != nil {
			return errors.Annotate(err, "DeleteAsset - failed to delete associated machine %s", asset.GetName()).Err()
		}
		// delete the asset
		err = registration.DeleteAsset(ctx, name)
		if err != nil {
			return errors.Annotate(err, "DeleteAsset - failed to delete %s", name).Err()
		}
		hc.LogAssetChanges(asset, nil)
		err = hc.SaveChangeEvents(ctx)
		if err != nil {
			return errors.Annotate(err, "DeleteAsset- unable to record delete history").Err()
		}
		return nil
	}
	return datastore.RunInTransaction(ctx, f, nil)
}

// RenameAsset renames a given asset (and corresponding machine if available) with new name.
func RenameAsset(ctx context.Context, oldName, newName string) (res *ufspb.Asset, err error) {
	f := func(ctx context.Context) error {
		asset, err := registration.GetAsset(ctx, oldName)
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("RenameAsset - %s not found", oldName))
		}
		// Validate if we can rename the asset
		if err := validateRenameAsset(ctx, asset, newName); err != nil {
			return err
		}
		oldAsset := proto.Clone(asset).(*ufspb.Asset)
		hc := &HistoryClient{}
		// Delete the asset.
		if err := registration.DeleteAsset(ctx, oldName); err != nil {
			return errors.Annotate(err, "RenameAsset - unable to delete asset %s", oldName).Err()
		}
		// Rename the asset to the new name given.
		asset.Name = newName
		if info := asset.GetInfo(); info != nil {
			asset.Info.AssetTag = newName
		}
		// Write the renamed asset to DB.
		if _, err := registration.BatchUpdateAssets(ctx, []*ufspb.Asset{asset}); err != nil {
			return err
		}
		// Rename the machine if it's a DUT or a Labstation.
		if asset.GetType() == ufspb.AssetType_DUT || asset.GetType() == ufspb.AssetType_LABSTATION {
			if _, err := renameMachineInner(ctx, oldName, newName); err != nil {
				return errors.Annotate(err, "RenameAsset [%s -> %s] - unable to rename machine", oldName, newName).Err()
			}
		}
		hc.LogAssetChanges(oldAsset, asset)
		if err := hc.SaveChangeEvents(ctx); err != nil {
			return errors.Annotate(err, "RenameAsset - unable to save changes to asset").Err()
		}
		// Return the asset back to the caller
		res = asset
		return nil
	}
	return res, datastore.RunInTransaction(ctx, f, nil)
}

// addMachineHelper adds a machine for the newly added asset.
//
// asset should be a DUT or Labstation.
// This should be run inside a transaction.
func addMachineHelper(ctx context.Context, asset *ufspb.Asset) error {
	// Check if machine already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetMachineResource(asset.Name)}, nil); err != nil {
		return err
	}
	machine := CreateMachineFromAsset(asset)
	hc := GetMachineHistoryClient(machine)
	if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
		return errors.Annotate(err, "unable to create machine").Err()
	}
	hc.LogMachineChanges(nil, machine)
	hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_REGISTERED)
	return hc.SaveChangeEvents(ctx)
}

// updateMachineHelper updates a machine for the updated asset.
//
// This should be run inside a transaction.
func updateMachineHelper(ctx context.Context, asset *ufspb.Asset) error {
	// Get the existing machine.
	machine, err := registration.GetMachine(ctx, asset.GetName())
	if err != nil {
		if util.IsNotFoundError(err) {
			// Create a new machine if the updated asset is a
			// DUT or Labstation.
			if asset.GetType() == ufspb.AssetType_DUT || asset.GetType() == ufspb.AssetType_LABSTATION {
				return addMachineHelper(ctx, asset)
			}
			// Nothing to do if its a servo type.
			// No machine is created for a servo asset.
			return nil
		}
		return err
	}
	// If the machine exists and the updated asset is a servo
	// then delete the associated machine.
	if asset.GetType() == ufspb.AssetType_SERVO {
		if err := validateDeleteAsset(ctx, asset); err != nil {
			return errors.Annotate(err, "failed to update %s to SERVO type as there is a DUT associated with this asset.", asset.GetName()).Err()
		}
		return deleteMachineHelper(ctx, asset.GetName())
	}
	// Copy for logging
	oldMachineCopy := proto.Clone(machine).(*ufspb.Machine)
	hc := GetMachineHistoryClient(machine)
	//update the machine from the asset
	if err := updateMachineFromAsset(ctx, machine, asset, hc); err != nil {
		return err
	}
	if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
		return errors.Annotate(err, "unable to create machine").Err()
	}
	hc.LogMachineChanges(oldMachineCopy, machine)
	return hc.SaveChangeEvents(ctx)
}

// deleteMachineHelper deletes a machine. If the machine is not found it return nil.
// This should be run inside a transaction.
func deleteMachineHelper(ctx context.Context, id string) error {
	hc := GetMachineHistoryClient(&ufspb.Machine{Name: id})
	machine, err := registration.GetMachine(ctx, id)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return err
	}
	// delete the machine
	if err := registration.DeleteMachine(ctx, id); err != nil {
		return err
	}
	hc.stUdt.deleteStateHelper(ctx)
	hc.LogMachineChanges(machine, nil)
	return hc.SaveChangeEvents(ctx)
}

func validateUpdateAsset(ctx context.Context, oldAsset *ufspb.Asset, asset *ufspb.Asset, mask *field_mask.FieldMask) error {
	if err := util.CheckPermission(ctx, util.RegistrationsUpdate, oldAsset.GetRealm()); err != nil {
		return err
	}
	if asset.GetRealm() != "" && oldAsset.GetRealm() != asset.GetRealm() {
		if err := util.CheckPermission(ctx, util.RegistrationsUpdate, asset.GetRealm()); err != nil {
			return err
		}
	}
	if mask == nil || mask.Paths == nil {
		// If mask doesn't exist then validate the given asset
		return validateAsset(ctx, asset)
	}
	// Validate AssetUpdate Mask if it exists
	return validateAssetUpdateMask(ctx, asset, mask)
}

func validateAssetRegistration(ctx context.Context, asset *ufspb.Asset) error {
	if err := util.CheckPermission(ctx, util.RegistrationsCreate, asset.GetRealm()); err != nil {
		return err
	}
	if err := validateAsset(ctx, asset); err != nil {
		return err
	}
	var errMsg strings.Builder
	errMsg.WriteString("validateAsset - ")
	if err := ResourceExist(ctx, []*Resource{GetAssetResource(asset.GetName())}, &errMsg); err == nil {
		return status.Errorf(codes.FailedPrecondition, "validateAssetRegistration - Asset %s exists, cannot create another", asset.GetName())
	}
	return nil
}

func validateAsset(ctx context.Context, asset *ufspb.Asset) error {
	if asset.GetName() == "" {
		return status.Error(codes.InvalidArgument, "validateAsset - Missing name")
	}
	if asset.GetLocation() == nil {
		return status.Error(codes.InvalidArgument, "validateAsset - Location unspecified")
	}
	if asset.GetLocation().GetZone() == ufspb.Zone_ZONE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "validateAsset - Zone unspecified")
	}
	// Check if the rack exists
	if r := asset.GetLocation().GetRack(); r != "" {
		var errMsg strings.Builder
		errMsg.WriteString("validateAsset - ")
		return ResourceExist(ctx, []*Resource{GetRackResource(r)}, &errMsg)
	}
	return status.Error(codes.InvalidArgument, "validateAsset - Rack unspecified")
}

func validateAssetUpdateMask(ctx context.Context, asset *ufspb.Asset, mask *field_mask.FieldMask) error {
	if mask != nil {
		for _, path := range mask.Paths {
			switch path {
			case "name":
				return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - name cannot be updated, delete and create new asset")
			case "info.asset_tag":
				return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - asset_tag cannot be updated, delete and create new asset")
			case "update_time":
				return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Invalid update, cannot update update_time")
			case "location":
				fallthrough
			case "location.zone":
				if asset.GetLocation() == nil || asset.GetLocation().GetZone() == ufspb.Zone_ZONE_UNSPECIFIED {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Zone is unspecified so cannot be updated")
				} else if asset.GetLocation().GetRack() == "" {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Zone is updated without updating rack")
				}
			case "location.rack":
				if asset.GetLocation() == nil {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Rack is unset so cannot be updated")
				}
				if asset.GetLocation().GetRack() == "" {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Rack is empty so cannot be updated")
				}
				var errMsg strings.Builder
				errMsg.WriteString("validateAssetUpdateMask - ")
				return ResourceExist(ctx, []*Resource{GetRackResource(asset.GetLocation().GetRack())}, &errMsg)
			case "location.aisle":
				fallthrough
			case "location.row":
				fallthrough
			case "location.rack_number":
				fallthrough
			case "location.shelf":
				fallthrough
			case "location.position":
				fallthrough
			case "location.barcode_name":
				if asset.GetLocation() == nil {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Barcode name is unset so cannot be updated")
				}
			case "type":
				if asset.GetType() == ufspb.AssetType_UNDEFINED {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Type is undefined so cannot be updated")
				}
			case "model":
				if asset.GetModel() == "" {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Model is unset so cannot be updated")
				}
			case "info.cost_center":
				fallthrough
			case "info.google_code_name":
				fallthrough
			case "info.build_target":
				fallthrough
			case "info.reference_board":
				fallthrough
			case "info.ethernet_mac_address":
				fallthrough
			case "info.phase":
				if asset.GetInfo() == nil {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Phase is unset so cannot be updated")
				}
			default:
				return status.Errorf(codes.InvalidArgument, "validateAssetUpdateMask - unsupported mask %s", path)
			}
		}
	}
	return nil
}

func validateDeleteAsset(ctx context.Context, asset *ufspb.Asset) error {
	if err := util.CheckPermission(ctx, util.RegistrationsDelete, asset.GetRealm()); err != nil {
		return err
	}
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", asset.GetName(), true)
	if err != nil {
		return err
	}
	if len(machinelses) > 0 {
		return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("Asset %s cannot be deleted because DUT %s is referring this Asset.", asset.GetName(), machinelses[0].GetName()))
	}
	// TODO(anushruth): Add validation for servo resources
	return nil
}

func validateRenameAsset(ctx context.Context, asset *ufspb.Asset, newName string) error {
	// Check permission
	if err := util.CheckPermission(ctx, util.RegistrationsCreate, asset.GetRealm()); err != nil {
		return err
	}
	if err := util.CheckPermission(ctx, util.RegistrationsDelete, asset.GetRealm()); err != nil {
		return err
	}
	// Ensure that the asset with newName doesn't exist
	if err := resourceAlreadyExists(ctx, []*Resource{GetAssetResource(newName)}, nil); err != nil {
		return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("Failed to rename %s. %s already exists", asset.GetName(), newName))
	}
	return nil
}

func processAssetUpdateMask(updatedAsset, oldAsset *ufspb.Asset, mask *field_mask.FieldMask) (*ufspb.Asset, error) {
	// Add empty asset info both messages to avoid segfaults
	if oldAsset.GetInfo() == nil {
		oldAsset.Info = &ufspb.AssetInfo{}
	}
	if updatedAsset.GetInfo() == nil {
		updatedAsset.Info = &ufspb.AssetInfo{}
	}
	// If we are updating zone. We need to reset all the fields in the Location
	if util.ContainsAnyStrings(mask.GetPaths(), "location.zone") && updatedAsset.GetLocation().GetZone() != oldAsset.GetLocation().GetZone() {
		oldAsset.Location = &ufspb.Location{}
	}
	if mask != nil {
		for _, path := range mask.Paths {
			switch path {
			case "type":
				oldAsset.Type = updatedAsset.Type
			case "model":
				oldAsset.Model = updatedAsset.Model
				oldAsset.Info.Model = updatedAsset.Model
			case "location":
				oldAsset.Location = updatedAsset.Location
				oldAsset.Realm = updatedAsset.Realm
			case "location.aisle":
				oldAsset.Location.Aisle = updatedAsset.Location.Aisle
			case "location.row":
				oldAsset.Location.Row = updatedAsset.Location.Row
			case "location.rack":
				oldAsset.Location.Rack = updatedAsset.Location.Rack
			case "location.rack_number":
				oldAsset.Location.RackNumber = updatedAsset.Location.RackNumber
			case "location.shelf":
				oldAsset.Location.Shelf = updatedAsset.Location.Shelf
			case "location.position":
				oldAsset.Location.Position = updatedAsset.Location.Position
			case "location.barcode_name":
				oldAsset.Location.BarcodeName = updatedAsset.Location.BarcodeName
			case "location.zone":
				oldAsset.Location.Zone = updatedAsset.Location.Zone
				oldAsset.Realm = updatedAsset.Realm
			case "info.cost_center":
				oldAsset.Info.CostCenter = updatedAsset.Info.CostCenter
			case "info.google_code_name":
				oldAsset.Info.GoogleCodeName = updatedAsset.Info.GoogleCodeName
			case "info.build_target":
				oldAsset.Info.BuildTarget = updatedAsset.Info.BuildTarget
			case "info.reference_board":
				oldAsset.Info.ReferenceBoard = updatedAsset.Info.ReferenceBoard
			case "info.ethernet_mac_address":
				oldAsset.Info.EthernetMacAddress = updatedAsset.Info.EthernetMacAddress
			case "info.phase":
				oldAsset.Info.Phase = updatedAsset.Info.Phase
			}
		}
		return oldAsset, nil
	}
	return nil, status.Error(codes.InvalidArgument, "processAssetUpdateMask - Invalid Input, No mask found")
}

// UpdateAssetMeta updates only dut meta data portion of the Asset.
//
// It's a temporary method to correct Serial number, HWID and Sku.
// Will remove once HaRT could provide us the correct info.
func UpdateAssetMeta(ctx context.Context, meta *ufspb.DutMeta) error {
	if meta == nil {
		return nil
	}
	f := func(ctx context.Context) error {
		machine, err := registration.GetMachine(ctx, meta.GetChromeosDeviceId())
		if err != nil {
			return err
		}
		if machine.GetChromeosMachine() == nil {
			logging.Warningf(ctx, "%s is not a valid Chromeos machine", meta.GetChromeosDeviceId())
			return nil
		}

		asset, err := registration.GetAsset(ctx, meta.GetChromeosDeviceId())
		if err != nil {
			return err
		}
		hc := &HistoryClient{}
		// Copy for logging
		oldAsset := proto.Clone(asset).(*ufspb.Asset)
		if asset.GetInfo() == nil {
			asset.Info = &ufspb.AssetInfo{}
		}

		if asset.GetInfo().GetSerialNumber() == meta.GetSerialNumber() &&
			asset.GetInfo().GetHwid() == meta.GetHwID() &&
			asset.GetInfo().GetSku() == meta.GetDeviceSku() {
			logging.Warningf(ctx, "nothing to update: old serial number %q, old hwid %q, old device-sku %q", meta.GetSerialNumber(), meta.GetHwID(), meta.GetDeviceSku())
			return nil
		}

		asset.GetInfo().SerialNumber = meta.GetSerialNumber()
		asset.GetInfo().Hwid = meta.GetHwID()
		asset.GetInfo().Sku = meta.GetDeviceSku()
		// Update the asset
		if _, err := registration.BatchUpdateAssets(ctx, []*ufspb.Asset{asset}); err != nil {
			return errors.Annotate(err, "Unable to update dut meta for %s", asset.Name).Err()
		}
		hc.LogAssetChanges(oldAsset, asset)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "UpdateAssetMeta (%s, %s) - %s", meta.GetChromeosDeviceId(), meta.GetHostname(), err.Error())
		return err
	}
	return nil
}

// CreateMachineFromAsset creates machine from asset
//
// If the asset is either a DUT or Labstation, machine is returned, nil otherwise.
func CreateMachineFromAsset(asset *ufspb.Asset) *ufspb.Machine {
	if asset == nil {
		return nil
	}
	device := &ufspb.ChromeOSMachine{
		ReferenceBoard: asset.GetInfo().GetReferenceBoard(),
		BuildTarget:    asset.GetInfo().GetBuildTarget(),
		Model:          asset.GetInfo().GetModel(),
		GoogleCodeName: asset.GetInfo().GetGoogleCodeName(),
		CostCenter:     asset.GetInfo().GetCostCenter(),
		Gpn:            asset.GetInfo().GetGpn(),
	}
	switch asset.GetType() {
	case ufspb.AssetType_DUT:
		device.DeviceType = ufspb.ChromeOSDeviceType_DEVICE_CHROMEBOOK
	case ufspb.AssetType_LABSTATION:
		device.DeviceType = ufspb.ChromeOSDeviceType_DEVICE_LABSTATION
	default:
		// Only DUTs and Labstations are stored as machines
		return nil
	}
	machine := &ufspb.Machine{
		Name:         asset.GetName(),
		SerialNumber: asset.GetInfo().GetSerialNumber(),
		Location:     asset.GetLocation(),
		Device: &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: device,
		},
		Realm:         asset.GetRealm(),
		ResourceState: ufspb.State_STATE_REGISTERED,
	}
	return machine
}

// updateMachineFromAsset updates a machine from asset
//
// This must be used only for DUT or a Labstation.
func updateMachineFromAsset(ctx context.Context, machine *ufspb.Machine, asset *ufspb.Asset, hc *HistoryClient) error {
	if asset == nil {
		return nil
	}
	if machine.GetChromeosMachine() == nil {
		machine.Device = &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{},
		}
	}

	// Update MachineLSE zone/rack info
	if machine.GetLocation().GetRack() != asset.GetLocation().GetRack() ||
		machine.GetLocation().GetZone() != asset.GetLocation().GetZone() {
		// Check if asset zone/rack information is updated.
		// If zone/rack info is updated, update the MachineLSE table with new zone/rack info.
		indexMap := map[string]string{
			"zone": asset.GetLocation().GetZone().String(), "rack": asset.GetLocation().GetRack()}
		oldIndexMap := map[string]string{
			"zone": machine.GetLocation().GetZone().String(), "rack": machine.GetLocation().GetRack()}
		if err := updateIndexingForMachineResources(ctx, machine, indexMap, oldIndexMap, hc); err != nil {
			return errors.Annotate(err, "UpdateAsset - update zone and rack indexing failed").Err()
		}
	}

	// Serial number of UFS machine is updated by SSW in
	// UpdateDutMeta https://source.corp.google.com/chromium_infra/go/src/infra/unifiedfleet/app/controller/machine.go;l=182
	// Dont rely on Asset(user provided) for Serial number, Hwid, Sku.
	// Especially for HWID, HaRT doesn't contain 100% trustable data. See b/185404595 for context.
	// Leave them with original values and dont update them.
	machine.Location = asset.GetLocation()
	machine.Realm = asset.GetRealm()
	machine.GetChromeosMachine().ReferenceBoard = asset.GetInfo().GetReferenceBoard()
	machine.GetChromeosMachine().BuildTarget = asset.GetInfo().GetBuildTarget()
	machine.GetChromeosMachine().Model = asset.GetInfo().GetModel()
	machine.GetChromeosMachine().GoogleCodeName = asset.GetInfo().GetGoogleCodeName()
	machine.GetChromeosMachine().MacAddress = asset.GetInfo().GetEthernetMacAddress()
	machine.GetChromeosMachine().Phase = asset.GetInfo().GetPhase()
	machine.GetChromeosMachine().CostCenter = asset.GetInfo().GetCostCenter()
	switch asset.GetType() {
	case ufspb.AssetType_DUT:
		machine.GetChromeosMachine().DeviceType = ufspb.ChromeOSDeviceType_DEVICE_CHROMEBOOK
	case ufspb.AssetType_LABSTATION:
		machine.GetChromeosMachine().DeviceType = ufspb.ChromeOSDeviceType_DEVICE_LABSTATION
	default:
		// Only DUTs and Labstations are stored as machines
	}
	return nil
}
