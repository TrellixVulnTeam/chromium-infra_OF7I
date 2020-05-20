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

// DracKind is the datastore entity kind Drac.
const DracKind string = "Drac"

// DracEntity is a datastore entity that tdracs Drac.
type DracEntity struct {
	_kind    string `gae:"$kind,Drac"`
	ID       string `gae:"$id"`
	SwitchID string `gae:"switch_id"`
	// fleet.Drac cannot be directly used as it contains pointer.
	Drac []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Drac.
func (e *DracEntity) GetProto() (proto.Message, error) {
	var p fleet.Drac
	if err := proto.Unmarshal(e.Drac, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newDracEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.Drac)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Drac ID").Err()
	}
	drac, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal Drac %s", p).Err()
	}
	return &DracEntity{
		ID:       p.GetName(),
		SwitchID: p.GetSwitchInterface().GetSwitch(),
		Drac:     drac,
	}, nil
}

// QueryDracByPropertyName query's Drac Entity in the datastore
//
// If keysOnly is true, then only key field is populated in returned dracs
func QueryDracByPropertyName(ctx context.Context, propertyName, id string, keysOnly bool) ([]*fleet.Drac, error) {
	q := datastore.NewQuery(DracKind).KeysOnly(keysOnly)
	var entities []*DracEntity
	if err := datastore.GetAll(ctx, q.Eq(propertyName, id), &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No dracs found for the query: %s", id)
		return nil, nil
	}
	dracs := make([]*fleet.Drac, 0, len(entities))
	for _, entity := range entities {
		if keysOnly {
			drac := &fleet.Drac{
				Name: entity.ID,
			}
			dracs = append(dracs, drac)
		} else {
			pm, perr := entity.GetProto()
			if perr != nil {
				logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
				continue
			}
			dracs = append(dracs, pm.(*fleet.Drac))
		}
	}
	return dracs, nil
}

// CreateDrac creates a new drac in datastore.
func CreateDrac(ctx context.Context, drac *fleet.Drac) (*fleet.Drac, error) {
	return putDrac(ctx, drac, false)
}

// UpdateDrac updates drac in datastore.
func UpdateDrac(ctx context.Context, drac *fleet.Drac) (*fleet.Drac, error) {
	return putDrac(ctx, drac, true)
}

// GetDrac returns drac for the given id from datastore.
func GetDrac(ctx context.Context, id string) (*fleet.Drac, error) {
	pm, err := fleetds.Get(ctx, &fleet.Drac{Name: id}, newDracEntity)
	if err == nil {
		return pm.(*fleet.Drac), err
	}
	return nil, err
}

// ListDracs lists the dracs
//
// Does a query over Drac entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListDracs(ctx context.Context, pageSize int32, pageToken string) (res []*fleet.Drac, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, DracKind, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *DracEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*fleet.Drac))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List Dracs %s", err)
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteDrac deletes the drac in datastore
func DeleteDrac(ctx context.Context, id string) error {
	return fleetds.Delete(ctx, &fleet.Drac{Name: id}, newDracEntity)
}

func putDrac(ctx context.Context, drac *fleet.Drac, update bool) (*fleet.Drac, error) {
	drac.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, drac, newDracEntity, update)
	if err == nil {
		return pm.(*fleet.Drac), err
	}
	return nil, err
}

// BatchUpdateDracs updates dracs in datastore.
//
// This is a non-atomic operation and doesnt check if the object already exists before
// update. Must be used within a Transaction where objects are checked before update.
// Will lead to partial updates if not used in a transaction.
func BatchUpdateDracs(ctx context.Context, dracs []*fleet.Drac) ([]*fleet.Drac, error) {
	return putAllDrac(ctx, dracs, true)
}

func putAllDrac(ctx context.Context, dracs []*fleet.Drac, update bool) ([]*fleet.Drac, error) {
	protos := make([]proto.Message, len(dracs))
	updateTime := ptypes.TimestampNow()
	for i, drac := range dracs {
		drac.UpdateTime = updateTime
		protos[i] = drac
	}
	_, err := fleetds.PutAll(ctx, protos, newDracEntity, update)
	if err == nil {
		return dracs, err
	}
	return nil, err
}

// ImportDracs creates or updates a batch of dracs in datastore.
func ImportDracs(ctx context.Context, dracs []*fleet.Drac) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(dracs))
	utime := ptypes.TimestampNow()
	for i, m := range dracs {
		m.UpdateTime = utime
		protos[i] = m
	}
	return fleetds.Insert(ctx, protos, newDracEntity, true, true)
}
