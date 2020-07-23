// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/registration"
)

// RackRegistration creates a new rack, switches, kvms and rpms in datastore.
func RackRegistration(ctx context.Context, rack *ufspb.Rack, switches []*ufspb.Switch, kvms []*ufspb.KVM, rpms []*ufspb.RPM) (*ufspb.Rack, []*ufspb.Switch, []*ufspb.KVM, []*ufspb.RPM, error) {
	f := func(ctx context.Context) error {
		// 1. Validate the input
		if err := validateRackRegistration(ctx, rack, switches, kvms, rpms); err != nil {
			return err
		}

		// 2. Make sure OUTPUT_ONLY fields are set to empty
		if rack.GetChromeBrowserRack() != nil {
			// These are output only field. User is not allowed to set these value.
			// Overwrite it with empty values.
			rack.GetChromeBrowserRack().Rpms = nil
			rack.GetChromeBrowserRack().Kvms = nil
			rack.GetChromeBrowserRack().Switches = nil
		}

		// 3. Create switches
		if switches != nil {
			var n []string = make([]string, 0, len(switches))
			for _, s := range switches {
				n = append(n, s.Name)
			}
			// This is output only field. Assign new value.
			rack.GetChromeBrowserRack().Switches = n

			// we use this func as it is a non-atomic operation and can be used to
			// run within a transaction to make it atomic. Datastore doesnt allow
			// nested transactions.
			if _, err := registration.BatchUpdateSwitches(ctx, switches); err != nil {
				return errors.Annotate(err, "Failed to create switches %s", n).Err()
			}
		}

		// 4. Create kvms
		if kvms != nil {
			var n []string = make([]string, 0, len(kvms))
			for _, kvm := range kvms {
				n = append(n, kvm.Name)
			}
			// This is output only field. Assign new value.
			rack.GetChromeBrowserRack().Kvms = n

			// we use this func as it is a non-atomic operation and can be used to
			// run within a transaction to make it atomic. Datastore doesnt allow
			// nested transactions.
			if _, err := registration.BatchUpdateKVMs(ctx, kvms); err != nil {
				return errors.Annotate(err, "Failed to create KVMs %s", n).Err()
			}
		}

		// 5. Create rpm
		if rpms != nil {
			var n []string = make([]string, 0, len(rpms))
			for _, rpm := range rpms {
				n = append(n, rpm.Name)
			}
			// This is output only field. Assign new value.
			rack.GetChromeBrowserRack().Rpms = n

			// we use this func as it is a non-atomic operation and can be used to
			// run within a transaction to make it atomic. Datastore doesnt allow
			// nested transactions.
			if _, err := registration.BatchUpdateRPMs(ctx, rpms); err != nil {
				return errors.Annotate(err, "Failed to create RPMs %s", n).Err()
			}
		}

		// 6. Create rack
		// we use this func as it is a non-atomic operation and can be used to
		// run within a transaction to make it atomic. Datastore doesnt allow
		// nested transactions.
		if _, err := registration.BatchUpdateRacks(ctx, []*ufspb.Rack{rack}); err != nil {
			return errors.Annotate(err, "Failed to create rack %s", rack.Name).Err()
		}
		return nil
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to register rack: %s", err)
		return nil, nil, nil, nil, err
	}
	changes := LogRackChanges(nil, rack)
	for _, sw := range switches {
		changes = append(changes, LogSwitchChanges(nil, sw)...)
	}
	for _, kvm := range kvms {
		changes = append(changes, LogKVMChanges(nil, kvm)...)
	}
	for _, rpm := range rpms {
		changes = append(changes, LogRPMChanges(nil, rpm)...)
	}
	SaveChangeEvents(ctx, changes)
	return rack, switches, kvms, rpms, nil
}

// validateRackRegistration validates if a rack, switches, kvms and rpms can be created in the datastore.
//
// checks if the resources rack/switches/kvms/rpms already exists in the system.
// checks if resources referenced by rack/switches/kvms/rpms does not exist in the system.
func validateRackRegistration(ctx context.Context, rack *ufspb.Rack, switches []*ufspb.Switch, kvms []*ufspb.KVM, rpms []*ufspb.RPM) error {
	if rack == nil {
		return errors.New("rack cannot be empty")
	}
	if err := validateCreateRack(ctx, rack); err != nil {
		return err
	}

	var resourcesAlreadyExists []*Resource
	var resourcesNotFound []*Resource
	var errorMsg strings.Builder
	errorMsg.WriteString(fmt.Sprintf("Cannot create rack %s:\n", rack.Name))
	// Aggregate resources to check if rack already exists
	resourcesAlreadyExists = append(resourcesAlreadyExists, GetRackResource(rack.Name))

	if switches != nil {
		for _, s := range switches {
			// Aggregate resources to check if switch already exists
			resourcesAlreadyExists = append(resourcesAlreadyExists, GetSwitchResource(s.Name))
		}
	}

	if kvms != nil {
		for _, kvm := range kvms {
			// Aggregate resources to check if kvm already exists
			resourcesAlreadyExists = append(resourcesAlreadyExists, GetKVMResource(kvm.Name))

			// Aggregate resource to check if resources referenced by the kvm does not exist
			if chromePlatformID := kvm.GetChromePlatform(); chromePlatformID != "" {
				resourcesNotFound = append(resourcesNotFound, GetChromePlatformResource(chromePlatformID))
			}
		}
	}

	if rpms != nil {
		for _, rpm := range rpms {
			// Aggregate resources to check if rpm already exists
			resourcesAlreadyExists = append(resourcesAlreadyExists, GetRPMResource(rpm.Name))
		}
	}

	// Check if rack/switches/kvms/rpms already exists
	if err := resourceAlreadyExists(ctx, resourcesAlreadyExists, &errorMsg); err != nil {
		return err
	}

	// Check if resources referenced by rack/switches/kvms/rpms does not exist
	return ResourceExist(ctx, resourcesNotFound, &errorMsg)
}
