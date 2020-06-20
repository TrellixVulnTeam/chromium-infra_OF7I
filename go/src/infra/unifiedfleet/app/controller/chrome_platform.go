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
	"infra/unifiedfleet/app/model/configuration"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/registration"
)

// CreateChromePlatform creates a new chromeplatform in datastore.
func CreateChromePlatform(ctx context.Context, chromeplatform *fleet.ChromePlatform) (*fleet.ChromePlatform, error) {
	return configuration.CreateChromePlatform(ctx, chromeplatform)
}

// UpdateChromePlatform updates chromeplatform in datastore.
func UpdateChromePlatform(ctx context.Context, chromeplatform *fleet.ChromePlatform) (*fleet.ChromePlatform, error) {
	return configuration.UpdateChromePlatform(ctx, chromeplatform)
}

// GetChromePlatform returns chromeplatform for the given id from datastore.
func GetChromePlatform(ctx context.Context, id string) (*fleet.ChromePlatform, error) {
	return configuration.GetChromePlatform(ctx, id)
}

// ListChromePlatforms lists the chromeplatforms
func ListChromePlatforms(ctx context.Context, pageSize int32, pageToken string) ([]*fleet.ChromePlatform, string, error) {
	return configuration.ListChromePlatforms(ctx, pageSize, pageToken)
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
func ImportChromePlatforms(ctx context.Context, platforms []*fleet.ChromePlatform) (*datastore.OpResults, error) {
	return configuration.ImportChromePlatforms(ctx, platforms)
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
func ReplaceChromePlatform(ctx context.Context, oldChromePlatform *fleet.ChromePlatform, newChromePlatform *fleet.ChromePlatform) (*fleet.ChromePlatform, error) {
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
