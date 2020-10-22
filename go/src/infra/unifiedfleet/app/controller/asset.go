// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/model/registration"
)

// AssetRegistration registers the given asset to the datastore after validation
func AssetRegistration(ctx context.Context, asset *ufspb.Asset) (*ufspb.Asset, error) {
	hc := &HistoryClient{}
	var err error
	f := func(ctx context.Context) error {
		if err := validateAssetRegistration(ctx, asset); err != nil {
			return err
		}
		_, err := registration.BatchUpdateAssets(ctx, []*ufspb.Asset{asset})
		if err != nil {
			return err
		}
		hc.LogAssetChanges(nil, asset)
		return hc.SaveChangeEvents(ctx)
	}
	if err = datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "AddAsset - unable to update asset %s", asset.GetName()).Err()
	}
	return asset, err
}

// UpdateAsset updates the asset record to the datastore after validation
func UpdateAsset(ctx context.Context, asset *ufspb.Asset, mask *field_mask.FieldMask) (*ufspb.Asset, error) {
	var oldAsset *ufspb.Asset
	var err error
	hc := &HistoryClient{}
	f := func(ctx context.Context) error {
		err := validateUpdateAsset(ctx, asset, mask)
		if err != nil {
			return err
		}
		// TODO(anshruth): Support validation of DUT/Labstation/Servo
		// created using this asset. And update them accordingly or fail.
		oldAsset, err = registration.GetAsset(ctx, asset.GetName())
		if err != nil {
			return err
		}
		// updatableAsset will be used to update the asset
		updatableAsset := asset
		if mask != nil && mask.Paths != nil {
			// Construct updatableAsset from mask if given
			updatableAsset = proto.Clone(proto.MessageV1(oldAsset)).(*ufspb.Asset)
			if asset, err = processAssetUpdateMask(asset, updatableAsset, mask); err != nil {
				return err
			}
		}
		_, err = registration.BatchUpdateAssets(ctx, []*ufspb.Asset{updatableAsset})
		if err != nil {
			return err
		}
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
		filterMap, err = getFilterMap(filter, getAssetIndexedFieldName)
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
		if err := validateDeleteAsset(ctx, name); err != nil {
			return errors.Annotate(err, "DeleteAsset - failed to delete %s", name).Err()
		}
		asset, err := registration.GetAsset(ctx, name)
		if err != nil {
			return errors.Annotate(err, "DeleteAsset - cannot find asset %s", name).Err()
		}
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

// getAssetIndexedFieldName returns the same string as the mapping is 1:1
func getAssetIndexedFieldName(name string) (string, error) {
	return name, nil
}

func validateUpdateAsset(ctx context.Context, asset *ufspb.Asset, mask *field_mask.FieldMask) error {
	if mask == nil || mask.Paths == nil {
		// If mask doesn't exist then validate the given asset
		return validateAsset(ctx, asset)
	}
	// Validate AssetUpdate Mask if it exists
	return validateAssetUpdateMask(ctx, asset, mask)
}

func validateAssetRegistration(ctx context.Context, asset *ufspb.Asset) error {
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
			case "location.zone":
				if asset.GetLocation() == nil || asset.GetLocation().GetZone() == ufspb.Zone_ZONE_UNSPECIFIED {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Invalid location")
				}
			case "location.rack":
				if asset.GetLocation() == nil {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Invalid location")
				}
				if asset.GetLocation().GetRack() == "" {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Invalid update, cannot clear rack")
				}
				var errMsg strings.Builder
				errMsg.WriteString("validateAssetUpdateMask - ")
				return ResourceExist(ctx, []*Resource{GetRackResource(asset.GetLocation().GetRack())}, &errMsg)
			case "type":
				if asset.GetType() == ufspb.AssetType_UNDEFINED {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Invalid update, no type given")
				}
			case "model":
				if asset.GetModel() == "" {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Invalid update, no model given")
				}
			case "info.serial_number":
				fallthrough
			case "info.cost_center":
				fallthrough
			case "info.google_code_name":
				fallthrough
			case "info.model":
				fallthrough
			case "info.build_target":
				fallthrough
			case "info.reference_board":
				fallthrough
			case "info.ethernet_mac_address":
				fallthrough
			case "info.sku":
				fallthrough
			case "info.phase":
				if asset.GetInfo() == nil {
					return status.Error(codes.InvalidArgument, "validateAssetUpdateMask - Invalid asset info")
				}
			default:
				return status.Errorf(codes.InvalidArgument, "validateAssetUpdateMask - unsupported mask %s", path)
			}
		}
	}
	return nil
}

func validateDeleteAsset(ctx context.Context, name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "validateDeleteAsset - Missing asset name")
	}

	var errMsg strings.Builder
	errMsg.WriteString("validateDeleteAsset - ")
	if err := ResourceExist(ctx, []*Resource{GetMachineResource(name)}, &errMsg); err == nil {
		// Cannot delete asset if its registered as a machine
		return status.Error(codes.FailedPrecondition, "validateDeleteAsset - Asset registered as DUT/Labstation")
	}
	// TODO(anushruth): Add validation for servo resources
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
	if mask != nil {
		for _, path := range mask.Paths {
			switch path {
			case "type":
				oldAsset.Type = updatedAsset.Type
			case "model":
				oldAsset.Model = updatedAsset.Model
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
			case "info.serial_number":
				oldAsset.Info.SerialNumber = updatedAsset.Info.SerialNumber
			case "info.cost_center":
				oldAsset.Info.CostCenter = updatedAsset.Info.CostCenter
			case "info.google_code_name":
				oldAsset.Info.GoogleCodeName = updatedAsset.Info.GoogleCodeName
			case "info.model":
				oldAsset.Info.Model = updatedAsset.Info.Model
			case "info.build_target":
				oldAsset.Info.BuildTarget = updatedAsset.Info.BuildTarget
			case "info.reference_board":
				oldAsset.Info.ReferenceBoard = updatedAsset.Info.ReferenceBoard
			case "info.ethernet_mac_address":
				oldAsset.Info.EthernetMacAddress = updatedAsset.Info.EthernetMacAddress
			case "info.sku":
				oldAsset.Info.Sku = updatedAsset.Info.Sku
			case "info.phase":
				oldAsset.Info.Phase = updatedAsset.Info.Phase
			}
		}
		return oldAsset, nil
	}
	return nil, status.Error(codes.InvalidArgument, "processAssetUpdateMask - Invalid Input, No mask found")
}
