// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	ufspb "infra/unifiedfleet/api/v1/proto"
	"infra/unifiedfleet/app/model/history"
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

type stateUpdater struct {
	ResourceName string
	Changes      []*ufspb.ChangeEvent
	Msgs         []*history.SnapshotMsgEntity
}

func (su *stateUpdater) logChanges(changes []*ufspb.ChangeEvent, msg *history.SnapshotMsgEntity) {
	su.Changes = append(su.Changes, changes...)
	if msg != nil {
		su.Msgs = append(su.Msgs, msg)
	}
}

// Delete a state record
//
// Can be used in a transaction
func (su *stateUpdater) deleteStateHelper(ctx context.Context) error {
	old, _ := state.GetStateRecord(ctx, su.ResourceName)
	state.DeleteStates(ctx, []string{su.ResourceName})
	su.logChanges(LogStateChanges(old, nil))
	return nil
}

func (su *stateUpdater) updateStateHelper(ctx context.Context, newS ufspb.State) error {
	old, _ := state.GetStateRecord(ctx, su.ResourceName)
	if old.GetState() == newS {
		return nil
	}
	state.DeleteStates(ctx, []string{su.ResourceName})
	newRecord := &ufspb.StateRecord{
		State:        newS,
		ResourceName: su.ResourceName,
		User:         util.CurrentUser(ctx),
	}
	if _, err := state.BatchUpdateStates(ctx, []*ufspb.StateRecord{newRecord}); err != nil {
		return err
	}
	su.logChanges(LogStateChanges(old, newRecord))
	return nil
}

func (su *stateUpdater) replaceStateHelper(ctx context.Context, oldR string) error {
	old, _ := state.GetStateRecord(ctx, oldR)
	state.DeleteStates(ctx, []string{oldR})
	newRecord := &ufspb.StateRecord{
		State:        old.GetState(),
		ResourceName: su.ResourceName,
		User:         util.CurrentUser(ctx),
	}
	if _, err := state.BatchUpdateStates(ctx, []*ufspb.StateRecord{newRecord}); err != nil {
		return err
	}
	su.logChanges(LogStateChanges(old, nil))
	su.logChanges(LogStateChanges(nil, newRecord))
	return nil
}

func (su *stateUpdater) addLseStateHelper(ctx context.Context, lse *ufspb.MachineLSE, machine *ufspb.Machine) error {
	stateRecords := make([]*ufspb.StateRecord, 0)
	rn := util.AddPrefix(util.MachineCollection, machine.GetName())
	s := &ufspb.StateRecord{
		State:        machine.GetResourceState(),
		ResourceName: rn,
		User:         util.CurrentUser(ctx),
	}
	oldS, _ := state.GetStateRecord(ctx, rn)
	stateRecords = append(stateRecords, s)
	su.logChanges(LogStateChanges(oldS, s))
	for _, vm := range lse.GetChromeBrowserMachineLse().GetVms() {
		rn := util.AddPrefix(util.VMCollection, vm.GetName())
		s := &ufspb.StateRecord{
			State:        vm.GetResourceState(),
			ResourceName: rn,
			User:         util.CurrentUser(ctx),
		}
		oldS, _ := state.GetStateRecord(ctx, rn)
		stateRecords = append(stateRecords, s)
		su.logChanges(LogStateChanges(oldS, s))
	}
	newS := &ufspb.StateRecord{
		State:        lse.GetResourceState(),
		ResourceName: util.AddPrefix(util.HostCollection, lse.GetName()),
		User:         util.CurrentUser(ctx),
	}
	stateRecords = append(stateRecords, newS)
	su.logChanges(LogStateChanges(nil, newS))
	if _, err := state.BatchUpdateStates(ctx, stateRecords); err != nil {
		return err
	}
	return nil
}

func (su *stateUpdater) deleteLseStateHelper(ctx context.Context, lse *ufspb.MachineLSE, machine *ufspb.Machine) error {
	stateRecords := make([]*ufspb.StateRecord, 0)
	su.ResourceName = util.AddPrefix(util.HostCollection, lse.GetName())
	// Update attached machines' state to registered
	if machine != nil {
		rn := util.AddPrefix(util.MachineCollection, machine.GetName())
		s := &ufspb.StateRecord{
			State:        machine.GetResourceState(),
			ResourceName: rn,
			User:         util.CurrentUser(ctx),
		}
		oldS, _ := state.GetStateRecord(ctx, rn)
		stateRecords = append(stateRecords, s)
		su.logChanges(LogStateChanges(oldS, s))
		if _, err := state.BatchUpdateStates(ctx, stateRecords); err != nil {
			return err
		}
	}

	// Delete host & vm states
	toDeleteResources := make([]string, 0)
	for _, m := range lse.GetChromeBrowserMachineLse().GetVms() {
		r := util.AddPrefix(util.VMCollection, m.GetName())
		toDeleteResources = append(toDeleteResources, r)
		oldS, _ := state.GetStateRecord(ctx, r)
		su.logChanges(LogStateChanges(oldS, nil))
	}
	toDeleteResources = append(toDeleteResources, su.ResourceName)
	oldS, _ := state.GetStateRecord(ctx, su.ResourceName)
	su.logChanges(LogStateChanges(oldS, nil))
	state.DeleteStates(ctx, toDeleteResources)
	return nil
}

func (su *stateUpdater) addRackStateHelper(ctx context.Context, rack *ufspb.Rack) error {
	stateRecords := make([]*ufspb.StateRecord, 0)
	for _, m := range rack.GetChromeBrowserRack().GetSwitchObjects() {
		s := &ufspb.StateRecord{
			State:        m.GetResourceState(),
			ResourceName: util.AddPrefix(util.SwitchCollection, m.Name),
			User:         util.CurrentUser(ctx),
		}
		stateRecords = append(stateRecords, s)
		su.logChanges(LogStateChanges(nil, s))
	}
	for _, m := range rack.GetChromeBrowserRack().GetKvmObjects() {
		s := &ufspb.StateRecord{
			State:        m.GetResourceState(),
			ResourceName: util.AddPrefix(util.KVMCollection, m.GetName()),
			User:         util.CurrentUser(ctx),
		}
		stateRecords = append(stateRecords, s)
		su.logChanges(LogStateChanges(nil, s))
	}
	for _, m := range rack.GetChromeBrowserRack().GetRpmObjects() {
		s := &ufspb.StateRecord{
			State:        m.GetResourceState(),
			ResourceName: util.AddPrefix(util.RPMCollection, m.GetName()),
			User:         util.CurrentUser(ctx),
		}
		stateRecords = append(stateRecords, s)
		su.logChanges(LogStateChanges(nil, s))
	}
	newS := &ufspb.StateRecord{
		State:        rack.GetResourceState(),
		ResourceName: util.AddPrefix(util.RackCollection, rack.GetName()),
		User:         util.CurrentUser(ctx),
	}
	stateRecords = append(stateRecords, newS)
	su.logChanges(LogStateChanges(nil, newS))
	if _, err := state.BatchUpdateStates(ctx, stateRecords); err != nil {
		return err
	}
	return nil
}
