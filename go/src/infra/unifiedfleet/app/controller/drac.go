// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"

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
	f := func(ctx context.Context) error {
		hc := &HistoryClient{}
		hc.LogDracChanges(nil, drac)
		// 1. Validate input
		if err := validateCreateDrac(ctx, drac, machineName); err != nil {
			return errors.Annotate(err, "CreateDrac - validation failed").Err()
		}

		// Get browser machine to associate the drac
		machine, err := getBrowserMachine(ctx, machineName)
		if err != nil {
			return errors.Annotate(err, "CreateDrac - failed to get machine %s", machineName).Err()
		}

		// Fill the machine/rack/lab to drac OUTPUT only fields for drac table indexing
		drac.Machine = machine.GetName()
		drac.Rack = machine.GetLocation().GetRack()
		drac.Lab = machine.GetLocation().GetLab().String()

		// Create a drac entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err = registration.BatchUpdateDracs(ctx, []*ufspb.Drac{drac}); err != nil {
			return errors.Annotate(err, "CreateDrac - unable to batch update drac %s", drac.Name).Err()
		}
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "CreateDrac - unable to create drac %s", drac.Name).Err()
	}
	return drac, nil
}

// UpdateDrac updates drac in datastore.
//
// Checks if the resources referenced by the Drac input already exists
// in the system before updating a Drac
func UpdateDrac(ctx context.Context, drac *ufspb.Drac, machineName string, mask *field_mask.FieldMask) (*ufspb.Drac, error) {
	f := func(ctx context.Context) error {
		hc := &HistoryClient{}
		// 1. Validate the input
		if err := validateUpdateDrac(ctx, drac, machineName, mask); err != nil {
			return errors.Annotate(err, "UpdateDrac - validation failed").Err()
		}

		// Get old/existing drac
		oldDrac, err := registration.GetDrac(ctx, drac.GetName())
		if err != nil {
			return errors.Annotate(err, "UpdateDrac - get drac %s failed", drac.GetName()).Err()
		}
		// Copy the machine/rack/lab to drac OUTPUT only fields from already existing drac
		drac.Machine = oldDrac.GetMachine()
		drac.Rack = oldDrac.GetRack()
		drac.Lab = oldDrac.GetLab()
		if machineName != "" {
			// Get the old browser machine associated with drac
			oldMachine, err := getBrowserMachine(ctx, oldDrac.GetMachine())
			if err != nil {
				return errors.Annotate(err, "UpdateDrac - failed to get machine %s", machineName).Err()
			}

			// User is trying to associate this drac with a different browser machine.
			if oldMachine.Name != machineName {
				// A machine can have only one drac. If there is a old drac associated with this machine already, error out.
				// A drac cannot exist in the system without being associated to a machine.
				dracs, err := registration.QueryDracByPropertyName(ctx, "machine", machineName, true)
				if err != nil {
					return errors.Annotate(err, "UpdateDrac - failed to query old drac for machine %s", machineName).Err()
				}
				if dracs != nil && len(dracs) > 0 {
					return status.Error(codes.InvalidArgument, fmt.Sprintf("validateUpdateDrac - There is already a drac %s associated with machine %s.\n"+
						"Please delete that drac and then associate this drac to the machine.", dracs[0].GetName(), machineName))
				}

				// Get new browser machine to associate the drac
				machine, err := getBrowserMachine(ctx, machineName)
				if err != nil {
					return errors.Annotate(err, "UpdateDrac - get browser machine %s failed", machineName).Err()
				}

				// Fill the machine/rack/lab to drac OUTPUT only fields
				drac.Machine = machine.GetName()
				drac.Rack = machine.GetLocation().GetRack()
				drac.Lab = machine.GetLocation().GetLab().String()
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

		// Update drac entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction. Datastore doesnt allow nested transactions.
		if _, err := registration.BatchUpdateDracs(ctx, []*ufspb.Drac{drac}); err != nil {
			return errors.Annotate(err, "UpdateDrac - unable to batch update drac %s", drac.Name).Err()
		}
		hc.LogDracChanges(oldDrac, drac)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "UpdateDrac - failed to update drac %s in datastore", drac.Name).Err()
	}
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
		case "portName":
			if oldDrac.GetSwitchInterface() == nil {
				oldDrac.SwitchInterface = &ufspb.SwitchInterface{
					PortName: drac.GetSwitchInterface().GetPortName(),
				}
			} else {
				oldDrac.GetSwitchInterface().PortName = drac.GetSwitchInterface().GetPortName()
			}
		case "tags":
			oldDrac.Tags = mergeTags(oldDrac.GetTags(), drac.GetTags())
		}
	}
	// return existing/old drac with new updated values
	return oldDrac, nil
}

// DeleteDracHost deletes the host of a drac in datastore.
func DeleteDracHost(ctx context.Context, dracName string) error {
	f := func(ctx context.Context) error {
		hc := &HistoryClient{
			netUdt: &networkUpdater{
				Hostname: dracName,
			},
		}
		if err := hc.netUdt.deleteDHCPHelper(ctx); err != nil {
			return err
		}
		return hc.SaveChangeEvents(ctx)
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
		hc := &HistoryClient{
			netUdt: &networkUpdater{
				Hostname: drac.GetName(),
			},
		}
		// 1. Validate the input
		if err := validateUpdateDracHost(ctx, drac, nwOpt.GetVlan(), nwOpt.GetIp()); err != nil {
			return err
		}

		// 2. Verify if the hostname is already set with IP. if yes, remove the current dhcp.
		if err := hc.netUdt.deleteDHCPHelper(ctx); err != nil {
			return err
		}

		// 3. Find free ip, set IP and DHCP config
		if _, err := hc.netUdt.addHostHelper(ctx, nwOpt.GetVlan(), nwOpt.GetIp(), drac.GetMacAddress()); err != nil {
			return err
		}
		return hc.SaveChangeEvents(ctx)
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
func DeleteDrac(ctx context.Context, id string) error {
	f := func(ctx context.Context) error {
		hc := &HistoryClient{
			netUdt: &networkUpdater{
				Hostname: id,
			},
		}
		hc.LogDracChanges(&ufspb.Drac{Name: id}, nil)
		// 1. Delete the drac
		if err := registration.DeleteDrac(ctx, id); err != nil {
			return errors.Annotate(err, "DeleteDrac - unable to delete drac %s", id).Err()
		}

		// 2. Delete its ip configs
		if err := hc.netUdt.deleteDHCPHelper(ctx); err != nil {
			return errors.Annotate(err, "DeleteDrac - unable to delete ip configs for drac %s", id).Err()
		}

		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return errors.Annotate(err, "DeleteDrac - failed to delete drac in datastore: %s", id).Err()
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

// validateCreateDrac validates if a drac can be created
//
// check if the drac already exists
// checks if the machine and resources referenced by the drac does not exist
func validateCreateDrac(ctx context.Context, drac *ufspb.Drac, machineName string) error {
	// Check if Drac already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetDracResource(drac.Name)}, nil); err != nil {
		return err
	}

	// A machine can have only one drac. If there is a old drac associated with this machine already, error out.
	// A drac cannot exist in the system without being associated to a machine.
	dracs, err := registration.QueryDracByPropertyName(ctx, "machine", machineName, true)
	if err != nil {
		return errors.Annotate(err, "validateCreateDrac - failed to query old drac for machine %s", machineName).Err()
	}
	if dracs != nil && len(dracs) > 0 {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("validateCreateDrac - There is already a drac %s associated with machine %s.\n"+
			"Please delete that drac and then add the new drac to the machine.", dracs[0].GetName(), machineName))
	}

	// Aggregate resource to check if machine does not exist
	resourcesNotFound := []*Resource{GetMachineResource(machineName)}
	// Aggregate resource to check if resources referenced by the drac does not exist
	if switchID := drac.GetSwitchInterface().GetSwitch(); switchID != "" {
		resourcesNotFound = append(resourcesNotFound, GetSwitchResource(switchID))
	}
	// Check if resources does not exist
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
				return status.Error(codes.InvalidArgument, "validateDracUpdateMask - name cannot be updated, delete and create a new drac instead")
			case "update_time":
				return status.Error(codes.InvalidArgument, "validateDracUpdateMask - update_time cannot be updated, it is a Output only field")
			case "switch":
			case "portName":
			case "machine":
			case "macAddress":
			case "tags":
				// valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "validateDracUpdateMask - unsupported update mask path %q", path)
			}
		}
	}
	return nil
}

func getDracHistoryClient(m *ufspb.Drac) *HistoryClient {
	return &HistoryClient{
		netUdt: &networkUpdater{
			Hostname: m.Name,
		},
	}
}
