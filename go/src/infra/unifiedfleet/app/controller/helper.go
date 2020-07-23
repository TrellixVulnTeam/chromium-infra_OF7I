// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chromeosLab "infra/unifiedfleet/api/v1/proto/chromeos/lab"
	"infra/unifiedfleet/app/model/configuration"
	ufsds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/inventory"
	"infra/unifiedfleet/app/model/registration"
	"infra/unifiedfleet/app/util"
)

//Generalized error messages for resources in the system
var (
	NotFoundErrorMessage      string = "There is no %s with %sID %s in the system.\n"
	AlreadyExistsErrorMessage string = "%s %s already exists in the system.\n"
)

// Resource contains the fleet entity to be checked and the ID and Kind
type Resource struct {
	Kind   string
	ID     string
	Entity ufsds.FleetEntity
}

// GetChromePlatformResource returns a Resource with ChromePlatformEntity
func GetChromePlatformResource(chromePlatformID string) *Resource {
	return &Resource{
		Kind: configuration.ChromePlatformKind,
		ID:   chromePlatformID,
		Entity: &configuration.ChromePlatformEntity{
			ID: chromePlatformID,
		},
	}
}

// GetMachineLSEProtoTypeResource returns a Resource with MachineLSEProtoTypeEntity
func GetMachineLSEProtoTypeResource(machineLSEProtoTypeID string) *Resource {
	return &Resource{
		Kind: configuration.MachineLSEPrototypeKind,
		ID:   machineLSEProtoTypeID,
		Entity: &configuration.MachineLSEPrototypeEntity{
			ID: machineLSEProtoTypeID,
		},
	}
}

// GetRackLSEProtoTypeResource returns a Resource with RackLSEProtoTypeEntity
func GetRackLSEProtoTypeResource(rackLSEProtoTypeID string) *Resource {
	return &Resource{
		Kind: configuration.RackLSEPrototypeKind,
		ID:   rackLSEProtoTypeID,
		Entity: &configuration.RackLSEPrototypeEntity{
			ID: rackLSEProtoTypeID,
		},
	}
}

//GetMachineResource returns a Resource with MachineEntity
func GetMachineResource(machineID string) *Resource {
	return &Resource{
		Kind: registration.MachineKind,
		ID:   machineID,
		Entity: &registration.MachineEntity{
			ID: machineID,
		},
	}
}

//GetMachineLSEResource returns a Resource with MachineLSEEntity
func GetMachineLSEResource(machinelseID string) *Resource {
	return &Resource{
		Kind: inventory.MachineLSEKind,
		ID:   machinelseID,
		Entity: &inventory.MachineLSEEntity{
			ID: machinelseID,
		},
	}
}

//GetRackResource returns a Resource with RackEntity
func GetRackResource(rackID string) *Resource {
	return &Resource{
		Kind: registration.RackKind,
		ID:   rackID,
		Entity: &registration.RackEntity{
			ID: rackID,
		},
	}
}

//GetKVMResource returns a Resource with KVMEntity
func GetKVMResource(kvmID string) *Resource {
	return &Resource{
		Kind: registration.KVMKind,
		ID:   kvmID,
		Entity: &registration.KVMEntity{
			ID: kvmID,
		},
	}
}

// GetRPMResource returns a Resource with RPMEntity
func GetRPMResource(rpmID string) *Resource {
	return &Resource{
		Kind: registration.RPMKind,
		ID:   rpmID,
		Entity: &registration.RPMEntity{
			ID: rpmID,
		},
	}
}

// GetSwitchResource returns a Resource with SwitchEntity
func GetSwitchResource(switchID string) *Resource {
	return &Resource{
		Kind: registration.SwitchKind,
		ID:   switchID,
		Entity: &registration.SwitchEntity{
			ID: switchID,
		},
	}
}

// GetNicResource returns a Resource with NicEntity
func GetNicResource(nicID string) *Resource {
	return &Resource{
		Kind: registration.NicKind,
		ID:   nicID,
		Entity: &registration.NicEntity{
			ID: nicID,
		},
	}
}

// GetDracResource returns a Resource with DracEntity
func GetDracResource(dracID string) *Resource {
	return &Resource{
		Kind: registration.DracKind,
		ID:   dracID,
		Entity: &registration.DracEntity{
			ID: dracID,
		},
	}
}

// GetVlanResource returns a Resource with VlanEntity
func GetVlanResource(vlanID string) *Resource {
	return &Resource{
		Kind: configuration.VlanKind,
		ID:   vlanID,
		Entity: &configuration.VlanEntity{
			ID: vlanID,
		},
	}
}

// ResourceExist checks if the given resources exists in the datastore
//
// Returns error if any one resource does not exist in the system.
// Appends error messages to the the given error message for resources
// that does not exist in the datastore.
func ResourceExist(ctx context.Context, resources []*Resource, errorMsg *strings.Builder) error {
	if len(resources) == 0 {
		return nil
	}
	if errorMsg == nil {
		errorMsg = &strings.Builder{}
	}
	var NotFound bool = false
	checkEntities := make([]ufsds.FleetEntity, 0, len(resources))
	for _, resource := range resources {
		logging.Debugf(ctx, "checking resource existence: %#v", resource)
		checkEntities = append(checkEntities, resource.Entity)
	}
	exists, err := ufsds.Exists(ctx, checkEntities)
	if err == nil {
		for i := range checkEntities {
			if !exists[i] {
				NotFound = true
				errorMsg.WriteString(fmt.Sprintf(NotFoundErrorMessage, resources[i].Kind, resources[i].Kind, resources[i].ID))
			}
		}
	} else {
		logging.Errorf(ctx, "Failed to check existence: %s", err)
		return status.Errorf(codes.Internal, err.Error())
	}
	if NotFound {
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

// resourceAlreadyExists checks if the given resources already exists in the datastore
//
// Returns error if any of the resource already exists.
// Appends error messages to the the given error message for resources
// that already exist in the datastore.
func resourceAlreadyExists(ctx context.Context, resources []*Resource, errorMsg *strings.Builder) error {
	if len(resources) == 0 {
		return nil
	}
	if errorMsg == nil {
		errorMsg = &strings.Builder{}
	}
	var alreadyExists bool = false
	checkEntities := make([]ufsds.FleetEntity, 0, len(resources))
	for _, resource := range resources {
		logging.Debugf(ctx, "checking resource existence: %#v", resource)
		checkEntities = append(checkEntities, resource.Entity)
	}
	exists, err := ufsds.Exists(ctx, checkEntities)
	if err == nil {
		for i := range checkEntities {
			if exists[i] {
				alreadyExists = true
				errorMsg.WriteString(fmt.Sprintf(AlreadyExistsErrorMessage, resources[i].Kind, resources[i].ID))
			}
		}
	} else {
		logging.Errorf(ctx, "Failed to check existence: %s", err)
		return status.Errorf(codes.Internal, err.Error())
	}
	if alreadyExists {
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return nil
}

// testServoEq checks if the 2 slice of servos are equal
func testServoEq(a, b []*chromeosLab.Servo) bool {
	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !proto.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func deleteByPage(ctx context.Context, toDelete []string, pageSize int, deletFunc func(ctx context.Context, resourceNames []string) *ufsds.OpResults) *ufsds.OpResults {
	var allRes ufsds.OpResults
	for i := 0; ; i += pageSize {
		end := util.Min(i+pageSize, len(toDelete))
		res := deletFunc(ctx, toDelete[i:end])
		allRes = append(allRes, *res...)
		if i+pageSize >= len(toDelete) {
			break
		}
	}
	return &allRes
}
