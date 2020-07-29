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
	"infra/unifiedfleet/app/model/registration"
)

// MachineRegistration creates a new machine, new nic and a new drac in datastore.
func MachineRegistration(ctx context.Context, machine *ufspb.Machine, nics []*ufspb.Nic, drac *ufspb.Drac) (*ufspb.Machine, []*ufspb.Nic, *ufspb.Drac, error) {
	f := func(ctx context.Context) error {
		// 1. Validate the input
		if err := validateMachineRegistration(ctx, machine, nics, drac); err != nil {
			return err
		}

		// 2. Make sure OUTPUT_ONLY fields are set to empty
		if machine.GetChromeBrowserMachine() != nil {
			// These are output only field. User is not allowed to set these value.
			// Overwrite it with empty values.
			machine.GetChromeBrowserMachine().Nics = nil
			machine.GetChromeBrowserMachine().Drac = ""
		}

		// 3. Create nic
		if nics != nil {
			var n []string = make([]string, 0, len(nics))
			for _, nic := range nics {
				n = append(n, nic.Name)
				// Fill the machine/rack/lab to nic OUTPUT only fields
				nic.Machine = machine.GetName()
				nic.Rack = machine.GetLocation().GetRack()
				nic.Lab = machine.GetLocation().GetLab().String()
			}
			// This is output only field. Assign new value.
			machine.GetChromeBrowserMachine().Nics = n

			// we use this func as it is a non-atomic operation and can be used to
			// run within a transaction to make it atomic. Datastore doesnt allow
			// nested transactions.
			if _, err := registration.BatchUpdateNics(ctx, nics); err != nil {
				return err
			}
		}

		// 4. Create drac
		if drac != nil {
			// This is output only field. Assign new value.
			machine.GetChromeBrowserMachine().Drac = drac.Name
			// Fill the machine/rack/lab to drac OUTPUT only fields
			drac.Machine = machine.GetName()
			drac.Rack = machine.GetLocation().GetRack()
			drac.Lab = machine.GetLocation().GetLab().String()

			// we use this func as it is a non-atomic operation and can be used to
			// run within a transaction to make it atomic. Datastore doesnt allow
			// nested transactions.
			if _, err := registration.BatchUpdateDracs(ctx, []*ufspb.Drac{drac}); err != nil {
				return err
			}
		}

		// 5. Create machine
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine}); err != nil {
			return err
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to register machine: %s", err)
		return nil, nil, nil, err
	}
	// Log the changes
	changes := LogMachineChanges(nil, machine)
	for _, nic := range nics {
		changes = append(changes, LogNicChanges(nil, nic)...)
	}
	changes = append(changes, LogDracChanges(nil, drac)...)
	SaveChangeEvents(ctx, changes)
	return machine, nics, drac, nil
}

// validateMachineRegistration validates if a machine, nic and drac can be created in the datastore.
//
// checks if the resources machine/nic/drac already exists in the system.
// checks if resources referenced by machine/nic/drac does not exist in the system.
func validateMachineRegistration(ctx context.Context, machine *ufspb.Machine, nics []*ufspb.Nic, drac *ufspb.Drac) error {
	if machine == nil {
		return errors.New("machine cannot be empty")
	}
	var resourcesAlreadyExists []*Resource
	var resourcesNotFound []*Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create machine %s:\n", machine.Name))
	// Aggregate resources to check if machine already exists
	resourcesAlreadyExists = append(resourcesAlreadyExists, GetMachineResource(machine.Name))

	// Aggregate resources to check if resources referenced by the machine exists
	if kvmID := machine.GetChromeBrowserMachine().GetKvmInterface().GetKvm(); kvmID != "" {
		resourcesNotFound = append(resourcesNotFound, GetKVMResource(kvmID))
	}
	if rpmID := machine.GetChromeBrowserMachine().GetRpmInterface().GetRpm(); rpmID != "" {
		resourcesNotFound = append(resourcesNotFound, GetRPMResource(rpmID))
	}
	if chromePlatformID := machine.GetChromeBrowserMachine().GetChromePlatform(); chromePlatformID != "" {
		resourcesNotFound = append(resourcesNotFound, GetChromePlatformResource(chromePlatformID))
	}

	if nics != nil {
		// Check if machine is not a browser machine
		if machine.GetChromeBrowserMachine() == nil {
			errorMsg := fmt.Sprintf("Machine %s is not a browser machine. "+
				"Nics can only be added to a browser machine.", machine.Name)
			return status.Errorf(codes.FailedPrecondition, errorMsg)
		}

		for _, nic := range nics {
			// Aggregate resources to check if nic already exists
			resourcesAlreadyExists = append(resourcesAlreadyExists, GetNicResource(nic.Name))

			// Aggregate resources to check if resources referenced by the nic exists
			if switchID := nic.GetSwitchInterface().GetSwitch(); switchID != "" {
				resourcesNotFound = append(resourcesNotFound, GetSwitchResource(switchID))
			}
		}
	}

	if drac != nil {
		// Check if machine is not a browser machine
		if machine.GetChromeBrowserMachine() == nil {
			errorMsg := fmt.Sprintf("Machine %s is not a browser machine. "+
				"Drac can only be added to a browser machine.", machine.Name)
			return status.Errorf(codes.FailedPrecondition, errorMsg)
		}

		// Aggregate resources to check if drac already exists
		resourcesAlreadyExists = append(resourcesAlreadyExists, GetDracResource(drac.Name))

		// Aggregate resources to check if resources referenced by the drac exists
		if switchID := drac.GetSwitchInterface().GetSwitch(); switchID != "" {
			resourcesNotFound = append(resourcesNotFound, GetSwitchResource(switchID))
		}
	}

	// Check if machine/nic/drac already exists
	if err := resourceAlreadyExists(ctx, resourcesAlreadyExists, &errorMsg); err != nil {
		return err
	}

	// Check if resources referenced by machine/nic/drac does not exist
	return ResourceExist(ctx, resourcesNotFound, &errorMsg)
}
