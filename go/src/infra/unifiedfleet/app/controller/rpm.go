// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
)

// CreateRPM creates a new rpm in datastore.
func CreateRPM(ctx context.Context, rpm *ufspb.RPM) (*ufspb.RPM, error) {
	return registration.CreateRPM(ctx, rpm)
}

// UpdateRPM updates rpm in datastore.
func UpdateRPM(ctx context.Context, rpm *ufspb.RPM) (*ufspb.RPM, error) {
	return registration.UpdateRPM(ctx, rpm)
}

// GetRPM returns rpm for the given id from datastore.
func GetRPM(ctx context.Context, id string) (*ufspb.RPM, error) {
	return registration.GetRPM(ctx, id)
}

// ListRPMs lists the rpms
func ListRPMs(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.RPM, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, registration.GetRPMIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing rpms").Err()
		}
	}
	return registration.ListRPMs(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteRPM deletes the rpm in datastore
//
// For referential data intergrity,
// Delete if this RPM is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteRPM(ctx context.Context, id string) error {
	err := validateDeleteRPM(ctx, id)
	if err != nil {
		return err
	}
	return registration.DeleteRPM(ctx, id)
}

// ReplaceRPM replaces an old RPM with new RPM in datastore
//
// It does a delete of old rpm and create of new RPM.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old RPM, it will get all the resources referencing
// the old RPM. It will update all the resources which were referencing
// the old RPM(got in the last step) with new RPM.
// Deletes the old RPM.
// Creates the new RPM.
// This will preserve data integrity in the system.
func ReplaceRPM(ctx context.Context, oldRPM *ufspb.RPM, newRPM *ufspb.RPM) (*ufspb.RPM, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateDeleteRPM validates if a RPM can be deleted
//
// Checks if this RPM(RPMID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteRPM(ctx context.Context, id string) error {
	machines, err := registration.QueryMachineByPropertyName(ctx, "rpm_id", id, true)
	if err != nil {
		return err
	}
	racks, err := registration.QueryRackByPropertyName(ctx, "rpm_ids", id, true)
	if err != nil {
		return err
	}
	racklses, err := inventory.QueryRackLSEByPropertyName(ctx, "rpm_ids", id, true)
	if err != nil {
		return err
	}
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "rpm_id", id, true)
	if err != nil {
		return err
	}
	if len(machines) > 0 || len(racks) > 0 || len(racklses) > 0 || len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("RPM %s cannot be deleted because there are other resources which are referring this RPM.", id))
		if len(machines) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachines referring the RPM:\n"))
			for _, machine := range machines {
				errorMsg.WriteString(machine.Name + ", ")
			}
		}
		if len(racks) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nRacks referring the RPM:\n"))
			for _, rack := range racks {
				errorMsg.WriteString(rack.Name + ", ")
			}
		}
		if len(racklses) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nRackLSEs referring the RPM:\n"))
			for _, racklse := range racklses {
				errorMsg.WriteString(racklse.Name + ", ")
			}
		}
		if len(machinelses) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachineLSEs referring the RPM:\n"))
			for _, machinelse := range machinelses {
				errorMsg.WriteString(machinelse.Name + ", ")
			}
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}
