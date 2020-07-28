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

// CreateMachineLSEPrototype creates a new machinelseprototype in datastore.
func CreateMachineLSEPrototype(ctx context.Context, machinelseprototype *ufspb.MachineLSEPrototype) (*ufspb.MachineLSEPrototype, error) {
	return configuration.CreateMachineLSEPrototype(ctx, machinelseprototype)
}

// UpdateMachineLSEPrototype updates machinelseprototype in datastore.
func UpdateMachineLSEPrototype(ctx context.Context, machinelseprototype *ufspb.MachineLSEPrototype) (*ufspb.MachineLSEPrototype, error) {
	return configuration.UpdateMachineLSEPrototype(ctx, machinelseprototype)
}

// GetMachineLSEPrototype returns machinelseprototype for the given id from datastore.
func GetMachineLSEPrototype(ctx context.Context, id string) (*ufspb.MachineLSEPrototype, error) {
	return configuration.GetMachineLSEPrototype(ctx, id)
}

// ListMachineLSEPrototypes lists the machinelseprototypes
func ListMachineLSEPrototypes(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.MachineLSEPrototype, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, configuration.GetMachineLSEPrototypeIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing machinelseprototypes").Err()
		}
	}
	return configuration.ListMachineLSEPrototypes(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteMachineLSEPrototype deletes the machinelseprototype in datastore
//
// For referential data intergrity,
// Delete if this MachineLSEPrototype is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteMachineLSEPrototype(ctx context.Context, id string) error {
	err := validateDeleteMachineLSEPrototype(ctx, id)
	if err != nil {
		return err
	}
	return configuration.DeleteMachineLSEPrototype(ctx, id)
}

// ReplaceMachineLSEPrototype replaces an old MachineLSEPrototype with new MachineLSEPrototype in datastore
//
// It does a delete of old machinelseprototype and create of new MachineLSEPrototype.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old MachineLSEPrototype, it will get all the resources referencing
// the old MachineLSEPrototype. It will update all the resources which were referencing
// the old MachineLSEPrototype(got in the last step) with new MachineLSEPrototype.
// Deletes the old MachineLSEPrototype.
// Creates the new MachineLSEPrototype.
// This will preserve data integrity in the system.
func ReplaceMachineLSEPrototype(ctx context.Context, oldMachineLSEPrototype *ufspb.MachineLSEPrototype, newMachineLSEPrototype *ufspb.MachineLSEPrototype) (*ufspb.MachineLSEPrototype, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateDeleteMachineLSEPrototype validates if a MachineLSEPrototype can be deleted
//
// Checks if this MachineLSEPrototype(MachineLSEPrototypeID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteMachineLSEPrototype(ctx context.Context, id string) error {
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "machinelse_prototype_id", id, true)
	if err != nil {
		return err
	}
	if len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("MachineLSEPrototype %s cannot be deleted because there are other resources which are referring this MachineLSEPrototype.", id))
		errorMsg.WriteString(fmt.Sprintf("\nMachineLSEs referring the MachineLSEPrototype:\n"))
		for _, machinelse := range machinelses {
			errorMsg.WriteString(machinelse.Name + ", ")
		}
		logging.Infof(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}
