// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"google.golang.org/genproto/protobuf/field_mask"

	ufspb "infra/unifiedfleet/api/v1/models"
)

// CreateSchedulingUnit creates a new SchedulingUnit in datastore.
func CreateSchedulingUnit(ctx context.Context, su *ufspb.SchedulingUnit) (*ufspb.SchedulingUnit, error) {
	return nil, nil
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
