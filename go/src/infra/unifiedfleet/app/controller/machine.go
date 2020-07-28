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
	"infra/unifiedfleet/app/util"
)

// CreateMachine creates a new machine in datastore.
//
// Checks if the resources referenced by the Machine input already exists
// in the system before creating a new Machine
func CreateMachine(ctx context.Context, machine *ufspb.Machine) (*ufspb.Machine, error) {
	f := func(ctx context.Context) error {
		// 1. Validate input
		if err := validateCreateMachine(ctx, machine); err != nil {
			return err
		}

		// 2. Make sure OUTPUT_ONLY fields are set to empty
		if machine.GetChromeBrowserMachine() != nil {
			// These are output only field. User is not allowed to set these value.
			// Overwrite it with empty values.
			machine.GetChromeBrowserMachine().Nics = nil
			machine.GetChromeBrowserMachine().Drac = ""
		}

		// 3. Create the machine
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
			return err
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create machine in datastore: %s", err)
		return nil, err
	}
	SaveChangeEvents(ctx, LogMachineChanges(nil, machine))
	return machine, nil
}

// UpdateMachine updates machine in datastore.
//
// Checks if the resources referenced by the Machine input already exists
// in the system before updating a Machine
func UpdateMachine(ctx context.Context, machine *ufspb.Machine) (*ufspb.Machine, error) {
	var oldMachine *ufspb.Machine
	var err error
	f := func(ctx context.Context) error {
		// 1. Validate input
		if err := validateUpdateMachine(ctx, machine); err != nil {
			return err
		}

		// 2. Make sure OUTPUT_ONLY fields are set to empty
		if machine.GetChromeBrowserMachine() != nil {
			// These are output only field. User is not allowed to set these value.
			// Overwrite it with empty values.
			machine.GetChromeBrowserMachine().Nics = nil
			machine.GetChromeBrowserMachine().Drac = ""
		}

		// 3. Get the existing/old machine
		oldMachine, err = registration.GetMachine(ctx, machine.GetName())
		if err != nil {
			return err
		}

		// 4. Make sure OUTPUT_ONLY fields are overwritten with old values
		if oldMachine.GetChromeBrowserMachine() != nil {
			if machine.GetChromeBrowserMachine() == nil {
				machine.Device = &ufspb.Machine_ChromeBrowserMachine{
					ChromeBrowserMachine: &ufspb.ChromeBrowserMachine{},
				}
			}
			// These are output only fields. Not allowed to update by the user.
			// Overwrite the input values with existing values.
			machine.GetChromeBrowserMachine().Nics = oldMachine.GetChromeBrowserMachine().Nics
			machine.GetChromeBrowserMachine().Drac = oldMachine.GetChromeBrowserMachine().Drac
		}

		// 5. Create the machine
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
			return err
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to update machine in datastore: %s", err)
		return nil, err
	}
	SaveChangeEvents(ctx, LogMachineChanges(oldMachine, machine))
	return machine, nil
}

// GetMachine returns machine for the given id from datastore.
func GetMachine(ctx context.Context, id string) (*ufspb.Machine, error) {
	return registration.GetMachine(ctx, id)
}

// ListMachines lists the machines
func ListMachines(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.Machine, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, registration.GetMachineIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing machines").Err()
		}
	}
	return registration.ListMachines(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// GetAllMachines returns all machines in datastore.
func GetAllMachines(ctx context.Context) (*ufsds.OpResults, error) {
	return registration.GetAllMachines(ctx)
}

// DeleteMachine deletes the machine and its associated nics and drac in datastore
//
// For referential data intergrity,
// Delete if this Machine is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteMachine(ctx context.Context, id string) error {
	var machine *ufspb.Machine
	var err error
	f := func(ctx context.Context) error {
		// 1. Get the machine
		machine, err = registration.GetMachine(ctx, id)
		if status.Code(err) == codes.Internal {
			return err
		}
		if machine == nil {
			return status.Errorf(codes.NotFound, ufsds.NotFound)
		}

		// 2. Check if any other resource references this machine.
		if err = validateDeleteMachine(ctx, id); err != nil {
			return err
		}

		//Only for a browser machine
		if machine.GetChromeBrowserMachine() != nil {
			// 3. Delete the nics
			if nicIDs := machine.GetChromeBrowserMachine().Nics; nicIDs != nil {
				// we use this func as it is a non-atomic operation and can be used to
				// run within a transaction to make it atomic. Datastore doesnt allow
				// nested transactions.
				if err = registration.BatchDeleteNics(ctx, nicIDs); err != nil {
					return err
				}
			}

			// 4. Delete the drac
			if dracID := machine.GetChromeBrowserMachine().Drac; dracID != "" {
				if err = registration.DeleteDrac(ctx, dracID); err != nil {
					return err
				}
			}
		}

		// 5. Delete the machine
		return registration.DeleteMachine(ctx, id)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete machine and its associated nics and drac in datastore: %s", err)
		return err
	}
	SaveChangeEvents(ctx, LogMachineChanges(machine, nil))
	return nil
}

// ImportMachines creates or updates a batch of machines in datastore
func ImportMachines(ctx context.Context, machines []*ufspb.Machine, pageSize int) (*ufsds.OpResults, error) {
	deleteNonExistingMachines(ctx, machines, pageSize)
	allRes := make(ufsds.OpResults, 0)
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(machines))
		res, err := registration.ImportMachines(ctx, machines[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(machines) {
			break
		}
	}
	return &allRes, nil
}

func deleteNonExistingMachines(ctx context.Context, machines []*ufspb.Machine, pageSize int) (*ufsds.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range machines {
		resMap[r.GetName()] = true
	}
	resp, err := registration.GetAllMachines(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.Machine)
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
		}
	}
	logging.Debugf(ctx, "Deleting %d non-existing machines", len(toDelete))
	return deleteByPage(ctx, toDelete, pageSize, registration.DeleteMachines), nil
}

// ReplaceMachine replaces an old Machine with new Machine in datastore
//
// It does a delete of old machine and create of new Machine.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old Machine, it will get all the resources referencing
// the old Machine. It will update all the resources which were referencing
// the old Machine(got in the last step) with new Machine.
// Deletes the old Machine.
// Creates the new Machine.
// This will preserve data integrity in the system.
func ReplaceMachine(ctx context.Context, oldMachine *ufspb.Machine, newMachine *ufspb.Machine) (*ufspb.Machine, error) {
	f := func(ctx context.Context) error {
		machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", oldMachine.Name, false)
		if err != nil {
			return err
		}
		if machinelses != nil {
			for _, machinelse := range machinelses {
				machines := machinelse.GetMachines()
				for i := range machines {
					if machines[i] == oldMachine.Name {
						machines[i] = newMachine.Name
						break
					}
				}
				machinelse.Machines = machines
			}
			_, err := inventory.BatchUpdateMachineLSEs(ctx, machinelses)
			if err != nil {
				return err
			}
		}

		err = registration.DeleteMachine(ctx, oldMachine.Name)
		if err != nil {
			return err
		}

		err = validateMachineReferences(ctx, newMachine)
		if err != nil {
			return err
		}
		entity := &registration.MachineEntity{
			ID: newMachine.Name,
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

		_, err = registration.BatchUpdateMachines(ctx, []*ufspb.Machine{newMachine})
		if err != nil {
			return err
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to replace entity in datastore: %s", err)
		return nil, err
	}

	changes := LogMachineChanges(oldMachine, nil)
	changes = append(changes, LogMachineChanges(nil, newMachine)...)
	SaveChangeEvents(ctx, changes)
	return newMachine, nil
}

// validateMachineReferences validates if the resources referenced by the machine
// are in the system.
//
// Checks if the resources referenced by the given Machine input already exists
// in the system. Returns an error if any resource referenced by the Machine input
// does not exist in the system.
func validateMachineReferences(ctx context.Context, machine *ufspb.Machine) error {
	var resources []*Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create Machine %s:\n", machine.Name))

	if kvmID := machine.GetChromeBrowserMachine().GetKvmInterface().GetKvm(); kvmID != "" {
		resources = append(resources, GetKVMResource(kvmID))
	}
	if rpmID := machine.GetChromeBrowserMachine().GetRpmInterface().GetRpm(); rpmID != "" {
		resources = append(resources, GetRPMResource(rpmID))
	}
	if chromePlatformID := machine.GetChromeBrowserMachine().GetChromePlatform(); chromePlatformID != "" {
		resources = append(resources, GetChromePlatformResource(chromePlatformID))
	}

	return ResourceExist(ctx, resources, &errorMsg)
}

// validateDeleteMachine validates if a Machine can be deleted
//
// Checks if this Machine(MachineID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteMachine(ctx context.Context, id string) error {
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", id, true)
	if err != nil {
		return err
	}
	if len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Machine %s cannot be deleted because there are other resources which are referring this Machine.", id))
		errorMsg.WriteString(fmt.Sprintf("\nHosts referring the Machine:\n"))
		for _, machinelse := range machinelses {
			errorMsg.WriteString(machinelse.Name + ", ")
		}
		errorMsg.WriteString(fmt.Sprintf("\nPlease delete the hosts and then delete the machine.\n"))
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

// getBrowserMachine gets the browser machine
func getBrowserMachine(ctx context.Context, machineName string) (*ufspb.Machine, error) {
	machine, err := registration.GetMachine(ctx, machineName)
	if err != nil {
		return nil, errors.Annotate(err, "Unable to get browser machine %s", machineName).Err()
	}
	if machine.GetChromeBrowserMachine() == nil {
		errorMsg := fmt.Sprintf("Machine %s is not a browser machine.", machineName)
		return nil, status.Errorf(codes.FailedPrecondition, errorMsg)
	}
	return machine, nil
}

// validateCreateMachine validates if a machine can be created
func validateCreateMachine(ctx context.Context, machine *ufspb.Machine) error {
	// 1. Check if machine already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetMachineResource(machine.Name)}, nil); err != nil {
		return err
	}
	// 2. Validate machine for referenced resources
	var resourcesNotfound []*Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create machine %s:\n", machine.Name))
	// Aggregate resources referenced by the machine to check if they do not exist
	if kvmID := machine.GetChromeBrowserMachine().GetKvmInterface().GetKvm(); kvmID != "" {
		resourcesNotfound = append(resourcesNotfound, GetKVMResource(kvmID))
	}
	if rpmID := machine.GetChromeBrowserMachine().GetRpmInterface().GetRpm(); rpmID != "" {
		resourcesNotfound = append(resourcesNotfound, GetRPMResource(rpmID))
	}
	if chromePlatformID := machine.GetChromeBrowserMachine().GetChromePlatform(); chromePlatformID != "" {
		resourcesNotfound = append(resourcesNotfound, GetChromePlatformResource(chromePlatformID))
	}
	// check if resources does not exist
	return ResourceExist(ctx, resourcesNotfound, &errorMsg)
}

// validateUpdateMachine validates if a machine can be updated
//
// checks if the machine and the resources referenced  by the machine
// does not exist in the system.
func validateUpdateMachine(ctx context.Context, machine *ufspb.Machine) error {
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot update machine %s:\n", machine.Name))
	// Aggregate resource to check if machine does not exist
	resourcesNotfound := []*Resource{GetMachineResource(machine.Name)}
	// Aggregate resources referenced by the machine to check if they do not exist
	if kvmID := machine.GetChromeBrowserMachine().GetKvmInterface().GetKvm(); kvmID != "" {
		resourcesNotfound = append(resourcesNotfound, GetKVMResource(kvmID))
	}
	if rpmID := machine.GetChromeBrowserMachine().GetRpmInterface().GetRpm(); rpmID != "" {
		resourcesNotfound = append(resourcesNotfound, GetRPMResource(rpmID))
	}
	if chromePlatformID := machine.GetChromeBrowserMachine().GetChromePlatform(); chromePlatformID != "" {
		resourcesNotfound = append(resourcesNotfound, GetChromePlatformResource(chromePlatformID))
	}
	// check if resources does not exist
	return ResourceExist(ctx, resourcesNotfound, &errorMsg)
}
