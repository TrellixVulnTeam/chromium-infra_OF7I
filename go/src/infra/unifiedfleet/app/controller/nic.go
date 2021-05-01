// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
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
func CreateNic(ctx context.Context, nic *ufspb.Nic) (*ufspb.Nic, error) {
	f := func(ctx context.Context) error {
		hc := getNicHistoryClient(nic)

		// Get browser machine to associate the nic
		machine, err := getBrowserMachine(ctx, nic.GetMachine())
		if err != nil {
			return errors.Annotate(err, "CreateNic - failed to get machine %s", nic.GetMachine()).Err()
		}

		// Validate the input
		if err := validateCreateNic(ctx, nic, machine); err != nil {
			return errors.Annotate(err, "CreateNic - validation failed").Err()
		}

		// Fill the rack/zone to nic OUTPUT only fields for indexing nic table
		nic.Rack = machine.GetLocation().GetRack()
		nic.Zone = machine.GetLocation().GetZone().String()

		// Create a nic entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err = registration.BatchUpdateNics(ctx, []*ufspb.Nic{nic}); err != nil {
			return errors.Annotate(err, "CreateNic - unable to batch update nic %s", nic.Name).Err()
		}
		hc.LogNicChanges(nil, nic)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "CreateNic - unable to create nic %s", nic.Name).Err()
	}
	return nic, nil
}

// UpdateNic updates nic in datastore.
//
// Checks if the resources referenced by the Nic input already exists
// in the system before updating a Nic
func UpdateNic(ctx context.Context, nic *ufspb.Nic, mask *field_mask.FieldMask) (*ufspb.Nic, error) {
	f := func(ctx context.Context) error {
		hc := getNicHistoryClient(nic)

		// Get old/existing nic
		oldNic, err := registration.GetNic(ctx, nic.GetName())
		if err != nil {
			return errors.Annotate(err, "UpdateNic - get nic %s failed", nic.GetName()).Err()
		}

		// Validate the input
		if err := validateUpdateNic(ctx, oldNic, nic, mask); err != nil {
			return errors.Annotate(err, "UpdateNic - validation failed").Err()
		}

		oldNicCopy := proto.Clone(oldNic).(*ufspb.Nic)
		// Copy the rack/zone to nic OUTPUT only fields from already existing nic
		nic.Rack = oldNic.GetRack()
		nic.Zone = oldNic.GetZone()

		// Partial update by field mask
		if mask != nil && len(mask.Paths) > 0 {
			nic, err = processNicUpdateMask(ctx, oldNic, nic, mask)
			if err != nil {
				return errors.Annotate(err, "UpdateNic - processing update mask failed").Err()
			}
		} else {
			// This is for complete object input
			if nic.GetMachine() == "" {
				return status.Error(codes.InvalidArgument, "Machine cannot be empty for updating a drac")
			}
		}

		// Check if user provided new machine to associate the nic
		if nic.GetMachine() != oldNicCopy.GetMachine() {
			newMachine, err := verifyNewMachine(ctx, nic.GetMachine())
			if err != nil {
				return err
			}
			// Fill the rack/zone to nic OUTPUT only fields
			nic.Rack = newMachine.GetLocation().GetRack()
			nic.Zone = newMachine.GetLocation().GetZone().String()

			// Update corresponding lses/dhcps/mac address
			if err := updateForMachineChange(ctx, nic, oldNicCopy.GetMachine(), nic.GetMachine(), newMachine.GetRealm(), hc); err != nil {
				return err
			}
		} else if nic.GetMacAddress() != oldNicCopy.GetMacAddress() {
			lses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", nic.GetMachine(), false)
			if err != nil {
				return err
			}
			if len(lses) > 0 {
				logging.Infof(ctx, "updating corresponding dhcp record for host %q", lses[0].GetName())
				// Set the correct hostname for the nic's corresponding dhcp configs
				hc.netUdt.Hostname = lses[0].GetName()
				if _, err := hc.netUdt.updateDHCPWithMac(ctx, nic.GetMacAddress()); err != nil {
					return err
				}
			}
		}

		// Update nic entry
		if _, err := registration.BatchUpdateNics(ctx, []*ufspb.Nic{nic}); err != nil {
			return errors.Annotate(err, "UpdateNic - unable to batch update nic %s", nic.Name).Err()
		}
		hc.LogNicChanges(oldNicCopy, nic)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "UpdateNic - failed to update nic %s in datastore", nic.Name).Err()
	}
	return nic, nil
}

func verifyNewMachine(ctx context.Context, newMachineName string) (*ufspb.Machine, error) {
	// Get browser machine to associate the nic
	machine, err := getBrowserMachine(ctx, newMachineName)
	if err != nil {
		return nil, errors.Annotate(err, "UpdateNic - failed to get browser machine %s", newMachineName).Err()
	}
	// check permission for the new machine realm
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsUpdate, machine.GetRealm()); err != nil {
		return nil, err
	}
	return machine, nil
}

func updateForMachineChange(ctx context.Context, nic *ufspb.Nic, oldMachineName, newMachineName, realm string, hc *HistoryClient) error {
	logging.Infof(ctx, "updating corresponding machine lses")
	lses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", newMachineName, false)
	if err != nil {
		return err
	}
	oldLses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", oldMachineName, false)
	if err != nil {
		return err
	}
	var newLseName string
	var lsesToUpdate []*ufspb.MachineLSE
	var lsesBeforeUpdate []*ufspb.MachineLSE
	if len(lses) > 0 {
		newLseName = lses[0].GetName()
		lsesBeforeUpdate = append(lsesBeforeUpdate, proto.Clone(lses[0]).(*ufspb.MachineLSE))
		lses[0].Nic = nic.GetName()
		lsesToUpdate = append(lsesToUpdate, lses[0])
	}
	var oldLseName string
	if len(oldLses) > 0 {
		oldLseName = oldLses[0].GetName()
		lsesBeforeUpdate = append(lsesBeforeUpdate, proto.Clone(oldLses[0]).(*ufspb.MachineLSE))
		oldLses[0].Nic = ""
		lsesToUpdate = append(lsesToUpdate, oldLses[0])
	}
	if len(lsesToUpdate) > 0 {
		// check permission for potential updating LSE/dhcp
		if err := ufsUtil.CheckPermission(ctx, ufsUtil.InventoriesUpdate, realm); err != nil {
			return err
		}
		if _, err := inventory.BatchUpdateMachineLSEs(ctx, lsesToUpdate); err != nil {
			return err
		}
		// log lse change event
		for i, old := range lsesBeforeUpdate {
			hc.LogMachineLSEChanges(old, lsesToUpdate[i])
		}
	}

	// Update corresponding dhcp records
	logging.Infof(ctx, "Updating dhcp record from %q to %q", oldLseName, newLseName)
	if _, err := hc.netUdt.renameDHCP(ctx, oldLseName, newLseName, nic.GetMacAddress()); err != nil {
		return err
	}
	return nil
}

// processNicUpdateMask process update field mask to get only specific update
// fields and return a complete nic object with updated and existing fields
func processNicUpdateMask(ctx context.Context, oldNic *ufspb.Nic, nic *ufspb.Nic, mask *field_mask.FieldMask) (*ufspb.Nic, error) {
	// update the fields in the existing/old nic
	for _, path := range mask.Paths {
		switch path {
		case "machine":
			oldNic.Machine = nic.GetMachine()
		case "macAddress":
			oldNic.MacAddress = nic.GetMacAddress()
		case "switch":
			if oldNic.GetSwitchInterface() == nil {
				oldNic.SwitchInterface = &ufspb.SwitchInterface{
					Switch: nic.GetSwitchInterface().GetSwitch(),
				}
			} else {
				oldNic.GetSwitchInterface().Switch = nic.GetSwitchInterface().GetSwitch()
			}
		case "portName":
			if oldNic.GetSwitchInterface() == nil {
				oldNic.SwitchInterface = &ufspb.SwitchInterface{
					PortName: nic.GetSwitchInterface().GetPortName(),
				}
			} else {
				oldNic.GetSwitchInterface().PortName = nic.GetSwitchInterface().GetPortName()
			}
		case "tags":
			oldNic.Tags = mergeTags(oldNic.GetTags(), nic.GetTags())
		}
	}
	// For partial update, validate switch interface just before updating in case
	// before we checks the incompleted interface
	if err := validateNicSwitchPort(ctx, oldNic.GetName(), oldNic.GetMachine(), oldNic.GetSwitchInterface()); err != nil {
		return oldNic, err
	}
	// return existing/old nic with new updated values
	return oldNic, nil
}

// GetNic returns nic for the given id from datastore.
func GetNic(ctx context.Context, id string) (*ufspb.Nic, error) {
	return registration.GetNic(ctx, id)
}

// BatchGetNics returns a batch of nics from datastore.
func BatchGetNics(ctx context.Context, ids []string) ([]*ufspb.Nic, error) {
	return registration.BatchGetNics(ctx, ids)
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
	filterMap = resetZoneFilter(filterMap)
	return registration.ListNics(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteNic deletes the nic in datastore
func DeleteNic(ctx context.Context, id string) error {
	return deleteNicHelper(ctx, id, true)
}

func deleteNicHelper(ctx context.Context, id string, inTransaction bool) error {
	f := func(ctx context.Context) error {
		hc := getNicHistoryClient(&ufspb.Nic{Name: id})

		nic, err := registration.GetNic(ctx, id)
		if err != nil {
			return errors.Annotate(err, "DeleteNic - Unable to get nic").Err()
		}

		// Validate the input
		if err := validateDeleteNic(ctx, nic); err != nil {
			return errors.Annotate(err, "DeleteNic - validation failed").Err()
		}

		// Delete the nic
		if err := registration.DeleteNic(ctx, id); err != nil {
			return errors.Annotate(err, "DeleteNic - unable to delete nic %s", id).Err()
		}
		hc.LogNicChanges(nic, nil)
		return hc.SaveChangeEvents(ctx)
	}
	if inTransaction {
		if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
			return errors.Annotate(err, "DeleteNic - failed to delete nic in datastore: %s", id).Err()
		}
		return nil
	}
	return f(ctx)
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
	logging.Infof(ctx, "Importing %d nics", len(newNics))
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
	logging.Infof(ctx, "Importing %d dracs", len(newDracs))
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
	logging.Infof(ctx, "Importing %d dhcps", len(dhcps))
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
	logging.Infof(ctx, "Deleting %d non-existing nics", len(toDelete))
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
	logging.Infof(ctx, "Deleting %d non-existing dracs", len(toDelete))
	logging.Infof(ctx, "Deleting %d non-existing drac-related dhcps", len(toDelete))
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

// validateDeleteNic validates if a nic can be deleted
func validateDeleteNic(ctx context.Context, nic *ufspb.Nic) error {
	machine, err := registration.GetMachine(ctx, nic.GetMachine())
	if err != nil {
		return errors.Annotate(err, "validateDeleteNic - unable to get machine %s", nic.GetMachine()).Err()
	}
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsDelete, machine.GetRealm()); err != nil {
		return err
	}

	// Get the machinelse associated with the nic
	lses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machine_ids", nic.GetMachine(), false)
	if err != nil {
		return errors.Annotate(err, "validateDeleteNic - failed to query host by machine %s", nic.GetMachine()).Err()
	}
	for _, lse := range lses {
		if lse.GetNic() == nic.GetName() {
			return status.Errorf(codes.InvalidArgument, "validateDeleteNic - nic %s is used by host %s", nic.GetName(), lse.GetName())
		}
	}
	return nil
}

// validateCreateNic validates if a nic can be created
//
// check if the nic already exists
// checks if the machine and resources referenced by the nic does not exist
func validateCreateNic(ctx context.Context, nic *ufspb.Nic, machine *ufspb.Machine) error {
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsCreate, machine.GetRealm()); err != nil {
		return err
	}
	// Check if nic already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetNicResource(nic.Name)}, nil); err != nil {
		return err
	}
	if err := validateMacAddress(ctx, nic.GetName(), nic.GetMacAddress()); err != nil {
		return err
	}
	if err := validateNicSwitchPort(ctx, nic.GetName(), nic.GetMachine(), nic.GetSwitchInterface()); err != nil {
		return err
	}
	// Aggregate resource to check if resources referenced by the nic does not exist
	if switchID := nic.GetSwitchInterface().GetSwitch(); switchID != "" {
		if err := ResourceExist(ctx, []*Resource{GetSwitchResource(switchID)}, nil); err != nil {
			return err
		}
	}
	return nil
}

// validateUpdateNic validates if a nic can be updated
//
// checks if nic, machine and resources referecned by the nic does not exist
func validateUpdateNic(ctx context.Context, oldNic *ufspb.Nic, nic *ufspb.Nic, mask *field_mask.FieldMask) error {
	machine, err := registration.GetMachine(ctx, oldNic.GetMachine())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "machine %s not found", oldNic.GetMachine())
	}
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsUpdate, machine.GetRealm()); err != nil {
		return err
	}
	// Aggregate resource to check if does not exist
	var resourcesNotFound []*Resource
	// Aggregate resource to check if machine does not exist
	if nic.GetMachine() != "" {
		resourcesNotFound = append(resourcesNotFound, GetMachineResource(nic.GetMachine()))
	}
	// Aggregate resource to check if resources referenced by the nic does not exist
	if switchID := nic.GetSwitchInterface().GetSwitch(); switchID != "" {
		resourcesNotFound = append(resourcesNotFound, GetSwitchResource(switchID))
	}

	// check if resources does not exist
	if err := ResourceExist(ctx, resourcesNotFound, nil); err != nil {
		return err
	}
	// Check partial update first to avoid unnecessary validations
	if err := validateNicUpdateMask(ctx, nic, mask); err != nil {
		return err
	}
	if err := validateMacAddress(ctx, nic.GetName(), nic.GetMacAddress()); err != nil {
		return err
	}
	if err := validateNicSwitchPort(ctx, nic.GetName(), nic.GetMachine(), nic.GetSwitchInterface()); err != nil {
		return err
	}
	return nil
}

// validateNicUpdateMask validates the update mask for nic update
func validateNicUpdateMask(ctx context.Context, nic *ufspb.Nic, mask *field_mask.FieldMask) error {
	if mask != nil {
		// validate the give field mask
		for _, path := range mask.Paths {
			switch path {
			case "name":
				return status.Error(codes.InvalidArgument, "validateNicUpdateMask - name cannot be updated, delete and create a new nic instead")
			case "update_time":
				return status.Error(codes.InvalidArgument, "validateNicUpdateMask - update_time cannot be updated, it is a Output only field")
			case "switch":
				fallthrough
			case "portName":
				// Check switch interface validity in processNicUpdateMask later.
				if nic.GetSwitchInterface() == nil {
					return status.Error(codes.InvalidArgument, "validateNicUpdateMask - switch interface cannot be empty/nil.")
				}
			case "machine":
				if nic.GetMachine() == "" {
					status.Error(codes.InvalidArgument, "validateNicUpdateMask - machine cannot be empty")
				}
			case "macAddress":
				if err := validateMacAddress(ctx, nic.GetName(), nic.GetMacAddress()); err != nil {
					return err
				}
			case "tags":
				// valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "validateNicUpdateMask - unsupported update mask path %q", path)
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

// RenameNic renames the nic and updates the associated machinelse in datastore
func RenameNic(ctx context.Context, oldNicName, newNicName string) (*ufspb.Nic, error) {
	var nic *ufspb.Nic
	var err error
	f := func(ctx context.Context) error {
		// Get the old Nic
		nic, err = registration.GetNic(ctx, oldNicName)
		if err != nil {
			return errors.Annotate(err, "Unable to get nic %s", oldNicName).Err()
		}

		// Validate the input
		if err := validateRenameNic(ctx, nic, newNicName); err != nil {
			return errors.Annotate(err, "validation failed").Err()
		}

		// Copy for logging
		oldNicCopy := proto.Clone(nic).(*ufspb.Nic)
		hc := getNicHistoryClient(oldNicCopy)

		// Update MachineLSE with new nic name
		if err := updateIndexingForMachineLSE(ctx, "nic", oldNicName, newNicName, hc); err != nil {
			return errors.Annotate(err, "failed to update indexing for hosts").Err()
		}

		// Delete the old nic
		if err := registration.DeleteNic(ctx, oldNicName); err != nil {
			return errors.Annotate(err, "unable to delete nic %s", oldNicName).Err()
		}

		// Create a new nic
		nic.Name = newNicName
		if _, err = registration.BatchUpdateNics(ctx, []*ufspb.Nic{nic}); err != nil {
			return errors.Annotate(err, "unable to batch update nic %s", nic.Name).Err()
		}

		hc.LogNicChanges(oldNicCopy, nic)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "failed to rename nic: %s", oldNicName).Err()
	}
	return nic, nil
}

// validateRenameNic validates if a Nic can be renamed
func validateRenameNic(ctx context.Context, oldNic *ufspb.Nic, newNicName string) error {
	machine, err := registration.GetMachine(ctx, oldNic.GetMachine())
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "machine %s not found", oldNic.GetMachine())
	}
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsUpdate, machine.GetRealm()); err != nil {
		return err
	}
	// Check if new nic name already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetNicResource(newNicName)}, nil); err != nil {
		return err
	}
	return nil
}

// updateIndexingForNic updates indexing for Nic tables
// can be used inside a transaction
func updateIndexingForNic(ctx context.Context, property, oldValue, newValue string, hc *HistoryClient) error {
	var nics []*ufspb.Nic
	var err error
	switch property {
	case "switch":
		nics, err = registration.QueryNicByPropertyName(ctx, "switch_id", oldValue, false)
		if err != nil {
			return errors.Annotate(err, "failed to query nics for switch %s", newValue).Err()
		}
		for _, nic := range nics {
			oldNicCopy := proto.Clone(nic).(*ufspb.Nic)
			nic.GetSwitchInterface().Switch = newValue
			hc.LogNicChanges(oldNicCopy, nic)
		}
	}
	if _, err := registration.BatchUpdateNics(ctx, nics); err != nil {
		return errors.Annotate(err, "unable to batch update nics").Err()
	}
	return nil
}
