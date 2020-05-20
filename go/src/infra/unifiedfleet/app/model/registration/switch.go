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

// SwitchKind is the datastore entity kind Switch.
const SwitchKind string = "Switch"

// SwitchEntity is a datastore entity that tracks switch.
type SwitchEntity struct {
	_kind string `gae:"$kind,Switch"`
	ID    string `gae:"$id"`
	// fleet.Switch cannot be directly used as it contains pointer (timestamp).
	Switch []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled switch.
func (e *SwitchEntity) GetProto() (proto.Message, error) {
	var p fleet.Switch
	if err := proto.Unmarshal(e.Switch, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newSwitchEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.Switch)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Switch ID").Err()
	}
	s, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal Switch %s", p).Err()
	}
	return &SwitchEntity{
		ID:     p.GetName(),
		Switch: s,
	}, nil
}

// CreateSwitch creates a new switch in datastore.
func CreateSwitch(ctx context.Context, s *fleet.Switch) (*fleet.Switch, error) {
	return putSwitch(ctx, s, false)
}

// UpdateSwitch updates switch in datastore.
func UpdateSwitch(ctx context.Context, s *fleet.Switch) (*fleet.Switch, error) {
	return putSwitch(ctx, s, true)
}

// GetSwitch returns switch for the given id from datastore.
func GetSwitch(ctx context.Context, id string) (*fleet.Switch, error) {
	pm, err := fleetds.Get(ctx, &fleet.Switch{Name: id}, newSwitchEntity)
	if err == nil {
		return pm.(*fleet.Switch), err
	}
	return nil, err
}

// ListSwitches lists the switches
//
// Does a query over switch entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListSwitches(ctx context.Context, pageSize int32, pageToken string) (res []*fleet.Switch, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, SwitchKind, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *SwitchEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*fleet.Switch))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List Switches %s", err)
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteSwitch deletes the switch in datastore
func DeleteSwitch(ctx context.Context, id string) error {
	return fleetds.Delete(ctx, &fleet.Switch{Name: id}, newSwitchEntity)
}

func putSwitch(ctx context.Context, s *fleet.Switch, update bool) (*fleet.Switch, error) {
	s.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, s, newSwitchEntity, update)
	if err == nil {
		return pm.(*fleet.Switch), err
	}
	return nil, err
}

// ImportSwitches creates or updates a batch of switches in datastore
func ImportSwitches(ctx context.Context, switches []*fleet.Switch) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(switches))
	utime := ptypes.TimestampNow()
	for i, m := range switches {
		m.UpdateTime = utime
		protos[i] = m
	}
	return fleetds.Insert(ctx, protos, newSwitchEntity, true, true)
}
