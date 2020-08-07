// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/registration"
)

// CreateDrac creates a new drac in datastore.
//
// Checks if the resources referenced by the Drac input already exists
// in the system before creating a new Drac
func CreateDrac(ctx context.Context, drac *ufspb.Drac, machineName string) (*ufspb.Drac, error) {
	changes := LogDracChanges(nil, drac)
	f := func(ctx context.Context) error {
		// 1. Validate input
		if err := validateCreateDrac(ctx, drac, machineName); err != nil {
			return err
		}

		// 2. Get browser machine to associate the drac
		machine, err := getBrowserMachine(ctx, machineName)
		if err != nil {
			return err
		}

		// Fill the machine/rack/lab to drac OUTPUT only fields
		drac.Machine = machine.GetName()
		drac.Rack = machine.GetLocation().GetRack()
		drac.Lab = machine.GetLocation().GetLab().String()

		// 3. Remove the drac associated with browser machine from the system.
		// The existing old drac in the browser machine will be replaced with this new drac,
		// A drac cannot exist in the system without being associated to a machine,
		// so we delete the old drac from the system.
		if oldDracName := machine.GetChromeBrowserMachine().GetDrac(); oldDracName != "" {
			if err := registration.DeleteDrac(ctx, oldDracName); err != nil {
				return errors.Annotate(err, "%s drac not dound in the system. Deletion for drac %s failed.", oldDracName, oldDracName).Err()
			}
			changes = append(changes, LogDracChanges(&ufspb.Drac{Name: oldDracName}, nil)...)
		}

		// 4. Update the browser machine with new drac information
		if cs, err := addDracToBrowserMachine(ctx, machine, drac.Name); err == nil {
			changes = append(changes, cs...)
		} else {
			return err
		}

		// 5. Create a drac entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err = registration.BatchUpdateDracs(ctx, []*ufspb.Drac{drac}); err != nil {
			return errors.Annotate(err, "Unable to create drac %s", drac.Name).Err()
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create drac in datastore: %s", err)
		return nil, err
	}
	// Log the changes
	SaveChangeEvents(ctx, changes)
	return drac, nil
}

// UpdateDrac updates drac in datastore.
//
// Checks if the resources referenced by the Drac input already exists
// in the system before updating a Drac
func UpdateDrac(ctx context.Context, drac *ufspb.Drac, machineName string, mask *field_mask.FieldMask) (*ufspb.Drac, error) {
	changes := make([]*ufspb.ChangeEvent, 0)
	f := func(ctx context.Context) error {
		// 1. Validate the input
		if err := validateUpdateDrac(ctx, drac, machineName, mask); err != nil {
			return errors.Annotate(err, "UpdateDrac - validation failed").Err()
		}

		oldDrac, err := registration.GetDrac(ctx, drac.GetName())
		if err != nil {
			return errors.Annotate(err, "UpdateDrac - get drac %s failed", drac.GetName()).Err()
		}
		// Copy the machine/rack/lab to drac OUTPUT only fields from already existing drac
		drac.Machine = oldDrac.GetMachine()
		drac.Rack = oldDrac.GetRack()
		drac.Lab = oldDrac.GetLab()
		changes = append(changes, LogDracChanges(oldDrac, drac)...)
		if machineName != "" {
			// 2. Get the old browser machine associated with drac
			oldMachine, err := getBrowserMachineForDrac(ctx, drac.Name)
			if err != nil {
				return errors.Annotate(err, "UpdateDrac - query machine for drac %s failed", drac.GetName()).Err()
			}

			// User is trying to associate this drac with a different browser machine.
			if oldMachine.Name != machineName {
				// 3. Remove the association between old browser machine and this drac.
				if cs, err := removeDracFromBrowserMachines(ctx, []*ufspb.Machine{oldMachine}); err == nil {
					changes = append(changes, cs...)
				} else {
					return errors.Annotate(err, "UpdateDrac - remove drac %s from browser machine %s failed", drac.Name, oldMachine.GetName()).Err()
				}

				// 4. Get new browser machine to associate the drac
				machine, err := getBrowserMachine(ctx, machineName)
				if err != nil {
					return errors.Annotate(err, "UpdateDrac - get browser machine %s failed", machineName).Err()
				}

				// Fill the machine/rack/lab to drac OUTPUT only fields
				drac.Machine = machine.GetName()
				drac.Rack = machine.GetLocation().GetRack()
				drac.Lab = machine.GetLocation().GetLab().String()

				// 5. Remove the drac associated with new browser machine from the system.
				// The existing old drac in the new browser machine will be replaced with this new drac,
				// A drac cannot exist in the system without being associated to a machine,
				// so we delete the old drac from the system.
				if oldDracName := machine.GetChromeBrowserMachine().GetDrac(); oldDracName != "" {
					if err := registration.DeleteDrac(ctx, oldDracName); err != nil {
						return errors.Annotate(err, "UpdateDrac - drac %s not dound in the system. Deletion for drac %s failed.", oldDracName, oldDracName).Err()
					}
					changes = append(changes, LogDracChanges(&ufspb.Drac{Name: oldDracName}, nil)...)
				}

				// 6. Update the new browser machine with new drac information
				if cs, err := addDracToBrowserMachine(ctx, machine, drac.Name); err == nil {
					changes = append(changes, cs...)
				} else {
					return errors.Annotate(err, "UpdateDrac - add drac %s from browser machine %s failed", drac.Name, machine.GetName()).Err()
				}
			}
		}

		// Partial update by field mask
		if mask != nil && len(mask.Paths) > 0 {
			drac, err = processDracUpdateMask(oldDrac, drac, mask)
			if err != nil {
				return errors.Annotate(err, "UpdateDrac - processing update mask failed").Err()
			}
		} else {
			// This check is for json file input
			// User is not allowed to update mac address of a drac
			// instead user has to delete the old drac and add new drac with new mac address
			// macaddress is associated with DHCP config, so we dont allow mac address update for a drac
			if oldDrac.GetMacAddress() != "" && oldDrac.GetMacAddress() != drac.GetMacAddress() {
				return status.Error(codes.InvalidArgument, "UpdateDrac - This drac's mac address is already set. "+
					"Updating mac address for the drac is not allowed.\nInstead delete the drac and add a new drac with updated mac address.")
			}
		}

		// 7. Update drac entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction. Datastore doesnt allow nested transactions.
		if _, err := registration.BatchUpdateDracs(ctx, []*ufspb.Drac{drac}); err != nil {
			return errors.Annotate(err, "UpdateDrac - unable to batch update drac %s", drac.Name).Err()
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "UpdateDrac - failed to update drac %s in datastore", drac.Name).Err()
	}
	SaveChangeEvents(ctx, changes)
	return drac, nil
}

// processDracUpdateMask process update field mask to get only specific update
// fields and return a complete drac object with updated and existing fields
func processDracUpdateMask(oldDrac *ufspb.Drac, drac *ufspb.Drac, mask *field_mask.FieldMask) (*ufspb.Drac, error) {
	// update the fields in the existing drac
	for _, path := range mask.Paths {
		switch path {
		case "machine":
			// In the previous step we have already checked for machineName != ""
			// and got the new values for OUTPUT only fields in new drac object,
			// assign them to oldDrac.
			oldDrac.Machine = drac.GetMachine()
			oldDrac.Rack = drac.GetRack()
			oldDrac.Lab = drac.GetLab()
		case "macAddress":
			if oldDrac.GetMacAddress() != "" {
				return oldDrac, status.Error(codes.InvalidArgument, "processDracUpdateMask - This drac's mac address is already set. "+
					"Updating mac address for the drac is not allowed.\nInstead delete the drac and add a new drac with updated mac address.")
			}
			oldDrac.MacAddress = drac.GetMacAddress()
		case "switch":
			if oldDrac.GetSwitchInterface() == nil {
				oldDrac.SwitchInterface = &ufspb.SwitchInterface{
					Switch: drac.GetSwitchInterface().GetSwitch(),
				}
			} else {
				oldDrac.GetSwitchInterface().Switch = drac.GetSwitchInterface().GetSwitch()
			}
		case "port":
			if oldDrac.GetSwitchInterface() == nil {
				oldDrac.SwitchInterface = &ufspb.SwitchInterface{
					Port: drac.GetSwitchInterface().GetPort(),
				}
			} else {
				oldDrac.GetSwitchInterface().Port = drac.GetSwitchInterface().GetPort()
			}
		case "tags":
			oldTags := oldDrac.GetTags()
			newTags := drac.GetTags()
			if newTags == nil || len(newTags) == 0 {
				oldTags = nil
			} else {
				for _, tag := range newTags {
					oldTags = append(oldTags, tag)
				}
			}
			oldDrac.Tags = oldTags
		}
	}
	// return existing/old drac with new updated values
	return oldDrac, nil
}

// DeleteDracHost deletes the host of a drac in datastore.
func DeleteDracHost(ctx context.Context, dracName string) error {
	f := func(ctx context.Context) error {
		nu := &networkUpdater{
			Hostname: dracName,
		}
		if err := nu.deleteDHCPHelper(ctx); err != nil {
			return err
		}
		return SaveChangeEvents(ctx, nu.Changes)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete the drac host: %s", err)
		return err
	}
	return nil
}

// UpdateDracHost updates the drac host in datastore.
func UpdateDracHost(ctx context.Context, drac *ufspb.Drac, nwOpt *ufsAPI.NetworkOption) error {
	f := func(ctx context.Context) error {
		// 1. Validate the input
		if err := validateUpdateDracHost(ctx, drac, nwOpt.GetVlan(), nwOpt.GetIp()); err != nil {
			return err
		}

		nu := &networkUpdater{
			Hostname: drac.GetName(),
		}
		// 2. Verify if the hostname is already set with IP. if yes, remove the current dhcp.
		if err := nu.deleteDHCPHelper(ctx); err != nil {
			return err
		}

		// 3. Find free ip, set IP and DHCP config
		if _, err := nu.addHostHelper(ctx, nwOpt.GetVlan(), nwOpt.GetIp(), drac.GetMacAddress()); err != nil {
			return err
		}
		return SaveChangeEvents(ctx, nu.Changes)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to assign IP to the drac: %s", err)
		return err
	}
	return nil
}

// GetDrac returns drac for the given id from datastore.
func GetDrac(ctx context.Context, id string) (*ufspb.Drac, error) {
	return registration.GetDrac(ctx, id)
}

// ListDracs lists the dracs
func ListDracs(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.Drac, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, registration.GetDracIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing dracs").Err()
		}
	}
	return registration.ListDracs(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteDrac deletes the drac in datastore
//
// For referential data intergrity,
// 1. Delete the drac
// 2. Get the machine associated with this drac
// 3. Update the machine by removing the association with this drac
func DeleteDrac(ctx context.Context, id string) error {
	f := func(ctx context.Context) error {
		changes := LogDracChanges(&ufspb.Drac{Name: id}, nil)
		// 1. Delete the drac
		if err := registration.DeleteDrac(ctx, id); err != nil {
			return err
		}

		nu := &networkUpdater{
			Hostname: id,
		}
		// 2. Delete its ip configs
		if err := nu.deleteDHCPHelper(ctx); err != nil {
			return err
		}
		changes = append(changes, nu.Changes...)

		// 3. Get the machine associated with drac
		machines, err := registration.QueryMachineByPropertyName(ctx, "drac_id", id, false)
		if err != nil {
			return errors.Annotate(err, "Unable to query machine for drac %s", id).Err()
		}
		if machines == nil || len(machines) == 0 {
			logging.Warningf(ctx, "No machine associated with the drac %s. Data discrepancy error.\n", id)
		}
		if len(machines) > 1 {
			logging.Warningf(ctx, "More than one machine associated the drac %s. Data discrepancy error.\n", id)
		}
		// 4. Remove the association between the browser machines and this drac.
		cs, err := removeDracFromBrowserMachines(ctx, machines)
		if err != nil {
			return err
		}
		changes = append(changes, cs...)
		return SaveChangeEvents(ctx, changes)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete drac in datastore: %s", err)
		return err
	}
	return nil
}

// ImportDracs creates or updates a batch of dracs in datastore
func ImportDracs(ctx context.Context, dracs []*ufspb.Drac) (*ufsds.OpResults, error) {
	return registration.ImportDracs(ctx, dracs)
}

// ReplaceDrac replaces an old Drac with new Drac in datastore
//
// It does a delete of old drac and create of new Drac.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old Drac, it will get all the resources referencing
// the old Drac. It will update all the resources which were referencing
// the old Drac(got in the last step) with new Drac.
// Deletes the old Drac.
// Creates the new Drac.
// This will preserve data integrity in the system.
func ReplaceDrac(ctx context.Context, oldDrac *ufspb.Drac, newDrac *ufspb.Drac) (*ufspb.Drac, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// getBrowserMachineForDrac return browser machine associated with the drac.
func getBrowserMachineForDrac(ctx context.Context, dracName string) (*ufspb.Machine, error) {
	machines, err := registration.QueryMachineByPropertyName(ctx, "drac_id", dracName, false)
	if err != nil {
		return nil, errors.Annotate(err, "Unable to query machine for drac %s", dracName).Err()
	}
	if machines == nil || len(machines) == 0 {
		errorMsg := fmt.Sprintf("No machine associated with the drac %s. Data discrepancy error.\n", dracName)
		return nil, status.Errorf(codes.Internal, errorMsg)
	}
	if len(machines) > 1 {
		errorMsg := fmt.Sprintf("More than one machine associated the drac %s. Data discrepancy error.\n", dracName)
		return nil, status.Errorf(codes.Internal, errorMsg)
	}
	return machines[0], nil
}

// removeDracFromBrowserMachines removes the drac info from browser machines and
// updates the machines in datastore.
// Must be called within a transaction as BatchUpdateMachines is a non-atomic operation
func removeDracFromBrowserMachines(ctx context.Context, machines []*ufspb.Machine) ([]*ufspb.ChangeEvent, error) {
	changes := make([]*ufspb.ChangeEvent, 0)
	if machines == nil || len(machines) == 0 {
		return changes, nil
	}
	for _, machine := range machines {
		oldMachine := proto.Clone(machine).(*ufspb.Machine)
		if machine.GetChromeBrowserMachine() == nil {
			errorMsg := fmt.Sprintf("Machine %s is not a browser machine", machine.Name)
			return nil, status.Errorf(codes.FailedPrecondition, errorMsg)
		}
		machine.GetChromeBrowserMachine().Drac = ""
		changes = append(changes, LogMachineChanges(oldMachine, machine)...)
	}
	_, err := registration.BatchUpdateMachines(ctx, machines)
	if err != nil {
		return nil, errors.Annotate(err, "Unable to remove drac information from machine").Err()
	}
	return changes, nil
}

// addDracToBrowserMachine adds the drac info to the browser machine an
// updates the machine in datastore.
// Must be called within a transaction as BatchUpdateMachines is a non-atomic operation
func addDracToBrowserMachine(ctx context.Context, machine *ufspb.Machine, dracName string) ([]*ufspb.ChangeEvent, error) {
	if machine == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Machine is nil")
	}
	if machine.GetChromeBrowserMachine() == nil {
		errorMsg := fmt.Sprintf("Machine %s is not a browser machine", machine.Name)
		return nil, status.Errorf(codes.FailedPrecondition, errorMsg)
	}
	oldMachine := proto.Clone(machine).(*ufspb.Machine)
	machine.GetChromeBrowserMachine().Drac = dracName
	_, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine})
	if err != nil {
		return nil, errors.Annotate(err, "Unable to update machine %s with drac %s information", machine.Name, dracName).Err()
	}
	return LogMachineChanges(oldMachine, machine), nil
}

// validateCreateDrac validates if a drac can be created
//
// check if the drac already exists
// checks if the machine and resources referenced by the drac does not exist
func validateCreateDrac(ctx context.Context, drac *ufspb.Drac, machineName string) error {
	// 1. Check if Drac already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetDracResource(drac.Name)}, nil); err != nil {
		return err
	}

	// Aggregate resource to check if machine does not exist
	resourcesNotFound := []*Resource{GetMachineResource(machineName)}
	// Aggregate resource to check if resources referenced by the drac does not exist
	if switchID := drac.GetSwitchInterface().GetSwitch(); switchID != "" {
		resourcesNotFound = append(resourcesNotFound, GetSwitchResource(switchID))
	}
	// 2. Check if resources does not exist
	return ResourceExist(ctx, resourcesNotFound, nil)
}

// validateUpdateDrac validates if a drac can be updated
//
// checks if drac, machine and resources referecned by the drac does not exist
func validateUpdateDrac(ctx context.Context, drac *ufspb.Drac, machineName string, mask *field_mask.FieldMask) error {
	// Aggregate resource to check if drac does not exist
	resourcesNotFound := []*Resource{GetDracResource(drac.Name)}
	// Aggregate resource to check if machine does not exist
	if machineName != "" {
		resourcesNotFound = append(resourcesNotFound, GetMachineResource(machineName))
	}
	// Aggregate resource to check if resources referenced by the drac does not exist
	if switchID := drac.GetSwitchInterface().GetSwitch(); switchID != "" {
		resourcesNotFound = append(resourcesNotFound, GetSwitchResource(switchID))
	}

	// check if resources does not exist
	if err := ResourceExist(ctx, resourcesNotFound, nil); err != nil {
		return err
	}

	return validateDracUpdateMask(mask)
}

// validateUpdateDracHost validates if a host can be added to a drac
func validateUpdateDracHost(ctx context.Context, drac *ufspb.Drac, vlanName, ipv4Str string) error {
	if drac.GetMacAddress() == "" {
		return errors.New("mac address of drac hasn't been specified")
	}
	if ipv4Str != "" {
		return nil
	}
	// Check if resources does not exist
	return ResourceExist(ctx, []*Resource{GetDracResource(drac.Name), GetVlanResource(vlanName)}, nil)
}

// validateDracUpdateMask validates the update mask for drac update
func validateDracUpdateMask(mask *field_mask.FieldMask) error {
	if mask != nil {
		// validate the give field mask
		for _, path := range mask.Paths {
			switch path {
			case "name":
				return status.Error(codes.InvalidArgument, "validateUpdateDrac - name cannot be updated, delete and create a new drac instead")
			case "update_time":
				return status.Error(codes.InvalidArgument, "validateUpdateDrac - update_time cannot be updated, it is a Output only field")
			case "switch":
			case "port":
			case "machine":
			case "macAddress":
			case "tags":
				// valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "validateUpdateDrac - unsupported update mask path %q", path)
			}
		}
	}
	return nil
}
