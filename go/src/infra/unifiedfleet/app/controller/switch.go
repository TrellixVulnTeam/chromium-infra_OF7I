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
	ufsUtil "infra/unifiedfleet/app/util"
)

// CreateSwitch creates switch in datastore.
func CreateSwitch(ctx context.Context, s *ufspb.Switch) (*ufspb.Switch, error) {
	// TODO(eshwarn): Add logic for Chrome OS
	f := func(ctx context.Context) error {
		hc := getSwitchHistoryClient(s)

		// Get rack to associate the switch
		rack, err := GetRack(ctx, s.GetRack())
		if err != nil {
			return err
		}

		// Validate the input
		if err := validateCreateSwitch(ctx, s, rack); err != nil {
			return err
		}

		// Fill the zone to switch OUTPUT only fields for indexing
		s.Zone = rack.GetLocation().GetZone().String()
		s.ResourceState = ufspb.State_STATE_SERVING

		// Create a switch entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err = registration.BatchUpdateSwitches(ctx, []*ufspb.Switch{s}); err != nil {
			return errors.Annotate(err, "Unable to create switch %s", s.Name).Err()
		}
		hc.LogSwitchChanges(nil, s)

		// Update state
		if err := hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_SERVING); err != nil {
			return err
		}
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create switch in datastore: %s", err)
		return nil, err
	}
	return s, nil
}

// UpdateSwitch updates switch in datastore.
func UpdateSwitch(ctx context.Context, s *ufspb.Switch, mask *field_mask.FieldMask) (*ufspb.Switch, error) {
	// TODO(eshwarn): Add logic for Chrome OS
	f := func(ctx context.Context) error {
		hc := getSwitchHistoryClient(s)

		// Get old/existing switch
		oldS, err := registration.GetSwitch(ctx, s.GetName())
		if err != nil {
			return errors.Annotate(err, "UpdateSwitch - get switch %s failed", s.GetName()).Err()
		}

		// Validate the input
		if err := validateUpdateSwitch(ctx, oldS, s, mask); err != nil {
			return errors.Annotate(err, "UpdateSwitch - validation failed").Err()
		}

		// Copy for logging
		oldSwitchCopy := proto.Clone(oldS).(*ufspb.Switch)
		// Fill the zone to switch OUTPUT only fields
		s.Zone = oldS.GetZone()

		// Partial update by field mask
		if mask != nil && len(mask.Paths) > 0 {
			s, err = processSwitchUpdateMask(ctx, oldS, s, mask)
			if err != nil {
				return errors.Annotate(err, "processing update mask failed").Err()
			}
		} else {
			// This is for complete object input
			if s.GetRack() == "" {
				return status.Error(codes.InvalidArgument, "rack cannot be empty for updating a switch")
			}
			if oldS.GetRack() != s.GetRack() {
				// User is trying to associate this switch with a different rack.
				// Get rack to associate the switch
				rack, err := GetRack(ctx, s.GetRack())
				if err != nil {
					return errors.Annotate(err, "UpdateSwitch - get rack %s failed", s.GetRack()).Err()
				}

				// check permission for the new rack realm
				if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsUpdate, rack.GetRealm()); err != nil {
					return err
				}
				// Fill the zone to switch OUTPUT only fields for indexing
				s.Zone = rack.GetLocation().GetZone().String()
			}
		}

		// Update state
		if err := hc.stUdt.updateStateHelper(ctx, s.GetResourceState()); err != nil {
			return errors.Annotate(err, "Fail to update state to switch %s", s.GetName()).Err()
		}

		// Update switch entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction. Datastore doesnt allow nested transactions.
		if _, err := registration.BatchUpdateSwitches(ctx, []*ufspb.Switch{s}); err != nil {
			return errors.Annotate(err, "UpdateSwitch - unable to batch update switch %s", s.Name).Err()
		}
		hc.LogSwitchChanges(oldSwitchCopy, s)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "UpdateSwitch - failed to update switch %s in datastore", s.Name).Err()
	}
	return s, nil
}

// processSwitchUpdateMask process update field mask to get only specific update
// fields and return a complete switch object with updated and existing fields
func processSwitchUpdateMask(ctx context.Context, oldSwitch *ufspb.Switch, s *ufspb.Switch, mask *field_mask.FieldMask) (*ufspb.Switch, error) {
	// update the fields in the existing/old switch
	for _, path := range mask.Paths {
		switch path {
		case "rack":
			if oldSwitch.GetRack() != s.GetRack() {
				// User is trying to associate this switch with a different rack.
				// Get rack to associate the switch
				rack, err := GetRack(ctx, s.GetRack())
				if err != nil {
					return oldSwitch, errors.Annotate(err, "UpdateSwitch - get rack %s failed", s.GetRack()).Err()
				}
				// check permission for the new rack realm
				if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsUpdate, rack.GetRealm()); err != nil {
					return oldSwitch, err
				}
				oldSwitch.Rack = s.GetRack()
				// Fill the zone to switch OUTPUT only fields for indexing
				oldSwitch.Zone = rack.GetLocation().GetZone().String()
			}
		case "resourceState":
			oldSwitch.ResourceState = s.GetResourceState()
		case "description":
			oldSwitch.Description = s.GetDescription()
		case "capacity":
			oldSwitch.CapacityPort = s.GetCapacityPort()
		case "tags":
			oldSwitch.Tags = mergeTags(oldSwitch.GetTags(), s.GetTags())
		}
	}
	// return existing/old switch with new updated values
	return oldSwitch, nil
}

// GetSwitch returns switch for the given id from datastore.
func GetSwitch(ctx context.Context, id string) (*ufspb.Switch, error) {
	return registration.GetSwitch(ctx, id)
}

// BatchGetSwitches returns a batch of switches from datastore.
func BatchGetSwitches(ctx context.Context, ids []string) ([]*ufspb.Switch, error) {
	return registration.BatchGetSwitches(ctx, ids)
}

// ListSwitches lists the switches
func ListSwitches(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.Switch, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, registration.GetSwitchIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing switches").Err()
		}
	}
	filterMap = resetStateFilter(filterMap)
	filterMap = resetZoneFilter(filterMap)
	return registration.ListSwitches(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteSwitch deletes the switch in datastore
//
// For referential data intergrity,
// 1. Validate if this switch is not referenced by other resources in the datastore.
// 2. Delete the switch
// 3. Get the rack associated with this switch
// 4. Update the rack by removing the association with this switch
func DeleteSwitch(ctx context.Context, id string) error {
	return deleteSwitchHelper(ctx, id, true)
}

func deleteSwitchHelper(ctx context.Context, id string, inTransaction bool) error {
	f := func(ctx context.Context) error {
		hc := getSwitchHistoryClient(&ufspb.Switch{Name: id})

		s, err := registration.GetSwitch(ctx, id)
		if err != nil {
			return errors.Annotate(err, "Unable to get switch").Err()
		}

		// Validate input
		if err := validateDeleteSwitch(ctx, s); err != nil {
			return errors.Annotate(err, "validation failed - Unable to delete switch %s", id).Err()
		}

		// Delete the switch
		if err := registration.DeleteSwitch(ctx, id); err != nil {
			return errors.Annotate(err, "delete failed - Unable to delete switch %s", id).Err()
		}

		// Update state
		hc.stUdt.deleteStateHelper(ctx)
		hc.LogSwitchChanges(s, nil)
		return hc.SaveChangeEvents(ctx)
	}
	if inTransaction {
		if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
			logging.Errorf(ctx, "Failed to delete switch in datastore: %s", err)
			return err
		}
		return nil
	}
	return f(ctx)
}

// ReplaceSwitch replaces an old Switch with new Switch in datastore
//
// It does a delete of old switch and create of new Switch.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old Switch, it will get all the resources referencing
// the old Switch. It will update all the resources which were referencing
// the old Switch(got in the last step) with new Switch.
// Deletes the old Switch.
// Creates the new Switch.
// This will preserve data integrity in the system.
func ReplaceSwitch(ctx context.Context, oldSwitch *ufspb.Switch, newSwitch *ufspb.Switch) (*ufspb.Switch, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateDeleteSwitch validates if a Switch can be deleted
//
// Checks if this Switch(SwitchID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteSwitch(ctx context.Context, s *ufspb.Switch) error {
	rack, err := registration.GetRack(ctx, s.GetRack())
	if err != nil {
		return errors.Annotate(err, "unable to get rack %s", s.GetRack()).Err()
	}
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsDelete, rack.GetRealm()); err != nil {
		return err
	}
	nics, err := registration.QueryNicByPropertyName(ctx, "switch_id", s.GetName(), true)
	if err != nil {
		return err
	}
	dracs, err := registration.QueryDracByPropertyName(ctx, "switch_id", s.GetName(), true)
	if err != nil {
		return err
	}
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "switch_id", s.GetName(), true)
	if err != nil {
		return err
	}
	if len(nics) > 0 || len(dracs) > 0 || len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Switch %s cannot be deleted because there are other resources which are referring to this Switch.", s.GetName()))
		if len(nics) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nNics referring to the Switch:\n"))
			for _, nic := range nics {
				errorMsg.WriteString(nic.Name + ", ")
			}
		}
		if len(dracs) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nDracs referring to the Switch:\n"))
			for _, drac := range dracs {
				errorMsg.WriteString(drac.Name + ", ")
			}
		}
		if len(machinelses) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nChromeOS hosts referring to the Switch:\n"))
			for _, machinelse := range machinelses {
				errorMsg.WriteString(machinelse.Name + ", ")
			}
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

// validateCreateSwitch validates if a switch can be created
//
// check if the switch already exists
// check if the rack does not exist
func validateCreateSwitch(ctx context.Context, s *ufspb.Switch, rack *ufspb.Rack) error {
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsCreate, rack.GetRealm()); err != nil {
		return err
	}
	// Check if switch already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetSwitchResource(s.Name)}, nil); err != nil {
		return err
	}
	return nil
}

// validateUpdateSwitch validates if a switch can be updated
//
// check if switch and rack does not exist
func validateUpdateSwitch(ctx context.Context, oldS *ufspb.Switch, s *ufspb.Switch, mask *field_mask.FieldMask) error {
	rack, err := registration.GetRack(ctx, oldS.GetRack())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "rack %s not found", oldS.GetRack())
	}
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsUpdate, rack.GetRealm()); err != nil {
		return err
	}

	// Aggregate resource to check if rack does not exist
	if s.GetRack() != "" {
		resourcesNotFound := []*Resource{GetRackResource(s.GetRack())}
		// check if resources does not exist
		if err := ResourceExist(ctx, resourcesNotFound, nil); err != nil {
			return err
		}
	}

	return validateSwitchUpdateMask(s, mask)
}

// validateSwitchUpdateMask validates the update mask for switch update
func validateSwitchUpdateMask(s *ufspb.Switch, mask *field_mask.FieldMask) error {
	if mask != nil {
		// validate the give field mask
		for _, path := range mask.Paths {
			switch path {
			case "name":
				return status.Error(codes.InvalidArgument, "validateUpdateSwitch - name cannot be updated, delete and create a new switch instead")
			case "update_time":
				return status.Error(codes.InvalidArgument, "validateUpdateSwitch - update_time cannot be updated, it is a Output only field")
			case "rack":
				if s.GetRack() == "" {
					return status.Error(codes.InvalidArgument, "rack cannot be empty for updating a switch")
				}
			case "capacity":
			case "description":
			case "tags":
			case "resourceState":
				// valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "validateUpdateSwitch - unsupported update mask path %q", path)
			}
		}
	}
	return nil
}

func getSwitchHistoryClient(m *ufspb.Switch) *HistoryClient {
	return &HistoryClient{
		stUdt: &stateUpdater{
			ResourceName: ufsUtil.AddPrefix(ufsUtil.SwitchCollection, m.Name),
		},
		netUdt: &networkUpdater{
			Hostname: m.Name,
		},
	}
}
