// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package state

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/gae/service/datastore"

	chromeosLab "infra/unifiedfleet/api/v1/models/chromeos/lab"
	ufsds "infra/unifiedfleet/app/model/datastore"
)

// DutStateKind is the datastore entity kind of dut state.
//
// Dut state is only specific to OS devices for now.
const DutStateKind string = "DutState"

// DutStateEntity is a datastore entity that tracks dut state.
type DutStateEntity struct {
	_kind string `gae:"$kind,DutState"`
	// refer to the device id
	ID string `gae:"$id"`
	// lab.DutState cannot be directly used as it contains pointer (timestamp).
	DutState []byte `gae:",noindex"`
}

// GetProto returns the unmarshaled dut state.
func (e *DutStateEntity) GetProto() (proto.Message, error) {
	var p chromeosLab.DutState
	if err := proto.Unmarshal(e.DutState, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func newDutStateEntity(ctx context.Context, pm proto.Message) (ufsds.FleetEntity, error) {
	p := pm.(*chromeosLab.DutState)
	if p.GetId().GetValue() == "" {
		return nil, errors.Reason("Empty ID in Dut state").Err()
	}
	s, err := proto.Marshal(p)
	if err != nil {
		return nil, errors.Annotate(err, "fail to marshal DutState %s", p).Err()
	}
	return &DutStateEntity{
		ID:       p.GetId().GetValue(),
		DutState: s,
	}, nil
}

// GetDutState returns dut state for the given id from datastore.
func GetDutState(ctx context.Context, id string) (*chromeosLab.DutState, error) {
	pm, err := ufsds.Get(ctx, &chromeosLab.DutState{Id: &chromeosLab.ChromeOSDeviceID{Value: id}}, newDutStateEntity)
	if err == nil {
		return pm.(*chromeosLab.DutState), err
	}
	return nil, err
}

// UpdateDutStates updates dut states in datastore.
func UpdateDutStates(ctx context.Context, dutStates []*chromeosLab.DutState) ([]*chromeosLab.DutState, error) {
	protos := make([]proto.Message, len(dutStates))
	utime := ptypes.TimestampNow()
	for i, ds := range dutStates {
		ds.UpdateTime = utime
		protos[i] = ds
	}
	_, err := ufsds.PutAll(ctx, protos, newDutStateEntity, true)
	if err == nil {
		return dutStates, err
	}
	return nil, err
}

func queryAllDutStates(ctx context.Context) ([]ufsds.FleetEntity, error) {
	var entities []*DutStateEntity
	q := datastore.NewQuery(DutStateKind)
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	fe := make([]ufsds.FleetEntity, len(entities))
	for i, e := range entities {
		fe[i] = e
	}
	return fe, nil
}

// GetAllDutStates returns all dut states in datastore.
func GetAllDutStates(ctx context.Context) (*ufsds.OpResults, error) {
	return ufsds.GetAll(ctx, queryAllDutStates)
}

// DeleteDutStates deletes a batch of dut states
func DeleteDutStates(ctx context.Context, resourceNames []string) *ufsds.OpResults {
	protos := make([]proto.Message, len(resourceNames))
	for i, m := range resourceNames {
		protos[i] = &chromeosLab.DutState{
			Id: &chromeosLab.ChromeOSDeviceID{
				Value: m,
			},
		}
	}
	return ufsds.DeleteAll(ctx, protos, newDutStateEntity)
}

// ImportDutStates creates or updates a batch of dut states in datastore
func ImportDutStates(ctx context.Context, dutStates []*chromeosLab.DutState) (*ufsds.OpResults, error) {
	protos := make([]proto.Message, len(dutStates))
	utime := ptypes.TimestampNow()
	for i, m := range dutStates {
		if m.UpdateTime == nil {
			m.UpdateTime = utime
		}
		protos[i] = m
	}
	return ufsds.Insert(ctx, protos, newDutStateEntity, true, true)
}
