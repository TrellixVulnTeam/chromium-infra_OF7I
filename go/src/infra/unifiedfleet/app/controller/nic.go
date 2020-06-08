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

// CreateNic creates a new nic in datastore.
func CreateNic(ctx context.Context, nic *fleet.Nic) (*fleet.Nic, error) {
	return registration.CreateNic(ctx, nic)
}

// UpdateNic updates nic in datastore.
func UpdateNic(ctx context.Context, nic *fleet.Nic) (*fleet.Nic, error) {
	return registration.UpdateNic(ctx, nic)
}

// GetNic returns nic for the given id from datastore.
func GetNic(ctx context.Context, id string) (*fleet.Nic, error) {
	return registration.GetNic(ctx, id)
}

// ListNics lists the nics
func ListNics(ctx context.Context, pageSize int32, pageToken string) ([]*fleet.Nic, string, error) {
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
func ImportNics(ctx context.Context, nics []*fleet.Nic) (*fleetds.OpResults, error) {
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
func ReplaceNic(ctx context.Context, oldNic *fleet.Nic, newNic *fleet.Nic) (*fleet.Nic, error) {
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
		errorMsg.WriteString(fmt.Sprintf("\nMachines referring the Nic:\n"))
		for _, machine := range machines {
			errorMsg.WriteString(machine.Name + ", ")
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}
