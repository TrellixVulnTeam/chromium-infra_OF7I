// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"

	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
)

// RackLSEKind is the datastore entity kind RackLSE.
const RackLSEKind string = "RackLSE"

// RackLSEEntity is a datastore entity that tracks RackLSE.
type RackLSEEntity struct {
	_kind              string   `gae:"$kind,RackLSE"`
	ID                 string   `gae:"$id"`
	RackIDs            []string `gae:"rack_ids"`
	RackLSEProtoTypeID string   `gae:"racklse_prototype_id"`
	KVMIDs             []string `gae:"kvm_ids"`
	RPMIDs             []string `gae:"rpm_ids"`
	// fleet.RackLSE cannot be directly used as it contains pointer.
	RackLSE []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled RackLSE.
func (e *RackLSEEntity) GetProto() (proto.Message, error) {
	var p fleet.RackLSE
	if err := proto.Unmarshal(e.RackLSE, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newRackLSEEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.RackLSE)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty RackLSE ID").Err()
	}
	rackLSE, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal RackLSE %s", p).Err()
	}
	return &RackLSEEntity{
		ID:                 p.GetName(),
		RackIDs:            p.GetRacks(),
		RackLSEProtoTypeID: p.GetRackLsePrototype(),
		KVMIDs:             p.GetChromeosRackLse().GetKvms(),
		RPMIDs:             p.GetChromeosRackLse().GetRpms(),
		RackLSE:            rackLSE,
	}, nil
}

// CreateRackLSE creates a new rackLSE in datastore.
func CreateRackLSE(ctx context.Context, rackLSE *fleet.RackLSE) (*fleet.RackLSE, error) {
	return putRackLSE(ctx, rackLSE, false)
}

// UpdateRackLSE updates rackLSE in datastore.
func UpdateRackLSE(ctx context.Context, rackLSE *fleet.RackLSE) (*fleet.RackLSE, error) {
	return putRackLSE(ctx, rackLSE, true)
}

func putRackLSE(ctx context.Context, rackLSE *fleet.RackLSE, update bool) (*fleet.RackLSE, error) {
	rackLSE.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, rackLSE, newRackLSEEntity, update)
	if err == nil {
		return pm.(*fleet.RackLSE), err
	}
	return nil, err
}
