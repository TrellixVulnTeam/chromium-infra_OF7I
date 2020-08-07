// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

type stateUpdater struct {
	ResourceName string
	Changes      []*ufspb.ChangeEvent
}

// Delete a state record
//
// Can be used in a transaction
func (su *stateUpdater) deleteStateHelper(ctx context.Context) error {
	old, _ := state.GetStateRecord(ctx, su.ResourceName)
	state.DeleteStates(ctx, []string{su.ResourceName})
	su.Changes = append(su.Changes, LogStateChanges(old, nil)...)
	return nil
}

func (su *stateUpdater) updateStateHelper(ctx context.Context, newS ufspb.State) error {
	old, _ := state.GetStateRecord(ctx, su.ResourceName)
	state.DeleteStates(ctx, []string{su.ResourceName})
	newRecord := &ufspb.StateRecord{
		State:        newS,
		ResourceName: su.ResourceName,
		User:         util.CurrentUser(ctx),
	}
	if _, err := state.BatchUpdateStates(ctx, []*ufspb.StateRecord{newRecord}); err != nil {
		return err
	}
	fmt.Println("old ", old, "new: ", newRecord)
	su.Changes = append(su.Changes, LogStateChanges(old, newRecord)...)
	return nil
}

func (su *stateUpdater) addLseStateHelper(ctx context.Context, lse *ufspb.MachineLSE) error {
	stateRecords := make([]*ufspb.StateRecord, 0)
	for _, m := range lse.Machines {
		s := &ufspb.StateRecord{
			State:        ufspb.State_STATE_SERVING,
			ResourceName: util.AddPrefix(util.MachineCollection, m),
			User:         util.CurrentUser(ctx),
		}
		oldS, _ := state.GetStateRecord(ctx, s.GetResourceName())
		stateRecords = append(stateRecords, s)
		su.Changes = append(su.Changes, LogStateChanges(oldS, s)...)
	}
	for _, vm := range lse.GetChromeBrowserMachineLse().GetVms() {
		s := &ufspb.StateRecord{
			State:        ufspb.State_STATE_DEPLOYED_PRE_SERVING,
			ResourceName: util.AddPrefix(util.VMCollection, vm.GetName()),
			User:         util.CurrentUser(ctx),
		}
		oldS, _ := state.GetStateRecord(ctx, s.GetResourceName())
		stateRecords = append(stateRecords, s)
		su.Changes = append(su.Changes, LogStateChanges(oldS, s)...)
	}
	newS := &ufspb.StateRecord{
		State:        ufspb.State_STATE_DEPLOYED_PRE_SERVING,
		ResourceName: util.AddPrefix(util.HostCollection, lse.GetName()),
		User:         util.CurrentUser(ctx),
	}
	stateRecords = append(stateRecords, newS)
	su.Changes = append(su.Changes, LogStateChanges(nil, newS)...)
	if _, err := state.BatchUpdateStates(ctx, stateRecords); err != nil {
		return err
	}
	return nil
}

func (su *stateUpdater) deleteLseStateHelper(ctx context.Context, lse *ufspb.MachineLSE) error {
	stateRecords := make([]*ufspb.StateRecord, 0)
	su.ResourceName = util.AddPrefix(util.HostCollection, lse.GetName())
	// Update attached machines' state to registered
	for _, m := range lse.Machines {
		s := &ufspb.StateRecord{
			State:        ufspb.State_STATE_REGISTERED,
			ResourceName: util.AddPrefix(util.MachineCollection, m),
			User:         util.CurrentUser(ctx),
		}
		oldS, _ := state.GetStateRecord(ctx, s.GetResourceName())
		stateRecords = append(stateRecords, s)
		su.Changes = append(su.Changes, LogStateChanges(oldS, s)...)
	}
	if _, err := state.BatchUpdateStates(ctx, stateRecords); err != nil {
		return err
	}

	// Delete host & vm states
	toDeleteResources := make([]string, 0)
	for _, m := range lse.GetChromeBrowserMachineLse().GetVms() {
		r := util.AddPrefix(util.VMCollection, m.GetName())
		toDeleteResources = append(toDeleteResources, r)
		oldS, _ := state.GetStateRecord(ctx, r)
		su.Changes = append(su.Changes, LogStateChanges(oldS, nil)...)
	}
	toDeleteResources = append(toDeleteResources, su.ResourceName)
	oldS, _ := state.GetStateRecord(ctx, su.ResourceName)
	su.Changes = append(su.Changes, LogStateChanges(oldS, nil)...)
	state.DeleteStates(ctx, toDeleteResources)
	return nil
}
