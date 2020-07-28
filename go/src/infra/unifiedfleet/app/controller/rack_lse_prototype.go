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
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/inventory"
)

// CreateRackLSEPrototype creates a new racklseprototype in datastore.
func CreateRackLSEPrototype(ctx context.Context, racklseprototype *ufspb.RackLSEPrototype) (*ufspb.RackLSEPrototype, error) {
	return configuration.CreateRackLSEPrototype(ctx, racklseprototype)
}

// UpdateRackLSEPrototype updates racklseprototype in datastore.
func UpdateRackLSEPrototype(ctx context.Context, racklseprototype *ufspb.RackLSEPrototype) (*ufspb.RackLSEPrototype, error) {
	return configuration.UpdateRackLSEPrototype(ctx, racklseprototype)
}

// GetRackLSEPrototype returns racklseprototype for the given id from datastore.
func GetRackLSEPrototype(ctx context.Context, id string) (*ufspb.RackLSEPrototype, error) {
	return configuration.GetRackLSEPrototype(ctx, id)
}

// ListRackLSEPrototypes lists the racklseprototypes
func ListRackLSEPrototypes(ctx context.Context, pageSize int32, pageToken string, filter string, keysOnly bool) ([]*ufspb.RackLSEPrototype, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, configuration.GetRackLSEPrototypeIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing racklseprototypes").Err()
		}
	}
	return configuration.ListRackLSEPrototypes(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteRackLSEPrototype deletes the racklseprototype in datastore
//
// For referential data intergrity,
// Delete if this RackLSEPrototype is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteRackLSEPrototype(ctx context.Context, id string) error {
	err := validateDeleteRackLSEPrototype(ctx, id)
	if err != nil {
		return err
	}
	return configuration.DeleteRackLSEPrototype(ctx, id)
}

// ReplaceRackLSEPrototype replaces an old RackLSEPrototype with new RackLSEPrototype in datastore
//
// It does a delete of old racklseprototype and create of new RackLSEPrototype.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old RackLSEPrototype, it will get all the resources referencing
// the old RackLSEPrototype. It will update all the resources which were referencing
// the old RackLSEPrototype(got in the last step) with new RackLSEPrototype.
// Deletes the old RackLSEPrototype.
// Creates the new RackLSEPrototype.
// This will preserve data integrity in the system.
func ReplaceRackLSEPrototype(ctx context.Context, oldRackLSEPrototype *ufspb.RackLSEPrototype, newRackLSEPrototype *ufspb.RackLSEPrototype) (*ufspb.RackLSEPrototype, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateDeleteRackLSEPrototype validates if a RackLSEPrototype can be deleted
//
// Checks if this RackLSEPrototype(RackLSEPrototypeID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteRackLSEPrototype(ctx context.Context, id string) error {
	racklses, err := inventory.QueryRackLSEByPropertyName(ctx, "racklse_prototype_id", id, true)
	if err != nil {
		return err
	}
	if len(racklses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("RackLSEPrototype %s cannot be deleted because there are other resources which are referring this RackLSEPrototype.", id))
		errorMsg.WriteString(fmt.Sprintf("\nRackLSEs referring the RackLSEPrototype:\n"))
		for _, racklse := range racklses {
			errorMsg.WriteString(racklse.Name + ", ")
		}
		logging.Infof(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}
