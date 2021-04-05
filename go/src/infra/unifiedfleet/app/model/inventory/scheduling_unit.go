// Copyright 2021 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/models"
	ufsds "infra/unifiedfleet/app/model/datastore"
)

// SchedulingUnitKind is the datastore entity kind for SchedulingUnit.
const SchedulingUnitKind string = "SchedulingUnit"

// SchedulingUnitEntity is a datastore entity that tracks a SchedulingUnit.
type SchedulingUnitEntity struct {
	_kind       string   `gae:"$kind,SchedulingUnit"`
	ID          string   `gae:"$id"`
	Type        string   `gae:"type"`
	Pools       []string `gae:"pools"`
	MachineLSEs []string `gae:"machinelses"`
	Tags        []string `gae:"tags"`
	// ufspb.SchedulingUnit cannot be directly used as it contains pointer.
	SchedulingUnit []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled SchedulingUnit.
func (e *SchedulingUnitEntity) GetProto() (proto.Message, error) {
	var p ufspb.SchedulingUnit
	if err := proto.Unmarshal(e.SchedulingUnit, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newSchedulingUnitEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*ufspb.SchedulingUnit)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty SchedulingUnit ID").Err()
	}
	su, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal SchedulingUnit %s", p).Err()
	}
	return &SchedulingUnitEntity{
		ID:             p.GetName(),
		Type:           p.GetType().String(),
		Tags:           p.GetTags(),
		Pools:          p.GetPools(),
		MachineLSEs:    p.GetMachineLSEs(),
		SchedulingUnit: su,
	}, nil
}

// QuerySchedulingUnitByPropertyNames queries SchedulingUnit Entity in the datastore
// If keysOnly is true, then only key field is populated in returned SchedulingUnits
func QuerySchedulingUnitByPropertyNames(ctx context.Context, propertyMap map[string]string, keysOnly bool) ([]*ufspb.SchedulingUnit, error) {
	q := datastore.NewQuery(SchedulingUnitKind).KeysOnly(keysOnly).FirestoreMode(true)
	var entities []*SchedulingUnitEntity
	for propertyName, id := range propertyMap {
		q = q.Eq(propertyName, id)
	}
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if len(entities) == 0 {
		return nil, nil
	}
	schedulingUnits := make([]*ufspb.SchedulingUnit, 0, len(entities))
	for _, entity := range entities {
		if keysOnly {
			schedulingUnit := &ufspb.SchedulingUnit{
				Name: entity.ID,
			}
			schedulingUnits = append(schedulingUnits, schedulingUnit)
		} else {
			pm, perr := entity.GetProto()
			if perr != nil {
				logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
				continue
			}
			schedulingUnits = append(schedulingUnits, pm.(*ufspb.SchedulingUnit))
		}
	}
	return schedulingUnits, nil
}

// CreateSchedulingUnit creates a new SchedulingUnit in datastore.
func CreateSchedulingUnit(ctx context.Context, su *ufspb.SchedulingUnit) (*ufspb.SchedulingUnit, error) {
	return putSchedulingUnit(ctx, su, false)
}

// BatchUpdateSchedulingUnits updates SchedulingUnits in datastore.
//
// This is a non-atomic operation and doesnt check if the object already exists before
// update. Must be used within a transaction where objects are checked before update.
// Will lead to partial updates if not used in a transaction.
func BatchUpdateSchedulingUnits(ctx context.Context, schedulingUnits []*ufspb.SchedulingUnit) ([]*ufspb.SchedulingUnit, error) {
	protos := make([]proto.Message, len(schedulingUnits))
	updateTime := ptypes.TimestampNow()
	for i, su := range schedulingUnits {
		su.UpdateTime = updateTime
		protos[i] = su
	}
	_, err := ufsds.PutAll(ctx, protos, newSchedulingUnitEntity, true)
	if err == nil {
		return schedulingUnits, err
	}
	return nil, err
}

func putSchedulingUnit(ctx context.Context, su *ufspb.SchedulingUnit, update bool) (*ufspb.SchedulingUnit, error) {
	su.UpdateTime = ptypes.TimestampNow()
	pm, err := ufsds.Put(ctx, su, newSchedulingUnitEntity, update)
	if err == nil {
		return pm.(*ufspb.SchedulingUnit), err
	}
	return nil, err
}
