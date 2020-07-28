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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	ufsUtil "infra/unifiedfleet/app/util"
)

// CreateRack creates a new rack in datastore.
func CreateRack(ctx context.Context, rack *ufspb.Rack) (*ufspb.Rack, error) {
	if err := validateCreateRack(ctx, rack); err != nil {
		return nil, err
	}
	// Make sure OUTPUT_ONLY fields are set to empty
	if rack.GetChromeBrowserRack() != nil {
		// These are output only field. User is not allowed to set these value.
		// Overwrite it with empty values.
		rack.GetChromeBrowserRack().Rpms = nil
		rack.GetChromeBrowserRack().Kvms = nil
		rack.GetChromeBrowserRack().Switches = nil
	}
	r, err := registration.CreateRack(ctx, rack)
	if err == nil {
		SaveChangeEvents(ctx, LogRackChanges(nil, rack))
	}
	return r, err
}

// UpdateRack updates rack in datastore.
func UpdateRack(ctx context.Context, rack *ufspb.Rack) (*ufspb.Rack, error) {
	var oldRack *ufspb.Rack
	var err error
	f := func(ctx context.Context) error {
		// 1. Check if rack does not exist
		if err := ResourceExist(ctx, []*Resource{GetRackResource(rack.Name)}, nil); err != nil {
			return err
		}

		// 2. Make sure OUTPUT_ONLY fields are set to empty
		// for OS rack we dont do anything as of now.
		if rack.GetChromeBrowserRack() != nil {
			// These are output only field. User is not allowed to set these value.
			// Overwrite it with empty values.
			rack.GetChromeBrowserRack().Rpms = nil
			rack.GetChromeBrowserRack().Kvms = nil
			rack.GetChromeBrowserRack().Switches = nil
		}

		// 3. Get the existing/old rack
		oldRack, err = registration.GetRack(ctx, rack.GetName())
		if err != nil {
			return err
		}

		// 4. Make sure OUTPUT_ONLY fields are overwritten with old values
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

		// 5. Update the rack
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err := registration.BatchUpdateRacks(ctx, []*ufspb.Rack{rack}); err != nil {
			return err
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to update rack in datastore: %s", err)
		return nil, err
	}
	SaveChangeEvents(ctx, LogRackChanges(oldRack, rack))
	return rack, err
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
	var rack *ufspb.Rack
	var err error
	f := func(ctx context.Context) error {
		// 1. Get the rack
		rack, err = registration.GetRack(ctx, id)
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
				// we use this func as it is a non-atomic operation and can be used to
				// run within a transaction to make it atomic. Datastore doesnt allow
				// nested transactions.
				if err = registration.BatchDeleteSwitches(ctx, switchIDs); err != nil {
					return errors.Annotate(err, "Failed to delete associated switches %s", switchIDs).Err()
				}
			}

			// 4. Delete the KVMs
			if kvmIDs := rack.GetChromeBrowserRack().GetKvms(); kvmIDs != nil {
				// we use this func as it is a non-atomic operation and can be used to
				// run within a transaction to make it atomic. Datastore doesnt allow
				// nested transactions.
				if err = registration.BatchDeleteKVMs(ctx, kvmIDs); err != nil {
					return errors.Annotate(err, "Failed to delete associated KVMs %s", kvmIDs).Err()
				}
			}

			// 5. Delete the RPMs
			if rpmIDs := rack.GetChromeBrowserRack().GetRpms(); rpmIDs != nil {
				// we use this func as it is a non-atomic operation and can be used to
				// run within a transaction to make it atomic. Datastore doesnt allow
				// nested transactions.
				if err = registration.BatchDeleteRPMs(ctx, rpmIDs); err != nil {
					return errors.Annotate(err, "Failed to delete associated RPMs %s", rpmIDs).Err()
				}
			}
		}

		// 6. Delete the rack
		return registration.DeleteRack(ctx, id)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete rack and its associated switches, rpms and kvms in datastore: %s", err)
		return err
	}
	SaveChangeEvents(ctx, LogRackChanges(rack, nil))
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
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to replace entity in datastore: %s", err)
		return nil, err
	}
	changes := LogRackChanges(oldRack, nil)
	changes = append(changes, LogRackChanges(nil, newRack)...)
	SaveChangeEvents(ctx, changes)
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
	if len(racklses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Rack %s cannot be deleted because there are other resources which are referring this Rack.", id))
		errorMsg.WriteString(fmt.Sprintf("\nRackLSEs referring the Rack:\n"))
		for _, racklse := range racklses {
			errorMsg.WriteString(racklse.Name + ", ")
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
