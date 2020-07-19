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
)

// CreateDrac creates a new drac in datastore.
//
// Checks if the resources referenced by the Drac input already exists
// in the system before creating a new Drac
func CreateDrac(ctx context.Context, drac *ufspb.Drac, machineName string) (*ufspb.Drac, error) {
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

		// 3. Remove the drac associated with browser machine from the system.
		// The existing old drac in the browser machine will be replaced with this new drac,
		// A drac cannot exist in the system without being associated to a machine,
		// so we delete the old drac from the system.
		if oldDracName := machine.GetChromeBrowserMachine().GetDrac(); oldDracName != "" {
			if err := registration.DeleteDrac(ctx, oldDracName); err != nil {
				return errors.Annotate(err, "%s drac not dound in the system. Deletion for drac %s failed.", oldDracName, oldDracName).Err()
			}
		}

		// 4. Update the browser machine with new drac information
		if err = addDracToBrowserMachine(ctx, machine, drac.Name); err != nil {
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
	return drac, nil
}

// UpdateDrac updates drac in datastore.
//
// Checks if the resources referenced by the Drac input already exists
// in the system before updating a Drac
func UpdateDrac(ctx context.Context, drac *ufspb.Drac, machineName string) (*ufspb.Drac, error) {
	f := func(ctx context.Context) error {
		// 1. Validate the input
		if err := validateUpdateDrac(ctx, drac, machineName); err != nil {
			return err
		}

		if machineName != "" {
			// 2. Get the old browser machine associated with drac
			oldMachine, err := getBrowserMachineForDrac(ctx, drac.Name)
			if err != nil {
				return err
			}

			// User is trying to associate this drac with a different browser machine.
			if oldMachine.Name != machineName {
				// 3. Remove the association between old browser machine and this drac.
				if err = removeDracFromBrowserMachines(ctx, []*ufspb.Machine{oldMachine}); err != nil {
					return err
				}

				// 4. Get new browser machine to associate the drac
				machine, err := getBrowserMachine(ctx, machineName)
				if err != nil {
					return err
				}

				// 5. Remove the drac associated with new browser machine from the system.
				// The existing old drac in the new browser machine will be replaced with this new drac,
				// A drac cannot exist in the system without being associated to a machine,
				// so we delete the old drac from the system.
				if oldDracName := machine.GetChromeBrowserMachine().GetDrac(); oldDracName != "" {
					if err := registration.DeleteDrac(ctx, oldDracName); err != nil {
						return errors.Annotate(err, "%s drac not dound in the system. Deletion for drac %s failed.", oldDracName, oldDracName).Err()
					}
				}

				// 6. Update the new browser machine with new drac information
				if err = addDracToBrowserMachine(ctx, machine, drac.Name); err != nil {
					return err
				}
			}
		}

		// 7. Update drac entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction. Datastore doesnt allow nested transactions.
		if _, err := registration.BatchUpdateDracs(ctx, []*ufspb.Drac{drac}); err != nil {
			return errors.Annotate(err, "Unable to create drac %s", drac.Name).Err()
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create drac in datastore: %s", err)
		return nil, err
	}
	return drac, nil
}

// GetDrac returns drac for the given id from datastore.
func GetDrac(ctx context.Context, id string) (*ufspb.Drac, error) {
	return registration.GetDrac(ctx, id)
}

// ListDracs lists the dracs
func ListDracs(ctx context.Context, pageSize int32, pageToken string) ([]*ufspb.Drac, string, error) {
	return registration.ListDracs(ctx, pageSize, pageToken)
}

// DeleteDrac deletes the drac in datastore
//
// For referential data intergrity,
// Delete if this Drac is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteDrac(ctx context.Context, id string) error {
	err := validateDeleteDrac(ctx, id)
	if err != nil {
		return err
	}
	return registration.DeleteDrac(ctx, id)
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

// validateDeleteDrac validates if a Drac can be deleted
//
// Checks if this Drac(DracID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteDrac(ctx context.Context, id string) error {
	machines, err := registration.QueryMachineByPropertyName(ctx, "drac_id", id, true)
	if err != nil {
		return err
	}
	if len(machines) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Drac %s cannot be deleted because there are other resources which are referring this Drac.", id))
		errorMsg.WriteString(fmt.Sprintf("\nMachines referring the Drac:\n"))
		for _, machine := range machines {
			errorMsg.WriteString(machine.Name + ", ")
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
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
func removeDracFromBrowserMachines(ctx context.Context, machines []*ufspb.Machine) error {
	for _, machine := range machines {
		if machine.GetChromeBrowserMachine() == nil {
			errorMsg := fmt.Sprintf("Machine %s is not a browser machine", machine.Name)
			return status.Errorf(codes.FailedPrecondition, errorMsg)
		}
		machine.GetChromeBrowserMachine().Drac = ""
	}
	_, err := registration.BatchUpdateMachines(ctx, machines)
	if err != nil {
		return errors.Annotate(err, "Unable to remove drac information from machine").Err()
	}
	return nil
}

// addDracToBrowserMachine adds the drac info to the browser machine an
// updates the machine in datastore.
// Must be called within a transaction as BatchUpdateMachines is a non-atomic operation
func addDracToBrowserMachine(ctx context.Context, machine *ufspb.Machine, dracName string) error {
	if machine == nil {
		return status.Errorf(codes.FailedPrecondition, "Machine is nil")
	}
	if machine.GetChromeBrowserMachine() == nil {
		errorMsg := fmt.Sprintf("Machine %s is not a browser machine", machine.Name)
		return status.Errorf(codes.FailedPrecondition, errorMsg)
	}
	machine.GetChromeBrowserMachine().Drac = dracName
	_, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine})
	if err != nil {
		return errors.Annotate(err, "Unable to update machine %s with drac %s information", machine.Name, dracName).Err()
	}
	return nil
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
func validateUpdateDrac(ctx context.Context, drac *ufspb.Drac, machineName string) error {
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

	// Check if resources does not exist
	return ResourceExist(ctx, resourcesNotFound, nil)
}
