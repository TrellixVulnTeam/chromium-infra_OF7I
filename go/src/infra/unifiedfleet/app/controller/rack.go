// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"

	"go.chromium.org/gae/service/datastore"
)

// CreateRack creates a new rack in datastore.
//
// Checks if the resources referenced by the Rack input already exists
// in the system before creating a new Rack
func CreateRack(ctx context.Context, rack *fleet.Rack) (*fleet.Rack, error) {
	err := validateRack(ctx, rack)
	if err != nil {
		return nil, err
	}
	return registration.CreateRack(ctx, rack)
}

// UpdateRack updates rack in datastore.
//
// Checks if the resources referenced by the Rack input already exists
// in the system before updating a Rack
func UpdateRack(ctx context.Context, rack *fleet.Rack) (*fleet.Rack, error) {
	err := validateRack(ctx, rack)
	if err != nil {
		return nil, err
	}
	return registration.UpdateRack(ctx, rack)
}

// GetRack returns rack for the given id from datastore.
func GetRack(ctx context.Context, id string) (*fleet.Rack, error) {
	return registration.GetRack(ctx, id)
}

// ListRacks lists the racks
func ListRacks(ctx context.Context, pageSize int32, pageToken string) ([]*fleet.Rack, string, error) {
	return registration.ListRacks(ctx, pageSize, pageToken)
}

// DeleteRack deletes the rack in datastore
//
// For referential data intergrity,
// Delete if this Rack is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteRack(ctx context.Context, id string) error {
	err := validateDeleteRack(ctx, id)
	if err != nil {
		return err
	}
	return registration.DeleteRack(ctx, id)
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
func ReplaceRack(ctx context.Context, oldRack *fleet.Rack, newRack *fleet.Rack) (*fleet.Rack, error) {
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

		err = validateRack(ctx, newRack)
		if err != nil {
			return err
		}
		entity := &registration.RackEntity{
			ID: newRack.Name,
		}
		existsResults, err := datastore.Exists(ctx, entity)
		if err == nil {
			if existsResults.All() {
				return status.Errorf(codes.AlreadyExists, fleetds.AlreadyExists)
			}
		} else {
			logging.Errorf(ctx, "Failed to check existence: %s", err)
			return status.Errorf(codes.Internal, fleetds.InternalError)
		}

		_, err = registration.BatchUpdateRacks(ctx, []*fleet.Rack{newRack})
		if err != nil {
			return err
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to replace entity in datastore: %s", err)
		return nil, err
	}
	return newRack, nil
}

// validateRack validates if a rack can be created/updated in the datastore.
//
// Checks if the resources referenced by the given Rack input already exists
// in the system. Returns an error if any resource referenced by the Rack input
// does not exist in the system.
func validateRack(ctx context.Context, rack *fleet.Rack) error {
	var resources []*Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create Rack %s:\n", rack.Name))

	kvmIDs := rack.GetChromeBrowserRack().GetKvms()
	rpmIDs := rack.GetChromeBrowserRack().GetRpms()
	switchIDs := rack.GetChromeBrowserRack().GetSwitches()
	if len(kvmIDs) != 0 {
		for _, kvmID := range kvmIDs {
			resources = append(resources, GetKVMResource(kvmID))
		}
	}
	if len(rpmIDs) != 0 {
		for _, rpmID := range rpmIDs {
			resources = append(resources, GetRPMResource(rpmID))
		}
	}
	if len(switchIDs) != 0 {
		for _, switchID := range switchIDs {
			resources = append(resources, GetSwitchResource(switchID))
		}
	}

	return ResourceExist(ctx, resources, &errorMsg)
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
