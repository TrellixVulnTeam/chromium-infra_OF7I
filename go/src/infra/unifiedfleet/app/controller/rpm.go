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
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	ufsUtil "infra/unifiedfleet/app/util"
)

// CreateRPM creates a new rpm in datastore.
func CreateRPM(ctx context.Context, rpm *ufspb.RPM) (*ufspb.RPM, error) {
	f := func(ctx context.Context) error {
		hc := getRPMHistoryClient(rpm)
		hc.LogRPMChanges(nil, rpm)

		// Get rack to associate the rpm
		rack, err := GetRack(ctx, rpm.GetRack())
		if err != nil {
			return err
		}

		// Validate the input
		if err := validateCreateRPM(ctx, rpm, rack); err != nil {
			return err
		}

		// Fill the zone to rpm OUTPUT only fields for indexing
		rpm.Zone = rack.GetLocation().GetZone().String()
		rpm.ResourceState = ufspb.State_STATE_REGISTERED

		// Create a rpm entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesn't allow
		// nested transactions.
		if _, err = registration.BatchUpdateRPMs(ctx, []*ufspb.RPM{rpm}); err != nil {
			return errors.Annotate(err, "Unable to create rpm %s", rpm.Name).Err()
		}

		// Update state
		if err := hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_REGISTERED); err != nil {
			return err
		}
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create rpm in datastore: %s", err)
		return nil, err
	}
	return rpm, nil
}

// UpdateRPM updates rpm in datastore.
func UpdateRPM(ctx context.Context, rpm *ufspb.RPM, mask *field_mask.FieldMask) (*ufspb.RPM, error) {
	f := func(ctx context.Context) error {
		hc := getRPMHistoryClient(rpm)

		// Get old/existing RPM
		oldRPM, err := registration.GetRPM(ctx, rpm.GetName())
		if err != nil {
			return errors.Annotate(err, "UpdateRPM - get rpm %s failed", rpm.GetName()).Err()
		}

		// Validate the input
		if err := validateUpdateRPM(ctx, oldRPM, rpm, mask); err != nil {
			return errors.Annotate(err, "UpdateRPM - validation failed").Err()
		}

		// Copy for logging
		oldRPMCopy := proto.Clone(oldRPM).(*ufspb.RPM)
		// Fill the zone to rpm OUTPUT only fields
		rpm.Zone = oldRPM.GetZone()

		// Partial update by field mask
		if mask != nil && len(mask.Paths) > 0 {
			rpm, err = processRPMUpdateMask(ctx, oldRPM, rpm, mask)
			if err != nil {
				return errors.Annotate(err, "UpdateRPM - processing update mask failed").Err()
			}
		} else {
			// This is for complete object input
			if rpm.GetRack() == "" {
				return status.Error(codes.InvalidArgument, "rack cannot be empty for updating a RPM")
			}
			if oldRPM.GetRack() != rpm.GetRack() {
				// User is trying to associate this rpm with a different rack.
				// Get rack to associate the rpm
				rack, err := GetRack(ctx, rpm.GetRack())
				if err != nil {
					return errors.Annotate(err, "UpdateRPM - get rack %s failed", rpm.GetRack()).Err()
				}

				// check permission for the new rack realm
				if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsUpdate, rack.GetRealm()); err != nil {
					return err
				}
				// Fill the zone to rpm OUTPUT only fields
				rpm.Zone = rack.GetLocation().GetZone().String()
			}
		}

		// Update state
		if err := hc.stUdt.updateStateHelper(ctx, rpm.GetResourceState()); err != nil {
			return errors.Annotate(err, "Fail to update state to rpm %s", rpm.GetName()).Err()
		}

		// Update rpm entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction. Datastore doesnt allow nested transactions.
		if _, err := registration.BatchUpdateRPMs(ctx, []*ufspb.RPM{rpm}); err != nil {
			return errors.Annotate(err, "UpdateRPM - unable to batch update rpm %s", rpm.Name).Err()
		}
		hc.LogRPMChanges(oldRPMCopy, rpm)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "UpdateRPM - failed to update rpm %s in datastore", rpm.Name).Err()
	}
	return rpm, nil
}

// GetRPM returns rpm for the given id from datastore.
func GetRPM(ctx context.Context, id string) (*ufspb.RPM, error) {
	return registration.GetRPM(ctx, id)
}

// BatchGetRPMs returns a batch of rpms from datastore.
func BatchGetRPMs(ctx context.Context, ids []string) ([]*ufspb.RPM, error) {
	return registration.BatchGetRPMs(ctx, ids)
}

// ListRPMs lists the rpms
func ListRPMs(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.RPM, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, registration.GetRPMIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing rpms").Err()
		}
	}
	filterMap = resetStateFilter(filterMap)
	filterMap = resetZoneFilter(filterMap)
	return registration.ListRPMs(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteRPM deletes the rpm in datastore
//
// For referential data intergrity,
// Delete if this RPM is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteRPM(ctx context.Context, id string) error {
	return deleteRPMHelper(ctx, id, true)
}

func deleteRPMHelper(ctx context.Context, id string, inTransaction bool) error {
	f := func(ctx context.Context) error {
		hc := getRPMHistoryClient(&ufspb.RPM{Name: id})

		// Get rpm
		rpm, err := registration.GetRPM(ctx, id)
		if err != nil {
			return errors.Annotate(err, "Unable to get RPM").Err()
		}

		// Validate input
		if err := validateDeleteRPM(ctx, rpm); err != nil {
			return errors.Annotate(err, "Validation failed - unable to delete rpm %s", id).Err()
		}

		// Delete the rpm
		if err := registration.DeleteRPM(ctx, id); err != nil {
			return errors.Annotate(err, "Delete failed - unable to delete rpm %s", id).Err()
		}

		// Update state
		hc.stUdt.deleteStateHelper(ctx)

		// Delete ip configs
		if err := hc.netUdt.deleteDHCPHelper(ctx); err != nil {
			return err
		}
		hc.LogRPMChanges(rpm, nil)
		return hc.SaveChangeEvents(ctx)
	}
	if inTransaction {
		if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
			logging.Errorf(ctx, "Failed to delete rpm in datastore: %s", err)
			return err
		}
		return nil
	}
	return f(ctx)
}

// ReplaceRPM replaces an old RPM with new RPM in datastore
//
// It does a delete of old rpm and create of new RPM.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old RPM, it will get all the resources referencing
// the old RPM. It will update all the resources which were referencing
// the old RPM(got in the last step) with new RPM.
// Deletes the old RPM.
// Creates the new RPM.
// This will preserve data integrity in the system.
func ReplaceRPM(ctx context.Context, oldRPM *ufspb.RPM, newRPM *ufspb.RPM) (*ufspb.RPM, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateDeleteRPM validates if a RPM can be deleted
//
// Checks if this RPM(RPMID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteRPM(ctx context.Context, rpm *ufspb.RPM) error {
	rack, err := registration.GetRack(ctx, rpm.GetRack())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "rack %s not found", rpm.GetRack())
	}
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsDelete, rack.GetRealm()); err != nil {
		return err
	}
	machines, err := registration.QueryMachineByPropertyName(ctx, "rpm_id", rpm.GetName(), true)
	if err != nil {
		return err
	}
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "rpm_id", rpm.GetName(), true)
	if err != nil {
		return err
	}
	if len(machines) > 0 || len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("RPM %s cannot be deleted because there are other resources which are referring this RPM.", rpm.GetName()))
		if len(machines) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachines referring the RPM:\n"))
			for _, machine := range machines {
				errorMsg.WriteString(machine.Name + ", ")
			}
		}
		if len(machinelses) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachineLSEs referring the RPM:\n"))
			for _, machinelse := range machinelses {
				errorMsg.WriteString(machinelse.Name + ", ")
			}
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

func getRPMHistoryClient(rpm *ufspb.RPM) *HistoryClient {
	return &HistoryClient{
		stUdt: &stateUpdater{
			ResourceName: ufsUtil.AddPrefix(ufsUtil.RPMCollection, rpm.Name),
		},
		netUdt: &networkUpdater{
			Hostname: rpm.Name,
		},
	}
}

// validateCreateRPM validates if a rpm can be created
//
// check if the rpm already exists
// check if the rack and resources referenced by rpm does not exist
func validateCreateRPM(ctx context.Context, rpm *ufspb.RPM, rack *ufspb.Rack) error {
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsCreate, rack.GetRealm()); err != nil {
		return err
	}
	// Check if rpm already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetRPMResource(rpm.Name)}, nil); err != nil {
		return err
	}
	if err := validateMacAddress(ctx, rpm.GetName(), rpm.GetMacAddress()); err != nil {
		return err
	}
	return nil
}

// validateUpdateRPM validates if a rpm can be updated
//
// check if rpm, rack and resources referenced rpm does not exist
func validateUpdateRPM(ctx context.Context, oldRPM *ufspb.RPM, rpm *ufspb.RPM, mask *field_mask.FieldMask) error {
	rack, err := registration.GetRack(ctx, oldRPM.GetRack())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "rack %s not found", oldRPM.GetRack())
	}
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsUpdate, rack.GetRealm()); err != nil {
		return err
	}
	// Aggregate resource to check if rpm does not exist
	var resourcesNotFound []*Resource
	// Aggregate resource to check if rack does not exist
	if rpm.GetRack() != "" {
		resourcesNotFound = append(resourcesNotFound, GetRackResource(rpm.GetRack()))
	}
	// check if resources does not exist
	if err := ResourceExist(ctx, resourcesNotFound, nil); err != nil {
		return err
	}

	return validateRPMUpdateMask(ctx, rpm, mask)
}

// validateRPMUpdateMask validates the update mask for rpm update
func validateRPMUpdateMask(ctx context.Context, rpm *ufspb.RPM, mask *field_mask.FieldMask) error {
	if mask != nil {
		// validate the give field mask
		for _, path := range mask.Paths {
			switch path {
			case "name":
				return status.Error(codes.InvalidArgument, "name cannot be updated, delete and create a new rpm instead")
			case "update_time":
				return status.Error(codes.InvalidArgument, "update_time cannot be updated, it is a Output only field")
			case "macAddress":
				if err := validateMacAddress(ctx, rpm.GetName(), rpm.GetMacAddress()); err != nil {
					return err
				}
			case "rack":
				if rpm.GetRack() == "" {
					return status.Error(codes.InvalidArgument, "rack cannot be empty for updating a RPM")
				}
			case "capacity":
			case "description":
			case "tags":
			case "resourceState":
				// valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "unsupported update mask path %q", path)
			}
		}
	}
	if err := validateMacAddress(ctx, rpm.GetName(), rpm.GetMacAddress()); err != nil {
		return err
	}
	return nil
}

// validateUpdateRPMHost validates if a host can be added to a rpm
func validateUpdateRPMHost(ctx context.Context, rpm *ufspb.RPM, vlanName, ipv4Str string) error {
	// during partial update, rpm object may not have rack info, so we get the old rpm to get the rack
	// to check the permission
	oldRPM, err := registration.GetRPM(ctx, rpm.GetName())
	if err != nil {
		return err
	}
	rack, err := registration.GetRack(ctx, oldRPM.GetRack())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "rack %s not found", oldRPM.GetRack())
	}
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsUpdate, rack.GetRealm()); err != nil {
		return err
	}
	if rpm.GetMacAddress() == "" {
		return errors.New("mac address of rpm hasn't been specified")
	}
	if ipv4Str != "" {
		return nil
	}
	// Check if resources does not exist
	return ResourceExist(ctx, []*Resource{GetRPMResource(rpm.Name), GetVlanResource(vlanName)}, nil)
}

// processRPMUpdateMask process update field mask to get only specific update
// fields and return a complete rpm object with updated and existing fields
func processRPMUpdateMask(ctx context.Context, oldRPM *ufspb.RPM, rpm *ufspb.RPM, mask *field_mask.FieldMask) (*ufspb.RPM, error) {
	// update the fields in the existing/old rpm
	for _, path := range mask.Paths {
		switch path {
		case "rack":
			if oldRPM.GetRack() != rpm.GetRack() {
				// User is trying to associate this rpm with a different rack.
				// Get rack to associate the rpm
				rack, err := GetRack(ctx, rpm.GetRack())
				if err != nil {
					return oldRPM, errors.Annotate(err, "get rack %s failed", rpm.GetRack()).Err()
				}
				// check permission for the new rack realm
				if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsUpdate, rack.GetRealm()); err != nil {
					return oldRPM, err
				}
				oldRPM.Rack = rpm.GetRack()
				// Fill the zone to rpm OUTPUT only fields
				oldRPM.Zone = rack.GetLocation().GetZone().String()
			}
		case "description":
			oldRPM.Description = rpm.GetDescription()
		case "capacity":
			oldRPM.CapacityPort = rpm.GetCapacityPort()
		case "resourceState":
			oldRPM.ResourceState = rpm.GetResourceState()
		case "macAddress":
			oldRPM.MacAddress = rpm.GetMacAddress()
		case "tags":
			oldRPM.Tags = mergeTags(oldRPM.GetTags(), rpm.GetTags())
		}
	}
	// return existing/old rpm with new updated values
	return oldRPM, nil
}

// DeleteRPMHost deletes the host of a rpm in datastore.
func DeleteRPMHost(ctx context.Context, rpmName string) error {
	f := func(ctx context.Context) error {
		hc := getRPMHistoryClient(&ufspb.RPM{Name: rpmName})
		if err := hc.netUdt.deleteDHCPHelper(ctx); err != nil {
			return err
		}
		if err := hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_REGISTERED); err != nil {
			return errors.Annotate(err, "Fail to update state to rpm %s", rpmName).Err()
		}
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete the rpm host: %s", err)
		return err
	}
	return nil
}

// UpdateRPMHost updates the rpm host in datastore.
func UpdateRPMHost(ctx context.Context, rpm *ufspb.RPM, nwOpt *ufsAPI.NetworkOption) error {
	f := func(ctx context.Context) error {
		hc := getRPMHistoryClient(rpm)
		// 1. Validate the input
		if err := validateUpdateRPMHost(ctx, rpm, nwOpt.GetVlan(), nwOpt.GetIp()); err != nil {
			return err
		}
		// 2. Verify if the hostname is already set with IP. if yes, remove the current dhcp.
		if err := hc.netUdt.deleteDHCPHelper(ctx); err != nil {
			return err
		}

		// 3. Find free ip, set IP and DHCP config
		if _, err := hc.netUdt.addHostHelper(ctx, nwOpt.GetVlan(), nwOpt.GetIp(), rpm.GetMacAddress()); err != nil {
			return err
		}

		if err := hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_DEPLOYING); err != nil {
			return errors.Annotate(err, "Fail to update state to rpm %s", rpm.GetName()).Err()
		}
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to assign IP to the rpm: %s", err)
		return err
	}
	return nil
}
