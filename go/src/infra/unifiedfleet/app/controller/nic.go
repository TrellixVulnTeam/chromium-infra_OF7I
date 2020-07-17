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
	"infra/unifiedfleet/app/model/registration"
	ufsUtil "infra/unifiedfleet/app/util"
)

// CreateNic creates a new nic in datastore.
//
// Checks if the resources referenced by the Nic input already exists
// in the system before creating a new Nic
func CreateNic(ctx context.Context, nic *ufspb.Nic, machineName string) (*ufspb.Nic, error) {
	f := func(ctx context.Context) error {
		// 1. Validate the input
		if err := validateCreateNic(ctx, nic, machineName); err != nil {
			return err
		}

		// 2. Get browser machine to associate the nic
		machine, err := getBrowserMachine(ctx, machineName)
		if err != nil {
			return err
		}

		// 3. Update the browser machine with new nic information
		if err = addNicToBrowserMachine(ctx, machine, nic.Name); err != nil {
			return err
		}

		// 4. Create a nic entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err = registration.BatchUpdateNics(ctx, []*ufspb.Nic{nic}); err != nil {
			return errors.Annotate(err, "Unable to create nic %s", nic.Name).Err()
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create entity in datastore: %s", err)
		return nil, err
	}
	return nic, nil
}

// UpdateNic updates nic in datastore.
//
// Checks if the resources referenced by the Nic input already exists
// in the system before updating a Nic
func UpdateNic(ctx context.Context, nic *ufspb.Nic, machineName string) (*ufspb.Nic, error) {
	f := func(ctx context.Context) error {
		// 1. Validate the input
		if err := validateUpdateNic(ctx, nic, machineName); err != nil {
			return err
		}

		if machineName != "" {
			// 2. Get the old browser machine associated with nic
			oldMachine, err := getBrowserMachineForNic(ctx, nic.Name)
			if err != nil {
				return err
			}

			// User is trying to associate this nic with a different browser machine.
			if oldMachine.Name != machineName {
				// 3. Get browser machine to associate the nic
				machine, err := getBrowserMachine(ctx, machineName)
				if err != nil {
					return err
				}

				// 4. Remove the association between old browser machine and this nic.
				if err = removeNicFromBrowserMachines(ctx, []*ufspb.Machine{oldMachine}, nic.Name); err != nil {
					return err
				}

				// 5. Update the browser machine with new nic information
				if err = addNicToBrowserMachine(ctx, machine, nic.Name); err != nil {
					return err
				}
			}
		}

		// 6. Update nic entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction. Datastore doesnt allow nested transactions.
		if _, err := registration.BatchUpdateNics(ctx, []*ufspb.Nic{nic}); err != nil {
			return errors.Annotate(err, "Unable to create nic %s", nic.Name).Err()
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create entity in datastore: %s", err)
		return nil, err
	}
	return nic, nil
}

// getBrowserMachineForNic return browser machine associated with the nic.
func getBrowserMachineForNic(ctx context.Context, nicName string) (*ufspb.Machine, error) {
	machines, err := registration.QueryMachineByPropertyName(ctx, "nic_ids", nicName, false)
	if err != nil {
		return nil, errors.Annotate(err, "Unable to query machine for nic %s", nicName).Err()
	}
	if machines == nil || len(machines) == 0 {
		errorMsg := fmt.Sprintf("No machine associated with the nic %s. Data discrepancy error.\n", nicName)
		return nil, status.Errorf(codes.Internal, errorMsg)
	}
	if len(machines) > 1 {
		errorMsg := fmt.Sprintf("More than one machine associated the nic %s. Data discrepancy error.\n", nicName)
		return nil, status.Errorf(codes.Internal, errorMsg)
	}
	return machines[0], nil
}

// removeNicFromBrowserMachines removes the nic info from browser machines and
// updates the machines in datastore.
// Must be called within a transaction as BatchUpdateMachines is a non-atomic operation
func removeNicFromBrowserMachines(ctx context.Context, machines []*ufspb.Machine, id string) error {
	for _, machine := range machines {
		if machine.GetChromeBrowserMachine() == nil {
			errorMsg := fmt.Sprintf("Machine %s is not a browser machine", machine.Name)
			return status.Errorf(codes.FailedPrecondition, errorMsg)
		}
		nics := machine.GetChromeBrowserMachine().GetNics()
		nics = ufsUtil.RemoveStringEntry(nics, id)
		machine.GetChromeBrowserMachine().Nics = nics
	}
	_, err := registration.BatchUpdateMachines(ctx, machines)
	if err != nil {
		return errors.Annotate(err, "Unable to remove nic information %s from machine", id).Err()
	}
	return nil
}

// addNicToBrowserMachine adds the nic info to the browser machine and updates
// the machine in datastore.
// Must be called within a transaction as BatchUpdateMachines is a non-atomic operation
func addNicToBrowserMachine(ctx context.Context, machine *ufspb.Machine, nicName string) error {
	if machine == nil {
		return status.Errorf(codes.FailedPrecondition, "Machine is nil")
	}
	if machine.GetChromeBrowserMachine() == nil {
		errorMsg := fmt.Sprintf("Machine %s is not a browser machine", machine.Name)
		return status.Errorf(codes.FailedPrecondition, errorMsg)
	}
	nics := []string{nicName}
	if machine.GetChromeBrowserMachine().GetNics() != nil {
		nics = machine.GetChromeBrowserMachine().GetNics()
		nics = append(nics, nicName)
	}
	machine.GetChromeBrowserMachine().Nics = nics
	_, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine})
	if err != nil {
		return errors.Annotate(err, "Unable to update machine %s with nic %s information", machine.Name, nicName).Err()
	}
	return nil
}

// GetNic returns nic for the given id from datastore.
func GetNic(ctx context.Context, id string) (*ufspb.Nic, error) {
	return registration.GetNic(ctx, id)
}

// ListNics lists the nics
func ListNics(ctx context.Context, pageSize int32, pageToken string) ([]*ufspb.Nic, string, error) {
	return registration.ListNics(ctx, pageSize, pageToken)
}

// DeleteNic deletes the nic in datastore
//
// For referential data intergrity,
// Delete if this Nic is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteNic(ctx context.Context, id string) error {
	err := validateDeleteNic(ctx, id)
	if err != nil {
		return err
	}
	return registration.DeleteNic(ctx, id)
}

// ImportNics creates or updates a batch of nics in datastore
func ImportNics(ctx context.Context, nics []*ufspb.Nic) (*ufsds.OpResults, error) {
	return registration.ImportNics(ctx, nics)
}

// ReplaceNic replaces an old Nic with new Nic in datastore
//
// It does a delete of old nic and create of new Nic.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old Nic, it will get all the resources referencing
// the old Nic. It will update all the resources which were referencing
// the old Nic(got in the last step) with new Nic.
// Deletes the old Nic.
// Creates the new Nic.
// This will preserve data integrity in the system.
func ReplaceNic(ctx context.Context, oldNic *ufspb.Nic, newNic *ufspb.Nic) (*ufspb.Nic, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateDeleteNic validates if a Nic can be deleted
//
// Checks if this Nic(NicID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteNic(ctx context.Context, id string) error {
	machines, err := registration.QueryMachineByPropertyName(ctx, "nic_ids", id, true)
	if err != nil {
		return err
	}
	if len(machines) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Nic %s cannot be deleted because there are other resources which are referring this Nic.", id))
		errorMsg.WriteString(fmt.Sprintf("\nMachine referring the Nic:\n"))
		for _, machine := range machines {
			errorMsg.WriteString(machine.Name + ", ")
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

// validateCreateNic validates if a nic can be created
//
// check if the nic already exists
// checks if the machine and resources referenced by the nic does not exist
func validateCreateNic(ctx context.Context, nic *ufspb.Nic, machineName string) error {
	// 1. Check if nic already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetNicResource(nic.Name)}, nil); err != nil {
		return err
	}

	// Aggregate resource to check if machine does not exist
	resourcesNotFound := []*Resource{GetMachineResource(machineName)}
	// Aggregate resource to check if resources referenced by the nic does not exist
	if switchID := nic.GetSwitchInterface().GetSwitch(); switchID != "" {
		resourcesNotFound = append(resourcesNotFound, GetSwitchResource(switchID))
	}
	// 2. Check if resources does not exist
	return ResourceExist(ctx, resourcesNotFound, nil)
}

// validateUpdateNic validates if a nic can be updated
//
// checks if nic, machine and resources referecned by the nic does not exist
func validateUpdateNic(ctx context.Context, nic *ufspb.Nic, machineName string) error {
	// Aggregate resource to check if nic does not exist
	resourcesNotFound := []*Resource{GetNicResource(nic.Name)}
	// Aggregate resource to check if machine does not exist
	if machineName != "" {
		resourcesNotFound = append(resourcesNotFound, GetMachineResource(machineName))
	}
	// Aggregate resource to check if resources referenced by the nic does not exist
	if switchID := nic.GetSwitchInterface().GetSwitch(); switchID != "" {
		resourcesNotFound = append(resourcesNotFound, GetSwitchResource(switchID))
	}

	// Check if resources does not exist
	return ResourceExist(ctx, resourcesNotFound, nil)
}
