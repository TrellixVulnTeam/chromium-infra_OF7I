// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"

	fleetds "infra/libs/fleet/datastore"
	fleet "infra/unifiedfleet/api/v1/proto"
)

// MachineKind is the datastore entity kind Machine.
const MachineKind string = "Machine"

// MachineEntity is a datastore entity that tracks Machine.
type MachineEntity struct {
	_kind string `gae:"$kind,Machine"`
	ID    string `gae:"$id"`
	// fleet.Machine cannot be directly used as it contains pointer.
	Machine []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Machine.
func (e *MachineEntity) GetProto() (proto.Message, error) {
	var p fleet.Machine
	if err := proto.Unmarshal(e.Machine, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.Machine)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Machine ID").Err()
	}
	machine, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal Machine %s", p).Err()
	}
	return &MachineEntity{
		ID:      p.GetName(),
		Machine: machine,
	}, nil
}

func queryAll(ctx context.Context) ([]fleetds.FleetEntity, error) {
	var entities []*MachineEntity
	q := datastore.NewQuery(MachineKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]fleetds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// CreateMachine creates a new machine in datastore.
func CreateMachine(ctx context.Context, machine *fleet.Machine) (*fleet.Machine, error) {
	return putMachine(ctx, machine, false)
}

// UpdateMachine updates machine in datastore.
func UpdateMachine(ctx context.Context, machine *fleet.Machine) (*fleet.Machine, error) {
	return putMachine(ctx, machine, true)
}

// GetMachine returns machine for the given id from datastore.
func GetMachine(ctx context.Context, id string) (*fleet.Machine, error) {
	pm, err := fleetds.Get(ctx, &fleet.Machine{Name: id}, newEntity)
	if err == nil {
		return pm.(*fleet.Machine), err
	}
	return nil, err
}

// GetAllMachines returns all machines in datastore.
func GetAllMachines(ctx context.Context) (*fleetds.OpResults, error) {
	return fleetds.GetAll(ctx, queryAll)
}

// DeleteMachine deletes the machine in datastore
func DeleteMachine(ctx context.Context, id string) error {
	return fleetds.Delete(ctx, &fleet.Machine{Name: id}, newEntity)
}

func putMachine(ctx context.Context, machine *fleet.Machine, update bool) (*fleet.Machine, error) {
	machine.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, machine, newEntity, update)
	if err == nil {
		return pm.(*fleet.Machine), err
	}
	return nil, err
}
