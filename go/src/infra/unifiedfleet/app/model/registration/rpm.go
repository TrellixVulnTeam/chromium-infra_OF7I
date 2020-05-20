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

// RPMKind is the datastore entity kind RPM.
const RPMKind string = "RPM"

// RPMEntity is a datastore entity that tracks RPM.
type RPMEntity struct {
	_kind string `gae:"$kind,RPM"`
	ID    string `gae:"$id"`
	// fleet.RPM cannot be directly used as it contains pointer.
	RPM []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled RPM.
func (e *RPMEntity) GetProto() (proto.Message, error) {
	var p fleet.RPM
	if err := proto.Unmarshal(e.RPM, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newRPMEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.RPM)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty RPM ID").Err()
	}
	rpm, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal RPM %s", p).Err()
	}
	return &RPMEntity{
		ID:  p.GetName(),
		RPM: rpm,
	}, nil
}

// CreateRPM creates a new RPM in datastore.
func CreateRPM(ctx context.Context, RPM *fleet.RPM) (*fleet.RPM, error) {
	return putRPM(ctx, RPM, false)
}

// UpdateRPM updates RPM in datastore.
func UpdateRPM(ctx context.Context, RPM *fleet.RPM) (*fleet.RPM, error) {
	return putRPM(ctx, RPM, true)
}

// GetRPM returns RPM for the given id from datastore.
func GetRPM(ctx context.Context, id string) (*fleet.RPM, error) {
	pm, err := fleetds.Get(ctx, &fleet.RPM{Name: id}, newRPMEntity)
	if err == nil {
		return pm.(*fleet.RPM), err
	}
	return nil, err
}

// ListRPMs lists the RPMs
//
// Does a query over RPM entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListRPMs(ctx context.Context, pageSize int32, pageToken string) (res []*fleet.RPM, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, RPMKind, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *RPMEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*fleet.RPM))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List RPMs %s", err)
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteRPM deletes the RPM in datastore
func DeleteRPM(ctx context.Context, id string) error {
	return fleetds.Delete(ctx, &fleet.RPM{Name: id}, newRPMEntity)
}

func putRPM(ctx context.Context, RPM *fleet.RPM, update bool) (*fleet.RPM, error) {
	RPM.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, RPM, newRPMEntity, update)
	if err == nil {
		return pm.(*fleet.RPM), err
	}
	return nil, err
}
