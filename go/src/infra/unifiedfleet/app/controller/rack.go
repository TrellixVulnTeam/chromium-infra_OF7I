// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/model/state"
	ufsUtil "infra/unifiedfleet/app/util"
)

// CreateRack creates a new rack in datastore.
func CreateRack(ctx context.Context, rack *ufspb.Rack) (*ufspb.Rack, error) {
	f := func(ctx context.Context) error {
		hc := getRackClientHistory(rack)
		if err := validateCreateRack(ctx, rack); err != nil {
			return err
		}
		// Make sure OUTPUT_ONLY fields are set to empty
		if rack.GetChromeBrowserRack() != nil {
			// These are output only field. User is not allowed to set these value.
			// Overwrite it with empty values.
			rack.GetChromeBrowserRack().Rpms = nil
			rack.GetChromeBrowserRack().Kvms = nil
			rack.GetChromeBrowserRack().Switches = nil
		}
		if _, err := registration.BatchUpdateRacks(ctx, []*ufspb.Rack{rack}); err != nil {
			return err
		}
		hc.LogRackChanges(nil, rack)
		if err := hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_SERVING); err != nil {
			return err
		}
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create entity in datastore: %s", err)
		return nil, err
	}
	return rack, nil
}

// UpdateRack updates rack in datastore.
func UpdateRack(ctx context.Context, rack *ufspb.Rack, mask *field_mask.FieldMask) (*ufspb.Rack, error) {
	f := func(ctx context.Context) error {
		hc := getRackClientHistory(rack)
		// Validate input
		if err := validateUpdateRack(ctx, rack, mask); err != nil {
			return errors.Annotate(err, "UpdateRack - validation failed").Err()
		}

		// Make sure OUTPUT_ONLY fields are set to empty
		// for OS rack we dont do anything as of now.
		if rack.GetChromeBrowserRack() != nil {
			// These are output only field. User is not allowed to set these value.
			// Overwrite it with empty values.
			rack.GetChromeBrowserRack().Rpms = nil
			rack.GetChromeBrowserRack().Kvms = nil
			rack.GetChromeBrowserRack().Switches = nil
		}

		// Get the existing/old rack
		oldRack, err := registration.GetRack(ctx, rack.GetName())
		if err != nil {
			return errors.Annotate(err, "UpdateRack - get rack %s failed", rack.GetName()).Err()
		}

		// Do not let updating from browser to os or vice versa change for rack.
		if oldRack.GetChromeBrowserRack() != nil && rack.GetChromeosRack() != nil {
			return status.Error(codes.InvalidArgument, "UpdateRack - cannot update a browser rack to os rack. Please delete the browser rack and create a new os rack")
		}
		if oldRack.GetChromeosRack() != nil && rack.GetChromeBrowserRack() != nil {
			return status.Error(codes.InvalidArgument, "UpdateRack - cannot update an os rack to browser rack. Please delete the os rack and create a new browser rack")
		}

		// Make sure OUTPUT_ONLY fields are overwritten with old values
		// Check if the existing rack is a browser rack and not an OS rack.
		// for OS rack we dont do anything as of now as the OS rack doesnt have any
		// OUTPUT_ONLY fields. Switches/kvms/rpms for OS rack are in RackLSE as of now.
		if oldRack.GetChromeBrowserRack() != nil {
			if rack.GetChromeBrowserRack() == nil {
				rack.Rack = &ufspb.Rack_ChromeBrowserRack{
					ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
				}
			}
			// These are output only fields. Not allowed to update by the user.
			// Overwrite the input values with existing values.
			rack.GetChromeBrowserRack().Rpms = oldRack.GetChromeBrowserRack().GetRpms()
			rack.GetChromeBrowserRack().Kvms = oldRack.GetChromeBrowserRack().GetKvms()
			rack.GetChromeBrowserRack().Switches = oldRack.GetChromeBrowserRack().GetSwitches()
		}

		// Partial update by field mask
		if mask != nil && len(mask.Paths) > 0 {
			rack, err = processRackUpdateMask(ctx, oldRack, rack, mask)
			if err != nil {
				return errors.Annotate(err, "UpdateRack - processing update mask failed").Err()
			}
		} else if rack.GetLocation().GetLab() != oldRack.GetLocation().GetLab() {
			// this check is for json input with complete update rack
			// Check if rack lab information is changed/updated
			if err = updateIndexingForRackResources(ctx, rack.GetName(), map[string]string{"lab": rack.GetLocation().GetLab().String()}); err != nil {
				return errors.Annotate(err, "UpdateRack - update lab indexing failed").Err()
			}
		}

		// Update the rack
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err := registration.BatchUpdateRacks(ctx, []*ufspb.Rack{rack}); err != nil {
			return errors.Annotate(err, "UpdateRack - unable to batch update rack %s", rack.Name).Err()
		}
		hc.LogRackChanges(oldRack, rack)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "UpdateRack - failed to update rack %s in datastore", rack.Name).Err()
	}
	return rack, nil
}

// processRackUpdateMask process update field mask to get only specific update
// fields and return a complete rack object with updated and existing fields
func processRackUpdateMask(ctx context.Context, oldRack *ufspb.Rack, rack *ufspb.Rack, mask *field_mask.FieldMask) (*ufspb.Rack, error) {
	// update the fields in the existing nic
	for _, path := range mask.Paths {
		switch path {
		case "lab":
			if err := updateIndexingForRackResources(ctx, rack.GetName(), map[string]string{"lab": rack.GetLocation().GetLab().String()}); err != nil {
				return nil, errors.Annotate(err, "processRackUpdateMask - failed to update lab indexing").Err()
			}
			if oldRack.GetLocation() == nil {
				oldRack.Location = &ufspb.Location{}
			}
			oldRack.GetLocation().Lab = rack.GetLocation().GetLab()
		case "capacity":
			oldRack.CapacityRu = rack.GetCapacityRu()
		case "tags":
			oldTags := oldRack.GetTags()
			newTags := rack.GetTags()
			if newTags == nil || len(newTags) == 0 {
				oldTags = nil
			} else {
				for _, tag := range newTags {
					oldTags = append(oldTags, tag)
				}
			}
			oldRack.Tags = oldTags
		}
	}
	// return existing/old rack with new updated values
	return oldRack, nil
}

// updateIndexingForRackResources updates indexing for kvm/rpm/switch tables
// can be used inside a transaction
func updateIndexingForRackResources(ctx context.Context, rackName string, indexMap map[string]string) error {
	// get KVMs for indexing
	kvms, err := registration.QueryKVMByPropertyName(ctx, "rack", rackName, false)
	if err != nil {
		return errors.Annotate(err, "updateIndexingForRackResources - Failed to query kvms for rack %s", rackName).Err()
	}
	// get RPMs for indexing
	rpms, err := registration.QueryRPMByPropertyName(ctx, "rack", rackName, false)
	if err != nil {
		return errors.Annotate(err, "updateIndexingForRackResources - Failed to query rpms for rack %s", rackName).Err()
	}
	// get Switches for indexing
	switches, err := registration.QuerySwitchByPropertyName(ctx, "rack", rackName, false)
	if err != nil {
		return errors.Annotate(err, "updateIndexingForRackResources - Failed to query switches for rack %s", rackName).Err()
	}
	for k, v := range indexMap {
		// These are output only fields used for indexing kvm/rpm/switch table
		switch k {
		case "lab":
			for _, kvm := range kvms {
				kvm.Lab = v
			}
			for _, rpm := range rpms {
				rpm.Lab = v
			}
			for _, s := range switches {
				s.Lab = v
			}
		}
	}
	if _, err := registration.BatchUpdateKVMs(ctx, kvms); err != nil {
		return errors.Annotate(err, "updateIndexingForRackResources - Unable to update kvms").Err()
	}
	if _, err := registration.BatchUpdateRPMs(ctx, rpms); err != nil {
		return errors.Annotate(err, "updateIndexingForRackResources - Unable to update rpms").Err()
	}
	if _, err := registration.BatchUpdateSwitches(ctx, switches); err != nil {
		return errors.Annotate(err, "updateIndexingForRackResources - Unable to update switches").Err()
	}
	return nil
}

// GetRack returns rack for the given id from datastore.
func GetRack(ctx context.Context, id string) (*ufspb.Rack, error) {
	return registration.GetRack(ctx, id)
}

// ListRacks lists the racks
func ListRacks(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.Rack, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, registration.GetRackIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing Racks").Err()
		}
	}
	return registration.ListRacks(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteRack deletes the rack in datastore
//
// For referential data intergrity,
// Delete if this Rack is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteRack(ctx context.Context, id string) error {
	// [TODO]: Add logic for Chrome OS
	f := func(ctx context.Context) error {
		hc := getRackClientHistory(&ufspb.Rack{Name: id})
		resourceNames := make([]string, 0)

		// 1. Get the rack
		rack, err := registration.GetRack(ctx, id)
		if status.Code(err) == codes.Internal {
			return err
		}
		if rack == nil {
			return status.Errorf(codes.NotFound, ufsds.NotFound)
		}

		// 2. Check if any other resource references this rack.
		if err = validateDeleteRack(ctx, id); err != nil {
			return err
		}

		//Only for a browser rack
		if rack.GetChromeBrowserRack() != nil {
			// 3. Delete the switches
			if switchIDs := rack.GetChromeBrowserRack().GetSwitches(); switchIDs != nil {
				for _, switchID := range switchIDs {
					resourceNames = append(resourceNames, ufsUtil.AddPrefix(ufsUtil.SwitchCollection, switchID))
					if err := validateDeleteSwitch(ctx, switchID); err != nil {
						return errors.Annotate(err, "validation failed - Unable to delete switch %s", switchID).Err()
					}
				}
				// we use this func as it is a non-atomic operation and can be used to
				// run within a transaction to make it atomic. Datastore doesnt allow
				// nested transactions.
				if err = registration.BatchDeleteSwitches(ctx, switchIDs); err != nil {
					return errors.Annotate(err, "Failed to delete associated switches %s", switchIDs).Err()
				}
			}

			// 4. Delete the KVMs
			if kvmIDs := rack.GetChromeBrowserRack().GetKvms(); kvmIDs != nil {
				for _, kvmID := range kvmIDs {
					resourceNames = append(resourceNames, ufsUtil.AddPrefix(ufsUtil.KVMCollection, kvmID))
					if err := validateDeleteKVM(ctx, kvmID); err != nil {
						return errors.Annotate(err, "validation failed - unable to delete kvm %s", kvmID).Err()
					}
				}
				// we use this func as it is a non-atomic operation and can be used to
				// run within a transaction to make it atomic. Datastore doesnt allow
				// nested transactions.
				if err = registration.BatchDeleteKVMs(ctx, kvmIDs); err != nil {
					return errors.Annotate(err, "Failed to delete associated KVMs %s", kvmIDs).Err()
				}
			}

			// 5. Delete the RPMs
			if rpmIDs := rack.GetChromeBrowserRack().GetRpms(); rpmIDs != nil {
				for _, rpmID := range rpmIDs {
					resourceNames = append(resourceNames, ufsUtil.AddPrefix(ufsUtil.RPMCollection, rpmID))
					if err := validateDeleteRPM(ctx, rpmID); err != nil {
						return errors.Annotate(err, "validation failed - unable to delete rpm %s", rpmID).Err()
					}
				}
				// we use this func as it is a non-atomic operation and can be used to
				// run within a transaction to make it atomic. Datastore doesnt allow
				// nested transactions.
				if err = registration.BatchDeleteRPMs(ctx, rpmIDs); err != nil {
					return errors.Annotate(err, "Failed to delete associated RPMs %s", rpmIDs).Err()
				}
			}
		}
		resourceNames = append(resourceNames, ufsUtil.AddPrefix(ufsUtil.RackCollection, id))
		state.DeleteStates(ctx, resourceNames)

		// 6. Delete the rack
		if err := registration.DeleteRack(ctx, id); err != nil {
			return err
		}
		hc.LogRackChanges(rack, nil)
		hc.stUdt.deleteStateHelper(ctx)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete rack and its associated switches, rpms and kvms in datastore: %s", err)
		return err
	}
	return nil
}

// ReplaceRack replaces an old Rack with new Rack in datastore
//
// It does a delete of old rack and create of new Rack.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old Rack, it will get all the resources referencing
// the old Rack. It will update all the resources which were referencing
// the old Rack(got in the last step) with new Rack.
// Deletes the old Rack.
// Creates the new Rack.
// This will preserve data integrity in the system.
func ReplaceRack(ctx context.Context, oldRack *ufspb.Rack, newRack *ufspb.Rack) (*ufspb.Rack, error) {
	f := func(ctx context.Context) error {
		hc := getRackClientHistory(newRack)
		racklses, err := inventory.QueryRackLSEByPropertyName(ctx, "rack_ids", oldRack.Name, false)
		if err != nil {
			return err
		}
		if racklses != nil {
			for _, racklse := range racklses {
				racks := racklse.GetRacks()
				for i := range racks {
					if racks[i] == oldRack.Name {
						racks[i] = newRack.Name
						break
					}
				}
				racklse.Racks = racks
			}
			_, err := inventory.BatchUpdateRackLSEs(ctx, racklses)
			if err != nil {
				return err
			}
		}

		err = registration.DeleteRack(ctx, oldRack.Name)
		if err != nil {
			return err
		}

		entity := &registration.RackEntity{
			ID: newRack.Name,
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

		_, err = registration.BatchUpdateRacks(ctx, []*ufspb.Rack{newRack})
		if err != nil {
			return err
		}
		hc.LogRackChanges(oldRack, nil)
		hc.LogRackChanges(nil, newRack)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to replace entity in datastore: %s", err)
		return nil, err
	}
	return newRack, nil
}

// validateDeleteRack validates if a Rack can be deleted
//
// Checks if this Rack(RackID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteRack(ctx context.Context, id string) error {
	racklses, err := inventory.QueryRackLSEByPropertyName(ctx, "rack_ids", id, true)
	if err != nil {
		return err
	}
	machines, err := registration.QueryMachineByPropertyName(ctx, "rack", id, true)
	if err != nil {
		return err
	}
	if len(racklses) > 0 || len(machines) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Rack %s cannot be deleted because there are other resources which are referring this Rack.", id))
		if len(racklses) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nRackLSEs referring the Rack:\n"))
			for _, racklse := range racklses {
				errorMsg.WriteString(racklse.Name + ", ")
			}
		}
		if len(machines) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachines referring to the Rack:\n"))
			for _, machine := range machines {
				errorMsg.WriteString(machine.Name + ", ")
			}
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

// validateCreateRack validates if a Rack can be created
//
// A rack cannot exist in the system with both ChromeBrowserRack/ChromeOSRack as nil
// checks if ChromeBrowserRack/ChromeOSRack is nil and initializes the object for rack
// checks the lab in the location to decide between browser/chromeos rack
func validateCreateRack(ctx context.Context, rack *ufspb.Rack) error {
	if rack.GetChromeBrowserRack() == nil && rack.GetChromeosRack() == nil {
		if rack.GetLocation() == nil || rack.GetLocation().GetLab() == ufspb.Lab_LAB_UNSPECIFIED {
			return errors.New("lab information in the location object cannot be empty/unspecified for a rack")
		}
		if ufsUtil.IsInBrowserLab(rack.GetLocation().GetLab().String()) {
			rack.Rack = &ufspb.Rack_ChromeBrowserRack{
				ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
			}
		} else {
			rack.Rack = &ufspb.Rack_ChromeosRack{
				ChromeosRack: &ufspb.ChromeOSRack{},
			}
		}
	}
	return nil
}

// validateUpdateRack validates if a rack can be updated
func validateUpdateRack(ctx context.Context, rack *ufspb.Rack, mask *field_mask.FieldMask) error {
	// check if resources does not exist
	if err := ResourceExist(ctx, []*Resource{GetRackResource(rack.Name)}, nil); err != nil {
		return err
	}

	return validateRackUpdateMask(rack, mask)
}

// validateRackUpdateMask validates the update mask for Rack update
func validateRackUpdateMask(rack *ufspb.Rack, mask *field_mask.FieldMask) error {
	if mask != nil {
		// validate the give field mask
		for _, path := range mask.Paths {
			switch path {
			case "name":
				return status.Error(codes.InvalidArgument, "validateUpdateRack - name cannot be updated, delete and create a new rack instead")
			case "update_time":
				return status.Error(codes.InvalidArgument, "validateUpdateRack - update_time cannot be updated, it is a output only field")
			case "lab":
				if rack.GetLocation() == nil {
					return status.Error(codes.InvalidArgument, "validateUpdateRack - location cannot be empty/nil.")
				}
			case "capacity":
			case "tags":
				// valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "validateUpdateRack - unsupported update mask path %q", path)
			}
		}
	}
	return nil
}

func getRackClientHistory(m *ufspb.Rack) *HistoryClient {
	return &HistoryClient{
		stUdt: &stateUpdater{
			ResourceName: ufsUtil.AddPrefix(ufsUtil.RackCollection, m.Name),
		},
		netUdt: &networkUpdater{
			Hostname: m.Name,
		},
	}
}
