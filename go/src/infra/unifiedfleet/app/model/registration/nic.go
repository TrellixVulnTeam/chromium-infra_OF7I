// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"

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

func putNic(ctx context.Context, nic *fleet.Nic, update bool) (*fleet.Nic, error) {
	nic.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, nic, newNicEntity, update)
	if err == nil {
		return pm.(*fleet.Nic), err
	}
	return nil, err
}
