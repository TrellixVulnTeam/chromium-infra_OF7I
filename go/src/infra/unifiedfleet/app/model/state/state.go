// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package state

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ufspb "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
)

// RecordKind is the datastore entity kind of state.
const RecordKind string = "State"

// RecordEntity is a datastore entity that tracks dhcp.
type RecordEntity struct {
	_kind string `gae:"$kind,State"`
	// refer to the hostname
	ResourceName string `gae:"$id"`
	State        string `gae:"state"`
	// ufspb.StateRecord cannot be directly used as it contains pointer (timestamp).
	StateRecord []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled DHCP.
func (e *RecordEntity) GetProto() (proto.Message, error) {
	var p ufspb.StateRecord
	if err := proto.Unmarshal(e.StateRecord, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newRecordEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*ufspb.StateRecord)
	if p.GetResourceName() == "" {
		return nil, errors.Reason("Empty resource name in state record").Err()
	}
	s, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal DHCPConfig %s", p).Err()
	}
	return &RecordEntity{
		ResourceName: p.GetResourceName(),
		State:        p.GetState().String(),
		StateRecord:  s,
	}, nil
}

// GetStateRecord returns the state for a given resource name.
func GetStateRecord(ctx context.Context, id string) (*ufspb.StateRecord, error) {
	pm, err := fleetds.Get(ctx, &ufspb.StateRecord{ResourceName: id}, newRecordEntity)
	if err == nil {
		return pm.(*ufspb.StateRecord), err
	}
	return nil, err
}

// UpdateStateRecord updates a state record in datastore.
func UpdateStateRecord(ctx context.Context, stateRecord *ufspb.StateRecord) (*ufspb.StateRecord, error) {
	stateRecord.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.PutSingle(ctx, stateRecord, newRecordEntity)
	if err == nil {
		return pm.(*ufspb.StateRecord), err
	}
	return nil, err
}

// ListStateRecords lists all the states
func ListStateRecords(ctx context.Context, pageSize int32, pageToken string) (res []*ufspb.StateRecord, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, RecordKind, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *RecordEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*ufspb.StateRecord))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List state records %s", err)
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// ImportStateRecords creates or updates a batch of state records in datastore
func ImportStateRecords(ctx context.Context, states []*ufspb.StateRecord) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(states))
	utime := ptypes.TimestampNow()
	for i, m := range states {
		m.UpdateTime = utime
		protos[i] = m
	}
	return fleetds.Insert(ctx, protos, newRecordEntity, true, true)
}
