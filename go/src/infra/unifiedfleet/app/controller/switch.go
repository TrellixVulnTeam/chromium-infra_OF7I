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
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
)

// CreateSwitch creates a new switch in datastore.
func CreateSwitch(ctx context.Context, s *fleet.Switch) (*fleet.Switch, error) {
	return registration.CreateSwitch(ctx, s)
}

// UpdateSwitch updates switch in datastore.
func UpdateSwitch(ctx context.Context, s *fleet.Switch) (*fleet.Switch, error) {
	return registration.UpdateSwitch(ctx, s)
}

// GetSwitch returns switch for the given id from datastore.
func GetSwitch(ctx context.Context, id string) (*fleet.Switch, error) {
	return registration.GetSwitch(ctx, id)
}

// ListSwitches lists the switches
func ListSwitches(ctx context.Context, pageSize int32, pageToken string) ([]*fleet.Switch, string, error) {
	return registration.ListSwitches(ctx, pageSize, pageToken)
}

// DeleteSwitch deletes the switch in datastore
//
// For referential data intergrity,
// Delete if this Switch is not referenced by other resources in the datastore.
// If there are any references, delete will be rejected and an error will be returned.
func DeleteSwitch(ctx context.Context, id string) error {
	err := validateDeleteSwitch(ctx, id)
	if err != nil {
		return err
	}
	return registration.DeleteSwitch(ctx, id)
}

// ReplaceSwitch replaces an old Switch with new Switch in datastore
//
// It does a delete of old switch and create of new Switch.
// All the steps are in done in a transaction to preserve consistency on failure.
// Before deleting the old Switch, it will get all the resources referencing
// the old Switch. It will update all the resources which were referencing
// the old Switch(got in the last step) with new Switch.
// Deletes the old Switch.
// Creates the new Switch.
// This will preserve data integrity in the system.
func ReplaceSwitch(ctx context.Context, oldSwitch *fleet.Switch, newSwitch *fleet.Switch) (*fleet.Switch, error) {
	// TODO(eshwarn) : implement replace after user testing the tool
	return nil, nil
}

// validateDeleteSwitch validates if a Switch can be deleted
//
// Checks if this Switch(SwitchID) is not referenced by other resources in the datastore.
// If there are any other references, delete will be rejected and an error will be returned.
func validateDeleteSwitch(ctx context.Context, id string) error {
	machines, err := registration.QueryMachineByPropertyName(ctx, "switch_id", id, true)
	if err != nil {
		return err
	}
	racks, err := registration.QueryRackByPropertyName(ctx, "switch_ids", id, true)
	if err != nil {
		return err
	}
	dracs, err := registration.QueryDracByPropertyName(ctx, "switch_id", id, true)
	if err != nil {
		return err
	}
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "switch_id", id, true)
	if err != nil {
		return err
	}
	if len(machines) > 0 || len(racks) > 0 || len(dracs) > 0 || len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Switch %s cannot be deleted because there are other resources which are referring this Switch.", id))
		if len(machines) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachines referring the Switch:\n"))
			for _, machine := range machines {
				errorMsg.WriteString(machine.Name + ", ")
			}
		}
		if len(racks) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nRacks referring the Switch:\n"))
			for _, rack := range racks {
				errorMsg.WriteString(rack.Name + ", ")
			}
		}
		if len(dracs) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nDracs referring the Switch:\n"))
			for _, drac := range dracs {
				errorMsg.WriteString(drac.Name + ", ")
			}
		}
		if len(machinelses) > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nMachineLSEs referring the Switch:\n"))
			for _, machinelse := range machinelses {
				errorMsg.WriteString(machinelse.Name + ", ")
			}
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}
