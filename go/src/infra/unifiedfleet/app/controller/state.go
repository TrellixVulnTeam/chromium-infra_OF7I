// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"go.chromium.org/luci/common/logging"
	crimson "go.chromium.org/luci/machine-db/api/crimson/v1"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

// ImportStates imports states of UFS resources.
func ImportStates(ctx context.Context, machines []*crimson.Machine, vms []*crimson.VM, pageSize int) (*datastore.OpResults, error) {
	states := make([]*ufspb.StateRecord, 0)
	logging.Debugf(ctx, "collecting states of machines")
	for _, m := range machines {
		states = append(states, &ufspb.StateRecord{
			ResourceName: util.AddPrefix(util.MachineCollection, m.GetName()),
			State:        util.ToState(m.GetState()),
			User:         util.DefaultImporter,
		})
	}
	logging.Debugf(ctx, "collecting states of vms")
	for _, vm := range vms {
		states = append(states, &ufspb.StateRecord{
			ResourceName: util.AddPrefix(util.VMCollection, vm.GetName()),
			State:        util.ToState(vm.GetState()),
			User:         util.DefaultImporter,
		})
	}

	allRes := make(datastore.OpResults, 0)
	logging.Debugf(ctx, "Importing %d states", len(states))
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(states))
		logging.Debugf(ctx, "importing states %dth - %dth", i, end-1)
		res, err := state.ImportStateRecords(ctx, states[i:end])
		allRes = append(allRes, *res...)
		if err != nil {
			return &allRes, err
		}
		if i+pageSize >= len(states) {
			break
		}
	}
	return &allRes, nil
}

// UpdateState updates state record for a resource.
func UpdateState(ctx context.Context, stateRecord *ufspb.StateRecord) (*ufspb.StateRecord, error) {
	return state.UpdateStateRecord(ctx, stateRecord)
}

// GetState returns state record for a resource.
func GetState(ctx context.Context, resourceName string) (*ufspb.StateRecord, error) {
	return state.GetStateRecord(ctx, resourceName)
}
