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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	ufsUtil "infra/unifiedfleet/app/util"
)

// CreateSwitch creates switch in datastore.
func CreateSwitch(ctx context.Context, s *ufspb.Switch, rackName string) (*ufspb.Switch, error) {
	// TODO(eshwarn): Add logic for Chrome OS
	changes := LogSwitchChanges(nil, s)
	f := func(ctx context.Context) error {
		// 1. Validate the input
		if err := validateCreateSwitch(ctx, s, rackName); err != nil {
			return err
		}

		// 2. Get rack to associate the switch
		rack, err := GetRack(ctx, rackName)
		if err != nil {
			return err
		}

		// 3. Update the rack with new switch information
		if cs, err := addSwitchToRack(ctx, rack, s.Name); err == nil {
			changes = append(changes, cs...)
		} else {
			return err
		}

		// 4. Create a switch entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err = registration.BatchUpdateSwitches(ctx, []*ufspb.Switch{s}); err != nil {
			return errors.Annotate(err, "Unable to create switch %s", s.Name).Err()
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create switch in datastore: %s", err)
		return nil, err
	}
	SaveChangeEvents(ctx, changes)
	return s, nil
}

// UpdateSwitch updates switch in datastore.
func UpdateSwitch(ctx context.Context, s *ufspb.Switch, rackName string) (*ufspb.Switch, error) {
	// TODO(eshwarn): Add logic for Chrome OS
	changes := make([]*ufspb.ChangeEvent, 0)
	f := func(ctx context.Context) error {
		// 1. Validate the input
		if err := validateUpdateSwitch(ctx, s, rackName); err != nil {
			return err
		}

		oldS, _ := registration.GetSwitch(ctx, s.GetName())
		changes = append(changes, LogSwitchChanges(oldS, s)...)
		if rackName != "" {
			// 2. Get the old rack associated with switch
			oldRack, err := getRackForSwitch(ctx, s.Name)
			if err != nil {
				return err
			}

			// User is trying to associate this switch with a different rack.
			if oldRack.Name != rackName {
				// 3. Get rack to associate the switch
				rack, err := GetRack(ctx, rackName)
				if err != nil {
					return err
				}

				// 4. Remove the association between old rack and this switch.
				if cs, err := removeSwitchFromRacks(ctx, []*ufspb.Rack{oldRack}, s.Name); err == nil {
					changes = append(changes, cs...)
				} else {
					return err
				}

				// 5. Update the rack with new switch information
				if cs, err := addSwitchToRack(ctx, rack, s.Name); err == nil {
					changes = append(changes, cs...)
				} else {
					return err
				}
			}
		}

		// 6. Update switch entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction. Datastore doesnt allow nested transactions.
		if _, err := registration.BatchUpdateSwitches(ctx, []*ufspb.Switch{s}); err != nil {
			return errors.Annotate(err, "Unable to update switch %s", s.Name).Err()
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to update switch in datastore: %s", err)
		return nil, err
	}
	SaveChangeEvents(ctx, changes)
	return s, nil
}

// GetSwitch returns switch for the given id from datastore.
func GetSwitch(ctx context.Context, id string) (*ufspb.Switch, error) {
	return registration.GetSwitch(ctx, id)
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
	changes := LogSwitchChanges(&ufspb.Switch{Name: id}, nil)
	f := func(ctx context.Context) error {
		// 1. Validate input
		if err := validateDeleteSwitch(ctx, id); err != nil {
			return errors.Annotate(err, "validation failed - Unable to delete switch %s", id).Err()
		}

		// 2. Delete the switch
		if err := registration.DeleteSwitch(ctx, id); err != nil {
			return errors.Annotate(err, "delete failed - Unable to delete switch %s", id).Err()
		}

		// 3. Get the rack associated with switch
		racks, err := registration.QueryRackByPropertyName(ctx, "switch_ids", id, false)
		if err != nil {
			return errors.Annotate(err, "Unable to query rack for switch %s", id).Err()
		}
		if racks == nil || len(racks) == 0 {
			logging.Warningf(ctx, "No rack associated with the switch %s. Data discrepancy error.\n", id)
			return nil
		}
		if len(racks) > 1 {
			logging.Warningf(ctx, "More than one rack associated with the switch %s. Data discrepancy error.\n", id)
		}

		// 4. Remove the association between the rack and this switch.
		cs, err := removeSwitchFromRacks(ctx, racks, id)
		if err != nil {
			return err
		}
		changes = append(changes, cs...)
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete switch in datastore: %s", err)
		return err
	}
	SaveChangeEvents(ctx, changes)
	return nil
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
func validateDeleteSwitch(ctx context.Context, id string) error {
	nics, err := registration.QueryNicByPropertyName(ctx, "switch_id", id, true)
	if err != nil {
		return err
	}
	dracs, err := registration.QueryDracByPropertyName(ctx, "switch_id", id, true)
	if err != nil {
		return err
	}
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "switch_id", id, true)
	if err != nil {
		return err
	}
	if len(nics) > 0 || len(dracs) > 0 || len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Switch %s cannot be deleted because there are other resources which are referring to this Switch.", id))
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
func validateCreateSwitch(ctx context.Context, s *ufspb.Switch, rackName string) error {
	// 1. Check if switch already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetSwitchResource(s.Name)}, nil); err != nil {
		return err
	}
	// 2. Check if rack does not exist
	return ResourceExist(ctx, []*Resource{GetRackResource(rackName)}, nil)
}

// validateUpdateSwitch validates if a switch can be updated
//
// check if switch and rack does not exist
func validateUpdateSwitch(ctx context.Context, s *ufspb.Switch, rackName string) error {
	// Aggregate resource to check if switch does not exist
	resourcesNotFound := []*Resource{GetSwitchResource(s.Name)}
	// Aggregate resource to check if rack does not exist
	if rackName != "" {
		resourcesNotFound = append(resourcesNotFound, GetRackResource(rackName))
	}
	// Check if resources does not exist
	return ResourceExist(ctx, resourcesNotFound, nil)
}

// addSwitchToRack adds the switch info to the rack and updates
// the rack in datastore.
// Must be called within a transaction as BatchUpdateRacks is a non-atomic operation
func addSwitchToRack(ctx context.Context, rack *ufspb.Rack, switchName string) ([]*ufspb.ChangeEvent, error) {
	if rack == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Rack is nil")
	}
	if rack.GetChromeBrowserRack() == nil {
		errorMsg := fmt.Sprintf("Rack %s is not a browser rack", rack.Name)
		return nil, status.Errorf(codes.FailedPrecondition, errorMsg)
	}
	switches := []string{switchName}
	if rack.GetChromeBrowserRack().GetSwitches() != nil {
		switches = rack.GetChromeBrowserRack().GetSwitches()
		switches = append(switches, switchName)
	}
	old := proto.Clone(rack).(*ufspb.Rack)
	rack.GetChromeBrowserRack().Switches = switches
	_, err := registration.BatchUpdateRacks(ctx, []*ufspb.Rack{rack})
	if err != nil {
		return nil, errors.Annotate(err, "Unable to update rack %s with switch %s information", rack.Name, switchName).Err()
	}
	return LogRackChanges(old, rack), nil
}

// getRackForSwitch return rack associated with the switch.
func getRackForSwitch(ctx context.Context, switchName string) (*ufspb.Rack, error) {
	racks, err := registration.QueryRackByPropertyName(ctx, "switch_ids", switchName, false)
	if err != nil {
		return nil, errors.Annotate(err, "Unable to query rack for switch %s", switchName).Err()
	}
	if racks == nil || len(racks) == 0 {
		errorMsg := fmt.Sprintf("No rack associated with the switch %s. Data discrepancy error.\n", switchName)
		return nil, status.Errorf(codes.Internal, errorMsg)
	}
	if len(racks) > 1 {
		errorMsg := fmt.Sprintf("More than one rack associated the switch %s. Data discrepancy error.\n", switchName)
		return nil, status.Errorf(codes.Internal, errorMsg)
	}
	return racks[0], nil
}

// removeSwitchFromRacks removes the switch info from racks and
// updates the racks in datastore.
// Must be called within a transaction as BatchUpdateRacks is a non-atomic operation
func removeSwitchFromRacks(ctx context.Context, racks []*ufspb.Rack, id string) ([]*ufspb.ChangeEvent, error) {
	changes := make([]*ufspb.ChangeEvent, 0)
	for _, rack := range racks {
		if rack.GetChromeBrowserRack() == nil {
			errorMsg := fmt.Sprintf("Rack %s is not a browser rack", rack.Name)
			return nil, status.Errorf(codes.FailedPrecondition, errorMsg)
		}
		switches := rack.GetChromeBrowserRack().GetSwitches()
		switches = ufsUtil.RemoveStringEntry(switches, id)
		old := proto.Clone(rack).(*ufspb.Rack)
		rack.GetChromeBrowserRack().Switches = switches
		changes = append(changes, LogRackChanges(old, rack)...)
	}
	_, err := registration.BatchUpdateRacks(ctx, racks)
	if err != nil {
		return nil, errors.Annotate(err, "Unable to remove switch information %s from rack", id).Err()
	}
	return changes, nil
}
