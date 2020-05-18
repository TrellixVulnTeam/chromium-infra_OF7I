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

	"go.chromium.org/gae/service/datastore"
	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
)

// CreateMachine creates a new machine in datastore.
//
// Checks if the resources referenced by the Machine input already exists
// in the system before creating a new Machine
func CreateMachine(ctx context.Context, machine *fleet.Machine) (*fleet.Machine, error) {
	err := validateMachine(ctx, machine)
	if err != nil {
		return nil, err
	}
	return registration.CreateMachine(ctx, machine)
}

// UpdateMachine updates machine in datastore.
//
// Checks if the resources referenced by the Machine input already exists
// in the system before updating a Machine
func UpdateMachine(ctx context.Context, machine *fleet.Machine) (*fleet.Machine, error) {
	err := validateMachine(ctx, machine)
	if err != nil {
		return nil, err
	}
	return registration.UpdateMachine(ctx, machine)
}

// GetMachine returns machine for the given id from datastore.
func GetMachine(ctx context.Context, id string) (*fleet.Machine, error) {
	return registration.GetMachine(ctx, id)
}

// ListMachines lists the machines
func ListMachines(ctx context.Context, pageSize int32, pageToken string) ([]*fleet.Machine, string, error) {
	return registration.ListMachines(ctx, pageSize, pageToken)
}

// GetAllMachines returns all machines in datastore.
func GetAllMachines(ctx context.Context) (*fleetds.OpResults, error) {
	return registration.GetAllMachines(ctx)
}

// DeleteMachine deletes the machine in datastore
//
// For referential data intergrity,
// Delete if this Machine is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error message will be thrown.
func DeleteMachine(ctx context.Context, id string) error {
	err := validateDeleteMachine(ctx, id)
	if err != nil {
		return err
	}
	return registration.DeleteMachine(ctx, id)
}

// ImportMachines creates or updates a batch of machines in datastore
func ImportMachines(ctx context.Context, machines []*fleet.Machine) (*fleetds.OpResults, error) {
	return registration.ImportMachines(ctx, machines)
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
func ReplaceMachine(ctx context.Context, oldMachine *fleet.Machine, newMachine *fleet.Machine) (*fleet.Machine, error) {
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

		err = validateMachine(ctx, newMachine)
		if err != nil {
			return err
		}
		entity := &registration.MachineEntity{
			ID: newMachine.Name,
		}
		existsResults, err := datastore.Exists(ctx, entity)
		if err == nil {
			if existsResults.All() {
				return status.Errorf(codes.AlreadyExists, fleetds.AlreadyExists)
			}
		} else {
			logging.Debugf(ctx, "Failed to check existence: %s", err)
			return status.Errorf(codes.Internal, fleetds.InternalError)
		}

		_, err = registration.BatchUpdateMachines(ctx, []*fleet.Machine{newMachine})
		if err != nil {
			return err
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to replace entity in datastore: %s", err)
		return nil, err
	}
	return newMachine, nil
}

// validateMachine validates if a machine can be created/updated in the datastore.
func validateMachine(ctx context.Context, machine *fleet.Machine) error {
	var NotFound bool = false
	var resources []Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create Machine %s:\n", machine.Name))

	kvmID := machine.GetChromeBrowserMachine().GetKvmInterface().GetKvm()
	rpmID := machine.GetChromeBrowserMachine().GetRpmInterface().GetRpm()
	switchID := machine.GetChromeBrowserMachine().GetNetworkDeviceInterface().GetSwitch()
	nicID := machine.GetChromeBrowserMachine().GetNic()
	dracID := machine.GetChromeBrowserMachine().GetDrac()
	chromePlatformID := machine.GetChromeBrowserMachine().GetChromePlatform()
	if kvmID != "" {
		resources = append(resources, GetKVMResource(kvmID))
	}
	if rpmID != "" {
		resources = append(resources, GetRPMResource(rpmID))
	}
	if switchID != "" {
		resources = append(resources, GetSwitchResource(switchID))
	}
	if nicID != "" {
		resources = append(resources, GetNicResource(nicID))
	}
	if dracID != "" {
		resources = append(resources, GetDracResource(dracID))
	}
	if chromePlatformID != "" {
		resources = append(resources, GetChromePlatformResource(chromePlatformID))
	}

	if len(resources) == 0 {
		return nil
	}
	checkEntities := make([]fleetds.FleetEntity, 0, len(resources))
	for _, resource := range resources {
		checkEntities = append(checkEntities, resource.Entity)
	}
	exists, err := fleetds.Exists(ctx, checkEntities)
	if err == nil {
		for i := range checkEntities {
			if !exists[i] {
				NotFound = true
				errorMsg.WriteString(fmt.Sprintf(ErrorMessage, resources[i].Kind, resources[i].Kind, resources[i].ID))
			}
		}
	}
	if NotFound {
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

//validateDeleteMachine validates if a Machine can be deleted
func validateDeleteMachine(ctx context.Context, id string) error {
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", id, true)
	if err != nil {
		return err
	}
	if len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Machine %s cannot be deleted because there are other resources which are referring this Machine.", id))
		errorMsg.WriteString(fmt.Sprintf("\nMachineLSEs referring the Machine:\n"))
		for _, machinelse := range machinelses {
			errorMsg.WriteString(machinelse.Name + ", ")
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}
