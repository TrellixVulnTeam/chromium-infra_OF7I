// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	ufsAPI "infra/unifiedfleet/api/v1/rpc"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/util"
)

// CreateVM creates a new vm in datastore.
func CreateVM(ctx context.Context, vm *ufspb.VM, nwOpt *ufsAPI.NetworkOption) (*ufspb.VM, error) {
	f := func(ctx context.Context) error {
		hc := getVMHistoryClient(vm)

		if err := validateCreateVM(ctx, vm, nwOpt); err != nil {
			return errors.Annotate(err, "Validation error - Failed to create MachineLSE").Err()
		}
		lse, err := inventory.GetMachineLSE(ctx, vm.GetMachineLseId())
		if err != nil {
			return errors.Annotate(err, "Fail to get host by %s", vm.GetMachineLseId()).Err()
		}
		vm.Zone = lse.Zone
		vm.ResourceState = ufspb.State_STATE_DEPLOYED_PRE_SERVING

		// Update states
		if err := hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_DEPLOYED_PRE_SERVING); err != nil {
			return errors.Annotate(err, "Fail to update state to vm %s", vm.GetName()).Err()
		}

		// Update ip configs
		if nwOpt.GetVlan() != "" || nwOpt.GetIp() != "" {
			if err := hc.netUdt.addVMHostHelper(ctx, nwOpt, vm); err != nil {
				return errors.Annotate(err, "Fail to assign ip to vm %s", vm.GetName()).Err()
			}
		}

		if _, err := inventory.BatchUpdateVMs(ctx, []*ufspb.VM{vm}); err != nil {
			return errors.Annotate(err, "Failed to create vm %q", vm.GetName()).Err()
		}
		hc.LogVMChanges(nil, vm)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create vm in datastore: %s", err)
		return nil, err
	}
	return vm, nil
}

// UpdateVM updates an existing vm in datastore.
func UpdateVM(ctx context.Context, vm *ufspb.VM, mask *field_mask.FieldMask) (*ufspb.VM, error) {
	f := func(ctx context.Context) error {
		hc := getVMHistoryClient(vm)

		// Validate input
		if err := validateUpdateVM(ctx, vm, mask); err != nil {
			return errors.Annotate(err, "UpdateVM - validation failed").Err()
		}

		// Get old/existing VM
		oldVM, err := inventory.GetVM(ctx, vm.GetName())
		if err != nil {
			return errors.Annotate(err, "Fail to get existing vm by %s", vm.GetName()).Err()
		}
		oldVMCopy := proto.Clone(oldVM).(*ufspb.VM)
		// Copy the vlan/zone to vm OUTPUT only fields from already existing vm
		vm.Zone = oldVM.GetZone()
		vm.Vlan = oldVM.GetVlan()

		// Partial update by field mask
		if mask != nil && len(mask.Paths) > 0 {
			vm, err = processVMUpdateMask(ctx, oldVM, vm, mask)
			if err != nil {
				return errors.Annotate(err, "UpdateVM - processing update mask failed").Err()
			}
		} else {
			// This is for complete object input
			if vm.MachineLseId == "" {
				return status.Error(codes.InvalidArgument, "UpdateVM - machineLseId cannot be empty/nil.")
			}
			// Check if user provided new host to associate the vm
			if vm.MachineLseId != oldVM.MachineLseId {
				lse, err := inventory.GetMachineLSE(ctx, vm.MachineLseId)
				if err != nil {
					return errors.Annotate(err, "UpdateVM - fail to get host by %s", vm.MachineLseId).Err()
				}
				// update the zone info for vm for vm table indexing
				vm.Zone = lse.Zone
			}
		}

		// update state
		if err := hc.stUdt.updateStateHelper(ctx, vm.GetResourceState()); err != nil {
			return errors.Annotate(err, "Fail to update state to vm %s", vm.GetName()).Err()
		}

		if _, err := inventory.BatchUpdateVMs(ctx, []*ufspb.VM{vm}); err != nil {
			return errors.Annotate(err, "Failed to create vm %q", vm.GetName()).Err()
		}
		hc.LogVMChanges(oldVMCopy, vm)
		return hc.SaveChangeEvents(ctx)
	}
	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to create vm in datastore: %s", err)
		return nil, err
	}
	return vm, nil
}

// UpdateVMHost updates the vm host(update ip assignment).
func UpdateVMHost(ctx context.Context, vmName string, nwOpt *ufsAPI.NetworkOption) (*ufspb.VM, error) {
	var vm *ufspb.VM
	var err error
	f := func(ctx context.Context) error {
		hc := getVMHistoryClient(&ufspb.VM{Name: vmName})
		// Validate the input
		if err := validateUpdateVMHost(ctx, vmName, nwOpt.GetVlan(), nwOpt.GetIp()); err != nil {
			return err
		}

		vm, err = GetVM(ctx, vmName)
		if err != nil {
			return err
		}
		// this is for logging changes
		oldVM := proto.Clone(vm).(*ufspb.VM)

		// Find free ip, set IP and DHCP config
		if err := hc.netUdt.addVMHostHelper(ctx, nwOpt, vm); err != nil {
			return errors.Annotate(err, "Fail to assign ip to vm %s", vm.Name).Err()
		}

		// update vm with new vlan info set in prev step by addVMHostHelper
		if _, err := inventory.BatchUpdateVMs(ctx, []*ufspb.VM{vm}); err != nil {
			return errors.Annotate(err, "Failed to update vm %q", vm.GetName()).Err()
		}
		hc.LogVMChanges(oldVM, vm)

		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to assign IP to the VM: %s", err)
		return nil, err
	}
	return vm, nil
}

// validateUpdateVMHost validates if an ip can be assigned to the VM
func validateUpdateVMHost(ctx context.Context, vmName string, vlanName, ipv4Str string) error {
	if ipv4Str != "" {
		return nil
	}
	// Check if resources does not exist
	return ResourceExist(ctx, []*Resource{GetVMResource(vmName), GetVlanResource(vlanName)}, nil)
}

// DeleteVMHost deletes the dhcp/ip of a vm in datastore.
func DeleteVMHost(ctx context.Context, vmName string) error {
	f := func(ctx context.Context) error {
		hc := getVMHistoryClient(&ufspb.VM{Name: vmName})
		if err := hc.netUdt.deleteDHCPHelper(ctx); err != nil {
			return err
		}

		oldVM, err := inventory.GetVM(ctx, vmName)
		if err != nil {
			return err
		}
		oldVMCopy := proto.Clone(oldVM).(*ufspb.VM)
		oldVM.Vlan = ""
		oldVM.ResourceState = ufspb.State_STATE_DEPLOYED_PRE_SERVING
		if _, err := inventory.BatchUpdateVMs(ctx, []*ufspb.VM{oldVM}); err != nil {
			return errors.Annotate(err, "Failed to update vm %q", vmName).Err()
		}
		hc.stUdt.updateStateHelper(ctx, ufspb.State_STATE_DEPLOYED_PRE_SERVING)
		hc.LogVMChanges(oldVMCopy, oldVM)
		return hc.SaveChangeEvents(ctx)
	}

	if err := datastore.RunInTransaction(ctx, f, nil); err != nil {
		logging.Errorf(ctx, "Failed to delete the vm dhcp: %s", err)
		return err
	}
	return nil
}

// processVMUpdateMask process update field mask to get only specific update
// fields and return a complete vm object with updated and existing fields
func processVMUpdateMask(ctx context.Context, oldVM *ufspb.VM, vm *ufspb.VM, mask *field_mask.FieldMask) (*ufspb.VM, error) {
	// update the fields in the existing vm
	for _, path := range mask.Paths {
		switch path {
		case "machineLseId":
			lse, err := inventory.GetMachineLSE(ctx, vm.GetMachineLseId())
			if err != nil {
				return oldVM, errors.Annotate(err, "fail to get host by %s", vm.GetMachineLseId()).Err()
			}
			oldVM.MachineLseId = vm.GetMachineLseId()
			oldVM.Zone = lse.GetZone()
		case "macAddress":
			oldVM.MacAddress = vm.GetMacAddress()
		case "resourceState":
			oldVM.ResourceState = vm.GetResourceState()
		case "osVersion":
			if oldVM.GetOsVersion() == nil {
				oldVM.OsVersion = &ufspb.OSVersion{
					Value: vm.GetOsVersion().GetValue(),
				}
			} else {
				oldVM.GetOsVersion().Value = vm.GetOsVersion().GetValue()
			}
		case "tags":
			oldVM.Tags = mergeTags(oldVM.GetTags(), vm.GetTags())
		case "description":
			oldVM.Description = vm.Description
		}
	}
	// return existing/old vm with new updated values
	return oldVM, nil
}

// DeleteVM deletes a vm in datastore.
func DeleteVM(ctx context.Context, id string) error {
	f := func(ctx context.Context) error {
		hc := getVMHistoryClient(&ufspb.VM{Name: id})

		if err := inventory.DeleteVM(ctx, id); err != nil {
			return errors.Annotate(err, "Unable to delete vm %s", id).Err()
		}
		hc.LogVMChanges(&ufspb.VM{Name: id}, nil)

		if err := hc.stUdt.deleteStateHelper(ctx); err != nil {
			return err
		}

		if err := hc.netUdt.deleteDHCPHelper(ctx); err != nil {
			return err
		}

		return hc.SaveChangeEvents(ctx)
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
	filterMap = resetStateFilter(filterMap)
	filterMap = resetOSFilter(filterMap)
	filterMap = resetZoneFilter(filterMap)
	return inventory.ListVMs(ctx, pageSize, pageSize, pageToken, filterMap, keysOnly, nil)
}

// GetVM returns vm for the given id from datastore.
func GetVM(ctx context.Context, id string) (*ufspb.VM, error) {
	return inventory.GetVM(ctx, id)
}

// BatchGetVMs returns a batch of vms from datastore.
func BatchGetVMs(ctx context.Context, ids []string) ([]*ufspb.VM, error) {
	return inventory.BatchGetVMs(ctx, ids)
}

func getVMHistoryClient(m *ufspb.VM) *HistoryClient {
	return &HistoryClient{
		stUdt: &stateUpdater{
			ResourceName: util.AddPrefix(util.VMCollection, m.Name),
		},
		netUdt: &networkUpdater{
			Hostname: m.Name,
		},
	}
}

// validateCreateVM validates if a vm can be created
func validateCreateVM(ctx context.Context, vm *ufspb.VM, nwOpt *ufsAPI.NetworkOption) error {
	// Aggregate resource to check if vm does not exist
	if err := resourceAlreadyExists(ctx, []*Resource{GetVMResource(vm.Name)}, nil); err != nil {
		return err
	}

	resourcesNotFound := make([]*Resource, 0)
	// Aggregate resource to check if machinelse does not exist
	if vm.GetMachineLseId() != "" {
		resourcesNotFound = append(resourcesNotFound, GetMachineLSEResource(vm.GetMachineLseId()))
	}
	if nwOpt.GetVlan() != "" || nwOpt.GetIp() != "" {
		if nwOpt.GetVlan() != "" {
			resourcesNotFound = append(resourcesNotFound, GetVlanResource(nwOpt.GetVlan()))
		}
	}
	// check if resources does not exist
	if err := ResourceExist(ctx, resourcesNotFound, nil); err != nil {
		return err
	}
	return nil
}

// validateUpdateVM validates if a vm can be updated
func validateUpdateVM(ctx context.Context, vm *ufspb.VM, mask *field_mask.FieldMask) error {
	// Aggregate resource to check if vm does not exist
	resourcesNotFound := []*Resource{GetVMResource(vm.Name)}
	// Aggregate resource to check if machinelse does not exist
	if vm.GetMachineLseId() != "" {
		resourcesNotFound = append(resourcesNotFound, GetMachineLSEResource(vm.GetMachineLseId()))
	}

	// check if resources does not exist
	if err := ResourceExist(ctx, resourcesNotFound, nil); err != nil {
		return err
	}

	return validateVMUpdateMask(vm, mask)
}

// validateVMUpdateMask validates the update mask for VM update
func validateVMUpdateMask(vm *ufspb.VM, mask *field_mask.FieldMask) error {
	if mask != nil {
		// validate the give field mask
		for _, path := range mask.Paths {
			switch path {
			case "name":
				return status.Error(codes.InvalidArgument, "validateVMUpdateMask - name cannot be updated, delete and create a new vm instead")
			case "update_time":
				return status.Error(codes.InvalidArgument, "validateVMUpdateMask - update_time cannot be updated, it is a output only field")
			case "machineLseId":
				if vm.MachineLseId == "" {
					return status.Error(codes.InvalidArgument, "validateVMUpdateMask - machineLseId cannot be empty/nil.")
				}
			case "macAddress":
			case "osVersion":
				if vm.GetOsVersion() == nil {
					return status.Error(codes.InvalidArgument, "validateUpdateVM - Osversion cannot be empty/nil.")
				}
			case "tags":
			case "description":
			case "resourceState":
				// valid fields, nothing to validate.
			default:
				return status.Errorf(codes.InvalidArgument, "validateUpdateVM - unsupported update mask path %q", path)
			}
		}
	}
	return nil
}
