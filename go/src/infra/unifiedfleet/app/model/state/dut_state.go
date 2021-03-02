// Copyright 2020 The Chromium OS Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package state

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	"go.chromium.org/luci/gae/service/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	ID       string `gae:"$id"`
	Hostname string `gae:"hostname"`
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
		Hostname: p.GetHostname(),
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

// ListAllDutStates returns all DutState in datastore.
func ListAllDutStates(ctx context.Context, keysOnly bool) (res []*chromeosLab.DutState, err error) {
	var entities []*DutStateEntity
	q := datastore.NewQuery(DutStateKind).KeysOnly(keysOnly).FirestoreMode(true)
	if err = datastore.GetAll(ctx, q, &entities); err != nil {
		return nil, err
	}
	for _, ent := range entities {
		if keysOnly {
			res = append(res, &chromeosLab.DutState{
				Id: &chromeosLab.ChromeOSDeviceID{Value: ent.ID},
			})
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil, err
			}
			dutState := pm.(*chromeosLab.DutState)
			res = append(res, dutState)
		}
	}
	return
}

// QueryDutStateByPropertyNames queries DutState Entity in the datastore.
// If keysOnly is true, then only key field is populated in returned DutStates.
func QueryDutStateByPropertyNames(ctx context.Context, propertyMap map[string]string, keysOnly bool) ([]*chromeosLab.DutState, error) {
	q := datastore.NewQuery(DutStateKind).KeysOnly(keysOnly).FirestoreMode(true)
	var entities []*DutStateEntity
	for propertyName, id := range propertyMap {
		q = q.Eq(propertyName, id)
	}
	if err := datastore.GetAll(ctx, q, &entities); err != nil {
		logging.Errorf(ctx, "Failed to query from datastore: %s", err)
		return nil, status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if len(entities) == 0 {
		logging.Infof(ctx, "No DutStates found for the query: %s", q.String())
		return nil, nil
	}
	dutStates := make([]*chromeosLab.DutState, 0, len(entities))
	for _, entity := range entities {
		if keysOnly {
			dutState := &chromeosLab.DutState{
				Id: &chromeosLab.ChromeOSDeviceID{Value: entity.ID},
			}
			dutStates = append(dutStates, dutState)
		} else {
			pm, perr := entity.GetProto()
			if perr != nil {
				logging.Errorf(ctx, "Failed to unmarshal proto: %s", perr)
				continue
			}
			dutStates = append(dutStates, pm.(*chromeosLab.DutState))
		}
	}
	return dutStates, nil
}

// ListDutStates lists the DutStates.
//
// Does a query over DutState entities. Returns up to pageSize entities, plus non-nil cursor (if
// there are more results). pageSize must be positive.
func ListDutStates(ctx context.Context, pageSize int32, pageToken string, filterMap map[string][]interface{}, keysOnly bool) (res []*chromeosLab.DutState, nextPageToken string, err error) {
	q, err := ufsds.ListQuery(ctx, DutStateKind, pageSize, pageToken, filterMap, keysOnly)
	if err != nil {
		return nil, "", err
	}
	var nextCur datastore.Cursor
	err = datastore.Run(ctx, q, func(ent *DutStateEntity, cb datastore.CursorCB) error {
		if keysOnly {
			DutState := &chromeosLab.DutState{
				Id: &chromeosLab.ChromeOSDeviceID{Value: ent.ID},
			}
			res = append(res, DutState)
		} else {
			pm, err := ent.GetProto()
			if err != nil {
				logging.Errorf(ctx, "Failed to UnMarshal: %s", err)
				return nil
			}
			res = append(res, pm.(*chromeosLab.DutState))
		}
		if len(res) >= int(pageSize) {
			if nextCur, err = cb(); err != nil {
				return err
			}
			return datastore.Stop
		}
		return nil
	})
	if err != nil {
		logging.Errorf(ctx, "Failed to list DutStates %s", err)
		return nil, "", status.Errorf(codes.Internal, ufsds.InternalError)
	}
	if nextCur != nil {
		nextPageToken = nextCur.String()
	}
	return
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
