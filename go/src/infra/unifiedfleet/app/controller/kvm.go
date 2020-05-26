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

	fleet "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
)

// CreateKVM creates a new kvm in datastore.
//
// Checks if the resources referenced by the KVM input already exists
// in the system before creating a new KVM
func CreateKVM(ctx context.Context, kvm *fleet.KVM) (*fleet.KVM, error) {
	err := validateKVM(ctx, kvm)
	if err != nil {
		return nil, err
	}
	return registration.CreateKVM(ctx, kvm)
}

// UpdateKVM updates kvm in datastore.
//
// Checks if the resources referenced by the KVM input already exists
// in the system before updating a KVM
func UpdateKVM(ctx context.Context, kvm *fleet.KVM) (*fleet.KVM, error) {
	err := validateKVM(ctx, kvm)
	if err != nil {
		return nil, err
	}
	return registration.UpdateKVM(ctx, kvm)
}

// GetKVM returns kvm for the given id from datastore.
func GetKVM(ctx context.Context, id string) (*fleet.KVM, error) {
	return registration.GetKVM(ctx, id)
}

// ListKVMs lists the kvms
func ListKVMs(ctx context.Context, pageSize int32, pageToken string) ([]*fleet.KVM, string, error) {
	return registration.ListKVMs(ctx, pageSize, pageToken)
}

// DeleteKVM deletes the kvm in datastore
//
// For referential data intergrity,
// Delete if this KVM is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteKVM(ctx context.Context, id string) error {
	err := validateDeleteKVM(ctx, id)
	if err != nil {
		return err
	}
	return registration.DeleteKVM(ctx, id)
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
func ReplaceKVM(ctx context.Context, oldKVM *fleet.KVM, newKVM *fleet.KVM) (*fleet.KVM, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateKVM validates if a kvm can be created/updated in the datastore.
//
// Checks if the resources referenced by the given KVM input already exists
// in the system. Returns an error if any resource referenced by the KVM input
// does not exist in the system.
func validateKVM(ctx context.Context, kvm *fleet.KVM) error {
	var resources []*Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create KVM %s:\n", kvm.Name))

	chromePlatformID := kvm.GetChromePlatform()
	if chromePlatformID != "" {
		resources = append(resources, GetChromePlatformResource(chromePlatformID))
	}

	return ResourceExist(ctx, resources, &errorMsg)
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
	racks, err := registration.QueryRackByPropertyName(ctx, "kvm_ids", id, true)
	if err != nil {
		return err
	}
	racklses, err := inventory.QueryRackLSEByPropertyName(ctx, "kvm_ids", id, true)
	if err != nil {
		return err
	}
	if len(machines) > 0 || len(racks) > 0 || len(racklses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("KVM %s cannot be deleted because there are other resources which are referring this KVM.", id))
		if len(machines) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachines referring the KVM:\n"))
			for _, machine := range machines {
				errorMsg.WriteString(machine.Name + ", ")
			}
		}
		if len(racks) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nRacks referring the KVM:\n"))
			for _, rack := range racks {
				errorMsg.WriteString(rack.Name + ", ")
			}
		}
		if len(racklses) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nRackLSEs referring the KVM:\n"))
			for _, racklse := range racklses {
				errorMsg.WriteString(racklse.Name + ", ")
			}
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}
