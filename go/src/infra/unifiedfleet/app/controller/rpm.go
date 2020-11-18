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
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	ufsUtil "infra/unifiedfleet/app/util"
)

// CreateRPM creates a new rpm in datastore.
func CreateRPM(ctx context.Context, rpm *ufspb.RPM) (*ufspb.RPM, error) {
	f := func(ctx context.Context) error {
		hc := getRPMHistoryClient(rpm)
		hc.LogRPMChanges(nil, rpm)

		// Get rack to associate the rpm
		rack, err := GetRack(ctx, rpm.GetRack())
		if err != nil {
			return err
		}

		// Validate the input
		if err := validateCreateRPM(ctx, rpm, rack); err != nil {
			return err
		}

		// Fill the zone to rpm OUTPUT only fields for indexing
		rpm.Zone = rack.GetLocation().GetZone().String()
		rpm.ResourceState = ufspb.State_STATE_REGISTERED

		// Create a rpm entry
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err = registration.BatchUpdateRPMs(ctx, []*ufspb.RPM{rpm}); err != nil {
			return errors.Annotate(err, "Unable to create rpm %s", rpm.Name).Err()
		}

		// Update state
		if err := hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_REGISTERED); err != nil {
			return err
		}
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create rpm in datastore: %s", err)
		return nil, err
	}
	return rpm, nil
}

// UpdateRPM updates rpm in datastore.
func UpdateRPM(ctx context.Context, rpm *ufspb.RPM) (*ufspb.RPM, error) {
	return registration.UpdateRPM(ctx, rpm)
}

// GetRPM returns rpm for the given id from datastore.
func GetRPM(ctx context.Context, id string) (*ufspb.RPM, error) {
	return registration.GetRPM(ctx, id)
}

// BatchGetRPMs returns a batch of rpms from datastore.
func BatchGetRPMs(ctx context.Context, ids []string) ([]*ufspb.RPM, error) {
	return registration.BatchGetRPMs(ctx, ids)
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
	filterMap = resetZoneFilter(filterMap)
	return registration.ListRPMs(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteRPM deletes the rpm in datastore
//
// For referential data intergrity,
// Delete if this RPM is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteRPM(ctx context.Context, id string) error {
	return deleteRPMHelper(ctx, id, true)
}

func deleteRPMHelper(ctx context.Context, id string, inTransaction bool) error {
	f := func(ctx context.Context) error {
		rpm := &ufspb.RPM{Name: id}
		hc := getRPMHistoryClient(rpm)
		hc.LogRPMChanges(rpm, nil)
		// Validate input
		if err := validateDeleteRPM(ctx, id); err != nil {
			return errors.Annotate(err, "Validation failed - unable to delete rpm %s", id).Err()
		}
		if err := registration.DeleteRPM(ctx, id); err != nil {
			return errors.Annotate(err, "Delete failed - unable to delete rpm %s", id).Err()
		}
		// Update state
		hc.stUdt.deleteStateHelper(ctx)
		// Delete ip configs
		if err := hc.netUdt.deleteDHCPHelper(ctx); err != nil {
			return err
		}
		return hc.SaveChangeEvents(ctx)
	}
	if inTransaction {
		if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
			logging.Errorf(ctx, "Failed to delete rpm in datastore: %s", err)
			return err
		}
		return nil
	}
	return f(ctx)
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
	racklses, err := inventory.QueryRackLSEByPropertyName(ctx, "rpm_ids", id, true)
	if err != nil {
		return err
	}
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "rpm_id", id, true)
	if err != nil {
		return err
	}
	if len(machines) > 0 || len(racklses) > 0 || len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("RPM %s cannot be deleted because there are other resources which are referring this RPM.", id))
		if len(machines) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachines referring the RPM:\n"))
			for _, machine := range machines {
				errorMsg.WriteString(machine.Name + ", ")
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

func getRPMHistoryClient(rpm *ufspb.RPM) *HistoryClient {
	return &HistoryClient{
		stUdt: &stateUpdater{
			ResourceName: ufsUtil.AddPrefix(ufsUtil.RPMCollection, rpm.Name),
		},
		netUdt: &networkUpdater{
			Hostname: rpm.Name,
		},
	}
}

// validateCreateRPM validates if a rpm can be created
//
// check if the rpm already exists
// check if the rack and resources referenced by rpm does not exist
func validateCreateRPM(ctx context.Context, rpm *ufspb.RPM, rack *ufspb.Rack) error {
	// Check permission
	if err := ufsUtil.CheckPermission(ctx, ufsUtil.RegistrationsCreate, rack.GetRealm()); err != nil {
		return err
	}
	// Check if rpm already exists
	if err := resourceAlreadyExists(ctx, []*Resource{GetRPMResource(rpm.Name)}, nil); err != nil {
		return err
	}
	if err := validateMacAddress(ctx, rpm.GetName(), rpm.GetMacAddress()); err != nil {
		return err
	}
	return nil
}
