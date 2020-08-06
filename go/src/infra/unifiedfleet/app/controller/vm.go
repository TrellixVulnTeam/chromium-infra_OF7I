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
	"infra/unifiedfleet/app/model/state"
	"infra/unifiedfleet/app/util"
)

// CreateVM creates a new vm in datastore.
func CreateVM(ctx context.Context, vm *ufspb.VM, host string, nwOpt *ufsAPI.NetworkOption) (*ufspb.VM, error) {
	f := func(ctx context.Context) error {
		stateRecords := make([]*ufspb.StateRecord, 0)
		lse, err := inventory.GetMachineLSE(ctx, host)
		if err != nil {
			return errors.Annotate(err, "Fail to get host by %s", host).Err()
		}
		vm.MachineLseId = host
		vm.Lab = lse.Lab
		vm.State = ufspb.State_STATE_DEPLOYED_PRE_SERVING.String()
		stateRecords = append(stateRecords, &ufspb.StateRecord{
			State:        ufspb.State_STATE_DEPLOYED_PRE_SERVING,
			ResourceName: util.AddPrefix(util.VMCollection, vm.GetName()),
			User:         util.CurrentUser(ctx),
		})

		// Assign ip configs
		if nwOpt.GetVlan() != "" || nwOpt.GetIp() != "" {
			if err := addVMHostHelper(ctx, nwOpt, vm); err != nil {
				return errors.Annotate(err, "Fail to assign ip to host %s", host).Err()
			}
		}

		if _, err := inventory.BatchUpdateVMs(ctx, []*ufspb.VM{vm}); err != nil {
			return errors.Annotate(err, "Failed to create vm %q", vm.GetName()).Err()
		}

		if _, err := state.BatchUpdateStates(ctx, stateRecords); err != nil {
			return err
		}
		return nil
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create vm in datastore: %s", err)
		return nil, err
	}
	SaveChangeEvents(ctx, LogVMChanges(nil, vm))
	return vm, nil
}

// UpdateVM updates an existing vm in datastore.
func UpdateVM(ctx context.Context, vm *ufspb.VM, host string, nwOpt *ufsAPI.NetworkOption, s ufspb.State) (*ufspb.VM, error) {
	var oldVM *ufspb.VM
	var err error
	f := func(ctx context.Context) error {
		stateRecords := make([]*ufspb.StateRecord, 0)

		oldVM, err = inventory.GetVM(ctx, vm.GetName())
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
		if vm.State == "" || s != ufspb.State_STATE_UNSPECIFIED {
			if s == ufspb.State_STATE_UNSPECIFIED {
				s = ufspb.State_STATE_DEPLOYED_PRE_SERVING
			}
			vm.State = s.String()
			stateRecords = append(stateRecords, &ufspb.StateRecord{
				State:        s,
				ResourceName: util.AddPrefix(util.VMCollection, vm.GetName()),
				User:         util.CurrentUser(ctx),
			})
		}

		// Assign ip configs
		if nwOpt.GetDelete() {
			if err := deleteDHCPHelper(ctx, vm.GetName()); err != nil {
				return err
			}
			vm.Vlan = ""
		} else if nwOpt.GetVlan() != "" || nwOpt.GetIp() != "" {
			if err := addVMHostHelper(ctx, nwOpt, vm); err != nil {
				return errors.Annotate(err, "Fail to assign ip to host %s", host).Err()
			}
		}

		if _, err := inventory.BatchUpdateVMs(ctx, []*ufspb.VM{vm}); err != nil {
			return errors.Annotate(err, "Failed to create vm %q", vm.GetName()).Err()
		}

		if _, err := state.BatchUpdateStates(ctx, stateRecords); err != nil {
			return err
		}
		return nil
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create vm in datastore: %s", err)
		return nil, err
	}
	SaveChangeEvents(ctx, LogVMChanges(oldVM, vm))
	return vm, nil
}

// DeleteVM deletes a vm in datastore.
func DeleteVM(ctx context.Context, id string) error {
	f := func(ctx context.Context) error {
		if err := inventory.DeleteVM(ctx, id); err != nil {
			return errors.Annotate(err, "Unable to delete vm %s", id).Err()
		}
		state.DeleteStates(ctx, []string{util.AddPrefix(util.VMCollection, id)})
		if err := deleteDHCPHelper(ctx, id); err != nil {
			return err
		}
		return nil
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete vm in datastore: %s", err)
		return err
	}
	SaveChangeEvents(ctx, LogVMChanges(&ufspb.VM{Name: id}, nil))
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
