// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"

	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"infra/unifiedfleet/app/model/inventory"

	ufspb "infra/unifiedfleet/api/v1/models"
)

// CreateSchedulingUnit creates a new SchedulingUnit in datastore.
func CreateSchedulingUnit(ctx context.Context, su *ufspb.SchedulingUnit) (*ufspb.SchedulingUnit, error) {
	f := func(ctx context.Context) error {
		if err := validateCreateSchedulingUnit(ctx, su); err != nil {
			return err
		}
		if _, err := inventory.BatchUpdateSchedulingUnits(ctx, []*ufspb.SchedulingUnit{su}); err != nil {
			return err
		}
		hc := &HistoryClient{}
		hc.logSchedulingUnitChanges(nil, su)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		return nil, errors.Annotate(err, "unable to create SchedulingUnit %s", su.Name).Err()
	}
	return su, nil
}

// UpdateSchedulingUnit updates existing SchedulingUnit in datastore.
func UpdateSchedulingUnit(ctx context.Context, su *ufspb.SchedulingUnit, mask *field_mask.FieldMask) (*ufspb.SchedulingUnit, error) {
	return nil, nil
}

// GetSchedulingUnit returns SchedulingUnit for the given id from datastore.
func GetSchedulingUnit(ctx context.Context, id string) (*ufspb.SchedulingUnit, error) {
	return nil, nil
}

// DeleteSchedulingUnit deletes the given SchedulingUnit in datastore.
func DeleteSchedulingUnit(ctx context.Context, id string) error {
	return nil
}

// ListSchedulingUnits lists the SchedulingUnits in datastore.
func ListSchedulingUnits(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.SchedulingUnit, string, error) {
	return nil, "", nil
}

// validateCreateSchedulingUnit validates if a SchedulingUnit can be created.
func validateCreateSchedulingUnit(ctx context.Context, su *ufspb.SchedulingUnit) error {
	// Check if SchedulingUnit already exists.
	if err := resourceAlreadyExists(ctx, []*Resource{GetSchedulingUnitResource(su.Name)}, nil); err != nil {
		return err
	}
	// Check if the DUTs/MachineLSEs not found.
	if err := checkIfMachineLSEsExists(ctx, su.GetMachineLSEs()); err != nil {
		return err
	}
	// Check if DUTs/MachineLSEs already used in other SchedulingUnits.
	for _, lse := range su.GetMachineLSEs() {
		schedulingUnits, err := inventory.QuerySchedulingUnitByPropertyNames(ctx, map[string]string{"machinelses": lse}, true)
		if err != nil {
			return errors.Annotate(err, "Failed to query SchedulingUnit for machinelses %s", lse).Err()
		}
		if len(schedulingUnits) > 0 {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("The DUT %s is already associated with SchedulingUnit %s.", lse, schedulingUnits[0].GetName()))
		}
	}
	return nil
}

func checkIfMachineLSEsExists(ctx context.Context, lseNames []string) error {
	var resourcesNotfound []*Resource
	for _, lseName := range lseNames {
		resourcesNotfound = append(resourcesNotfound, GetMachineLSEResource(lseName))
	}
	if err := ResourceExist(ctx, resourcesNotfound, nil); err != nil {
		return err
	}
	return nil
}
