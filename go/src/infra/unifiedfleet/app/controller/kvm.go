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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/registration"
	ufsUtil "infra/unifiedfleet/app/util"
)

// CreateKVM creates a new kvm in datastore.
func CreateKVM(ctx context.Context, kvm *ufspb.KVM, rackName string) (*ufspb.KVM, error) {
	// TODO(eshwarn): Add logic for Chrome OS
	changes := LogKVMChanges(nil, kvm)
	f := func(ctx context.Context) error {
		// 1. Validate the input
		if err := validateCreateKVM(ctx, kvm, rackName); err != nil {
			return err
		}

		// 2. Get rack to associate the kvm
		rack, err := GetRack(ctx, rackName)
		if err != nil {
			return err
		}

		// 3. Update the rack with new kvm information
		if cs, err := addKVMToRack(ctx, rack, kvm.Name); err == nil {
			changes = append(changes, cs...)
		} else {
			return err
		}

		// 4. Create a kvm entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err = registration.BatchUpdateKVMs(ctx, []*ufspb.KVM{kvm}); err != nil {
			return errors.Annotate(err, "Unable to create kvm %s", kvm.Name).Err()
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create kvm in datastore: %s", err)
		return nil, err
	}
	SaveChangeEvents(ctx, changes)
	return kvm, nil
}

// UpdateKVM updates kvm in datastore.
func UpdateKVM(ctx context.Context, kvm *ufspb.KVM, rackName string) (*ufspb.KVM, error) {
	// TODO(eshwarn): Add logic for Chrome OS
	changes := make([]*ufspb.ChangeEvent, 0)
	f := func(ctx context.Context) error {
		// 1. Validate the input
		if err := validateUpdateKVM(ctx, kvm, rackName); err != nil {
			return err
		}

		oldKVM, _ := registration.GetKVM(ctx, kvm.GetName())
		changes = append(changes, LogKVMChanges(oldKVM, kvm)...)
		if rackName != "" {
			// 2. Get the old rack associated with kvm
			oldRack, err := getRackForKVM(ctx, kvm.Name)
			if err != nil {
				return err
			}

			// User is trying to associate this kvm with a different rack.
			if oldRack.Name != rackName {
				// 3. Get rack to associate the kvm
				rack, err := GetRack(ctx, rackName)
				if err != nil {
					return err
				}

				// 4. Remove the association between old rack and this kvm.
				if cs, err := removeKVMFromRacks(ctx, []*ufspb.Rack{oldRack}, kvm.Name); err == nil {
					changes = append(changes, cs...)
				} else {
					return err
				}

				// 5. Update the rack with new kvm information
				if cs, err := addKVMToRack(ctx, rack, kvm.Name); err == nil {
					changes = append(changes, cs...)
				} else {
					return err
				}
			}
		}

		// 6. Update kvm entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction. Datastore doesnt allow nested transactions.
		if _, err := registration.BatchUpdateKVMs(ctx, []*ufspb.KVM{kvm}); err != nil {
			return errors.Annotate(err, "Unable to update kvm %s", kvm.Name).Err()
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to update kvm in datastore: %s", err)
		return nil, err
	}
	SaveChangeEvents(ctx, changes)
	return kvm, nil
}

// GetKVM returns kvm for the given id from datastore.
func GetKVM(ctx context.Context, id string) (*ufspb.KVM, error) {
	return registration.GetKVM(ctx, id)
}

// ListKVMs lists the kvms
func ListKVMs(ctx context.Context, pageSize int32, pageToken string) ([]*ufspb.KVM, string, error) {
	return registration.ListKVMs(ctx, pageSize, pageToken)
}

// DeleteKVM deletes the kvm in datastore
//
// For referential data intergrity,
// 1. Validate if this kvm is not referenced by other resources in the datastore.
// 2. Delete the kvm
// 3. Get the rack associated with this kvm
// 4. Update the rack by removing the association with this kvm
func DeleteKVM(ctx context.Context, id string) error {
	changes := LogKVMChanges(&ufspb.KVM{Name: id}, nil)
	f := func(ctx context.Context) error {
		// 1. Validate input
		if err := validateDeleteKVM(ctx, id); err != nil {
			return errors.Annotate(err, "Validation failed - unable to delete kvm %s", id).Err()
		}

		// 2. Delete the kvm
		if err := registration.DeleteKVM(ctx, id); err != nil {
			return errors.Annotate(err, "Delete failed - unable to delete kvm %s", id).Err()
		}

		// 3. Get the rack associated with kvm
		racks, err := registration.QueryRackByPropertyName(ctx, "kvm_ids", id, false)
		if err != nil {
			return errors.Annotate(err, "Unable to query rack for kvm %s", id).Err()
		}
		if racks == nil || len(racks) == 0 {
			logging.Warningf(ctx, "No rack associated with the kvm %s. Data discrepancy error.\n", id)
			return nil
		}
		if len(racks) > 1 {
			logging.Warningf(ctx, "More than one rack associated with the kvm %s. Data discrepancy error.\n", id)
		}

		// 4. Remove the association between the rack and this kvm.
		cs, err := removeKVMFromRacks(ctx, racks, id)
		if err != nil {
			return err
		}
		changes = append(changes, cs...)
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete kvm in datastore: %s", err)
		return err
	}
	SaveChangeEvents(ctx, changes)
	return nil
}

// ReplaceKVM replaces an old KVM with new KVM in datastore
//
// It does a delete of old kvm and create of new KVM.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old KVM, it will get all the resources referencing
// the old KVM. It will update all the resources which were referencing
// the old KVM(got in the last step) with new KVM.
// Deletes the old KVM.
// Creates the new KVM.
// This will preserve data integrity in the system.
func ReplaceKVM(ctx context.Context, oldKVM *ufspb.KVM, newKVM *ufspb.KVM) (*ufspb.KVM, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateDeleteKVM validates if a KVM can be deleted
//
// Checks if this KVM(KVMID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteKVM(ctx context.Context, id string) error {
	machines, err := registration.QueryMachineByPropertyName(ctx, "kvm_id", id, true)
	if err != nil {
		return err
	}
	if len(machines) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("KVM %s cannot be deleted because there are other resources which are referring this KVM.", id))
		if len(machines) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachines referring the KVM:\n"))
			for _, machine := range machines {
				errorMsg.WriteString(machine.Name + ", ")
			}
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

// validateCreateKVM validates if a kvm can be created
//
// check if the kvm already exists
// check if the rack and resources referenced by kvm does not exist
func validateCreateKVM(ctx context.Context, kvm *ufspb.KVM, rackName string) error {
	// 1. Check if kvm already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetKVMResource(kvm.Name)}, nil); err != nil {
		return err
	}

	// Aggregate resource to check if rack does not exist
	resourcesNotFound := []*Resource{GetRackResource(rackName)}
	// Aggregate resource to check if resources referenced by the kvm does not exist
	if chromePlatformID := kvm.GetChromePlatform(); chromePlatformID != "" {
		resourcesNotFound = append(resourcesNotFound, GetChromePlatformResource(chromePlatformID))
	}
	// 2. Check if resources does not exist
	return ResourceExist(ctx, resourcesNotFound, nil)
}

// validateUpdateKVM validates if a kvm can be updated
//
// check if kvm, rack and resources referenced kvm does not exist
func validateUpdateKVM(ctx context.Context, kvm *ufspb.KVM, rackName string) error {
	// Aggregate resource to check if kvm does not exist
	resourcesNotFound := []*Resource{GetKVMResource(kvm.Name)}
	// Aggregate resource to check if rack does not exist
	if rackName != "" {
		resourcesNotFound = append(resourcesNotFound, GetRackResource(rackName))
	}
	// Aggregate resource to check if resources referenced by the kvm does not exist
	if chromePlatformID := kvm.GetChromePlatform(); chromePlatformID != "" {
		resourcesNotFound = append(resourcesNotFound, GetChromePlatformResource(chromePlatformID))
	}
	// Check if resources does not exist
	return ResourceExist(ctx, resourcesNotFound, nil)
}

// addKVMToRack adds the kvm info to the rack and updates
// the rack in datastore.
// Must be called within a transaction as BatchUpdateRacks is a non-atomic operation
func addKVMToRack(ctx context.Context, rack *ufspb.Rack, kvmName string) ([]*ufspb.ChangeEvent, error) {
	if rack == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Rack is nil")
	}
	if rack.GetChromeBrowserRack() == nil {
		errorMsg := fmt.Sprintf("Rack %s is not a browser rack", rack.Name)
		return nil, status.Errorf(codes.FailedPrecondition, errorMsg)
	}
	kvms := []string{kvmName}
	if rack.GetChromeBrowserRack().GetKvms() != nil {
		kvms = rack.GetChromeBrowserRack().GetKvms()
		kvms = append(kvms, kvmName)
	}
	old := proto.Clone(rack).(*ufspb.Rack)
	rack.GetChromeBrowserRack().Kvms = kvms
	_, err := registration.BatchUpdateRacks(ctx, []*ufspb.Rack{rack})
	if err != nil {
		return nil, errors.Annotate(err, "Unable to update rack %s with kvm %s information", rack.Name, kvmName).Err()
	}
	return LogRackChanges(old, rack), nil
}

// getRackForKVM return rack associated with the kvm.
func getRackForKVM(ctx context.Context, kvmName string) (*ufspb.Rack, error) {
	racks, err := registration.QueryRackByPropertyName(ctx, "kvm_ids", kvmName, false)
	if err != nil {
		return nil, errors.Annotate(err, "Unable to query rack for kvm %s", kvmName).Err()
	}
	if racks == nil || len(racks) == 0 {
		errorMsg := fmt.Sprintf("No rack associated with the kvm %s. Data discrepancy error.\n", kvmName)
		return nil, status.Errorf(codes.Internal, errorMsg)
	}
	if len(racks) > 1 {
		errorMsg := fmt.Sprintf("More than one rack associated the kvm %s. Data discrepancy error.\n", kvmName)
		return nil, status.Errorf(codes.Internal, errorMsg)
	}
	return racks[0], nil
}

// removeKVMFromRacks removes the kvm info from racks and
// updates the racks in datastore.
// Must be called within a transaction as BatchUpdateRacks is a non-atomic operation
func removeKVMFromRacks(ctx context.Context, racks []*ufspb.Rack, id string) ([]*ufspb.ChangeEvent, error) {
	changes := make([]*ufspb.ChangeEvent, 0)
	for _, rack := range racks {
		if rack.GetChromeBrowserRack() == nil {
			errorMsg := fmt.Sprintf("Rack %s is not a browser rack", rack.Name)
			return nil, status.Errorf(codes.FailedPrecondition, errorMsg)
		}
		kvms := rack.GetChromeBrowserRack().GetKvms()
		kvms = ufsUtil.RemoveStringEntry(kvms, id)
		old := proto.Clone(rack).(*ufspb.Rack)
		rack.GetChromeBrowserRack().Kvms = kvms
		changes = append(changes, LogRackChanges(old, rack)...)
	}
	_, err := registration.BatchUpdateRacks(ctx, racks)
	if err != nil {
		return nil, errors.Annotate(err, "Unable to remove kvm information %s from rack", id).Err()
	}
	return changes, nil
}
