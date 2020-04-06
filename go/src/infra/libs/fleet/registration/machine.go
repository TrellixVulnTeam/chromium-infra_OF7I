// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"

	fleetds "infra/libs/fleet/datastore"
	fleet "infra/libs/fleet/protos/go"
)

// MachineKind is the datastore entity kind Machine.
const MachineKind string = "Machine"

// MachineEntity is a datastore entity that tracks Machine.
type MachineEntity struct {
	_kind string `gae:"$kind,Machine"`
	ID    string `gae:"$id"`
	// fleet.Machine cannot be directly used as it contains pointer.
	Machine []byte `gae:",noindex"`
	// Should be in UTC timezone.
	Updated time.Time
	Parent  *datastore.Key `gae:"$parent"`
}

// GetProto returns the unmarshaled Machine.
func (e *MachineEntity) GetProto() (proto.Message, error) {
	var p fleet.Machine
	if err := proto.Unmarshal(e.Machine, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetUpdated returns the updated time of the entity.
func (e *MachineEntity) GetUpdated() time.Time {
	return e.Updated
}

func newEntity(ctx context.Context, pm proto.Message, updateTime time.Time) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.Machine)
	if p.GetId().GetValue() == "" {
		return nil, errors.Reason("Empty Machine ID").Err()
	}
	machine, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal Machine %s", p).Err()
	}
	parent := fleetds.FakeAncestorKey(ctx, MachineKind)
	return &MachineEntity{
		ID:      p.GetId().GetValue(),
		Machine: machine,
		Updated: updateTime,
		Parent:  parent,
	}, nil
}

func exists(ctx context.Context, entities []fleetds.FleetEntity) ([]bool, error) {
	cpEntities := make([]*MachineEntity, len(entities))
	for i, e := range entities {
		cpEntities[i] = e.(*MachineEntity)
	}
	res, err := datastore.Exists(ctx, cpEntities)
	if err != nil {
		return nil, err
	}
	return res.List(0), nil
}

func queryAll(ctx context.Context) ([]fleetds.FleetEntity, error) {
	var entities []*MachineEntity
	q := datastore.NewQuery(MachineKind).Ancestor(fleetds.FakeAncestorKey(ctx, MachineKind))
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]fleetds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// CreateMachines inserts machines to datastore.
func CreateMachines(ctx context.Context, machines []*fleet.Machine) (*fleetds.OpResults, error) {
	return putMachines(ctx, machines, false)
}

// UpdateMachines updates machines to datastore.
func UpdateMachines(ctx context.Context, machines []*fleet.Machine) (*fleetds.OpResults, error) {
	return putMachines(ctx, machines, true)
}

// GetAllMachines returns all machines in datastore.
func GetAllMachines(ctx context.Context) (*fleetds.OpResults, error) {
	return fleetds.GetAll(ctx, queryAll)
}

// GetMachinesByID returns machines for the given ids from datastore.
func GetMachinesByID(ctx context.Context, ids []string) *fleetds.OpResults {
	protos := make([]proto.Message, len(ids))
	for i, id := range ids {
		protos[i] = &fleet.Machine{
			Id: &fleet.MachineID{
				Value: id,
			},
		}
	}
	return fleetds.GetByID(ctx, protos, newEntity)
}

// DeleteMachines returns the deleted machines
func DeleteMachines(ctx context.Context, ids []string) *fleetds.OpResults {
	protos := make([]proto.Message, len(ids))
	for i, id := range ids {
		protos[i] = &fleet.Machine{
			Id: &fleet.MachineID{
				Value: id,
			},
		}
	}
	return fleetds.Delete(ctx, protos, newEntity, exists)
}

func putMachines(ctx context.Context, machines []*fleet.Machine, update bool) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(machines))
	for i, p := range machines {
		protos[i] = p
	}
	return fleetds.Insert(ctx, protos, newEntity, exists, update)
}
