// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package configuration

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

// RackLSEPrototypeKind is the datastore entity kind for chrome platforms.
const RackLSEPrototypeKind string = "RackLSEPrototype"

// RackLSEPrototypeEntity is a datastore entity that tracks a platform.
type RackLSEPrototypeEntity struct {
	_kind string `gae:"$kind,RackLSEPrototype"`
	ID    string `gae:"$id"`
	// fleet.RackLSEPrototype cannot be directly used as it contains pointer.
	RackLSEPrototype []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Chrome platform.
func (e *RackLSEPrototypeEntity) GetProto() (proto.Message, error) {
	var p fleet.RackLSEPrototype
	if err := proto.Unmarshal(e.RackLSEPrototype, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newRackLSEPrototypeEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.RackLSEPrototype)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Chrome Platform ID").Err()
	}
	rackLSEPrototype, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal RackLSEPrototype %s", p).Err()
	}
	return &RackLSEPrototypeEntity{
		ID:               p.GetName(),
		RackLSEPrototype: rackLSEPrototype,
	}, nil
}

// CreateRackLSEPrototype creates a new rackLSEPrototype in datastore.
func CreateRackLSEPrototype(ctx context.Context, rackLSEPrototype *fleet.RackLSEPrototype) (*fleet.RackLSEPrototype, error) {
	return putRackLSEPrototype(ctx, rackLSEPrototype, false)
}

// UpdateRackLSEPrototype updates rackLSEPrototype in datastore.
func UpdateRackLSEPrototype(ctx context.Context, rackLSEPrototype *fleet.RackLSEPrototype) (*fleet.RackLSEPrototype, error) {
	return putRackLSEPrototype(ctx, rackLSEPrototype, true)
}

// GetRackLSEPrototype returns rackLSEPrototype for the given id from datastore.
func GetRackLSEPrototype(ctx context.Context, id string) (*fleet.RackLSEPrototype, error) {
	pm, err := fleetds.Get(ctx, &fleet.RackLSEPrototype{Name: id}, newRackLSEPrototypeEntity)
	if err == nil {
		return pm.(*fleet.RackLSEPrototype), err
	}
	return nil, err
}

// ListRackLSEPrototypes lists the rackLSEPrototypes
//
// Does a query over RackLSEPrototype entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListRackLSEPrototypes(ctx context.Context, pageSize int32, pageToken string) (res []*fleet.RackLSEPrototype, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, RackLSEPrototypeKind, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *RackLSEPrototypeEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*fleet.RackLSEPrototype))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List RackLSEPrototypes %s", err)
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteRackLSEPrototype deletes the rackLSEPrototype in datastore
func DeleteRackLSEPrototype(ctx context.Context, id string) error {
	return fleetds.Delete(ctx, &fleet.RackLSEPrototype{Name: id}, newRackLSEPrototypeEntity)
}

func putRackLSEPrototype(ctx context.Context, rackLSEPrototype *fleet.RackLSEPrototype, update bool) (*fleet.RackLSEPrototype, error) {
	rackLSEPrototype.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, rackLSEPrototype, newRackLSEPrototypeEntity, update)
	if err == nil {
		return pm.(*fleet.RackLSEPrototype), err
	}
	return nil, err
}

// ImportRackLSEPrototypes creates or updates a batch of rack lse prototypes in datastore
func ImportRackLSEPrototypes(ctx context.Context, lps []*fleet.RackLSEPrototype) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(lps))
	utime := ptypes.TimestampNow()
	for i, m := range lps {
		m.UpdateTime = utime
		protos[i] = m
	}
	return fleetds.Insert(ctx, protos, newRackLSEPrototypeEntity, true, true)
}
