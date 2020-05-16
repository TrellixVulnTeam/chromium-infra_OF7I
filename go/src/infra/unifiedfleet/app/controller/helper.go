// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package controller

import (
	"infra/unifiedfleet/app/model/configuration"
	fleetds "infra/unifiedfleet/app/model/datastore"
	"infra/unifiedfleet/app/model/registration"
)

var (
	// ErrorMessage - generalized error message for resources not found in the system
	ErrorMessage string = "There is no %s with %sID %s in the system.\n"
)

// Resource contains the fleet entity to be checked and the ID and Kind
type Resource struct {
	Kind   string
	ID     string
	Entity fleetds.FleetEntity
}

// GetChromePlatformResource returns a Resource with ChromePlatformEntity
func GetChromePlatformResource(chromePlatformID string) Resource {
	return Resource{
		Kind: configuration.ChromePlatformKind,
		ID:   chromePlatformID,
		Entity: &configuration.ChromePlatformEntity{
			ID: chromePlatformID,
		},
	}
}

// GetMachineLSEProtoTypeResource returns a Resource with MachineLSEProtoTypeEntity
func GetMachineLSEProtoTypeResource(machineLSEProtoTypeID string) Resource {
	return Resource{
		Kind: configuration.MachineLSEPrototypeKind,
		ID:   machineLSEProtoTypeID,
		Entity: &configuration.MachineLSEPrototypeEntity{
			ID: machineLSEProtoTypeID,
		},
	}
}

// GetRackLSEProtoTypeResource returns a Resource with RackLSEProtoTypeEntity
func GetRackLSEProtoTypeResource(rackLSEProtoTypeID string) Resource {
	return Resource{
		Kind: configuration.RackLSEPrototypeKind,
		ID:   rackLSEProtoTypeID,
		Entity: &configuration.RackLSEPrototypeEntity{
			ID: rackLSEProtoTypeID,
		},
	}
}

//GetMachineResource returns a Resource with MachineEntity
func GetMachineResource(machineID string) Resource {
	return Resource{
		Kind: registration.MachineKind,
		ID:   machineID,
		Entity: &registration.MachineEntity{
			ID: machineID,
		},
	}
}

//GetRackResource returns a Resource with RackEntity
func GetRackResource(rackID string) Resource {
	return Resource{
		Kind: registration.RackKind,
		ID:   rackID,
		Entity: &registration.RackEntity{
			ID: rackID,
		},
	}
}

//GetKVMResource returns a Resource with KVMEntity
func GetKVMResource(kvmID string) Resource {
	return Resource{
		Kind: registration.KVMKind,
		ID:   kvmID,
		Entity: &registration.KVMEntity{
			ID: kvmID,
		},
	}
}

// GetRPMResource returns a Resource with RPMEntity
func GetRPMResource(rpmID string) Resource {
	return Resource{
		Kind: registration.RPMKind,
		ID:   rpmID,
		Entity: &registration.RPMEntity{
			ID: rpmID,
		},
	}
}

// GetSwitchResource returns a Resource with SwitchEntity
func GetSwitchResource(switchID string) Resource {
	return Resource{
		Kind: registration.SwitchKind,
		ID:   switchID,
		Entity: &registration.SwitchEntity{
			ID: switchID,
		},
	}
}

// GetNicResource returns a Resource with NicEntity
func GetNicResource(nicID string) Resource {
	return Resource{
		Kind: registration.NicKind,
		ID:   nicID,
		Entity: &registration.NicEntity{
			ID: nicID,
		},
	}
}

// GetDracResource returns a Resource with DracEntity
func GetDracResource(dracID string) Resource {
	return Resource{
		Kind: registration.DracKind,
		ID:   dracID,
		Entity: &registration.DracEntity{
			ID: dracID,
		},
	}
}

// GetVlanResource returns a Resource with VlanEntity
func GetVlanResource(vlanID string) Resource {
	return Resource{
		Kind: registration.VlanKind,
		ID:   vlanID,
		Entity: &registration.VlanEntity{
			ID: vlanID,
		},
	}
}
