// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	fleet "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/inventory"
)

// CreateRackLSE creates a new racklse in datastore.
//
// Checks if the resources referenced by the RackLSE input already exists
// in the system before creating a new RackLSE
func CreateRackLSE(ctx context.Context, racklse *fleet.RackLSE) (*fleet.RackLSE, error) {
	err := validateRackLSE(ctx, racklse)
	if err != nil {
		return nil, err
	}
	return inventory.CreateRackLSE(ctx, racklse)
}

// UpdateRackLSE updates racklse in datastore.
//
// Checks if the resources referenced by the RackLSE input already exists
// in the system before updating a RackLSE
func UpdateRackLSE(ctx context.Context, racklse *fleet.RackLSE) (*fleet.RackLSE, error) {
	err := validateRackLSE(ctx, racklse)
	if err != nil {
		return nil, err
	}
	return inventory.UpdateRackLSE(ctx, racklse)
}

// GetRackLSE returns racklse for the given id from datastore.
func GetRackLSE(ctx context.Context, id string) (*fleet.RackLSE, error) {
	return inventory.GetRackLSE(ctx, id)
}

// ListRackLSEs lists the racklses
func ListRackLSEs(ctx context.Context, pageSize int32, pageToken string) ([]*fleet.RackLSE, string, error) {
	return inventory.ListRackLSEs(ctx, pageSize, pageToken)
}

// DeleteRackLSE deletes the racklse in datastore
//
// For referential data intergrity,
// Delete if this RackLSE is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteRackLSE(ctx context.Context, id string) error {
	return inventory.DeleteRackLSE(ctx, id)
}

// validateRackLSE validates if a racklse can be created/updated in the datastore.
//
// Checks if the resources referenced by the given RackLSE input already exists
// in the system. Returns an error if any resource referenced by the RackLSE input
// does not exist in the system.
func validateRackLSE(ctx context.Context, racklse *fleet.RackLSE) error {
	var resources []*Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create RackLSE %s:\n", racklse.Name))

	rackIDs := racklse.GetRacks()
	rackLSEPrototypeID := racklse.GetRackLsePrototype()
	kvmIDs := racklse.GetChromeosRackLse().GetKvms()
	rpmIDs := racklse.GetChromeosRackLse().GetRpms()

	if len(rackIDs) != 0 {
		for _, rackID := range rackIDs {
			resources = append(resources, GetRackResource(rackID))
		}
	}
	if rackLSEPrototypeID != "" {
		resources = append(resources, GetRackLSEProtoTypeResource(rackLSEPrototypeID))
	}
	if len(kvmIDs) != 0 {
		for _, kvmID := range kvmIDs {
			resources = append(resources, GetKVMResource(kvmID))
		}
	}
	if len(rpmIDs) != 0 {
		for _, rpmID := range rpmIDs {
			resources = append(resources, GetRPMResource(rpmID))
		}
	}

	return ResourceExist(ctx, resources, &errorMsg)
}
