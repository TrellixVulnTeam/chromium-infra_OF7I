// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/util"
)

// CreateVM creates a new vm in datastore.
func CreateVM(ctx context.Context, vm *ufspb.VM, host string, nwOpt *ufsAPI.NetworkOption) (*ufspb.VM, error) {
	f := func(ctx context.Context) error {
		var changes []*ufspb.ChangeEvent
		nu := &networkUpdater{
			Hostname: vm.GetName(),
		}
		su := &stateUpdater{
			ResourceName: util.AddPrefix(util.VMCollection, vm.GetName()),
		}

		lse, err := inventory.GetMachineLSE(ctx, host)
		if err != nil {
			return errors.Annotate(err, "Fail to get host by %s", host).Err()
		}
		vm.MachineLseId = host
		vm.Lab = lse.Lab
		vm.State = ufspb.State_STATE_DEPLOYED_PRE_SERVING.String()
		// Update states
		if err := su.updateStateHelper(ctx, ufspb.State_STATE_DEPLOYED_PRE_SERVING); err != nil {
			return errors.Annotate(err, "Fail to update state to vm %s", vm.GetName()).Err()
		}

		// Update ip configs
		if nwOpt.GetVlan() != "" || nwOpt.GetIp() != "" {
			if err := nu.addVMHostHelper(ctx, nwOpt, vm); err != nil {
				return errors.Annotate(err, "Fail to assign ip to vm %s", vm.GetName()).Err()
			}
		}

		if _, err := inventory.BatchUpdateVMs(ctx, []*ufspb.VM{vm}); err != nil {
			return errors.Annotate(err, "Failed to create vm %q", vm.GetName()).Err()
		}
		changes = append(changes, nu.Changes...)
		changes = append(changes, su.Changes...)
		changes = append(changes, LogVMChanges(nil, vm)...)

		return SaveChangeEvents(ctx, changes)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create vm in datastore: %s", err)
		return nil, err
	}
	return vm, nil
}

// UpdateVM updates an existing vm in datastore.
func UpdateVM(ctx context.Context, vm *ufspb.VM, host string, nwOpt *ufsAPI.NetworkOption, s ufspb.State) (*ufspb.VM, error) {
	f := func(ctx context.Context) error {
		var changes []*ufspb.ChangeEvent
		nu := &networkUpdater{
			Hostname: vm.GetName(),
		}
		su := &stateUpdater{
			ResourceName: util.AddPrefix(util.VMCollection, vm.GetName()),
		}

		oldVM, err := inventory.GetVM(ctx, vm.GetName())
		if err != nil {
			return errors.Annotate(err, "Fail to get existing vm by %s", vm.GetName()).Err()
		}
		lse, err := inventory.GetMachineLSE(ctx, host)
		if err != nil {
			return errors.Annotate(err, "Fail to get host by %s", host).Err()
		}
		// Before partial update is enabled, not overwrite VM if nwOpt or s is specified
		if nwOpt != nil || s != ufspb.State_STATE_UNSPECIFIED {
			vm = oldVM
		}
		vm.MachineLseId = host
		vm.Lab = lse.Lab
		vm.State = oldVM.State
		newState := ufspb.State_STATE_UNSPECIFIED
		if s != ufspb.State_STATE_UNSPECIFIED {
			newState = s
			vm.State = s.String()
		} else {
			if oldVM.State == "" {
				newState = s
				vm.State = s.String()
			}
		}
		if newState != ufspb.State_STATE_UNSPECIFIED {
			if err := su.updateStateHelper(ctx, newState); err != nil {
				return errors.Annotate(err, "Fail to update state to vm %s", vm.GetName()).Err()
			}
		}

		if nwOpt.GetDelete() {
			if err := nu.deleteDHCPHelper(ctx); err != nil {
				return err
			}
			vm.Vlan = ""
		} else if nwOpt.GetVlan() != "" || nwOpt.GetIp() != "" {
			if err := nu.addVMHostHelper(ctx, nwOpt, vm); err != nil {
				return errors.Annotate(err, "Fail to assign ip to host %s", host).Err()
			}
		}

		if _, err := inventory.BatchUpdateVMs(ctx, []*ufspb.VM{vm}); err != nil {
			return errors.Annotate(err, "Failed to create vm %q", vm.GetName()).Err()
		}
		changes = append(changes, nu.Changes...)
		changes = append(changes, su.Changes...)
		changes = append(changes, LogVMChanges(oldVM, vm)...)
		return SaveChangeEvents(ctx, changes)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create vm in datastore: %s", err)
		return nil, err
	}
	return vm, nil
}

// DeleteVM deletes a vm in datastore.
func DeleteVM(ctx context.Context, id string) error {
	f := func(ctx context.Context) error {
		var changes []*ufspb.ChangeEvent
		nu := &networkUpdater{
			Hostname: id,
		}
		su := &stateUpdater{
			ResourceName: util.AddPrefix(util.VMCollection, id),
		}

		if err := inventory.DeleteVM(ctx, id); err != nil {
			return errors.Annotate(err, "Unable to delete vm %s", id).Err()
		}
		changes = append(changes, LogVMChanges(&ufspb.VM{Name: id}, nil)...)

		if err := su.deleteStateHelper(ctx); err != nil {
			return err
		}
		changes = append(changes, su.Changes...)

		if err := nu.deleteDHCPHelper(ctx); err != nil {
			return err
		}
		changes = append(changes, nu.Changes...)

		return SaveChangeEvents(ctx, changes)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete vm in datastore: %s", err)
		return err
	}
	return nil
}

// ListVMs lists the vms
func ListVMs(ctx context.Context, pageSize int32, pageToken, filter string, keysOnly bool) ([]*ufspb.VM, string, error) {
	var filterMap map[string][]interface{}
	var err error
	if filter != "" {
		filterMap, err = getFilterMap(filter, inventory.GetVMIndexedFieldName)
		if err != nil {
			return nil, "", errors.Annotate(err, "Failed to read filter for listing vms").Err()
		}
	}
	if v, ok := filterMap["state"]; ok {
		s := util.ToUFSState(fmt.Sprintf("%s", v[0]))
		filterMap["state"] = []interface{}{s.String()}
	}
	return inventory.ListVMs(ctx, pageSize, pageSize, pageToken, filterMap, keysOnly, nil)
}

// GetVM returns vm for the given id from datastore.
func GetVM(ctx context.Context, id string) (*ufspb.VM, error) {
	return inventory.GetVM(ctx, id)
}
