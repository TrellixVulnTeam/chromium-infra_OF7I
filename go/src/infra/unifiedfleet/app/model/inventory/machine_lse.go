// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package inventory

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"

	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
)

// MachineLSEKind is the datastore entity kind MachineLSE.
const MachineLSEKind string = "MachineLSE"

// MachineLSEEntity is a datastore entity that tracks MachineLSE.
type MachineLSEEntity struct {
	_kind                 string   `gae:"$kind,MachineLSE"`
	ID                    string   `gae:"$id"`
	MachineIDs            []string `gae:"machine_ids"`
	MachineLSEProtoTypeID string   `gae:"machinelse_prototype_id"`
	SwitchID              string   `gae:"switch_id"`
	RPMID                 string   `gae:"rpm_id"`
	VlanID                string   `gae:"vlan_id"`
	// fleet.MachineLSE cannot be directly used as it contains pointer.
	MachineLSE []byte         `gae:",noindex"`
	Parent     *datastore.Key `gae:"$parent"`
}

// GetProto returns the unmarshaled MachineLSE.
func (e *MachineLSEEntity) GetProto() (proto.Message, error) {
	var p fleet.MachineLSE
	if err := proto.Unmarshal(e.MachineLSE, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newMachineLSEEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.MachineLSE)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty MachineLSE ID").Err()
	}
	machineLSE, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal MachineLSE %s", p).Err()
	}
	return &MachineLSEEntity{
		ID:                    p.GetName(),
		MachineIDs:            p.GetMachines(),
		MachineLSEProtoTypeID: p.GetMachineLsePrototype(),
		SwitchID:              p.GetChromeosMachineLse().GetDut().GetNetworkDeviceInterface().GetSwitch(),
		RPMID:                 p.GetChromeosMachineLse().GetDut().GetRpmInterface().GetRpm(),
		VlanID:                p.GetChromeosMachineLse().GetServer().GetSupportedRestrictedVlan(),
		MachineLSE:            machineLSE,
		Parent:                fleetds.FakeAncestorKey(ctx, MachineLSEKind),
	}, nil
}

// CreateMachineLSE creates a new machineLSE in datastore.
func CreateMachineLSE(ctx context.Context, machineLSE *fleet.MachineLSE) (*fleet.MachineLSE, error) {
	return putMachineLSE(ctx, machineLSE, false)
}

// UpdateMachineLSE updates machineLSE in datastore.
func UpdateMachineLSE(ctx context.Context, machineLSE *fleet.MachineLSE) (*fleet.MachineLSE, error) {
	return putMachineLSE(ctx, machineLSE, true)
}

func putMachineLSE(ctx context.Context, machineLSE *fleet.MachineLSE, update bool) (*fleet.MachineLSE, error) {
	machineLSE.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, machineLSE, newMachineLSEEntity, update)
	if err == nil {
		return pm.(*fleet.MachineLSE), err
	}
	return nil, err
}
