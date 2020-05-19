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
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
)

// RackKind is the datastore entity kind Rack.
const RackKind string = "Rack"

// RackEntity is a datastore entity that tracks Rack.
type RackEntity struct {
	_kind     string   `gae:"$kind,Rack"`
	ID        string   `gae:"$id"`
	SwitchIDs []string `gae:"switch_ids"`
	KVMIDs    []string `gae:"kvm_ids"`
	RPMIDs    []string `gae:"rpm_ids"`
	// fleet.Rack cannot be directly used as it contains pointer.
	Rack []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Rack.
func (e *RackEntity) GetProto() (proto.Message, error) {
	var p fleet.Rack
	if err := proto.Unmarshal(e.Rack, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newRackEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.Rack)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Rack ID").Err()
	}
	rack, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal Rack %s", p).Err()
	}
	return &RackEntity{
		ID:        p.GetName(),
		SwitchIDs: p.GetChromeBrowserRack().GetSwitches(),
		KVMIDs:    p.GetChromeBrowserRack().GetKvms(),
		RPMIDs:    p.GetChromeBrowserRack().GetRpms(),
		Rack:      rack,
	}, nil
}

// QueryRackByPropertyName queries Rack Entity in the datastore
//
// If keysOnly is true, then only key field is populated in returned racks
func QueryRackByPropertyName(ctx context.Context, propertyName, id string, keysOnly bool) ([]*fleet.Rack, error) {
	q := datastore.NewQuery(RackKind).KeysOnly(keysOnly)
	var entities []*RackEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No racks found for the query: %s", id)
		return nil, nil
	}
	racks := make([]*fleet.Rack, 0, len(entities))
	for _, entity := range entities {
		if keysOnly {
			rack := &fleet.Rack{
				Name: entity.ID,
			}
			racks = append(racks, rack)
		} else {
			pm, perr := entity.GetProto()
			if perr != nil {
				logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
				continue
			}
			racks = append(racks, pm.(*fleet.Rack))
		}
	}
	return racks, nil
}

// CreateRack creates a new rack in datastore.
func CreateRack(ctx context.Context, rack *fleet.Rack) (*fleet.Rack, error) {
	return putRack(ctx, rack, false)
}

// UpdateRack updates rack in datastore.
func UpdateRack(ctx context.Context, rack *fleet.Rack) (*fleet.Rack, error) {
	return putRack(ctx, rack, true)
}

// GetRack returns rack for the given id from datastore.
func GetRack(ctx context.Context, id string) (*fleet.Rack, error) {
	pm, err := fleetds.Get(ctx, &fleet.Rack{Name: id}, newRackEntity)
	if err == nil {
		return pm.(*fleet.Rack), err
	}
	return nil, err
}

// ListRacks lists the racks
// Does a query over Rack entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListRacks(ctx context.Context, pageSize int32, pageToken string) (res []*fleet.Rack, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, RackKind, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *RackEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*fleet.Rack))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List Racks %s", err)
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteRack deletes the rack in datastore
func DeleteRack(ctx context.Context, id string) error {
	return fleetds.Delete(ctx, &fleet.Rack{Name: id}, newRackEntity)
}

// BatchUpdateRacks updates racks in datastore.
//
// This is a non-atomic operation and doesnt check if the object already exists before
// update. Must be used within a Transaction where objects are checked before update.
// Will lead to partial updates if not used in a transaction.
func BatchUpdateRacks(ctx context.Context, racks []*fleet.Rack) ([]*fleet.Rack, error) {
	return putAllRack(ctx, racks, true)
}

func putRack(ctx context.Context, rack *fleet.Rack, update bool) (*fleet.Rack, error) {
	rack.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, rack, newRackEntity, update)
	if err == nil {
		return pm.(*fleet.Rack), err
	}
	return nil, err
}

// ImportRacks creates or updates a batch of racks in datastore.
func ImportRacks(ctx context.Context, racks []*fleet.Rack) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(racks))
	utime := ptypes.TimestampNow()
	for i, m := range racks {
		m.UpdateTime = utime
		protos[i] = m
	}
	return fleetds.Insert(ctx, protos, newRackEntity, true, true)
}

func putAllRack(ctx context.Context, racks []*fleet.Rack, update bool) ([]*fleet.Rack, error) {
	protos := make([]proto.Message, len(racks))
	updateTime := ptypes.TimestampNow()
	for i, rack := range racks {
		rack.UpdateTime = updateTime
		protos[i] = rack
	}
	_, err := fleetds.PutAll(ctx, protos, newRackEntity, update)
	if err == nil {
		return racks, err
	}
	return nil, err
}
