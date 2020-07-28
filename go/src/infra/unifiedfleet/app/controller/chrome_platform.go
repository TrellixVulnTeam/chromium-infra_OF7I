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
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/registration"
)

// CreateChromePlatform creates a new chromeplatform in datastore.
func CreateChromePlatform(ctx context.Context, chromeplatform *ufspb.ChromePlatform) (*ufspb.ChromePlatform, error) {
	return configuration.CreateChromePlatform(ctx, chromeplatform)
}

// UpdateChromePlatform updates chromeplatform in datastore.
func UpdateChromePlatform(ctx context.Context, chromeplatform *ufspb.ChromePlatform) (*ufspb.ChromePlatform, error) {
	return configuration.UpdateChromePlatform(ctx, chromeplatform)
}

// GetChromePlatform returns chromeplatform for the given id from datastore.
func GetChromePlatform(ctx context.Context, id string) (*ufspb.ChromePlatform, error) {
	return configuration.GetChromePlatform(ctx, id)
}

// ListChromePlatforms lists the chromeplatforms
func ListChromePlatforms(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.ChromePlatform, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, configuration.GetChromePlatformIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing chromeplatforms").Err()
		}
	}
	return configuration.ListChromePlatforms(ctx, pageSize, pageToken, filterMap, keysOnly)
}

// DeleteChromePlatform deletes the chromeplatform in datastore
//
// For referential data intergrity,
// Delete if this ChromePlatform is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteChromePlatform(ctx context.Context, id string) error {
	err := validateDeleteChromePlatform(ctx, id)
	if err != nil {
		return err
	}
	return configuration.DeleteChromePlatform(ctx, id)
}

// ImportChromePlatforms inserts chrome platforms to datastore.
func ImportChromePlatforms(ctx context.Context, platforms []*ufspb.ChromePlatform, pageSize int) (*datastore.OpResults, error) {
	deleteNonExistingPlatforms(ctx, platforms, pageSize)
	return configuration.ImportChromePlatforms(ctx, platforms)
}

func deleteNonExistingPlatforms(ctx context.Context, platforms []*ufspb.ChromePlatform, pageSize int) (*datastore.OpResults, error) {
	resMap := make(map[string]bool)
	for _, r := range platforms {
		resMap[r.GetName()] = true
	}
	resp, err := configuration.GetAllChromePlatforms(ctx)
	if err != nil {
		return nil, err
	}
	var toDelete []string
	for _, sr := range resp.Passed() {
		s := sr.Data.(*ufspb.ChromePlatform)
		if _, ok := resMap[s.GetName()]; !ok {
			toDelete = append(toDelete, s.GetName())
		}
	}
	logging.Debugf(ctx, "Deleting %d non-existing platforms", len(toDelete))
	return deleteByPage(ctx, toDelete, pageSize, configuration.DeleteChromePlatforms), nil
}

// ReplaceChromePlatform replaces an old ChromePlatform with new ChromePlatform in datastore
//
// It does a delete of old chromeplatform and create of new ChromePlatform.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old ChromePlatform, it will get all the resources referencing
// the old ChromePlatform. It will update all the resources which were referencing
// the old ChromePlatform(got in the last step) with new ChromePlatform.
// Deletes the old ChromePlatform.
// Creates the new ChromePlatform.
// This will preserve data integrity in the system.
func ReplaceChromePlatform(ctx context.Context, oldChromePlatform *ufspb.ChromePlatform, newChromePlatform *ufspb.ChromePlatform) (*ufspb.ChromePlatform, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateDeleteChromePlatform validates if a ChromePlatform can be deleted
//
// Checks if this ChromePlatform(ChromePlatformID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteChromePlatform(ctx context.Context, id string) error {
	machines, err := registration.QueryMachineByPropertyName(ctx, "chrome_platform_id", id, true)
	if err != nil {
		return err
	}
	kvms, err := registration.QueryKVMByPropertyName(ctx, "chrome_platform_id", id, true)
	if err != nil {
		return err
	}
	if len(machines) > 0 || len(kvms) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("ChromePlatform %s cannot be deleted because there are other resources which are referring this ChromePlatform.", id))
		if len(machines) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachines referring the ChromePlatform:\n"))
			for _, machine := range machines {
				errorMsg.WriteString(machine.Name + ", ")
			}
		}
		if len(kvms) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nKVMs referring the ChromePlatform:\n"))
			for _, kvm := range kvms {
				errorMsg.WriteString(kvm.Name + ", ")
			}
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}
