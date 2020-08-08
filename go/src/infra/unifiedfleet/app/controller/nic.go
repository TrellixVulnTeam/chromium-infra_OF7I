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
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/configuration"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	ufsUtil "infra/unifiedfleet/app/util"
)

// CreateNic creates a new nic in datastore.
//
// Checks if the resources referenced by the Nic input already exists
// in the system before creating a new Nic
func CreateNic(ctx context.Context, nic *ufspb.Nic, machineName string) (*ufspb.Nic, error) {
	f := func(ctx context.Context) error {
		hc := getNicHistoryClient(nic)
		// 1. Validate the input
		if err := validateCreateNic(ctx, nic, machineName); err != nil {
			return err
		}

		// 2. Get browser machine to associate the nic
		machine, err := getBrowserMachine(ctx, machineName)
		if err != nil {
			return err
		}

		// Fill the machine/rack/lab to nic OUTPUT only fields
		nic.Machine = machine.GetName()
		nic.Rack = machine.GetLocation().GetRack()
		nic.Lab = machine.GetLocation().GetLab().String()

		// 3. Update the browser machine with new nic information
		if err := addNicToBrowserMachine(ctx, machine, nic.Name, hc); err != nil {
			return err
		}

		// 4. Create a nic entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err = registration.BatchUpdateNics(ctx, []*ufspb.Nic{nic}); err != nil {
			return errors.Annotate(err, "Unable to create nic %s", nic.Name).Err()
		}
		hc.LogNicChanges(nil, nic)
		return hc.SaveChangeEvents(ctx)
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
func UpdateNic(ctx context.Context, nic *ufspb.Nic, machineName string, mask *field_mask.FieldMask) (*ufspb.Nic, error) {
	f := func(ctx context.Context) error {
		hc := getNicHistoryClient(nic)
		// 1. Validate the input
		if err := validateUpdateNic(ctx, nic, machineName, mask); err != nil {
			return errors.Annotate(err, "UpdateNic - validation failed").Err()
		}

		oldNic, err := registration.GetNic(ctx, nic.GetName())
		if err != nil {
			return errors.Annotate(err, "UpdateNic - get nic %s failed", nic.GetName()).Err()
		}
		// Copy the machine/rack/lab to nic OUTPUT only fields from already existing nic
		nic.Machine = oldNic.GetMachine()
		nic.Rack = oldNic.GetRack()
		nic.Lab = oldNic.GetLab()

		if machineName != "" {
			// 2. Get the old browser machine associated with nic
			oldMachine, err := getBrowserMachineForNic(ctx, nic.Name)
			if err != nil {
				return errors.Annotate(err, "UpdateNic - query machine for nic %s failed", nic.GetName()).Err()
			}

			// User is trying to associate this nic with a different browser machine.
			if oldMachine.Name != machineName {
				// 3. Get browser machine to associate the nic
				machine, err := getBrowserMachine(ctx, machineName)
				if err != nil {
					return errors.Annotate(err, "UpdateNic - get browser machine %s failed", machineName).Err()
				}

				// Fill the machine/rack/lab to nic OUTPUT only fields
				nic.Machine = machine.GetName()
				nic.Rack = machine.GetLocation().GetRack()
				nic.Lab = machine.GetLocation().GetLab().String()

				// 4. Remove the association between old browser machine and this nic.
				if err := removeNicFromBrowserMachines(ctx, []*ufspb.Machine{oldMachine}, nic.Name, hc); err != nil {
					return err
				}

				// 5. Update the browser machine with new nic information
				if err := addNicToBrowserMachine(ctx, machine, nic.Name, hc); err != nil {
					return err
				}
			}
		}

		// Partial update by field mask
		if mask != nil && len(mask.Paths) > 0 {
			nic, err = processNicUpdateMask(oldNic, nic, mask)
			if err != nil {
				return errors.Annotate(err, "UpdateNic - processing update mask failed").Err()
			}
		} else {
			// This check is for json file input
			// User is not allowed to update mac address of a nic
			// instead user has to delete the old nic and add new nic with new mac address
			// macaddress is associated with DHCP config, so we dont allow mac address update for a nic
			if oldNic.GetMacAddress() != "" && oldNic.GetMacAddress() != nic.GetMacAddress() {
				return status.Error(codes.InvalidArgument, "UpdateNic - This nic's mac address is already set. "+
					"Updating mac address for the nic is not allowed.\nInstead delete the nic and add a new nic with updated mac address.")
			}
		}

		// 6. Update nic entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction. Datastore doesnt allow nested transactions.
		if _, err := registration.BatchUpdateNics(ctx, []*ufspb.Nic{nic}); err != nil {
			return errors.Annotate(err, "UpdateNic - unable to batch update nic %s", nic.Name).Err()
		}
		hc.LogNicChanges(oldNic, nic)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "UpdateNic - failed to update nic %s in datastore", nic.Name).Err()
	}
	return nic, nil
}

// processNicUpdateMask process update field mask to get only specific update
// fields and return a complete nic object with updated and existing fields
func processNicUpdateMask(oldNic *ufspb.Nic, nic *ufspb.Nic, mask *field_mask.FieldMask) (*ufspb.Nic, error) {
	// update the fields in the existing/old nic
	for _, path := range mask.Paths {
		switch path {
		case "machine":
			// In the previous step we have already checked for machineName != ""
			// and got the new values for OUTPUT only fields in new nic object,
			// assign them to oldnic.
			oldNic.Machine = nic.GetMachine()
			oldNic.Rack = nic.GetRack()
			oldNic.Lab = nic.GetLab()
		case "macAddress":
			if oldNic.GetMacAddress() != "" {
				return oldNic, status.Error(codes.InvalidArgument, "processNicUpdateMask - This nic's mac address is already set. "+
					"Updating mac address for the nic is not allowed.\nInstead delete the nic and add a new nic with updated mac address.")
			}
			oldNic.MacAddress = nic.GetMacAddress()
		case "switch":
			if oldNic.GetSwitchInterface() == nil {
				oldNic.SwitchInterface = &ufspb.SwitchInterface{
					Switch: nic.GetSwitchInterface().GetSwitch(),
				}
			} else {
				oldNic.GetSwitchInterface().Switch = nic.GetSwitchInterface().GetSwitch()
			}
		case "port":
			if oldNic.GetSwitchInterface() == nil {
				oldNic.SwitchInterface = &ufspb.SwitchInterface{
					Port: nic.GetSwitchInterface().GetPort(),
				}
			} else {
				oldNic.GetSwitchInterface().Port = nic.GetSwitchInterface().GetPort()
			}
		case "tags":
			oldNic.Tags = mergeTags(oldNic.GetTags(), nic.GetTags())
		}
	}
	// return existing/old nic with new updated values
	return oldNic, nil
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
func removeNicFromBrowserMachines(ctx context.Context, machines []*ufspb.Machine, id string, hc *HistoryClient) error {
	for _, machine := range machines {
		oldMachine := proto.Clone(machine).(*ufspb.Machine)
		if machine.GetChromeBrowserMachine() == nil {
			errorMsg := fmt.Sprintf("Machine %s is not a browser machine", machine.Name)
			return status.Errorf(codes.FailedPrecondition, errorMsg)
		}
		nics := machine.GetChromeBrowserMachine().GetNics()
		nics = ufsUtil.RemoveStringEntry(nics, id)
		machine.GetChromeBrowserMachine().Nics = nics
		hc.LogMachineChanges(oldMachine, machine)
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
func addNicToBrowserMachine(ctx context.Context, machine *ufspb.Machine, nicName string, hc *HistoryClient) error {
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
	oldMachine := proto.Clone(machine).(*ufspb.Machine)
	machine.GetChromeBrowserMachine().Nics = nics
	_, err := registration.BatchUpdateMachines(ctx, []*ufspb.Machine{machine})
	if err != nil {
		return errors.Annotate(err, "Unable to update machine %s with nic %s information", machine.Name, nicName).Err()
	}
	hc.LogMachineChanges(oldMachine, machine)
	return nil
}

// GetNic returns nic for the given id from datastore.
func GetNic(ctx context.Context, id string) (*ufspb.Nic, error) {
	return registration.GetNic(ctx, id)
}

// ListNics lists the nics
func ListNics(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.Nic, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, registration.GetNicIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing nics").Err()
		}
	}
	return registration.ListNics(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteNic deletes the nic in datastore
//
// For referential data intergrity,
// 1. Delete the nic
// 2. Get the machine associated with this nic
// 3. Update the machine by removing the association with this nic
func DeleteNic(ctx context.Context, id string) error {
	f := func(ctx context.Context) error {
		hc := getNicHistoryClient(&ufspb.Nic{Name: id})
		// Delete the nic
		if err := registration.DeleteNic(ctx, id); err != nil {
			return errors.Annotate(err, "Unable to delete nic %s", id).Err()
		}
		hc.LogNicChanges(&ufspb.Nic{Name: id}, nil)

		// Get the machine associated with nic
		machine, err := getBrowserMachineForNic(ctx, id)
		if err != nil {
			logging.Debugf(ctx, "Failed to query machine for nic %s", err.Error())
		}
		if machine != nil {
			lses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", machine.GetName(), false)
			if err != nil {
				return errors.Annotate(err, "Fail to query host by machine %s", machine.GetName()).Err()
			}
			for _, lse := range lses {
				if lse.GetNic() == id {
					return status.Errorf(codes.InvalidArgument, "nic %s is used by host %s", id, lse.GetName())
				}
			}

			// Remove the association between the machine and this nic.
			if err := removeNicFromBrowserMachines(ctx, []*ufspb.Machine{machine}, id, hc); err != nil {
				return err
			}
		}

		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete entity in datastore: %s", err)
		return err
	}
	return nil
}

// ImportNetworkInterfaces creates or updates a batch of nics, dracs, and dhcps in datastore
func ImportNetworkInterfaces(ctx context.Context, nics []*crimson.NIC, dracs []*crimson.DRAC, machines []*crimson.Machine, pageSize int) (*ufsds.OpResults, error) {
	var allRes ufsds.OpResults
	newNics, newDracs, dhcps, _, _ := ufsUtil.ProcessNetworkInterfaces(nics, dracs, machines)
	// Please note that the importing here is not in one transaction, which
	// actually may cause data incompleteness. But as the importing job
	// will be triggered periodically, such incompleteness that's caused by
	// potential failure will be ignored.
	deleteNonExistingNics(ctx, newNics, pageSize)
	logging.Debugf(ctx, "Importing %d nics", len(newNics))
	for i := 0; ; i += pageSize {
		end := ufsUtil.Min(i+pageSize, len(newNics))
		res, err := registration.ImportNics(ctx, newNics[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(newNics) {
			break
		}
	}
	deleteNonExistingDracs(ctx, newDracs, pageSize)
	logging.Debugf(ctx, "Importing %d dracs", len(newDracs))
	for i := 0; ; i += pageSize {
		end := ufsUtil.Min(i+pageSize, len(newDracs))
		res, err := registration.ImportDracs(ctx, newDracs[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(newDracs) {
			break
		}
	}
	logging.Debugf(ctx, "Importing %d dhcps", len(dhcps))
	for i := 0; ; i += pageSize {
		end := ufsUtil.Min(i+pageSize, len(dhcps))
		res, err := configuration.ImportDHCPConfigs(ctx, dhcps[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(dhcps) {
			break
		}
	}
	return &allRes, nil
}

func deleteNonExistingNics(ctx context.Context, nics []*ufspb.Nic, pageSize int) (*ufsds.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range nics {
		resMap[r.GetName()] = true
	}
	resp, err := registration.GetAllNics(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.Nic)
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
		}
	}
	logging.Debugf(ctx, "Deleting %d non-existing nics", len(toDelete))
	return deleteByPage(ctx, toDelete, pageSize, registration.DeleteNics), nil
}

func deleteNonExistingDracs(ctx context.Context, dracs []*ufspb.Drac, pageSize int) (*ufsds.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range dracs {
		resMap[r.GetName()] = true
	}
	resp, err := registration.GetAllDracs(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.Drac)
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
		}
	}
	logging.Debugf(ctx, "Deleting %d non-existing dracs", len(toDelete))
	logging.Debugf(ctx, "Deleting %d non-existing drac-related dhcps", len(toDelete))
	allRes := *deleteByPage(ctx, toDelete, pageSize, registration.DeleteDracs)
	allRes = append(allRes, *deleteByPage(ctx, toDelete, pageSize, configuration.DeleteDHCPs)...)
	return &allRes, nil
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
func validateUpdateNic(ctx context.Context, nic *ufspb.Nic, machineName string, mask *field_mask.FieldMask) error {
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

	// check if resources does not exist
	if err := ResourceExist(ctx, resourcesNotFound, nil); err != nil {
		return err
	}

	return validateNicUpdateMask(mask)
}

// validateNicUpdateMask validates the update mask for nic update
func validateNicUpdateMask(mask *field_mask.FieldMask) error {
	if mask != nil {
		// validate the give field mask
		for _, path := range mask.Paths {
			switch path {
			case "name":
				return status.Error(codes.InvalidArgument, "validateUpdateNic - name cannot be updated, delete and create a new nic instead")
			case "update_time":
				return status.Error(codes.InvalidArgument, "validateUpdateNic - update_time cannot be updated, it is a Output only field")
			case "switch":
			case "port":
			case "machine":
			case "macAddress":
			case "tags":
				// valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "validateUpdateNic - unsupported update mask path %q", path)
			}
		}
	}
	return nil
}

func getNicHistoryClient(m *ufspb.Nic) *HistoryClient {
	return &HistoryClient{
		stUdt: &stateUpdater{
			ResourceName: ufsUtil.AddPrefix(ufsUtil.NicCollection, m.Name),
		},
		netUdt: &networkUpdater{
			Hostname: m.Name,
		},
	}
}
