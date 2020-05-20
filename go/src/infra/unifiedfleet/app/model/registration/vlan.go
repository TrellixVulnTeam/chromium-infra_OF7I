// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package registration

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"infra/unifiedfleet/app/model/inventory"

	fleet "infra/unifiedfleet/api/v1/proto"
	fleetds "infra/unifiedfleet/app/model/datastore"
)

// VlanKind is the datastore entity kind Vlan.
const VlanKind string = "Vlan"

// VlanEntity is a datastore entity that tvlans Vlan.
type VlanEntity struct {
	_kind string `gae:"$kind,Vlan"`
	ID    string `gae:"$id"`
	// fleet.Vlan cannot be directly used as it contains pointer.
	Vlan []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled Vlan.
func (e *VlanEntity) GetProto() (proto.Message, error) {
	var p fleet.Vlan
	if err := proto.Unmarshal(e.Vlan, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newVlanEntity(ctx context.Context, pm proto.Message) (fleetds.FleetEntity, error) {
	p := pm.(*fleet.Vlan)
	if p.GetName() == "" {
		return nil, errors.Reason("Empty Vlan ID").Err()
	}
	vlan, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal Vlan %s", p).Err()
	}
	return &VlanEntity{
		ID:   p.GetName(),
		Vlan: vlan,
	}, nil
}

// CreateVlan creates a new vlan in datastore.
func CreateVlan(ctx context.Context, vlan *fleet.Vlan) (*fleet.Vlan, error) {
	return putVlan(ctx, vlan, false)
}

// UpdateVlan updates vlan in datastore.
func UpdateVlan(ctx context.Context, vlan *fleet.Vlan) (*fleet.Vlan, error) {
	return putVlan(ctx, vlan, true)
}

// GetVlan returns vlan for the given id from datastore.
func GetVlan(ctx context.Context, id string) (*fleet.Vlan, error) {
	pm, err := fleetds.Get(ctx, &fleet.Vlan{Name: id}, newVlanEntity)
	if err == nil {
		return pm.(*fleet.Vlan), err
	}
	return nil, err
}

// ListVlans lists the vlans
//
// Does a query over Vlan entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListVlans(ctx context.Context, pageSize int32, pageToken string) (res []*fleet.Vlan, nextPageToken string, err error) {
	q, err := fleetds.ListQuery(ctx, VlanKind, pageSize, pageToken)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *VlanEntity, cb datastore.CursorCB) error {
		pm, err := ent.GetProto()
		if err != nil {
			logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
			return nil
		}
		res = append(res, pm.(*fleet.Vlan))
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to List Vlans %s", err)
		return nil, "", status.Errorf(codes.Internal, fleetds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
}

// DeleteVlan deletes the vlan in datastore
//
// For referential data intergrity,
// Delete if there are no references to the Vlan by MachineLSE in the datastore.
// If there are any references, delete will be rejected and an error message will be thrown.
func DeleteVlan(ctx context.Context, id string) error {
	machinelses, err := inventory.QueryMachineLSEByPropertyName(ctx, "vlan_id", id, true)
	if err != nil {
		return err
	}
	if len(machinelses) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("Vlan %s cannot be deleted because there are other resources which are referring this Vlan.", id))
		errorMsg.WriteString(fmt.Sprintf("\nMachineLSEs referring the Vlan:\n"))
		for _, machinelse := range machinelses {
			errorMsg.WriteString(machinelse.Name + ", ")
		}
		logging.Errorf(ctx, errorMsg.String())
		return status.Errorf(codes.FailedPrecondition, errorMsg.String())
	}
	return fleetds.Delete(ctx, &fleet.Vlan{Name: id}, newVlanEntity)
}

func putVlan(ctx context.Context, vlan *fleet.Vlan, update bool) (*fleet.Vlan, error) {
	vlan.UpdateTime = ptypes.TimestampNow()
	pm, err := fleetds.Put(ctx, vlan, newVlanEntity, update)
	if err == nil {
		return pm.(*fleet.Vlan), err
	}
	return nil, err
}

// ImportVlans creates or updates a batch of vlan in datastore
func ImportVlans(ctx context.Context, vlans []*fleet.Vlan) (*fleetds.OpResults, error) {
	protos := make([]proto.Message, len(vlans))
	utime := ptypes.TimestampNow()
	for i, m := range vlans {
		m.UpdateTime = utime
		protos[i] = m
	}
	return fleetds.Insert(ctx, protos, newVlanEntity, true, true)
}
