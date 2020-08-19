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
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	ufsUtil "infra/unifiedfleet/app/util"
)

// RackRegistration creates a new rack, switches, kvms and rpms in datastore.
func RackRegistration(ctx context.Context, rack *ufspb.Rack) (*ufspb.Rack, error) {
	switches := rack.GetChromeBrowserRack().GetSwitchObjects()
	kvms := rack.GetChromeBrowserRack().GetKvmObjects()
	rpms := rack.GetChromeBrowserRack().GetRpmObjects()
	f := func(ctx context.Context) error {
		hc := getRackClientHistory(rack)
		// Validate the input
		if err := validateRackRegistration(ctx, rack); err != nil {
			return err
		}

		// Create switches
		if switches != nil {
			for _, sw := range switches {
				// Fill the rack/zone to switch OUTPUT only fields for indexing
				sw.Rack = rack.GetName()
				sw.Zone = rack.GetLocation().GetZone().String()
			}
			if _, err := registration.BatchUpdateSwitches(ctx, switches); err != nil {
				return errors.Annotate(err, "Failed to create switches").Err()
			}
		}

		// Create kvms
		if kvms != nil {
			for _, kvm := range kvms {
				// Fill the rack/zone to kvm OUTPUT only fields for indexing
				kvm.Rack = rack.GetName()
				kvm.Zone = rack.GetLocation().GetZone().String()
			}
			if _, err := registration.BatchUpdateKVMs(ctx, kvms); err != nil {
				return errors.Annotate(err, "Failed to create KVMs").Err()
			}
		}

		// Create rpms
		if rpms != nil {
			for _, rpm := range rpms {
				// Fill the rack/zone to rpm OUTPUT only fields for indexing
				rpm.Rack = rack.GetName()
				rpm.Zone = rack.GetLocation().GetZone().String()
			}
			if _, err := registration.BatchUpdateRPMs(ctx, rpms); err != nil {
				return errors.Annotate(err, "Failed to create RPMs").Err()
			}
		}

		// Create rack
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err := registration.BatchUpdateRacks(ctx, []*ufspb.Rack{rack}); err != nil {
			return errors.Annotate(err, "Failed to create rack %s", rack.Name).Err()
		}
		if rack.GetChromeBrowserRack() != nil {
			rack.GetChromeBrowserRack().SwitchObjects = switches
			rack.GetChromeBrowserRack().KvmObjects = kvms
			rack.GetChromeBrowserRack().RpmObjects = rpms
		}

		hc.LogAddRackChanges(rack, switches, kvms, rpms)
		hc.stUdt.addRackStateHelper(ctx, rack)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to register rack: %s", err)
		return nil, err
	}
	return rack, nil
}

// UpdateRack updates rack in datastore.
func UpdateRack(ctx context.Context, rack *ufspb.Rack, mask *field_mask.FieldMask) (*ufspb.Rack, error) {
	var oldRack *ufspb.Rack
	var err error
	f := func(ctx context.Context) error {
		hc := getRackClientHistory(rack)
		// Validate input
		if err := validateUpdateRack(ctx, rack, mask); err != nil {
			return errors.Annotate(err, "UpdateRack - validation failed").Err()
		}

		// Make sure OUTPUT_ONLY fields are set to empty
		// for OS rack we dont do anything as of now.
		if rack.GetChromeBrowserRack() != nil {
			// switches/kvms/rpms is not allowed to be updated in UpdateRack call.
			// We dont store switches/kvms/rpms object inside Rack object in Rack table.
			// switches/kvms/rpms objects are stored in their separate tables
			// user has to use switch/kvm/rpm CRUD apis to update switch/kvm/rpm
			rack.GetChromeBrowserRack().RpmObjects = nil
			rack.GetChromeBrowserRack().KvmObjects = nil
			rack.GetChromeBrowserRack().SwitchObjects = nil
		}

		// Get the existing/old rack
		oldRack, err = registration.GetRack(ctx, rack.GetName())
		oldRackCopy := proto.Clone(oldRack).(*ufspb.Rack)
		if err != nil {
			return errors.Annotate(err, "UpdateRack - get rack %s failed", rack.GetName()).Err()
		}
		// Fill the OUTPUT only fields with existing values
		rack.State = oldRack.GetState()

		// Do not let updating from browser to os or vice versa change for rack.
		if oldRack.GetChromeBrowserRack() != nil && rack.GetChromeosRack() != nil {
			return status.Error(codes.InvalidArgument, "UpdateRack - cannot update a browser rack to os rack. Please delete the browser rack and create a new os rack")
		}
		if oldRack.GetChromeosRack() != nil && rack.GetChromeBrowserRack() != nil {
			return status.Error(codes.InvalidArgument, "UpdateRack - cannot update an os rack to browser rack. Please delete the os rack and create a new browser rack")
		}

		// Partial update by field mask
		if mask != nil && len(mask.Paths) > 0 {
			rack, err = processRackUpdateMask(ctx, oldRack, rack, mask, hc)
			if err != nil {
				return errors.Annotate(err, "UpdateRack - processing update mask failed").Err()
			}
		} else if rack.GetLocation().GetZone() != oldRack.GetLocation().GetZone() {
			// this check is for json input with complete update rack
			// Check if rack zone information is changed/updated
			indexMap := map[string]string{"zone": rack.GetLocation().GetZone().String()}
			oldIndexMap := map[string]string{"zone": oldRack.GetLocation().GetZone().String()}
			if err = updateIndexingForRackResources(ctx, rack.GetName(), indexMap, oldIndexMap, hc); err != nil {
				return errors.Annotate(err, "UpdateRack - update zone indexing failed").Err()
			}
		}

		// Update the rack
		if _, err := registration.BatchUpdateRacks(ctx, []*ufspb.Rack{rack}); err != nil {
			return errors.Annotate(err, "UpdateRack - unable to batch update rack %s", rack.Name).Err()
		}
		hc.LogRackChanges(oldRackCopy, rack)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "UpdateRack - failed to update rack %s in datastore", rack.Name).Err()
	}
	if oldRack.GetChromeBrowserRack() != nil {
		// We fill the rack object with its switches/kvms/rpms from switch/kvm/rpm table
		setRack(ctx, rack)
	}
	return rack, nil
}

// processRackUpdateMask process update field mask to get only specific update
// fields and return a complete rack object with updated and existing fields
func processRackUpdateMask(ctx context.Context, oldRack *ufspb.Rack, rack *ufspb.Rack, mask *field_mask.FieldMask, hc *HistoryClient) (*ufspb.Rack, error) {
	// update the fields in the existing rack
	for _, path := range mask.Paths {
		switch path {
		case "zone":
			indexMap := map[string]string{"zone": rack.GetLocation().GetZone().String()}
			oldIndexMap := map[string]string{"zone": oldRack.GetLocation().GetZone().String()}
			if err := updateIndexingForRackResources(ctx, rack.GetName(), indexMap, oldIndexMap, hc); err != nil {
				return nil, errors.Annotate(err, "processRackUpdateMask - failed to update zone indexing").Err()
			}
			if oldRack.GetLocation() == nil {
				oldRack.Location = &ufspb.Location{}
			}
			oldRack.GetLocation().Zone = rack.GetLocation().GetZone()
		case "capacity":
			oldRack.CapacityRu = rack.GetCapacityRu()
		case "tags":
			oldRack.Tags = mergeTags(oldRack.GetTags(), rack.GetTags())
		}
	}
	// return existing/old rack with new updated values
	return oldRack, nil
}

// updateIndexingForRackResources updates indexing for kvm/rpm/switch tables
// can be used inside a transaction
func updateIndexingForRackResources(ctx context.Context, rackName string, indexMap, oldIndexMap map[string]string, hc *HistoryClient) error {
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
		case "zone":
			for _, kvm := range kvms {
				kvm.Zone = v
			}
			for _, rpm := range rpms {
				rpm.Zone = v
			}
			for _, s := range switches {
				s.Zone = v
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
	hc.LogRackLocationChanges(kvms, switches, rpms, indexMap, oldIndexMap)
	return nil
}

// GetRack returns rack for the given id from datastore.
func GetRack(ctx context.Context, id string) (*ufspb.Rack, error) {
	rack, err := registration.GetRack(ctx, id)
	if err != nil {
		return nil, err
	}
	setRack(ctx, rack)
	return rack, nil
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
	filterMap = resetStateFilter(filterMap)
	filterMap = resetZoneFilter(filterMap)
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

		// Get the rack
		rack, err := registration.GetRack(ctx, id)
		if status.Code(err) == codes.Internal {
			return err
		}
		if rack == nil {
			return status.Errorf(codes.NotFound, ufsds.NotFound)
		}

		// Check if any other resource references this rack.
		if err = validateDeleteRack(ctx, id); err != nil {
			return err
		}

		var switchIDs []string
		var kvmIDs []string
		var rpmIDs []string
		//Only for a browser rack
		if rack.GetChromeBrowserRack() != nil {
			switchIDs, err = getDeleteSwitchIDs(ctx, rack.GetName())
			if err != nil {
				return err
			}
			kvmIDs, err = getDeleteKVMIDs(ctx, rack.GetName())
			if err != nil {
				return err
			}
			rpmIDs, err = getDeleteRPMIDs(ctx, rack.GetName())
			if err != nil {
				return err
			}
			if switchIDs != nil && len(switchIDs) > 0 {
				if err = registration.BatchDeleteSwitches(ctx, switchIDs); err != nil {
					return errors.Annotate(err, "Failed to delete associated switches %s", switchIDs).Err()
				}
			}
			if kvmIDs != nil && len(kvmIDs) > 0 {
				if err = registration.BatchDeleteKVMs(ctx, kvmIDs); err != nil {
					return errors.Annotate(err, "Failed to delete associated KVMs %s", kvmIDs).Err()
				}
			}
			if rpmIDs != nil && len(rpmIDs) > 0 {
				if err = registration.BatchDeleteRPMs(ctx, rpmIDs); err != nil {
					return errors.Annotate(err, "Failed to delete associated RPMs %s", rpmIDs).Err()
				}
			}
		}

		// 6. Delete the rack
		if err := registration.DeleteRack(ctx, id); err != nil {
			return err
		}
		hc.LogDeleteRackChanges(id, switchIDs, kvmIDs, rpmIDs)
		hc.stUdt.deleteRackStateHelper(ctx, rack, switchIDs, kvmIDs, rpmIDs)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete rack and its associated switches, rpms and kvms in datastore: %s", err)
		return err
	}
	return nil
}

// can be called inside a transaction
func getDeleteSwitchIDs(ctx context.Context, rackName string) ([]string, error) {
	switches, err := registration.QuerySwitchByPropertyName(ctx, "rack", rackName, true)
	if err != nil {
		return nil, errors.Annotate(err, "DeleteRack - failed to query switches for rack %s", rackName).Err()
	}
	switchIDs := make([]string, 0, len(switches))
	for _, s := range switches {
		if err := validateDeleteSwitch(ctx, s.GetName()); err != nil {
			return nil, errors.Annotate(err, "validation failed - Unable to delete switch %s", s.GetName()).Err()
		}
		switchIDs = append(switchIDs, s.GetName())
	}
	return switchIDs, nil
}

// can be called inside a transaction
func getDeleteKVMIDs(ctx context.Context, rackName string) ([]string, error) {
	kvms, err := registration.QueryKVMByPropertyName(ctx, "rack", rackName, true)
	if err != nil {
		return nil, errors.Annotate(err, "DeleteRack - failed to query kvms for rack %s", rackName).Err()
	}
	var kvmIDs []string
	for _, kvm := range kvms {
		if err := validateDeleteKVM(ctx, kvm.GetName()); err != nil {
			return nil, errors.Annotate(err, "validation failed - Unable to delete kvm %s", kvm.GetName()).Err()
		}
		kvmIDs = append(kvmIDs, kvm.GetName())
	}
	return kvmIDs, nil
}

// can be called inside a transaction
func getDeleteRPMIDs(ctx context.Context, rackName string) ([]string, error) {
	rpms, err := registration.QueryRPMByPropertyName(ctx, "rack", rackName, true)
	if err != nil {
		return nil, errors.Annotate(err, "DeleteRack - failed to query rpms for rack %s", rackName).Err()
	}
	var rpmIDs []string
	for _, rpm := range rpms {
		if err := validateDeleteRPM(ctx, rpm.GetName()); err != nil {
			return nil, errors.Annotate(err, "validation failed - Unable to delete rpm %s", rpm.GetName()).Err()
		}
		rpmIDs = append(rpmIDs, rpm.GetName())
	}
	return rpmIDs, nil
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
// checks the zone in the location to decide between browser/chromeos rack
func validateCreateRack(ctx context.Context, rack *ufspb.Rack) error {
	if rack.GetChromeBrowserRack() == nil && rack.GetChromeosRack() == nil {
		if rack.GetLocation() == nil || rack.GetLocation().GetZone() == ufspb.Zone_ZONE_UNSPECIFIED {
			return errors.New("zone information in the location object cannot be empty/unspecified for a rack")
		}
		if ufsUtil.IsInBrowserZone(rack.GetLocation().GetZone().String()) {
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

// validateRackRegistration validates if a rack, switches, kvms and rpms can be created in the datastore.
//
// checks if the resources rack/switches/kvms/rpms already exists in the system.
// checks if resources referenced by rack/switches/kvms/rpms does not exist in the system.
func validateRackRegistration(ctx context.Context, rack *ufspb.Rack) error {
	if rack == nil {
		return errors.New("rack cannot be empty")
	}

	if rack.GetChromeBrowserRack() == nil && rack.GetChromeosRack() == nil {
		if rack.GetLocation() == nil || rack.GetLocation().GetZone() == ufspb.Zone_ZONE_UNSPECIFIED {
			return errors.New("zone information in the location object cannot be empty/unspecified for a rack")
		}
		if ufsUtil.IsInBrowserZone(rack.GetLocation().GetZone().String()) {
			rack.Rack = &ufspb.Rack_ChromeBrowserRack{
				ChromeBrowserRack: &ufspb.ChromeBrowserRack{},
			}
		} else {
			rack.Rack = &ufspb.Rack_ChromeosRack{
				ChromeosRack: &ufspb.ChromeOSRack{},
			}
		}
	}

	var resourcesAlreadyExists []*Resource
	var resourcesNotFound []*Resource
	var switches []*ufspb.Switch
	var kvms []*ufspb.KVM
	var rpms []*ufspb.RPM
	if rack.GetChromeBrowserRack() != nil {
		switches = rack.GetChromeBrowserRack().GetSwitchObjects()
		kvms = rack.GetChromeBrowserRack().GetKvmObjects()
		rpms = rack.GetChromeBrowserRack().GetRpmObjects()
	}
	// Aggregate resources to check if rack already exists
	resourcesAlreadyExists = append(resourcesAlreadyExists, GetRackResource(rack.Name))

	if switches != nil {
		for _, s := range switches {
			// Aggregate resources to check if switch already exists
			resourcesAlreadyExists = append(resourcesAlreadyExists, GetSwitchResource(s.Name))
		}
	}

	if kvms != nil {
		for _, kvm := range kvms {
			// Aggregate resources to check if kvm already exists
			resourcesAlreadyExists = append(resourcesAlreadyExists, GetKVMResource(kvm.Name))

			// Aggregate resource to check if resources referenced by the kvm does not exist
			if chromePlatformID := kvm.GetChromePlatform(); chromePlatformID != "" {
				resourcesNotFound = append(resourcesNotFound, GetChromePlatformResource(chromePlatformID))
			}
		}
	}

	if rpms != nil {
		for _, rpm := range rpms {
			// Aggregate resources to check if rpm already exists
			resourcesAlreadyExists = append(resourcesAlreadyExists, GetRPMResource(rpm.Name))
		}
	}

	// Check if rack/switches/kvms/rpms already exists
	if err := resourceAlreadyExists(ctx, resourcesAlreadyExists, nil); err != nil {
		return err
	}

	// Check if resources referenced by rack/switches/kvms/rpms does not exist
	return ResourceExist(ctx, resourcesNotFound, nil)
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
			case "zone":
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

//setRack populates the rack object with switches and drac
func setRack(ctx context.Context, rack *ufspb.Rack) {
	// get switches for rack
	switches, err := registration.QuerySwitchByPropertyName(ctx, "rack", rack.GetName(), false)
	if err != nil {
		// Just log a warning message and dont fail operation
		logging.Warningf(ctx, "GetRack - failed to query switches for rack %s: %s", rack.GetName(), err)
	}
	setSwitchesToRack(rack, switches)

	// get kvms for rack
	kvms, err := registration.QueryKVMByPropertyName(ctx, "rack", rack.GetName(), false)
	if err != nil {
		// Just log a warning message and dont fail operation
		logging.Warningf(ctx, "GetRack - failed to query kvms for rack %s: %s", rack.GetName(), err)
	}
	setKVMsToRack(rack, kvms)

	// get rpms for rack
	rpms, err := registration.QueryRPMByPropertyName(ctx, "rack", rack.GetName(), false)
	if err != nil {
		// Just log a warning message and dont fail operation
		logging.Warningf(ctx, "GetRack - failed to query rpms for rack %s: %s", rack.GetName(), err)
	}
	setRPMsToRack(rack, rpms)
}

func setSwitchesToRack(rack *ufspb.Rack, switches []*ufspb.Switch) {
	if len(switches) <= 0 {
		return
	}
	if rack.GetChromeBrowserRack() == nil {
		rack.Rack = &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{
				SwitchObjects: switches,
			},
		}
	} else {
		rack.GetChromeBrowserRack().SwitchObjects = switches
	}
}

func setKVMsToRack(rack *ufspb.Rack, kvms []*ufspb.KVM) {
	if len(kvms) <= 0 {
		return
	}
	if rack.GetChromeBrowserRack() == nil {
		rack.Rack = &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{
				KvmObjects: kvms,
			},
		}
	} else {
		rack.GetChromeBrowserRack().KvmObjects = kvms
	}
}

func setRPMsToRack(rack *ufspb.Rack, rpms []*ufspb.RPM) {
	if len(rpms) <= 0 {
		return
	}
	if rack.GetChromeBrowserRack() == nil {
		rack.Rack = &ufspb.Rack_ChromeBrowserRack{
			ChromeBrowserRack: &ufspb.ChromeBrowserRack{
				RpmObjects: rpms,
			},
		}
	} else {
		rack.GetChromeBrowserRack().RpmObjects = rpms
	}
}
