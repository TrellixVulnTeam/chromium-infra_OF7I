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

// NicKind is the datastore entity kind Nic.
const NicKind string = "Nic"

// NicEntity is a datastore entity that tnics Nic.
type NicEntity struct {
	_kind string `gae:"$kind,Nic"`
	ID    string `gae:"$id"`
	// fleet.Nic cannot be directly used as it contains pointer.
	Nic []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Nic.
func (e *NicEntity) GetProto() (proto.Message, error) {
	var p fleet.Nic
	if err := proto.Unmarshal(e.Nic, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newNicEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.Nic)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Nic ID").Err()
	}
	nic, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal Nic %s", p).Err()
	}
	return &NicEntity{
		ID:  p.GetName(),
		Nic: nic,
	}, nil
}

// CreateNic creates a new nic in datastore.
func CreateNic(ctx context.Context, nic *fleet.Nic) (*fleet.Nic, error) {
	return putNic(ctx, nic, false)
}

// UpdateNic updates nic in datastore.
func UpdateNic(ctx context.Context, nic *fleet.Nic) (*fleet.Nic, error) {
	return putNic(ctx, nic, true)
}

// GetNic returns nic for the given id from datastore.
func GetNic(ctx context.Context, id string) (*fleet.Nic, error) {
	pm, err := fleetds.Get(ctx, &fleet.Nic{Name: id}, newNicEntity)
	if err == nil {
		return pm.(*fleet.Nic), err
	}
	return nil, err
}

// ListNics lists the nics
//
// Does a query over Nic entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListNics(ctx context.Context, pageSize int32, pageToken string) (res []*fleet.Nic, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, NicKind, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *NicEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*fleet.Nic))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List Nics %s", err)
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

func putNic(ctx context.Context, nic *fleet.Nic, update bool) (*fleet.Nic, error) {
	nic.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, nic, newNicEntity, update)
	if err == nil {
		return pm.(*fleet.Nic), err
	}
	return nil, err
}
