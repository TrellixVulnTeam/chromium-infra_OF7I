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
	fleetds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/registration"
)

// CreateDrac creates a new drac in datastore.
//
// Checks if the resources referenced by the Drac input already exists
// in the system before creating a new Drac
func CreateDrac(ctx context.Context, drac *fleet.Drac) (*fleet.Drac, error) {
	err := validateDrac(ctx, drac)
	if err != nil {
		return nil, err
	}
	return registration.CreateDrac(ctx, drac)
}

// UpdateDrac updates drac in datastore.
//
// Checks if the resources referenced by the Drac input already exists
// in the system before updating a Drac
func UpdateDrac(ctx context.Context, drac *fleet.Drac) (*fleet.Drac, error) {
	err := validateDrac(ctx, drac)
	if err != nil {
		return nil, err
	}
	return registration.UpdateDrac(ctx, drac)
}

// GetDrac returns drac for the given id from datastore.
func GetDrac(ctx context.Context, id string) (*fleet.Drac, error) {
	return registration.GetDrac(ctx, id)
}

// ListDracs lists the dracs
func ListDracs(ctx context.Context, pageSize int32, pageToken string) ([]*fleet.Drac, string, error) {
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
func ImportDracs(ctx context.Context, dracs []*fleet.Drac) (*fleetds.OpResults, error) {
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
func ReplaceDrac(ctx context.Context, oldDrac *fleet.Drac, newDrac *fleet.Drac) (*fleet.Drac, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateDrac validates if a drac can be created/updated in the datastore.
//
// Checks if the resources referenced by the given Drac input already exists
// in the system. Returns an error if any resource referenced by the Drac input
// does not exist in the system.
func validateDrac(ctx context.Context, drac *fleet.Drac) error {
	var resources []*Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create Drac %s:\n", drac.Name))

	switchID := drac.GetSwitchInterface().GetSwitch()
	if switchID != "" {
		resources = append(resources, GetSwitchResource(switchID))
	}

	return ResourceExist(ctx, resources, &errorMsg)
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
